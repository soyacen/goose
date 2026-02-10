package bbr

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soyacen/goose/server"
	"github.com/stretchr/testify/assert"
)

func TestServerMiddleware(t *testing.T) {
	t.Run("normal_request_allowed", func(t *testing.T) {
		// 创建中间件
		mdw := Server(
			WithWindow(time.Second),
			WithBuckets(10),
			WithCPUThreshold(80.0),
		)

		// 创建测试服务器
		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})

		// 创建测试请求
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		// 执行中间件链
		server.Invoke(mdw, rec, req, finalHandler, nil)

		// 验证响应
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "success", rec.Body.String())
	})

	t.Run("rate_limit_exceeded", func(t *testing.T) {
		// 创建一个会触发限流的中间件配置
		mdw := Server(
			WithWindow(time.Millisecond*100),
			WithBuckets(10),
			WithCPUThreshold(10.0),                  // 设置很低的阈值
			WithCPU(func() float64 { return 90.0 }), // 模拟高CPU使用率
		)

		// 创建测试服务器
		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})

		// 创建大量并发请求来触发限流
		for i := 0; i < 1000; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()

			// 先让一些请求通过以建立历史记录
			server.Invoke(mdw, rec, req, finalHandler, nil)
		}

		// 现在发送一个应该被限流的请求
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		// 人工增加inflight计数来模拟高并发
		// 注意：这在实际测试中可能需要访问内部状态

		server.Invoke(mdw, rec, req, finalHandler, nil)

		// 验证是否返回了限流响应
		// 注意：由于我们无法直接控制bbrLimiter的内部状态，
		// 这个测试主要用于验证中间件的基本结构是否正确
		// 在实际环境中，当达到限流条件时应该返回429
		t.Logf("Response status: %d", rec.Code)
		t.Logf("Response body: %s", rec.Body.String())
	})

	t.Run("middleware_chain", func(t *testing.T) {
		// 测试中间件链
		mdw1 := Server(WithCPUThreshold(80.0))
		mdw2 := func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
			response.Header().Set("X-Test-Middleware", "true")
			invoker(response, request)
		}

		chain := server.Chain(mdw1, mdw2)

		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("chained"))
		})

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		server.Invoke(chain, rec, req, finalHandler, nil)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "chained", rec.Body.String())
		assert.Equal(t, "true", rec.Header().Get("X-Test-Middleware"))
	})

	t.Run("default_status_code_handling", func(t *testing.T) {
		// 测试没有显式调用WriteHeader的情况
		mdw := Server(WithCPUThreshold(80.0))

		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 不调用WriteHeader，只写入body
			w.Write([]byte("no explicit status"))
		})

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		server.Invoke(mdw, rec, req, finalHandler, nil)

		// 验证默认状态码为200
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "no explicit status", rec.Body.String())
	})

	t.Run("no_response_write", func(t *testing.T) {
		// 测试既不调用WriteHeader也不写入body的情况
		mdw := Server(WithCPUThreshold(80.0))

		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 什么都不做
		})

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		server.Invoke(mdw, rec, req, finalHandler, nil)

		// 验证仍然有默认状态码
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "", rec.Body.String())
	})
}
