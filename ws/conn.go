package ws

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// ConnConfig holds the configuration for a WebSocket connection.
type ConnConfig struct {
	// MaxReadBytes limits the maximum size of a single read message.
	MaxReadBytes int64
	// WriteBufferSize is the channel buffer size for outgoing messages.
	WriteBufferSize int
	// PingInterval is the interval between ping frames.
	PingInterval time.Duration
	// WriteTimeout is the timeout for a single write operation.
	WriteTimeout time.Duration
}

// DefaultConnConfig returns a ConnConfig with sensible production defaults.
func DefaultConnConfig() ConnConfig {
	return ConnConfig{
		MaxReadBytes:    1 << 20, // 1 MB
		WriteBufferSize: 256,
		PingInterval:    30 * time.Second,
		WriteTimeout:    10 * time.Second,
	}
}

// Conn wraps a github.com/coder/websocket.Conn with production-grade features:
//   - Buffered asynchronous writes (non-blocking send)
//   - Periodic ping/pong keepalive
//   - Graceful close with drain
//   - Context-based lifecycle
//   - Separate send-side close (CloseSend) for client-streaming scenarios
type Conn struct {
	ws     *websocket.Conn
	cfg    ConnConfig
	logger *slog.Logger

	writeCh   chan []byte
	closeOnce sync.Once
	closeErr  error
	sendDone  chan struct{} // closed by CloseSend to signal send-side is done
	startDone chan struct{} // closed when Start() returns
}

// AcceptConn upgrades the HTTP connection to WebSocket and starts the read loop.
func AcceptConn(response http.ResponseWriter, request *http.Request, cfg ConnConfig, logger *slog.Logger) (ctx context.Context, conn *Conn, cancel context.CancelFunc, err error) {
	ctx = request.Context()
	wsConn, err := websocket.Accept(response, request, AcceptOptions())
	if err != nil {
		return ctx, nil, func() {}, err
	}
	conn = NewConn(wsConn, cfg, logger)
	ctx, cancel = context.WithCancel(ctx)
	go conn.Start(context.Background())
	return ctx, conn, cancel, nil
}

// NewConn wraps a raw websocket.Conn into a managed Conn.
// The caller must call Start() to begin the write pump and ping loop.
func NewConn(ws *websocket.Conn, cfg ConnConfig, logger *slog.Logger) *Conn {
	if logger == nil {
		logger = slog.Default()
	}
	return &Conn{
		ws:        ws,
		cfg:       cfg,
		logger:    logger,
		writeCh:   make(chan []byte, cfg.WriteBufferSize),
		sendDone:  make(chan struct{}),
		startDone: make(chan struct{}),
	}
}

// Start launches the background write pump and ping loop.
// It blocks until the write pump exits and cleanup is complete.
//
// The write pump uses an internal context (independent of ctx) so that
// cancelling ctx does not abort in-flight writes. Start exits when:
//   - ctx is cancelled,
//   - Close() is called (writeCh closed), or
//   - CloseSend() is called (send-side done).
//
// On ctx cancellation or CloseSend, Start always performs the following
// before returning:
//  1. Cancels the internal writeCtx to stop writePump and pingLoop.
//  2. Drains any remaining buffered writes using a fresh 2-second timeout.
//  3. Writes an end-of-stream (EOS) marker — an empty text frame — to
//     signal the peer that no more messages will be sent.
//
// On Close() (writeCh closed), the writePump exits naturally and Start
// returns immediately without drain/EOS, since Close() implies the caller
// has already finished enqueueing messages.
//
// The WebSocket itself is NOT closed here; the peer is expected to close
// the connection after processing the EOS marker.
func (c *Conn) Start(ctx context.Context) {
	defer close(c.startDone)

	writeCtx, writeCancel := context.WithCancel(context.Background())

	writePumpDone := make(chan struct{})
	go func() {
		c.writePump(writeCtx)
		close(writePumpDone)
	}()

	pingDone := make(chan struct{})
	go func() {
		c.pingLoop(writeCtx)
		close(pingDone)
	}()

	// Wait for one of:
	//   - Parent context cancelled (e.g. HTTP request done)
	//   - Write pump exited (Close was called)
	//   - Send-side closed (CloseSend called, either by client or by drainAndClose)
	select {
	case <-ctx.Done():
	case <-writePumpDone:
		writeCancel()
		<-pingDone
		return
	case <-c.sendDone:
	}

	// Cancel writeCtx to stop the write pump and ping loop,
	// then wait for both to fully exit before draining.
	// This ensures no concurrent writes on the underlying WebSocket.
	writeCancel()
	<-writePumpDone
	<-pingDone

	// Drain remaining writes synchronously using a fresh context,
	// then write an empty message as an end-of-stream (EOS) marker.
	// The peer interprets this as "no more messages from this side"
	// (equivalent to gRPC's CloseSend / half-close).
	drainCtx, drainCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer drainCancel()
	c.drainWrites(drainCtx)
	_ = c.ws.Write(drainCtx, websocket.MessageText, nil)
}

