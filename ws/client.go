package ws

import (
	"context"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// RetryConfig controls the reconnection behavior.
type RetryConfig struct {
	// MaxRetries is the maximum number of reconnection attempts.
	// 0 means retry forever.
	MaxRetries int
	// InitialBackoff is the delay before the first retry.
	InitialBackoff time.Duration
	// MaxBackoff caps the maximum delay between retries.
	MaxBackoff time.Duration
	// Multiplier is the exponential backoff multiplier (typically 2).
	Multiplier float64
	// JitterFraction adds randomness to avoid thundering herd (0.0 ~ 1.0).
	JitterFraction float64
	// DialTimeout is the timeout for a single dial attempt.
	DialTimeout time.Duration
}

// DefaultRetryConfig returns production-ready retry defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     0, // retry forever
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     30 * time.Second,
		Multiplier:     2.0,
		JitterFraction: 0.25,
		DialTimeout:    10 * time.Second,
	}
}

// ConnState represents the lifecycle state of a client connection.
type ConnState int

const (
	ConnStateDisconnected ConnState = iota
	ConnStateConnecting
	ConnStateConnected
	ConnStateReconnecting
	ConnStateClosed
)

func (s ConnState) String() string {
	switch s {
	case ConnStateDisconnected:
		return "disconnected"
	case ConnStateConnecting:
		return "connecting"
	case ConnStateConnected:
		return "connected"
	case ConnStateReconnecting:
		return "reconnecting"
	case ConnStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// StateChangeFunc is called when the connection state transitions.
type StateChangeFunc func(from, to ConnState)

// MessageHandler is invoked for each received message.
type MessageHandler func(ctx context.Context, data []byte)

// ClientOptions holds all options for creating a resilient WebSocket client.
type ClientOptions struct {
	// URL is the WebSocket endpoint URL (ws:// or wss://).
	URL string
	// Headers are additional HTTP headers sent during the handshake.
	Headers http.Header
	// Retry controls reconnection behavior.
	Retry RetryConfig
	// ConnConfig is the underlying connection configuration.
	ConnConfig ConnConfig
	// Logger for structured logging.
	Logger *slog.Logger
	// OnStateChange is called on connection state transitions.
	OnStateChange StateChangeFunc
	// OnMessage handles incoming messages (used by server-stream and bidi-stream).
	OnMessage MessageHandler
	// OutgoingMessages is a channel the caller feeds messages into
	// (used by client-stream and bidi-stream). The client reads from
	// this channel and sends to the server.
	OutgoingMessages <-chan []byte
}

// Client is a production-grade WebSocket client with automatic reconnection.
// It supports all three streaming patterns:
//   - Client-only streaming: set OutgoingMessages, leave OnMessage nil
//   - Server-only streaming: set OnMessage, leave OutgoingMessages nil
//   - Bidirectional: set both OutgoingMessages and OnMessage
type Client struct {
	opts  ClientOptions
	state atomic.Int32

	// closed signals the retry loop to stop permanently.
	closed chan struct{}
	// reconnectCh triggers an immediate reconnect (used internally).
	reconnectCh chan struct{}
}

// NewClient creates a new resilient WebSocket client.
func NewClient(opts ClientOptions) *Client {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Retry.InitialBackoff == 0 {
		opts.Retry = DefaultRetryConfig()
	}
	c := &Client{
		opts:        opts,
		closed:      make(chan struct{}),
		reconnectCh: make(chan struct{}, 1),
	}
	c.state.Store(int32(ConnStateDisconnected))
	return c
}

// Run starts the client with automatic reconnection.
// It blocks until the context is cancelled or Close() is called.
func (c *Client) Run(ctx context.Context) {
	var attempt int
	for {
		select {
		case <-ctx.Done():
			c.setState(ConnStateClosed)
			return
		case <-c.closed:
			c.setState(ConnStateClosed)
			return
		default:
		}

		c.runOnceWithRetry(ctx, &attempt)

		// Check if we should stop before next attempt.
		select {
		case <-ctx.Done():
			c.setState(ConnStateClosed)
			return
		case <-c.closed:
			c.setState(ConnStateClosed)
			return
		default:
		}
	}
}

// Close permanently stops the client. No further reconnection attempts will be made.
func (c *Client) Close() {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
}

// Reconnect triggers an immediate reconnection (disconnect + reconnect).
func (c *Client) Reconnect() {
	select {
	case c.reconnectCh <- struct{}{}:
	default:
	}
}

// State returns the current connection state.
func (c *Client) State() ConnState {
	return ConnState(c.state.Load())
}

// runOnceWithRetry wraps runOnce with retry counting and backoff.
func (c *Client) runOnceWithRetry(parentCtx context.Context, attempt *int) {
	c.setState(ConnStateConnecting)

	dialCtx, dialCancel := context.WithTimeout(parentCtx, c.opts.Retry.DialTimeout)
	wsOpts := &websocket.DialOptions{
		HTTPHeader:      c.opts.Headers,
		CompressionMode: websocket.CompressionContextTakeover,
	}
	wsConn, _, err := websocket.Dial(dialCtx, c.opts.URL, wsOpts)
	dialCancel()
	if err != nil {
		c.opts.Logger.Warn("dial failed, will retry",
			slog.String("url", c.opts.URL),
			slog.String("error", err.Error()),
			slog.Int("attempt", *attempt+1),
		)
		c.backoffWait(parentCtx, *attempt)
		*attempt++
		return
	}

	// Reset attempt counter on successful connection.
	*attempt = 0
	c.setState(ConnStateConnected)
	c.opts.Logger.Info("websocket connected", slog.String("url", c.opts.URL))

	connCfg := c.opts.ConnConfig
	if connCfg.WriteBufferSize == 0 {
		connCfg = DefaultConnConfig()
	}
	conn := NewConn(wsConn, connCfg, c.opts.Logger)

	ctx, cancel := context.WithCancel(parentCtx)
	go conn.Start(ctx)

	connErr := c.messageLoop(ctx, conn)

	cancel()
	conn.Close()

	select {
	case <-c.closed:
		return
	case <-parentCtx.Done():
		return
	default:
		c.opts.Logger.Warn("connection lost, will retry",
			slog.String("url", c.opts.URL),
			slog.String("error", errStr(connErr)),
		)
		c.backoffWait(parentCtx, *attempt)
		*attempt++
	}
}

func errStr(err error) string {
	if err == nil {
		return "none"
	}
	return err.Error()
}

// messageLoop runs the appropriate read/write pumps based on the stream type.
func (c *Client) messageLoop(ctx context.Context, conn *Conn) error {
	errCh := make(chan error, 2)

	// Read pump (server-stream or bidi-stream).
	if c.opts.OnMessage != nil {
		go func() {
			for {
				data, err := conn.Read(ctx)
				if err != nil {
					errCh <- err
					return
				}
				c.opts.OnMessage(ctx, data)
			}
		}()
	}

	// Write pump (client-stream or bidi-stream).
	if c.opts.OutgoingMessages != nil {
		go func() {
			for {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				case msg, ok := <-c.opts.OutgoingMessages:
					if !ok {
						return
					}
					conn.Send(msg)
				}
			}
		}()
	} else if c.opts.OnMessage == nil {
		<-ctx.Done()
		return ctx.Err()
	}

	// For client-only stream (no OnMessage), still need a read pump to
	// detect server-side disconnect.
	if c.opts.OnMessage == nil {
		go func() {
			for {
				_, err := conn.Read(ctx)
				if err != nil {
					errCh <- err
					return
				}
			}
		}()
	}

	// Watch for external reconnect trigger.
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-c.reconnectCh:
			conn.Close()
			errCh <- nil
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) setState(newState ConnState) {
	old := ConnState(c.state.Swap(int32(newState)))
	if old != newState && c.opts.OnStateChange != nil {
		c.opts.OnStateChange(old, newState)
	}
}

// backoffWait implements exponential backoff with jitter and retry limit.
func (c *Client) backoffWait(ctx context.Context, attempt int) {
	if c.opts.Retry.MaxRetries > 0 && attempt >= c.opts.Retry.MaxRetries {
		c.opts.Logger.Error("max retries exceeded",
			slog.Int("max_retries", c.opts.Retry.MaxRetries),
		)
		c.Close()
		return
	}

	delay := c.calculateBackoff(attempt)
	c.setState(ConnStateReconnecting)
	c.opts.Logger.Info("backoff waiting",
		slog.Int("attempt", attempt+1),
		slog.Duration("delay", delay),
	)

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-c.closed:
	case <-timer.C:
	}
}

func (c *Client) calculateBackoff(attempt int) time.Duration {
	r := c.opts.Retry
	backoff := float64(r.InitialBackoff) * math.Pow(r.Multiplier, float64(attempt))
	if backoff > float64(r.MaxBackoff) {
		backoff = float64(r.MaxBackoff)
	}
	jitter := backoff * r.JitterFraction
	backoff = backoff - jitter + rand.Float64()*2*jitter
	return time.Duration(backoff)
}
