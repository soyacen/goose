package cors

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/soyacen/goose/server"
)

func assertEqual[T comparable](t *testing.T, want, got T, msg string) {
	t.Helper()
	if want != got {
		t.Errorf("%s: want %v, got %v", msg, want, got)
	}
}

func assertEmpty(t *testing.T, got, msg string) {
	t.Helper()
	if got != "" {
		t.Errorf("%s: expected empty, got %q", msg, got)
	}
}

func assertContains(t *testing.T, s, substr, msg string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("%s: expected %q to contain %q", msg, s, substr)
	}
}

func assertNotContains(t *testing.T, s, substr, msg string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("%s: expected %q to not contain %q", msg, s, substr)
	}
}

func TestServer_DefaultOptions(t *testing.T) {
	mdw := Server()

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	server.Invoke(mdw, rec, req, finalHandler, nil)

	assertEqual(t, http.StatusOK, rec.Code, "status code")
	assertEqual(t, "*", rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	assertEqual(t, "Origin", rec.Header().Get("Vary"), "vary")
}

func TestServer_AllowedOrigins(t *testing.T) {
	mdw := Server(AllowedOrigins([]string{"https://example.com", "https://app.example.com"}))

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("allowed_origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEqual(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})

	t.Run("disallowed_origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://evil.com")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})

	t.Run("case_insensitive", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "HTTPS://EXAMPLE.COM")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEqual(t, "HTTPS://EXAMPLE.COM", rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})
}

func TestServer_WildcardOrigins(t *testing.T) {
	mdw := Server(AllowedOrigins([]string{"https://*.example.com"}))

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("matching_subdomain", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://app.example.com")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEqual(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})

	t.Run("non_matching_domain", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://evil.com")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})

	t.Run("prefix_too_short", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://.example.com")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		// Note: https://.example.com matches https://*.example.com per rs/cors wildcard behavior
		assertEqual(t, "https://.example.com", rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})
}

func TestServer_AllowOriginFunc(t *testing.T) {
	mdw := Server(AllowOriginFunc(func(r *http.Request, origin string) bool {
		return r.Method == http.MethodGet && origin == "https://example.com"
	}))

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("allowed_by_request_func", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEqual(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})

	t.Run("disallowed_by_request_func", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})
}

func TestServer_AllowedMethods(t *testing.T) {
	mdw := Server(
		AllowedMethods([]string{http.MethodGet, http.MethodPost, http.MethodDelete}),
		AllowedOrigins([]string{"https://example.com"}),
	)

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("allowed_method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEqual(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})

	t.Run("disallowed_method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})
}

func TestServer_Preflight(t *testing.T) {
	mdw := Server(
		AllowedOrigins([]string{"https://example.com"}),
		AllowedMethods([]string{http.MethodGet, http.MethodPost, http.MethodDelete}),
		AllowedHeaders([]string{"Content-Type", "X-Custom-Header"}),
		MaxAge(time.Hour),
	)

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("valid_preflight", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEqual(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
		assertEqual(t, "POST", rec.Header().Get("Access-Control-Allow-Methods"), "allow-methods")
		assertEqual(t, "Content-Type", rec.Header().Get("Access-Control-Allow-Headers"), "allow-headers")
		assertEqual(t, "3600", rec.Header().Get("Access-Control-Max-Age"), "max-age")
	})

	t.Run("preflight_disallowed_method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodPatch)
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})

	t.Run("preflight_disallowed_header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		req.Header.Set("Access-Control-Request-Headers", "X-Unknown-Header")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})

	t.Run("preflight_disallowed_origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://evil.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})

	t.Run("preflight_empty_origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})
}

func TestServer_AllowedHeadersWildcard(t *testing.T) {
	mdw := Server(
		AllowedOrigins([]string{"https://example.com"}),
		AllowedHeaders([]string{"*"}),
	)

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "X-Any-Header, X-Another-Header")
	rec := httptest.NewRecorder()
	server.Invoke(mdw, rec, req, finalHandler, nil)

	assertEqual(t, http.StatusOK, rec.Code, "status code")
	assertEqual(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	assertEqual(t, "X-Any-Header, X-Another-Header", rec.Header().Get("Access-Control-Allow-Headers"), "allow-headers")
}

func TestServer_ExposedHeaders(t *testing.T) {
	mdw := Server(
		AllowedOrigins([]string{"*"}),
		ExposedHeaders([]string{"X-Request-Id", "X-Trace-Id"}),
	)

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	server.Invoke(mdw, rec, req, finalHandler, nil)

	assertEqual(t, http.StatusOK, rec.Code, "status code")
	assertEqual(t, "X-Request-Id, X-Trace-Id", rec.Header().Get("Access-Control-Expose-Headers"), "expose-headers")
}

func TestServer_AllowCredentials(t *testing.T) {
	mdw := Server(
		AllowedOrigins([]string{"https://example.com"}),
		AllowCredentials(),
	)

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("actual_request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEqual(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"), "allow-credentials")
	})

	t.Run("preflight_request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEqual(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"), "allow-credentials")
	})
}

