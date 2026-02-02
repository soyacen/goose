package otelgoose

import (
	"context"
	"net/http"

	"github.com/soyacen/goose"
	"github.com/soyacen/goose/client"
	"github.com/soyacen/goose/server"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type ContextFunc func(ctx context.Context) context.Context

func Server(contextFunc ContextFunc) server.Middleware {
	return func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
		var operation string
		routeInfo, ok := goose.ExtractRouteInfo(request.Context())
		if ok {
			operation = routeInfo.Pattern
		} else {
			operation = request.URL.Path
		}
		otelhttp.NewHandler(invoker, operation).ServeHTTP(response, request)
	}
}

func Client(contextFunc ContextFunc) client.Middleware {
	return func(cli *http.Client, request *http.Request, invoker client.Invoker) (*http.Response, error) {
		cli.Transport = otelhttp.NewTransport(cli.Transport)
		return invoker(cli, request)
	}
}
