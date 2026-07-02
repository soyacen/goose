package main

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/coder/websocket"
)

// ---------------------------------------------------------------------------
// serverStream — base implementation of ServerStream
// ---------------------------------------------------------------------------

// serverStream is the concrete implementation of the ServerStream interface.
// It wraps a *Conn and provides Codec-based SendMsg/RecvMsg.
type serverStream struct {
	conn    *Conn
	ctx     context.Context
	codec   Codec
	req     *http.Request
	header  http.Header
	trailer http.Header
}

// newServerStream creates a base server stream wrapping conn.
func newServerStream(ctx context.Context, conn *Conn, r *http.Request, codec Codec) *serverStream {
	return &serverStream{
		conn:    conn,
		ctx:     ctx,
		codec:   codec,
		req:     r,
		header:  make(http.Header),
		trailer: make(http.Header),
	}
}

// Header returns the response header metadata.
func (s *serverStream) Header() http.Header { return s.header }

// SetHeader sets the response header metadata.
func (s *serverStream) SetHeader(h http.Header) { s.header = h }

// Trailer returns the trailer metadata.
func (s *serverStream) Trailer() http.Header { return s.trailer }

// SetTrailer sets the trailer metadata.
func (s *serverStream) SetTrailer(t http.Header) { s.trailer = t }

// Context returns the stream's context.
func (s *serverStream) Context() context.Context { return s.ctx }

// SendMsg serializes m using the Codec and enqueues it for writing via Conn.
func (s *serverStream) SendMsg(m any) error {
	data, err := s.codec.Marshal(m)
	if err != nil {
		return err
	}
	if !s.conn.Send(data) {
		return io.ErrClosedPipe
	}
	return nil
}

// RecvMsg reads a message from Conn and deserializes it into m using the Codec.
// Returns io.EOF when the client closes the stream.
func (s *serverStream) RecvMsg(m any) error {
	data, err := s.conn.Read(s.ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		// Any websocket-level close (normal closure, going away, etc.)
		// maps to io.EOF — this is how the server detects that the client
		// has finished sending (equivalent to gRPC CloseSend).
		if websocket.CloseStatus(err) != -1 {
			return io.EOF
		}
		return err
	}
	return s.codec.Unmarshal(data, m)
}

// ---------------------------------------------------------------------------
// serverClientStream — client-streaming (many requests, one response)
// ---------------------------------------------------------------------------

// serverClientStream implements ServerClientStream[Req, Res].
type serverClientStream[Req any, Res any] struct {
	*serverStream
}

// newServerClientStream creates a typed client-streaming server stream.
func newServerClientStream[Req any, Res any](ctx context.Context, conn *Conn, r *http.Request, codec Codec) ServerClientStream[Req, Res] {
	return &serverClientStream[Req, Res]{
		serverStream: newServerStream(ctx, conn, r, codec),
	}
}

// Recv reads the next request from the client. Returns io.EOF when the
// client has closed the stream.
func (s *serverClientStream[Req, Res]) Recv() (*Req, error) {
	var req Req
	if err := s.RecvMsg(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

// SendAndClose sends the final response to the client and signals stream
// completion. It must be called exactly once after all Recv calls.
func (s *serverClientStream[Req, Res]) SendAndClose(res *Res) error {
	return s.SendMsg(res)
}

// ---------------------------------------------------------------------------
// serverServerStream — server-streaming (one request, many responses)
// ---------------------------------------------------------------------------

// serverServerStream implements ServerServerStream[Res].
type serverServerStream[Res any] struct {
	*serverStream
}

// newServerServerStream creates a typed server-streaming server stream.
func newServerServerStream[Res any](ctx context.Context, conn *Conn, r *http.Request, codec Codec) ServerServerStream[Res] {
	return &serverServerStream[Res]{
		serverStream: newServerStream(ctx, conn, r, codec),
	}
}

// Send writes a response message to the client stream.
func (s *serverServerStream[Res]) Send(res *Res) error {
	return s.SendMsg(res)
}

// ---------------------------------------------------------------------------
// serverBidiStream — bidirectional (many requests, many responses)
// ---------------------------------------------------------------------------

// serverBidiStream implements ServerBidiStream[Req, Res].
type serverBidiStream[Req any, Res any] struct {
	*serverStream
}

// newServerBidiStream creates a typed bidirectional server stream.
func newServerBidiStream[Req any, Res any](ctx context.Context, conn *Conn, r *http.Request, codec Codec) ServerBidiStream[Req, Res] {
	return &serverBidiStream[Req, Res]{
		serverStream: newServerStream(ctx, conn, r, codec),
	}
}

// Recv reads the next request from the client. Returns io.EOF when the
// client has closed the stream.
func (s *serverBidiStream[Req, Res]) Recv() (*Req, error) {
	var req Req
	if err := s.RecvMsg(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

// Send writes a response message to the client stream.
func (s *serverBidiStream[Req, Res]) Send(res *Res) error {
	return s.SendMsg(res)
}
