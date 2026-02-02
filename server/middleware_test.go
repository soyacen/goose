package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func makeWriteMiddleware(s string, callNext bool) Middleware {
	return func(w http.ResponseWriter, r *http.Request, invoker http.HandlerFunc) {
		_, _ = w.Write([]byte(s))
		if callNext && invoker != nil {
			invoker(w, r)
		}
	}
}

func TestInvokeWithNilMiddlewareCallsFinal(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)

	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("final"))
	})

	Invoke(nil, rec, req, final, nil)

	if rec.Body.String() != "final" {
		t.Fatalf("expected body %q, got %q", "final", rec.Body.String())
	}
}

func TestChainSingleMiddleware(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)

	m := makeWriteMiddleware("A", true)
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Z"))
	})

	chained := Chain(m)
	Invoke(chained, rec, req, final, nil)

	if got := rec.Body.String(); got != "AZ" {
		t.Fatalf("expected body %q, got %q", "AZ", got)
	}
}

func TestChainMultipleMiddlewaresOrder(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)

	m1 := makeWriteMiddleware("1", true)
	m2 := makeWriteMiddleware("2", true)
	m3 := makeWriteMiddleware("3", true)

	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("F"))
	})

	chained := Chain(m1, m2, m3)
	Invoke(chained, rec, req, final, nil)

	if got := rec.Body.String(); got != "123F" {
		t.Fatalf("expected body %q, got %q", "123F", got)
	}
}
