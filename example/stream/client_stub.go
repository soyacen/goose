package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
)

// ---------------------------------------------------------------------------
// clientStream — base implementation of ClientStream on the client side
// ---------------------------------------------------------------------------

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

// newClientStream dials a WebSocket and returns a base clientStream.
func newClientStream(ctx context.Context, url string, codec Codec, dialOpts *websocket.DialOptions, cfg ConnConfig, logger *slog.Logger) (*clientStream, error) {
	ws, _, err := websocket.Dial(ctx, url, dialOpts)
	if err != nil {
		return nil, err
	}

	streamCtx, cancel := context.WithCancel(ctx)
	conn := NewConn(ws, cfg, logger)
	go conn.Start(streamCtx)

	return &clientStream{
		conn:    conn,
		ctx:     streamCtx,
		cancel:  cancel,
		codec:   codec,
		header:  make(http.Header),
		trailer: make(http.Header),
	}, nil
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

// ---------------------------------------------------------------------------
// clientStreamingStub — implements ClientStreamingClient[Req, Res]
// ---------------------------------------------------------------------------

type clientStreamingStub[Req any, Res any] struct {
	*clientStream
}

// Send sends a request message to the server.
func (s *clientStreamingStub[Req, Res]) Send(req *Req) error {
	return s.SendMsg(req)
}

// CloseAndRecv closes the request stream and waits for the server's response.
func (s *clientStreamingStub[Req, Res]) CloseAndRecv() (*Res, error) {
	// Signal that we are done sending.
	s.conn.Close()

	var res Res
	if err := s.RecvMsg(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

// ---------------------------------------------------------------------------
// serverStreamingStub — implements ServerStreamingClient[Res]
// ---------------------------------------------------------------------------

type serverStreamingStub[Res any] struct {
	*clientStream
}

// Recv receives the next response message from the server.
// Returns io.EOF when the stream has completed.
func (s *serverStreamingStub[Res]) Recv() (*Res, error) {
	var res Res
	if err := s.RecvMsg(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

// ---------------------------------------------------------------------------
// bidiStreamingStub — implements BidiStreamingClient[Req, Res]
// ---------------------------------------------------------------------------

type bidiStreamingStub[Req any, Res any] struct {
	*clientStream
}

// Send sends a request message to the server.
func (s *bidiStreamingStub[Req, Res]) Send(req *Req) error {
	return s.SendMsg(req)
}

// Recv receives the next response message from the server.
// Returns io.EOF when the stream has completed.
func (s *bidiStreamingStub[Req, Res]) Recv() (*Res, error) {
	var res Res
	if err := s.RecvMsg(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

// ---------------------------------------------------------------------------
// streamServiceClient — implements StreamService (client-side)
// ---------------------------------------------------------------------------

// streamServiceClient implements the StreamService interface by creating
// WebSocket connections for each RPC call. This is what protoc-gen-goose
// would generate as the client stub.
type streamServiceClient struct {
	url      string
	codec    Codec
	dialOpts *websocket.DialOptions
	connCfg  ConnConfig
	logger   *slog.Logger
}

// NewStreamServiceClient creates a client that implements StreamService.
// url is the WebSocket endpoint (e.g., "ws://localhost:8080/ws/bidi-stream").
// Each method call dials a new connection for the corresponding streaming RPC.
func NewStreamServiceClient(url string, codec Codec, logger *slog.Logger) StreamService {
	if codec == nil {
		codec = JSONCodec{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &streamServiceClient{
		url:    url,
		codec:  codec,
		logger: logger,
		connCfg: DefaultConnConfig(),
		dialOpts: &websocket.DialOptions{
			CompressionMode: websocket.CompressionContextTakeover,
		},
	}
}

// ClientStrean opens a client-streaming RPC.
func (c *streamServiceClient) ClientStrean(ctx context.Context) (ClientStreamingClient[ListExpiredCreditBucketsRequest, ListExpiredCreditBucketsResponse], error) {
	cs, err := newClientStream(ctx, c.url, c.codec, c.dialOpts, c.connCfg, c.logger)
	if err != nil {
		return nil, err
	}
	return &clientStreamingStub[ListExpiredCreditBucketsRequest, ListExpiredCreditBucketsResponse]{clientStream: cs}, nil
}

// ServerStrean opens a server-streaming RPC. It sends the initial request
// and returns a stream for receiving multiple responses.
func (c *streamServiceClient) ServerStrean(ctx context.Context, in *ListExpiredCreditBucketsRequest) (ServerStreamingClient[ListExpiredCreditBucketsResponse], error) {
	cs, err := newClientStream(ctx, c.url, c.codec, c.dialOpts, c.connCfg, c.logger)
	if err != nil {
		return nil, err
	}

	// Send the initial request.
	if err := cs.SendMsg(in); err != nil {
		cs.close()
		return nil, err
	}

	return &serverStreamingStub[ListExpiredCreditBucketsResponse]{clientStream: cs}, nil
}

// Bid opens a bidirectional-streaming RPC.
func (c *streamServiceClient) Bid(ctx context.Context) (BidiStreamingClient[ListExpiredCreditBucketsRequest, ListExpiredCreditBucketsResponse], error) {
	cs, err := newClientStream(ctx, c.url, c.codec, c.dialOpts, c.connCfg, c.logger)
	if err != nil {
		return nil, err
	}
	return &bidiStreamingStub[ListExpiredCreditBucketsRequest, ListExpiredCreditBucketsResponse]{clientStream: cs}, nil
}
