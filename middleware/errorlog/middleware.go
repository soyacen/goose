// Package errorlog 提供HTTP错误日志记录中间件功能
// 用于记录发生错误的HTTP请求，支持配置是否打印请求和响应
package errorlog

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"

	"github.com/soyacen/goose"
	"github.com/soyacen/goose/client"
	"github.com/soyacen/goose/server"
)

// Server 创建服务器端错误日志中间件
// 该中间件捕获HTTP请求的错误（4xx和5xx状态码），并记录错误日志
//
// 参数:
//   - opts: 可选的配置选项
//
// 返回值:
//   - server.Middleware: HTTP服务器中间件
func Server(opts ...Option) server.Middleware {
	// 获取默认选项并应用用户配置
	o := defaultOptions()
	o.apply(opts...)

	return func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
		// 包装响应写入器以捕获状态码和响应体
		wrappedResponse := &responseWriter{
			ResponseWriter: response,
			statusCode:     http.StatusOK,
			body:           &bytes.Buffer{},
		}

		// 获取请求体（如果配置为打印请求）
		var requestBody []byte
		if o.printRequest && request.Body != nil {
			requestBody, _ = io.ReadAll(request.Body)
			request.Body = io.NopCloser(bytes.NewReader(requestBody))
		}

		// 执行原始处理器
		invoker(wrappedResponse, request)

		// 检查是否发生错误（状态码 >= 400）
		if wrappedResponse.statusCode >= 400 {
			ctx := request.Context()

			// 构建日志属性
			attrs := buildLogAttrs(request, wrappedResponse, requestBody, o)

			// 记录错误日志
			slog.LogAttrs(ctx, slog.LevelError, "HTTP error", attrs...)
		}
	}
}

// Client 创建客户端错误日志中间件
// 该中间件捕获HTTP客户端调用的错误（4xx和5xx状态码），并记录错误日志
//
// 参数:
//   - opts: 可选的配置选项
//
// 返回值:
//   - client.Middleware: HTTP客户端中间件
func Client(opts ...Option) client.Middleware {
	// 获取默认选项并应用用户配置
	o := defaultOptions()
	o.apply(opts...)

	return func(cli *http.Client, request *http.Request, invoker client.Invoker) (*http.Response, error) {
		// 获取请求体（如果配置为打印请求）
		var requestBody []byte
		if o.printRequest && request.Body != nil {
			requestBody, _ = io.ReadAll(request.Body)
			request.Body = io.NopCloser(bytes.NewReader(requestBody))
		}

		// 执行HTTP调用
		response, err := invoker(cli, request)

		// 获取响应体（如果配置为打印响应且响应不为空）
		var responseBody []byte
		if o.printResponse && response != nil && response.Body != nil {
			responseBody, _ = io.ReadAll(response.Body)
			response.Body = io.NopCloser(bytes.NewReader(responseBody))
		}

		// 检查是否发生错误（HTTP错误或非nil错误）
		isError := err != nil || (response != nil && response.StatusCode >= 400)

		if isError {
			ctx := request.Context()

			// 构建日志属性
			attrs := buildClientLogAttrs(request, response, err, requestBody, responseBody, o)

			// 记录错误日志
			slog.LogAttrs(ctx, slog.LevelError, "HTTP client error", attrs...)
		}

		return response, err
	}
}

// responseWriter 包装 http.ResponseWriter 以捕获状态码和响应体
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

// WriteHeader 捕获状态码
func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write 捕获响应体
func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// buildLogAttrs 构建服务器端错误日志属性
func buildLogAttrs(request *http.Request, response *responseWriter, requestBody []byte, opts *options) []slog.Attr {
	ctx := request.Context()
	var route string
	if routeInfo, ok := goose.ExtractRouteInfo(ctx); ok {
		route = routeInfo.Pattern
	} else {
		route = request.URL.Path
	}

	attrs := []slog.Attr{
		slog.String("system", "http.server"),
		slog.String("route", route),
		slog.String("method", request.Method),
		slog.String("path", request.URL.Path),
		slog.Int("status", response.statusCode),
		slog.String("remote_address", request.RemoteAddr),
		slog.String("user_agent", request.Header.Get("User-Agent")),
		slog.String("request_id", request.Header.Get("X-Request-Id")),
	}

	if opts.printRequest && len(requestBody) > 0 {
		attrs = append(attrs, slog.String("request_body", string(requestBody)))
	}

	if opts.printResponse && response.body.Len() > 0 {
		attrs = append(attrs, slog.String("response_body", response.body.String()))
	}

	return attrs
}

// buildClientLogAttrs 构建客户端错误日志属性
func buildClientLogAttrs(request *http.Request, response *http.Response, err error, requestBody, responseBody []byte, opts *options) []slog.Attr {
	ctx := request.Context()
	var route string
	if routeInfo, ok := goose.ExtractRouteInfo(ctx); ok {
		route = routeInfo.Pattern
	} else {
		route = request.URL.Path
	}

	attrs := []slog.Attr{
		slog.String("system", "http.client"),
		slog.String("route", route),
		slog.String("method", request.Method),
		slog.String("url", request.URL.String()),
		slog.String("request_id", request.Header.Get("X-Request-Id")),
	}

	if response != nil {
		attrs = append(attrs, slog.Int("status", response.StatusCode))
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	if opts.printRequest && len(requestBody) > 0 {
		attrs = append(attrs, slog.String("request_body", string(requestBody)))
	}

	if opts.printResponse && len(responseBody) > 0 {
		attrs = append(attrs, slog.String("response_body", string(responseBody)))
	}

	return attrs
}
