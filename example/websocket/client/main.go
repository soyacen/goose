package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/soyacen/goose/example/websocket"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	baseURL := "ws://localhost:8080"
	if env := os.Getenv("SERVER_URL"); env != "" {
		baseURL = env
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	marshalOpts := protojson.MarshalOptions{}
	unmarshalOpts := protojson.UnmarshalOptions{}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	switch os.Args[1] {
	case "client-stream":
		runClientStream(ctx, baseURL, logger, marshalOpts, unmarshalOpts)
	case "server-stream":
		runServerStream(ctx, baseURL, logger, marshalOpts, unmarshalOpts)
	case "bidi":
		runBidStream(ctx, baseURL, logger, marshalOpts, unmarshalOpts)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`WebSocket Streaming Client

Usage:
  client <command>

Commands:
  client-stream   Send multiple messages, receive one aggregated response
  server-stream   Send one message, receive multiple responses
  bidi            Bidirectional streaming (send and receive concurrently)
  help            Show this help message

Environment:
  SERVER_URL      Server base URL (default: ws://localhost:8080)`)
}

// ---------------------------------------------------------------------------
// client-stream: send multiple messages → receive one response
// ---------------------------------------------------------------------------

func runClientStream(ctx context.Context, baseURL string, logger *slog.Logger, marshalOpts protojson.MarshalOptions, unmarshalOpts protojson.UnmarshalOptions) {
	client := websocket.NewStreamServiceClient(baseURL+"/ws/client-stream", logger, marshalOpts, unmarshalOpts)

	stream, err := client.ClientStream(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Client-Stream: type messages to send (empty line to finish)")

	scanner := bufio.NewScanner(os.Stdin)
	var count int
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}
		if err := stream.Send(&websocket.Request{Name: line}); err != nil {
			fmt.Fprintf(os.Stderr, "send error: %v\n", err)
			os.Exit(1)
		}
		count++
		fmt.Printf("  sent: %s\n", line)
	}

	// Signal we're done sending, but keep read side open.
	if err := stream.CloseSend(); err != nil {
		fmt.Fprintf(os.Stderr, "close send error: %v\n", err)
		os.Exit(1)
	}

	// Small delay to let the server process and send its response.
	time.Sleep(100 * time.Millisecond)

	var resp websocket.Response
	if err := stream.RecvMsg(&resp); err != nil {
		fmt.Fprintf(os.Stderr, "recv error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nresponse: %s (sent %d messages)\n", resp.Message, count)
}

// ---------------------------------------------------------------------------
// server-stream: send one message → receive multiple responses
// ---------------------------------------------------------------------------

func runServerStream(ctx context.Context, baseURL string, logger *slog.Logger, marshalOpts protojson.MarshalOptions, unmarshalOpts protojson.UnmarshalOptions) {
	client := websocket.NewStreamServiceClient(baseURL+"/ws/server-stream", logger, marshalOpts, unmarshalOpts)

	fmt.Print("Enter your name: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return
	}
	name := strings.TrimSpace(scanner.Text())
	if name == "" {
		name = "world"
	}

	stream, err := client.ServerStream(ctx, &websocket.Request{Name: name})
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nReceiving messages:")
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("\nstream completed")
			return
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "recv error: %v\n", err)
			return
		}
		fmt.Printf("  <- %s\n", resp.Message)
	}
}

// ---------------------------------------------------------------------------
// bidi: full-duplex bidirectional streaming
// ---------------------------------------------------------------------------

func runBidStream(ctx context.Context, baseURL string, logger *slog.Logger, marshalOpts protojson.MarshalOptions, unmarshalOpts protojson.UnmarshalOptions) {
	client := websocket.NewStreamServiceClient(baseURL+"/ws/bidi-stream", logger, marshalOpts, unmarshalOpts)

	stream, err := client.BidStream(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect failed: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Receiver goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				if ctx.Err() == nil {
					fmt.Fprintf(os.Stderr, "recv error: %v\n", err)
				}
				return
			}
			fmt.Printf("  <- %s\n", resp.Message)
		}
	}()

	fmt.Println("Bidirectional Stream: type messages to send (Ctrl+D or empty line to finish)")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}
		if err := stream.Send(&websocket.Request{Name: line}); err != nil {
			fmt.Fprintf(os.Stderr, "send error: %v\n", err)
			break
		}
	}

	// Close send direction, then wait for receiver to finish
	_ = stream.CloseSend()
	<-done

	fmt.Println("bidi stream completed")
}
