// Package timeout provides server and client timeout middleware
package timeout

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/soyacen/goose/client"
	"github.com/soyacen/goose/server"
)

// Key is the HTTP header key used to pass timeout settings
const Key = "X-Leo-Timeout"

// Server creates a server timeout middleware
// Parameters:
//   - duration: Default timeout duration
//
// Returns:
//   - server.Middleware: Server middleware function
//
// Behavior:
//  1. Checks for incoming timeout settings in request header
//  2. Uses the smaller of incoming timeout and default timeout
//  3. Creates context with timeout and invokes next handler
func Server(duration time.Duration) server.Middleware {
	return func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
		// Set initial timeout to default value
		timeout := duration

		// Check for custom timeout setting in request header
		if value := request.Header.Get(Key); value != "" {
			// Parse incoming timeout setting
			incomingDuration, err := time.ParseDuration(value)
			if err != nil {
				// Log error but continue with default timeout if parsing fails
				slog.Error("timeout parse error", slog.String("timeout", value), slog.String("error", err.Error()))
			} else {
				// Use smaller timeout value if parsed successfully and greater than 0
				if incomingDuration > 0 {
					timeout = min(incomingDuration, timeout)
				}
			}
		}

		// Create context with timeout
		ctx := request.Context()
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel() // Cancel context on function exit

		// Associate new context with request
		request = request.WithContext(ctx)

		// Invoke next handler
		invoker(response, request)
	}
}

// Client creates a client timeout middleware
// Parameters:
//   - duration: Default timeout duration
//
// Returns:
//   - client.Middleware: Client middleware function
//
// Behavior:
//  1. Calculates timeout based on context deadline
//  2. Sets timeout value in request header for server
//  3. Creates context with timeout and invokes next handler
func Client(duration time.Duration) client.Middleware {
	return func(cli *http.Client, request *http.Request, invoker client.Invoker) (*http.Response, error) {
		// Get context from request
		ctx := request.Context()

		// Set initial timeout to default value
		outgoingDuration := duration

		// Check if context has deadline set
		deadline, ok := ctx.Deadline()
		if ok {
			// Calculate remaining time until deadline
			timeout := time.Until(deadline)
			if timeout <= 0 {
				// Return deadline exceeded error if already timed out
				return nil, context.DeadlineExceeded
			}
			// Use smaller timeout value
			outgoingDuration = min(outgoingDuration, timeout)
		}

		// Set calculated timeout value in request header for server
		request.Header.Set(Key, outgoingDuration.String())

		// Create context with timeout
		ctx, cancel := context.WithTimeout(ctx, outgoingDuration)
		defer cancel() // Cancel context on function exit

		// Associate new context with request
		request = request.WithContext(ctx)

		// Invoke next handler
		return invoker(cli, request)
	}
}
