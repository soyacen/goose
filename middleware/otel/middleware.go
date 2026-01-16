package context

import (
	"context"
	"net/http"

	"github.com/soyacen/goose/client"
	"github.com/soyacen/goose/server"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type ContextFunc func(ctx context.Context) context.Context

func Server(contextFunc ContextFunc) server.Middleware {
	return func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
		ctx := request.Context()
		xinvoker := otelhttp.NewHandler(invoker, "Hello")

		if contextFunc != nil {
			ctx = contextFunc(ctx)
			request = request.WithContext(ctx)
		}
		invoker(response, request)
	}
}

func Client(contextFunc ContextFunc) client.Middleware {
	return func(cli *http.Client, request *http.Request, invoker client.Invoker) (*http.Response, error) {
		// Get context from request
		ctx := request.Context()
		if contextFunc != nil {
			ctx = contextFunc(ctx)
			request = request.WithContext(ctx)
		}
		return invoker(cli, request)
	}
}
