package main

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
type Conn struct {
	ws     *websocket.Conn
	cfg    ConnConfig
	logger *slog.Logger

	writeCh chan []byte
	closeOnce sync.Once
	closeErr  error
}

// NewConn wraps a raw websocket.Conn into a managed Conn.
// The caller must call Start() to begin the write pump and ping loop.
func NewConn(ws *websocket.Conn, cfg ConnConfig, logger *slog.Logger) *Conn {
	if logger == nil {
		logger = slog.Default()
	}
	return &Conn{
		ws:      ws,
		cfg:     cfg,
		logger:  logger,
		writeCh: make(chan []byte, cfg.WriteBufferSize),
	}
}

// Start launches the background write pump and ping loop.
// It blocks until the connection is closed.
func (c *Conn) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := make(chan struct{})
	go func() {
		c.writePump(ctx)
		close(done)
	}()

	go c.pingLoop(ctx)

	// Wait for context cancellation or write pump exit.
	select {
	case <-ctx.Done():
	case <-done:
	}

	// Drain remaining writes with a short timeout.
	drainCtx, drainCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer drainCancel()
	c.drainWrites(drainCtx)

	_ = c.ws.Close(websocket.StatusNormalClosure, "connection closed")
}

// Send enqueues a message for asynchronous writing.
// Returns false if the write buffer is full or the connection is closing.
func (c *Conn) Send(data []byte) bool {
	select {
	case c.writeCh <- data:
		return true
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

func (c *Conn) writePump(ctx context.Context) {
	for msg := range c.writeCh {
		writeCtx, cancel := context.WithTimeout(ctx, c.cfg.WriteTimeout)
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
			pingCtx, cancel := context.WithTimeout(ctx, c.cfg.WriteTimeout)
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
