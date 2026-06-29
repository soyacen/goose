package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
	"github.com/coder/websocket"
)

// acceptOptions returns the websocket.AcceptOptions for CORS and subprotocol.
func acceptOptions() *websocket.AcceptOptions {
	return &websocket.AcceptOptions{
		InsecureSkipVerify: false,
	}
}

// ------------------------------------------------------------------
// 1. Client-Stream: client sends, server only receives
// ------------------------------------------------------------------

// ClientStreamHandler handles WebSocket connections where only the client
// sends messages to the server (unidirectional: client -> server).
// Typical use cases: log ingestion, telemetry upload, event reporting.
type ClientStreamHandler struct {
	cfg     ConnConfig
	logger  *slog.Logger
	active  atomic.Int64
	maxConn int64
}

// NewClientStreamHandler creates a new ClientStreamHandler.
// maxConn limits the number of concurrent connections (0 = unlimited).
func NewClientStreamHandler(cfg ConnConfig, logger *slog.Logger, maxConn int64) *ClientStreamHandler {
	return &ClientStreamHandler{cfg: cfg, logger: logger, maxConn: maxConn}
}

// ServeHTTP implements http.Handler.
func (h *ClientStreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.maxConn > 0 && h.active.Load() >= h.maxConn {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}

	ws, err := websocket.Accept(w, r, acceptOptions())
	if err != nil {
		h.logger.Error("websocket accept failed", slog.String("error", err.Error()))
		return
	}

	h.active.Add(1)
	defer h.active.Add(-1)

	conn := NewConn(ws, h.cfg, h.logger)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go conn.Start(ctx)

	h.logger.Info("client-stream connected", slog.String("remote", r.RemoteAddr))
	defer h.logger.Info("client-stream disconnected", slog.String("remote", r.RemoteAddr))

	for {
		data, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway ||
				errors.Is(err, context.Canceled) {
				return
			}
			h.logger.Error("client-stream read error", slog.String("error", err.Error()))
			return
		}
		h.onMessage(r.Context(), data)
	}
}

// onMessage processes each incoming message from the client.
func (h *ClientStreamHandler) onMessage(_ context.Context, data []byte) {
	h.logger.Info("client-stream received", slog.Int("bytes", len(data)))
	// In production, dispatch to a message queue or processing pipeline here.
}

// ActiveConnections returns the current number of active connections.
func (h *ClientStreamHandler) ActiveConnections() int64 {
	return h.active.Load()
}

// ------------------------------------------------------------------
// 2. Server-Stream: server sends, client only receives
// ------------------------------------------------------------------

// ServerStreamHandler handles WebSocket connections where only the server
// pushes messages to the client (unidirectional: server -> client).
// Typical use cases: real-time notifications, live feeds, server-sent events.
type ServerStreamHandler struct {
	cfg         ConnConfig
	logger      *slog.Logger
	active      atomic.Int64
	maxConn     int64
	interval    time.Duration
	messageFunc func(seq uint64) []byte
}

// NewServerStreamHandler creates a new ServerStreamHandler.
// interval controls how frequently the server pushes messages.
// messageFunc generates the payload for each push; if nil a default JSON is used.
func NewServerStreamHandler(cfg ConnConfig, logger *slog.Logger, maxConn int64, interval time.Duration, messageFunc func(seq uint64) []byte) *ServerStreamHandler {
	if messageFunc == nil {
		messageFunc = defaultServerMessage
	}
	return &ServerStreamHandler{
		cfg:         cfg,
		logger:      logger,
		maxConn:     maxConn,
		interval:    interval,
		messageFunc: messageFunc,
	}
}

