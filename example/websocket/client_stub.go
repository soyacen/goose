package websocket

import (
	"context"
	"log/slog"

	"github.com/coder/websocket"
	"github.com/soyacen/goose/ws"
)

// ---------------------------------------------------------------------------
// streamServiceClient — implements StreamServiceClient
// ---------------------------------------------------------------------------

// Compile-time check: streamServiceClient implements StreamServiceClient.
var _ StreamServiceClient = (*streamServiceClient)(nil)

// streamServiceClient implements the StreamServiceClient interface by creating
// WebSocket connections for each RPC call. This is what protoc-gen-goose
// would generate as the client stub.
type streamServiceClient struct {
	url      string
	codec    ws.Codec
	dialOpts *websocket.DialOptions
	connCfg  ws.ConnConfig
	logger   *slog.Logger
}

// NewStreamServiceClient creates a client that implements StreamServiceClient.
// url is the WebSocket endpoint (e.g., "ws://localhost:8080/ws/bidi-stream").
// Each method call dials a new connection for the corresponding streaming RPC.
func NewStreamServiceClient(url string, codec ws.Codec, logger *slog.Logger) StreamServiceClient {
	if codec == nil {
		codec = ws.JSONCodec{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &streamServiceClient{
		url:     url,
		codec:   codec,
		logger:  logger,
		connCfg: ws.DefaultConnConfig(),
		dialOpts: &websocket.DialOptions{
			CompressionMode: websocket.CompressionContextTakeover,
		},
	}
}

// dialAndConnect dials the WebSocket endpoint and returns a ClientStream ready
// for use. The caller is responsible for the returned cancel function if the
// stream is not fully consumed.
func (c *streamServiceClient) dialAndConnect(ctx context.Context) (ws.ClientStream, error) {
	wsConn, _, err := websocket.Dial(ctx, c.url, c.dialOpts)
	if err != nil {
		return nil, err
	}

	conn := ws.NewConn(wsConn, c.connCfg, c.logger)
	go conn.Start(ctx)

	return ws.NewClientStream(ctx, conn, c.codec), nil
}

// ClientStream opens a client-streaming RPC.
func (c *streamServiceClient) ClientStream(ctx context.Context) (ws.ClientStreamingClient[Request, Response], error) {
	cs, err := c.dialAndConnect(ctx)
	if err != nil {
		return nil, err
	}
	return &ws.GenericClientStream[Request, Response]{ClientStream: cs}, nil
}

// ServerStream opens a server-streaming RPC. It sends the initial request
// and returns a stream for receiving multiple responses.
func (c *streamServiceClient) ServerStream(ctx context.Context, in *Request) (ws.ServerStreamingClient[Response], error) {
	cs, err := c.dialAndConnect(ctx)
	if err != nil {
		return nil, err
	}

	// Send the initial request.
	if err := cs.SendMsg(in); err != nil {
		_ = cs.CloseSend()
		return nil, err
	}

	return &ws.GenericClientStream[Request, Response]{ClientStream: cs}, nil
}

// Bid opens a bidirectional-streaming RPC.
func (c *streamServiceClient) BidStream(ctx context.Context) (ws.BidiStreamingClient[Request, Response], error) {
	cs, err := c.dialAndConnect(ctx)
	if err != nil {
		return nil, err
	}
	return &ws.GenericClientStream[Request, Response]{ClientStream: cs}, nil
}
