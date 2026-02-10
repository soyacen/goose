package bbr

import (
	"net/http"

	"github.com/soyacen/goose/server"
)

func Server(opts ...Option) server.Middleware {
	limiter := defaultOptions().apply(opts...).init().newLimiter()
	return func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
		done, err := limiter.Allow()
		if err != nil {
			// 请求被限流，直接返回错误
			return nil, err
		}
		// Invoke the next handler
		invoker(response, request)
	}
}