func defaultServerMessage(seq uint64) []byte {
	msg := map[string]interface{}{
		"type":      "server-push",
		"seq":       seq,
		"timestamp": time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(msg)
	return data
}

// ServeHTTP implements http.Handler.
func (h *ServerStreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.maxConn > 0 && h.active.Load() >= h.maxConn {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}

	ws, err := websocket.Accept(w, r, acceptOptions())
	if err != nil {
		h.logger.Error("websocket accept failed", slog.String("error", err.Error()))
		return
	}

	h.active.Add(1)
	defer h.active.Add(-1)

	conn := NewConn(ws, h.cfg, h.logger)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go conn.Start(ctx)

	h.logger.Info("server-stream connected", slog.String("remote", r.RemoteAddr))
	defer h.logger.Info("server-stream disconnected", slog.String("remote", r.RemoteAddr))

	g, gCtx := errgroup.WithContext(ctx)

	// Read pump: detect client close / errors.
	g.Go(func() error {
		for {
			_, err := conn.Read(gCtx)
			if err != nil {
				return err
			}
		}
	})

	// Write pump: server pushes messages at configured interval.
	g.Go(func() error {
		var seq uint64
		ticker := time.NewTicker(h.interval)
		defer ticker.Stop()
		for {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			case <-ticker.C:
				seq++
				if !conn.Send(h.messageFunc(seq)) {
					return fmt.Errorf("send buffer full")
				}
			}
		}
	})

	if err := g.Wait(); err != nil {
		closeStatus := websocket.CloseStatus(err)
		if closeStatus == websocket.StatusNormalClosure ||
			closeStatus == websocket.StatusGoingAway ||
			errors.Is(err, context.Canceled) {
			return
		}
		h.logger.Error("server-stream error", slog.String("error", err.Error()))
	}
}

// ActiveConnections returns the current number of active connections.
func (h *ServerStreamHandler) ActiveConnections() int64 {
	return h.active.Load()
}

// ------------------------------------------------------------------
// 3. Bidirectional-Stream: both sides send messages
// ------------------------------------------------------------------

// BidiStreamHandler handles WebSocket connections where both the client
// and the server can send messages concurrently (full-duplex).
// Typical use cases: chat, collaborative editing, gaming.
type BidiStreamHandler struct {
	cfg     ConnConfig
	logger  *slog.Logger
	active  atomic.Int64
	maxConn int64
}

// NewBidiStreamHandler creates a new BidiStreamHandler.
func NewBidiStreamHandler(cfg ConnConfig, logger *slog.Logger, maxConn int64) *BidiStreamHandler {
	return &BidiStreamHandler{cfg: cfg, logger: logger, maxConn: maxConn}
}

// ServeHTTP implements http.Handler.
func (h *BidiStreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.maxConn > 0 && h.active.Load() >= h.maxConn {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}

	ws, err := websocket.Accept(w, r, acceptOptions())
	if err != nil {
		h.logger.Error("websocket accept failed", slog.String("error", err.Error()))
		return
	}

	h.active.Add(1)
	defer h.active.Add(-1)

	conn := NewConn(ws, h.cfg, h.logger)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go conn.Start(ctx)

	h.logger.Info("bidi-stream connected", slog.String("remote", r.RemoteAddr))
	defer h.logger.Info("bidi-stream disconnected", slog.String("remote", r.RemoteAddr))

	g, gCtx := errgroup.WithContext(ctx)

	// Read pump: receive messages from client.
	g.Go(func() error {
		for {
			data, err := conn.Read(gCtx)
			if err != nil {
				return err
			}
			h.onClientMessage(conn, data)
		}
	})

	// Server push pump: periodically send heartbeat/status to client.
	g.Go(func() error {
		var seq uint64
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			case <-ticker.C:
				seq++
				msg := map[string]interface{}{
					"type":      "server-heartbeat",
					"seq":       seq,
					"timestamp": time.Now().UnixMilli(),
				}
				data, _ := json.Marshal(msg)
				if !conn.Send(data) {
					return fmt.Errorf("send buffer full")
				}
			}
		}
	})

	if err := g.Wait(); err != nil {
		closeStatus := websocket.CloseStatus(err)
		if closeStatus == websocket.StatusNormalClosure ||
			closeStatus == websocket.StatusGoingAway ||
			errors.Is(err, context.Canceled) {
			return
		}
		h.logger.Error("bidi-stream error", slog.String("error", err.Error()))
	}
}

// onClientMessage processes an incoming client message and optionally responds.
func (h *BidiStreamHandler) onClientMessage(conn *Conn, data []byte) {
	h.logger.Info("bidi-stream received", slog.Int("bytes", len(data)))

	// Echo-back acknowledgement (can be replaced with business logic).
	ack := map[string]interface{}{
		"type":      "ack",
		"size":      len(data),
		"timestamp": time.Now().UnixMilli(),
	}
	ackData, _ := json.Marshal(ack)
	conn.Send(ackData)
}

// ActiveConnections returns the current number of active connections.
func (h *BidiStreamHandler) ActiveConnections() int64 {
	return h.active.Load()
}
