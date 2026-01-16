package recovery

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/soyacen/goose/server"
)

type options struct {
	handler HandlerFunc
}
type Option func(*options)

func defaultOptions() *options {
	return &options{
		handler: defaultHandler,
	}
}

func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request, p any)

// RecoveryHandler customizes the function for recovering from a panic.
func RecoveryHandler(f HandlerFunc) Option {
	return func(o *options) {
		o.handler = f
	}
}

func Server(opts ...Option) server.Middleware {
	opt := defaultOptions().apply(opts...)
	return func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
		defer func() {
			p := recover()
			if p == nil {
				return
			}
			opt.handler(response, request, p)
		}()
		invoker(response, request)
	}
}

func defaultHandler(response http.ResponseWriter, request *http.Request, p any) {
	slog.ErrorContext(request.Context(), "panic caught", "panic", p, "stack", string(debug.Stack()))
}
