package ws

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/coder/websocket"
)

// Compile-time check: clientStream implements the ClientStream interface.
var _ ClientStream = (*clientStream)(nil)

// clientStream is the base struct for all client-side stream stubs.
// It wraps a *Conn and provides Codec-based SendMsg/RecvMsg.
type clientStream struct {
	conn    *Conn
	ctx     context.Context
	cancel  context.CancelFunc
	codec   Codec
	header  http.Header
	trailer http.Header
}

// NewClientStream creates a base client stream wrapping an established Conn.
// The caller is responsible for dialing the WebSocket and calling conn.Start()
// before passing the Conn here. The cancel function is called when CloseSend
// is invoked or the stream is otherwise torn down.
func NewClientStream(ctx context.Context, conn *Conn, codec Codec) *clientStream {
	streamCtx, cancel := context.WithCancel(ctx)
	return &clientStream{
		conn:    conn,
		ctx:     streamCtx,
		cancel:  cancel,
		codec:   codec,
		header:  make(http.Header),
		trailer: make(http.Header),
	}
}

// Header implements ClientStream.
func (s *clientStream) Header() (http.Header, error) {
	return s.header, nil
}

// Trailer implements ClientStream.
func (s *clientStream) Trailer() http.Header {
	return s.trailer
}

// CloseSend closes the send direction of the stream.
func (s *clientStream) CloseSend() error {
	s.conn.Close()
	return nil
}

// Context implements ClientStream.
func (s *clientStream) Context() context.Context {
	return s.ctx
}

// SendMsg serializes m and enqueues it for sending.
func (s *clientStream) SendMsg(m any) error {
	data, err := s.codec.Marshal(m)
	if err != nil {
		return err
	}
	if !s.conn.Send(data) {
		return io.ErrClosedPipe
	}
	return nil
}

// RecvMsg reads a message from the connection and deserializes it into m.
// Returns io.EOF when the server closes the stream.
func (s *clientStream) RecvMsg(m any) error {
	data, err := s.conn.Read(s.ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		// Map websocket close to io.EOF, mirroring gRPC behavior.
		if websocket.CloseStatus(err) != -1 {
			return io.EOF
		}
		return err
	}
	return s.codec.Unmarshal(data, m)
}

// close tears down the stream.
func (s *clientStream) close() {
	s.conn.Close()
	s.cancel()
}