func TestServer_AllowPrivateNetwork(t *testing.T) {
	mdw := Server(
		AllowedOrigins([]string{"https://example.com"}),
		AllowPrivateNetwork(),
	)

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("private_network_requested", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		req.Header.Set("Access-Control-Request-Private-Network", "true")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEqual(t, "true", rec.Header().Get("Access-Control-Allow-Private-Network"), "allow-private-network")
	})

	t.Run("private_network_not_requested", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEmpty(t, rec.Header().Get("Access-Control-Allow-Private-Network"), "allow-private-network")
	})
}

func TestServer_VaryHeader(t *testing.T) {
	mdw := Server(
		AllowedOrigins([]string{"https://example.com"}),
		AllowPrivateNetwork(),
	)

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("actual_request_vary", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, "Origin", rec.Header().Get("Vary"), "vary")
	})

	t.Run("preflight_vary_with_private_network", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		vary := rec.Header()["Vary"]
		if len(vary) != 1 {
			t.Fatalf("expected 1 vary header value, got %d", len(vary))
		}
		assertContains(t, vary[0], "Origin", "vary")
		assertContains(t, vary[0], "Access-Control-Request-Method", "vary")
		assertContains(t, vary[0], "Access-Control-Request-Headers", "vary")
		assertContains(t, vary[0], "Access-Control-Request-Private-Network", "vary")
	})

	t.Run("preflight_vary_without_private_network", func(t *testing.T) {
		mdwNoPrivate := Server(AllowedOrigins([]string{"https://example.com"}))

		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		rec := httptest.NewRecorder()
		server.Invoke(mdwNoPrivate, rec, req, finalHandler, nil)

		vary := rec.Header()["Vary"]
		if len(vary) != 1 {
			t.Fatalf("expected 1 vary header value, got %d", len(vary))
		}
		assertContains(t, vary[0], "Origin", "vary")
		assertContains(t, vary[0], "Access-Control-Request-Method", "vary")
		assertContains(t, vary[0], "Access-Control-Request-Headers", "vary")
		assertNotContains(t, vary[0], "Access-Control-Request-Private-Network", "vary")
	})
}

func TestServer_EmptyOrigin(t *testing.T) {
	mdw := Server(AllowedOrigins([]string{"*"}))

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("actual_request_no_origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})

	t.Run("preflight_no_origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		rec := httptest.NewRecorder()
		server.Invoke(mdw, rec, req, finalHandler, nil)

		assertEqual(t, http.StatusOK, rec.Code, "status code")
		assertEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
	})
}

func TestServer_OptionsWithoutPreflight(t *testing.T) {
	mdw := Server(AllowedOrigins([]string{"*"}))

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// OPTIONS request without Access-Control-Request-Method is treated as actual request
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	server.Invoke(mdw, rec, req, finalHandler, nil)

	assertEqual(t, http.StatusOK, rec.Code, "status code")
	assertEqual(t, "*", rec.Header().Get("Access-Control-Allow-Origin"), "allow-origin")
}

func TestWildcard_Match(t *testing.T) {
	tests := []struct {
		name    string
		w       wildcard
		origin  string
		allowed bool
	}{
		{"exact_prefix_suffix", wildcard{prefix: "https://", suffix: ".example.com"}, "https://app.example.com", true},
		{"multiple_subdomains", wildcard{prefix: "https://", suffix: ".example.com"}, "https://api.v1.example.com", true},
		{"no_match_wrong_domain", wildcard{prefix: "https://", suffix: ".example.com"}, "https://evil.com", false},
		// Note: https://.example.com matches https://*.example.com per rs/cors wildcard behavior
		{"zero_chars_between", wildcard{prefix: "https://", suffix: ".example.com"}, "https://.example.com", true},
		{"prefix_only", wildcard{prefix: "https://", suffix: ""}, "https://anything", true},
		{"suffix_only", wildcard{prefix: "", suffix: ".example.com"}, "app.example.com", true},
		{"empty_wildcard", wildcard{prefix: "", suffix: ""}, "anything", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.w.match(tt.origin); got != tt.allowed {
				t.Errorf("wildcard{%q, %q}.match(%q) = %v, want %v", tt.w.prefix, tt.w.suffix, tt.origin, got, tt.allowed)
			}
		})
	}
}
