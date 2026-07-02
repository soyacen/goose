package websocket

import (
	"context"

	"github.com/soyacen/goose/ws"
)

type StreamServiceClient interface {
	ClientStream(ctx context.Context) (ws.ClientStreamingClient[Request, Response], error)
	ServerStream(ctx context.Context, in *Request) (ws.ServerStreamingClient[Response], error)
	BidStream(ctx context.Context) (ws.BidiStreamingClient[Request, Response], error)
}

type StreamServiceServer interface {
	ClientStream(ws.ClientStreamingServer[Request, Response]) error
	ServerStream(*Request, ws.ServerStreamingServer[Response]) error
	BidStream(ws.BidiStreamingServer[Request, Response]) error
}
