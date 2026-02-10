package bbr

import (
	"net/http"

	"github.com/soyacen/goose/server"
)

// statusCodeResponseWriter wraps http.ResponseWriter to capture the status code
// This allows us to track the actual HTTP status code returned by handlers
// for more accurate rate limiting statistics
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

// Server creates a server-side BBR (ByteDance BBR) rate limiting middleware
// Parameters:
//   - opts: Variable number of Option functions for configuration
//
// Returns:
//   - server.Middleware: Server middleware function
//
// Behavior:
//  1. Uses BBR algorithm to determine if request should be allowed
//  2. Returns 429 Too Many Requests when rate limit is exceeded
//  3. Tracks request completion with actual response status codes for adaptive rate limiting
func Server(opts ...Option) server.Middleware {
	limiter := defaultOptions().apply(opts...).init().newLimiter()
	return func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
		done, err := limiter.Allow()
		if err != nil {
			// 请求被限流，返回429状态码
			response.Header().Set("Content-Type", "text/plain; charset=utf-8")
			response.WriteHeader(http.StatusTooManyRequests)
			_, _ = response.Write([]byte("rate limit exceeded"))
			return
		}

		// Wrap the response writer to capture the status code
		statusCodeResponse := &statusCodeResponseWriter{ResponseWriter: response}

		// Invoke the next handler
		invoker(statusCodeResponse, request)

		// 确保状态码被正确设置
		// 如果既没有调用WriteHeader也没有调用Write，默认为200 OK
		if statusCodeResponse.statusCode == 0 {
			statusCodeResponse.statusCode = http.StatusOK
		}

		// 调用完成回调，传递实际的响应状态码
		done(DoneInfo{Status: statusCodeResponse.statusCode})
	}
}
