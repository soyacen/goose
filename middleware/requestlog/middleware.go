// Package requestlog provides HTTP access logging middleware for both server and client requests
package requestlog

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/soyacen/goose/client"
)

// LoggerFactory is a function type that creates a logger instance from a context
// Parameters:
//   - ctx: Context that may contain information for creating the logger
//
// Returns:
//   - *slog.Logger: Logger instance
//   - error: Error if logger creation fails
type LoggerFactory func(ctx context.Context) (*slog.Logger, error)

// options holds configuration options for the access log middleware
type options struct {
	loggerFactory LoggerFactory // Factory function to create loggers
	level         slog.Level    // Log level for access log entries
}

// apply applies the given options to the options struct
// Parameters:
//   - opts: Variable number of Option functions
//
// Returns:
//   - *options: Pointer to the updated options struct
func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Option is a function type for configuring access log middleware options
type Option func(o *options)

// defaultOptions returns the default configuration options
// Returns:
//   - *options: Default options with nil logger factory and zero log level
func defaultOptions() *options {
	return &options{}
}

// WithLoggerFactory sets the logger factory function
// Parameters:
//   - loggerFactory: Function to create loggers from context
//
// Returns:
//   - Option: Function to set the logger factory option
func WithLoggerFactory(loggerFactory LoggerFactory) Option {
	return func(o *options) {
		o.loggerFactory = loggerFactory
	}
}

// WithLevel sets the log level for access log entries
// Parameters:
//   - level: Log level for access logs
//
// Returns:
//   - Option: Function to set the log level option
func WithLevel(level slog.Level) Option {
	return func(o *options) {
		o.level = level
	}
}

// Client creates a client-side access logging middleware
// Parameters:
//   - opts: Variable number of Option functions for configuration
//
// Returns:
//   - client.Middleware: Client middleware function
//
// Behavior:
//  1. Records the start time of the request
//  2. Invokes the next handler
//  3. Logs request details including latency and response status
func Client(opts ...Option) client.Middleware {
	opt := defaultOptions().apply(opts...)

	// Create a sync.Pool to reuse slog.Attr slices for better performance
	pool := sync.Pool{
		New: func() interface{} {
			return make([]slog.Attr, 0, 10)
		},
	}

	return func(cli *http.Client, request *http.Request, invoker client.Invoker) (*http.Response, error) {
		// Skip logging if no logger factory is configured
		if opt.loggerFactory == nil {
			return invoker(cli, request)
		}

		// Get context from the request
		ctx := request.Context()

		// Create a logger using the logger factory
		logger, err := opt.loggerFactory(ctx)
		if err != nil {
			// Log error and continue with request processing if logger creation fails
			slog.Error("accesslog: failed to get logger", slog.String("error", err.Error()))
			return invoker(cli, request)
		}

		// Record the start time for latency calculation
		startTime := time.Now()

		// Invoke the next handler
		response, err := invoker(cli, request)

		// Calculate request processing latency
		latency := time.Since(startTime)

		// Get a reusable slice of slog.Attr from the pool
		fields := pool.Get().([]slog.Attr)

		// Add system identifier
		fields = append(fields, slog.String("system", "client"))

		// Add timestamp when the request started
		fields = append(fields, slog.String("timestamp", startTime.Format(time.RFC3339Nano)))

		// Add deadline information if context has a deadline
		if d, ok := ctx.Deadline(); ok {
			fields = append(fields, slog.String("deadline", d.Format(time.RFC3339Nano)))
		}

		// Add request information
		fields = append(fields, slog.String("latency", latency.String()))
		fields = append(fields, slog.String("method", request.Method))
		fields = append(fields, slog.String("uri", request.RequestURI))
		fields = append(fields, slog.String("proto", request.Proto))
		fields = append(fields, slog.String("host", request.Host))

		// Add response information if available
		if response != nil {
			fields = append(fields, slog.Int("response_status", response.StatusCode))
		} else if err != nil {
			fields = append(fields, slog.String("error", err.Error()))
		}

		// Log the access information
		logger.LogAttrs(ctx, opt.level, request.URL.Path, fields...)

		// Reset the slice length to 0 to reuse the underlying array
		fields = fields[:0]

		// Put the slice back into the pool for reuse
		pool.Put(fields)

		return response, err
	}
}
