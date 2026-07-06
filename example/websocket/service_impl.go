package websocket

import (
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/soyacen/goose/ws"
)

// ---------------------------------------------------------------------------
// streamServiceImpl implements StreamServiceServer.
//
// This is what a user would write after protoc-gen-goose generates the
// interface and handler layer. Each method contains the business logic for
// the corresponding streaming RPC.
// ---------------------------------------------------------------------------

// Compile-time check: streamServiceImpl implements StreamServiceServer.
var _ StreamServiceServer = (*streamServiceImpl)(nil)

// streamServiceImpl is the user-defined service implementation.
type streamServiceImpl struct {
	logger *slog.Logger
}

// NewStreamServiceImpl creates a new service implementation.
func NewStreamServiceImpl(logger *slog.Logger) StreamServiceServer {
	return &streamServiceImpl{logger: logger}
}

// ---------------------------------------------------------------------------
// ClientStream: client sends many requests, server responds with one
// aggregated response (e.g., batch upload, log ingestion).
// ---------------------------------------------------------------------------

func (s *streamServiceImpl) ClientStream(stream ws.ClientStreamingServer[*Request, *Response]) error {
	var count int

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// Client has finished sending. Send the aggregated response.
			s.logger.Info("client-stream complete", slog.Int("count", count))
			return stream.SendAndClose(&Response{
				Message: fmt.Sprintf("received %d messages", count),
			})
		}
		if err != nil {
			return err
		}

		count++
		s.logger.Info("client-stream received",
			slog.String("name", req.Name),
		)
	}
}

// ---------------------------------------------------------------------------
// ServerStream: client sends one request, server streams back many responses
// (e.g., real-time feed, paginated list push).
// ---------------------------------------------------------------------------

func (s *streamServiceImpl) ServerStream(req *Request, stream ws.ServerStreamingServer[*Response]) error {
	s.logger.Info("server-stream started", slog.String("name", req.Name))

	// Simulate streaming 5 responses back to the client.
	for i := 1; i <= 5; i++ {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}

		resp := &Response{
			Message: fmt.Sprintf("hello %s, message #%d", req.Name, i),
		}

		if err := stream.Send(resp); err != nil {
			return err
		}

		s.logger.Info("server-stream sent", slog.Int("seq", i))

		// Simulate some processing delay between pushes.
		time.Sleep(500 * time.Millisecond)
	}

	s.logger.Info("server-stream complete")
	return nil
}

// ---------------------------------------------------------------------------
// BidStream: full-duplex bidirectional communication (e.g., chat, collab).
// ---------------------------------------------------------------------------

func (s *streamServiceImpl) BidStream(stream ws.BidiStreamingServer[*Request, *Response]) error {
	s.logger.Info("bidi-stream started")

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			s.logger.Info("bidi-stream client closed")
			return nil
		}
		if err != nil {
			return err
		}

		s.logger.Info("bidi-stream received",
			slog.String("name", req.Name),
		)

		// Echo back an acknowledgement.
		resp := &Response{
			Message: fmt.Sprintf("ack: %s", req.Name),
		}

		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}
