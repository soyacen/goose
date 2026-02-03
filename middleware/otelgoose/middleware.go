package otelgoose

import (
	"context"
	"net/http"

	"github.com/soyacen/goose"
	"github.com/soyacen/goose/client"
	"github.com/soyacen/goose/server"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

func ExtractTraceId(ctx context.Context) (string, bool) {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String(), true
	}
	return "", false
}

func Server() server.Middleware {
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

func Client() client.Middleware {
	return func(cli *http.Client, request *http.Request, invoker client.Invoker) (*http.Response, error) {
		cli.Transport = otelhttp.NewTransport(cli.Transport)
		return invoker(cli, request)
	}
}
