// Package accesslog provides HTTP access logging middleware for server requests
package accesslog

import (
	"context"
	"log/slog"
	"net/http"
	"reflect"
	"runtime/debug"
	"sync"
	"time"
	"unsafe"

	"github.com/soyacen/goose/server"
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

// Server creates a server-side access logging middleware
// Parameters:
//   - opts: Variable number of Option functions for configuration
//
// Returns:
//   - server.Middleware: Server middleware function
//
// Behavior:
//  1. Records the start time of the request
//  2. Wraps the response writer to capture the status code
//  3. Invokes the next handler
//  4. Logs request details including latency and response status
func Server(opts ...Option) server.Middleware {
	opt := defaultOptions().apply(opts...)

	// Create a sync.Pool to reuse slog.Attr slices for better performance
	pool := sync.Pool{
		New: func() interface{} {
			return make([]slog.Attr, 0, 10)
		},
	}

	return func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
		// Skip logging if no logger factory is configured
		if opt.loggerFactory == nil {
			invoker(response, request)
			return
		}

		// Get the context from the request
		ctx := request.Context()

		// Create a logger using the logger factory
		logger, err := opt.loggerFactory(ctx)
		if err != nil {
			// Log error and continue with request processing if logger creation fails
			slog.Error("accesslog: failed to get logger", slog.String("error", err.Error()))
			invoker(response, request)
			return
		}

		// Record the start time for latency calculation
		startTime := time.Now()

		// Wrap the response writer to capture the status code
		statusCodeResponse := &statusCodeResponseWriter{ResponseWriter: response}

		// Invoke the next handler
		invoker(statusCodeResponse, request)

		// Calculate request processing latency
		latency := time.Since(startTime)

		// Get the route information (uses unsafe reflection)
		route := getRoute(request)

		// Get a reusable slice of slog.Attr from the pool
		fields := pool.Get().([]slog.Attr)

		// Add system identifier
		fields = append(fields, slog.String("system", "server"))

		// Add timestamp when the request started
		fields = append(fields, slog.String("timestamp", startTime.Format(time.RFC3339Nano)))

		// Add deadline information if context has a deadline
		if d, ok := ctx.Deadline(); ok {
			fields = append(fields, slog.String("deadline", d.Format(time.RFC3339Nano)))
		}

		// Add request and response information
		fields = append(fields, slog.String("latency", latency.String()))
		fields = append(fields, slog.String("method", request.Method))
		fields = append(fields, slog.String("uri", request.RequestURI))
		fields = append(fields, slog.String("proto", request.Proto))
		fields = append(fields, slog.String("host", request.Host))
		fields = append(fields, slog.String("remote_address", request.RemoteAddr))
		fields = append(fields, slog.Int("response_status", statusCodeResponse.statusCode))

		// Log the access information
		logger.LogAttrs(ctx, opt.level, route, fields...)

		// Reset the slice length to 0 to reuse the underlying array
		fields = fields[:0]

		// Put the slice back into the pool for reuse
		pool.Put(fields)
	}
}

// statusCodeResponseWriter wraps http.ResponseWriter to capture the status code
type statusCodeResponseWriter struct {
	http.ResponseWriter
	statusCode int // Captured HTTP status code
}

// WriteHeader captures the status code before calling the wrapped WriteHeader
// Parameters:
//   - statusCode: HTTP status code to be written
func (r *statusCodeResponseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// getRoute extracts route information from the HTTP request using reflection
// Parameters:
//   - r: HTTP request
//
// Returns:
//   - string: Route pattern or request URI
//
// Note: This function uses unsafe reflection to access private fields and may be unstable
func getRoute(r *http.Request) string {
	// Recover from potential panics during reflection
	defer func() {
		if p := recover(); p != nil {
			slog.Error("accesslog: failed to get route", slog.String("stack", string(debug.Stack())))
		}
	}()

	// Use reflection to access private fields of the request (unsafe)
	strVal := reflect.ValueOf(r).Elem().FieldByName("pat").Elem().FieldByName("str")
	return reflect.NewAt(strVal.Type(), unsafe.Pointer(strVal.UnsafeAddr())).Elem().String()
}