func (c *Conn) drainAndClose() {
	// Signal Start() to stop the writePump and pingLoop.
	c.CloseSend()

	// Wait for Start() to finish stopping the writePump, pingLoop,
	// and draining writes. This ensures no concurrent writes on the
	// underlying WebSocket when we call ws.Close().
	<-c.startDone

	// Close the WebSocket.
	_ = c.ws.Close(websocket.StatusNormalClosure, "connection closed")
}

// drainAndClose drains pending writes and closes the WebSocket.
// Used by ServerStream.CloseSend to gracefully terminate the connection
// after sending the final response.
func (c *Conn) DrainAndClose() {
	c.drainAndClose()
}

// Send enqueues a message for asynchronous writing.
// Returns false if the write buffer is full or the connection is closing.
func (c *Conn) Send(data []byte) bool {
	select {
	case c.writeCh <- data:
		return true
	case <-c.sendDone:
		return false
	default:
		c.logger.Warn("write buffer full, dropping message")
		return false
	}
}

// Read reads a single message from the connection.
// It respects MaxReadBytes and the provided context.
func (c *Conn) Read(ctx context.Context) ([]byte, error) {
	c.ws.SetReadLimit(c.cfg.MaxReadBytes)
	_, data, err := c.ws.Read(ctx)
	return data, err
}

// Close initiates a graceful close of the connection.
func (c *Conn) Close() {
	c.closeOnce.Do(func() {
		close(c.writeCh)
	})
}

// CloseSend signals that no more messages will be sent on this connection.
// Unlike Close, it does not close the write channel or the WebSocket.
// Instead, it triggers the Start() goroutine to drain pending writes and
// write an end-of-stream (EOS) marker (an empty text frame) to the peer.
// The read side remains open, allowing the caller to still receive messages.
//
// Graceful shutdown: server-side handlers call stream.CloseSend() (which
// delegates here) before returning, ensuring all buffered messages are
// flushed and the EOS marker is written. The generated handler code uses
// context.Background() for Start() precisely so that HTTP request context
// cancellation (e.g. during server shutdown) does not abort the drain/EOS
// sequence — the connection lifecycle is fully controlled by CloseSend.
func (c *Conn) CloseSend() {
	select {
	case <-c.sendDone:
		// already closed
	default:
		close(c.sendDone)
	}
}

func (c *Conn) writePump(ctx context.Context) {
	for {
		// Wait for next message or cancellation.
		var msg []byte
		var ok bool
		select {
		case msg, ok = <-c.writeCh:
			if !ok {
				return
			}
		case <-ctx.Done():
			return
		}

		// Write using an independent timeout context so that cancelling ctx
		// does not abort an in-flight write.
		writeCtx, cancel := context.WithTimeout(context.Background(), c.cfg.WriteTimeout)
		err := c.ws.Write(writeCtx, websocket.MessageText, msg)
		cancel()
		if err != nil {
			c.logger.Error("write error", slog.String("error", err.Error()))
			c.Close()
			return
		}
	}
}

func (c *Conn) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(context.Background(), c.cfg.WriteTimeout)
			err := c.ws.Ping(pingCtx)
			cancel()
			if err != nil {
				c.logger.Error("ping error", slog.String("error", err.Error()))
				c.Close()
				return
			}
		}
	}
}

func (c *Conn) drainWrites(ctx context.Context) {
	for {
		select {
		case msg, ok := <-c.writeCh:
			if !ok {
				return
			}
			_ = c.ws.Write(ctx, websocket.MessageText, msg)
		default:
			return
		}
	}
}
