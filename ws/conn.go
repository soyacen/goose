package ws

import (
	"context"
	"log/slog"
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
	ws        *websocket.Conn
	cfg       ConnConfig
	logger    *slog.Logger

	writeCh   chan []byte
	closeOnce sync.Once
	closeErr  error
	sendDone  chan struct{} // closed by CloseSend to signal send-side is done
}

// NewConn wraps a raw websocket.Conn into a managed Conn.
// The caller must call Start() to begin the write pump and ping loop.
func NewConn(ws *websocket.Conn, cfg ConnConfig, logger *slog.Logger) *Conn {
	if logger == nil {
		logger = slog.Default()
	}
	return &Conn{
		ws:       ws,
		cfg:      cfg,
		logger:   logger,
		writeCh:  make(chan []byte, cfg.WriteBufferSize),
		sendDone: make(chan struct{}),
	}
}

// Start launches the background write pump and ping loop.
// It blocks until all pending writes are drained and the write pump exits.
//
// The write pump uses an internal context (independent of ctx) so that
// cancelling ctx does not abort in-flight writes. The write pump exits when:
//   - ctx is cancelled,
//   - Close() is called (writeCh closed), or
//   - CloseSend() is called (send-side done).
//
// After the write pump exits, pending writes are drained synchronously.
// The WebSocket itself is NOT closed here; the caller (or the remote peer)
// is responsible for closing the connection.
func (c *Conn) Start(ctx context.Context) {
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
	//   - Send-side closed (CloseSend called)
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
	// then send a WebSocket close frame to signal the peer.
	drainCtx, drainCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer drainCancel()
	c.drainWrites(drainCtx)
	_ = c.ws.Close(websocket.StatusNormalClosure, "connection closed")
}

func (c *Conn) drainAndClose() {
	drainCtx, drainCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer drainCancel()
	c.drainWrites(drainCtx)
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
// Unlike Close, it does not close the write channel immediately.
// The write pump drains pending writes and then closes the WebSocket.
// The read side remains open, allowing the caller to still receive messages.
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
