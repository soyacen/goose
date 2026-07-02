package main

import (
	"context"
)

type Request struct {
	Name string `json:"name,omitempty"`
}

type Response struct {
	Message string `json:"message,omitempty"`
}

type StreamServiceClient interface {
	ClientStrean(ctx context.Context) (ClientStreamingClient[Request, Response], error)
	ServerStrean(ctx context.Context, in *Request) (ServerStreamingClient[Response], error)
	Bid(ctx context.Context) (BidiStreamingClient[Request, Response], error)
}

type StreamServiceServer interface {
	ClientStream(ClientStreamingServer[Request, Response]) error
	ServerStream(*Request, ServerStreamingServer[Response]) error
	BidStream(BidiStreamingServer[Request, Response]) error
}
