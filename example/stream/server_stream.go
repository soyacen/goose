package main

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/coder/websocket"
)

// Compile-time check: serverStream implements the ServerStream interface.
var _ ServerStream = (*serverStream)(nil)

// serverStream is the concrete implementation of the ServerStream interface
// defined in stream_interfaces.go. It wraps a *Conn and provides Codec-based
// SendMsg/RecvMsg for use by GenericServerStream.
type serverStream struct {
	conn    *Conn
	ctx     context.Context
	codec   Codec
	header  http.Header
	trailer http.Header
}

// newServerStream creates a base server stream wrapping conn.
func newServerStream(ctx context.Context, conn *Conn, codec Codec) *serverStream {
	return &serverStream{
		conn:    conn,
		ctx:     ctx,
		codec:   codec,
		header:  make(http.Header),
		trailer: make(http.Header),
	}
}

// SetHeader sets the header metadata. It may be called multiple times.
func (s *serverStream) SetHeader(h http.Header) error {
	for k, v := range h {
		s.header[k] = v
	}
	return nil
}

// SendHeader sends the header metadata. For WebSocket streams this is a no-op
// since headers are sent during the HTTP upgrade handshake.
func (s *serverStream) SendHeader(h http.Header) error {
	for k, v := range h {
		s.header[k] = v
	}
	return nil
}

// SetTrailer sets the trailer metadata which will be sent with the RPC status.
func (s *serverStream) SetTrailer(t http.Header) {
	for k, v := range t {
		s.trailer[k] = v
	}
}

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
