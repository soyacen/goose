package websocket

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/soyacen/goose/server"
	"github.com/soyacen/goose/ws"

	"google.golang.org/protobuf/encoding/protojson"
)

type mockStreamService struct {
	logger *slog.Logger
}

var _ ResponseBodyStreamServer = (*mockStreamService)(nil)

func (s *mockStreamService) ClientStream(stream ws.ClientStreamingServer[*Request, *Response]) error {
	var count int
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&Response{
				Message: fmt.Sprintf("received %d messages", count),
			})
		}
		if err != nil {
			return err
		}
		count++
		s.logger.Info("client-stream received", slog.String("name", req.Name))
	}
}

func (s *mockStreamService) ServerStream(req *Request, stream ws.ServerStreamingServer[*Response]) error {
	for i := 1; i <= 3; i++ {
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
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

func (s *mockStreamService) BidStream(stream ws.BidiStreamingServer[*Request, *Response]) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		resp := &Response{
			Message: fmt.Sprintf("ack: %s", req.Name),
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}

func startServer(t *testing.T, port int) *http.Server {
	t.Helper()
	logger := slog.Default()
	connCfg := ws.DefaultConnConfig()
	marshalOpts := protojson.MarshalOptions{}
	unmarshalOpts := protojson.UnmarshalOptions{}

	svc := &mockStreamService{logger: logger}

	mux := http.NewServeMux()
	AppendResponseBodyWebsocketRoute(mux, svc, server.Chain(), marshalOpts, unmarshalOpts, ws.AcceptOptions(), connCfg, logger)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("server error: %v", err)
		}
	}()

	time.Sleep(200 * time.Millisecond)
	return srv
}

func newTestClient(t *testing.T, port int) ResponseBodyStreamClient {
	t.Helper()
	logger := slog.Default()
	marshalOpts := protojson.MarshalOptions{}
	unmarshalOpts := protojson.UnmarshalOptions{}
	return NewResponseBodyStreamClient(fmt.Sprintf("ws://localhost:%d", port), logger, marshalOpts, unmarshalOpts, ws.DialOptions())
}

func TestClientStream(t *testing.T) {
	srv := startServer(t, 39081)
	defer srv.Shutdown(context.Background())

	client := newTestClient(t, 39081)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.ClientStream(ctx)
	if err != nil {
		t.Fatalf("ClientStream connect failed: %v", err)
	}

	messages := []string{"alice", "bob", "charlie"}
	for _, msg := range messages {
		if err := stream.Send(&Request{Name: msg}); err != nil {
			t.Fatalf("Send failed: %v", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatalf("CloseAndRecv failed: %v", err)
	}

	expected := fmt.Sprintf("received %d messages", len(messages))
	if resp.Message != expected {
		t.Fatalf("expected %q, got %q", expected, resp.Message)
	}
}

func TestServerStream(t *testing.T) {
	srv := startServer(t, 39082)
	defer srv.Shutdown(context.Background())

	client := newTestClient(t, 39082)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.ServerStream(ctx, &Request{Name: "world"})
	if err != nil {
		t.Fatalf("ServerStream connect failed: %v", err)
	}

	var received []string
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv failed: %v", err)
		}
		received = append(received, resp.Message)
	}

	if len(received) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(received))
	}

	for i, msg := range received {
		expected := fmt.Sprintf("hello world, message #%d", i+1)
		if msg != expected {
			t.Fatalf("message %d: expected %q, got %q", i+1, expected, msg)
		}
	}
}

func TestBidStream(t *testing.T) {
	srv := startServer(t, 39083)
	defer srv.Shutdown(context.Background())

	client := newTestClient(t, 39083)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.BidStream(ctx)
	if err != nil {
		t.Fatalf("BidStream connect failed: %v", err)
	}

	messages := []string{"hello", "world", "test"}
	for _, msg := range messages {
		if err := stream.Send(&Request{Name: msg}); err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		resp, err := stream.Recv()
		if err != nil {
			t.Fatalf("Recv failed: %v", err)
		}

		expected := fmt.Sprintf("ack: %s", msg)
		if resp.Message != expected {
			t.Fatalf("expected %q, got %q", expected, resp.Message)
		}
	}

	if err := stream.CloseSend(); err != nil {
		t.Fatalf("CloseSend failed: %v", err)
	}
}
