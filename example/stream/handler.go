package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/coder/websocket"
)

// acceptOptions returns the websocket.AcceptOptions for CORS and subprotocol.
func acceptOptions() *websocket.AcceptOptions {
	return &websocket.AcceptOptions{
		InsecureSkipVerify: false,
	}
}

// isNormalClose returns true if the error represents a normal WebSocket close.
func isNormalClose(err error) bool {
	status := websocket.CloseStatus(err)
	return status == websocket.StatusNormalClosure ||
		status == websocket.StatusGoingAway ||
		errors.Is(err, context.Canceled)
}

// ------------------------------------------------------------------
// 1. Client-Stream: client sends, server only receives
// ------------------------------------------------------------------

// ClientStreamHandler handles WebSocket connections where only the client
// sends messages to the server (unidirectional: client -> server).
// It delegates business logic to the StreamServiceServer implementation.
type ClientStreamHandler struct {
	service StreamServiceServer
	codec   Codec
	cfg     ConnConfig
	logger  *slog.Logger
	active  atomic.Int64
	maxConn int64
}

// NewClientStreamHandler creates a new ClientStreamHandler.
// service is the user-implemented StreamServiceServer that contains the
// business logic for this streaming pattern.
func NewClientStreamHandler(service StreamServiceServer, codec Codec, cfg ConnConfig, logger *slog.Logger, maxConn int64) *ClientStreamHandler {
	return &ClientStreamHandler{
		service: service,
		codec:   codec,
		cfg:     cfg,
		logger:  logger,
		maxConn: maxConn,
	}
}

// ServeHTTP implements http.Handler.
// It upgrades the connection, creates a Conn and a typed server stream,
// then delegates to the StreamServiceServer.ClientStream method.
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

	ss := newServerStream(ctx, conn, h.codec)
	stream := &GenericServerStream[Request, Response]{ServerStream: ss}
	if err := h.service.ClientStream(stream); err != nil && !isNormalClose(err) {
		h.logger.Error("client-stream error", slog.String("error", err.Error()))
	}
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
// It delegates business logic to the StreamServiceServer implementation.
type ServerStreamHandler struct {
	service StreamServiceServer
	codec   Codec
	cfg     ConnConfig
	logger  *slog.Logger
	active  atomic.Int64
	maxConn int64
}

// NewServerStreamHandler creates a new ServerStreamHandler.
func NewServerStreamHandler(service StreamServiceServer, codec Codec, cfg ConnConfig, logger *slog.Logger, maxConn int64) *ServerStreamHandler {
	return &ServerStreamHandler{
		service: service,
		codec:   codec,
		cfg:     cfg,
		logger:  logger,
		maxConn: maxConn,
	}
}

// ServeHTTP implements http.Handler.
// It upgrades the connection, reads the initial request from the client,
// creates a server-streaming stream, and delegates to the service.
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

	// Read the initial request from the client.
	var req Request
	data, err := conn.Read(ctx)
	if err != nil {
		if !isNormalClose(err) {
			h.logger.Error("server-stream read request error", slog.String("error", err.Error()))
		}
		return
	}
	if err := h.codec.Unmarshal(data, &req); err != nil {
		h.logger.Error("server-stream unmarshal error", slog.String("error", err.Error()))
		return
	}

	ss := newServerStream(ctx, conn, h.codec)
	stream := &GenericServerStream[Request, Response]{ServerStream: ss}
	if err := h.service.ServerStream(&req, stream); err != nil && !isNormalClose(err) {
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
// It delegates business logic to the StreamServiceServer implementation.
type BidiStreamHandler struct {
	service StreamServiceServer
	codec   Codec
	cfg     ConnConfig
	logger  *slog.Logger
	active  atomic.Int64
	maxConn int64
}

// NewBidiStreamHandler creates a new BidiStreamHandler.
func NewBidiStreamHandler(service StreamServiceServer, codec Codec, cfg ConnConfig, logger *slog.Logger, maxConn int64) *BidiStreamHandler {
	return &BidiStreamHandler{
		service: service,
		codec:   codec,
		cfg:     cfg,
		logger:  logger,
		maxConn: maxConn,
	}
}

// ServeHTTP implements http.Handler.
// It upgrades the connection, creates a bidirectional stream, and delegates
// to the StreamServiceServer.BidStream method.
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

	ss := newServerStream(ctx, conn, h.codec)
	stream := &GenericServerStream[Request, Response]{ServerStream: ss}
	if err := h.service.BidStream(stream); err != nil && !isNormalClose(err) {
		h.logger.Error("bidi-stream error", slog.String("error", err.Error()))
	}
}

// ActiveConnections returns the current number of active connections.
func (h *BidiStreamHandler) ActiveConnections() int64 {
	return h.active.Load()
}
