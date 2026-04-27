package cors

import (
	"net/http"
	"strings"
	"time"
)

type options struct {
	AllowedOrigins         []string
	AllowOriginFunc func(r *http.Request, origin string) bool
	AllowedMethods         []string
	AllowedHeaders         []string
	ExposedHeaders         []string
	MaxAge                 time.Duration
	AllowCredentials       bool
	AllowPrivateNetwork    bool
}

type Option func(*options)

func defaultOptions() *options {
	return &options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodHead},
		AllowedHeaders: []string{"Accept", "Content-Type", "X-Requested-With"},
		MaxAge:         10 * time.Minute,
	}
}

func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// AllowedOrigins sets the allowed origins. Use ["*"] to allow all origins.
// Wildcards are supported, e.g., "https://*.example.com".
func AllowedOrigins(origins []string) Option {
	return func(o *options) {
		o.AllowedOrigins = origins
	}
}

// AllowOriginFunc sets a function to determine if an origin is allowed based on the request.
func AllowOriginFunc(f func(r *http.Request, origin string) bool) Option {
	return func(o *options) {
		o.AllowOriginFunc = f
	}
}

// AllowedMethods sets the allowed HTTP methods.
func AllowedMethods(methods []string) Option {
	return func(o *options) {
		o.AllowedMethods = methods
	}
}

// AllowedHeaders sets the allowed headers. Use ["*"] to allow all headers.
func AllowedHeaders(headers []string) Option {
	return func(o *options) {
		o.AllowedHeaders = headers
	}
}

// ExposedHeaders sets the headers exposed to the client.
func ExposedHeaders(headers []string) Option {
	return func(o *options) {
		o.ExposedHeaders = headers
	}
}

// MaxAge sets the max age for preflight caching.
func MaxAge(d time.Duration) Option {
	return func(o *options) {
		o.MaxAge = d
	}
}

// AllowCredentials enables credentials in CORS requests.
func AllowCredentials() Option {
	return func(o *options) {
		o.AllowCredentials = true
	}
}

// AllowPrivateNetwork enables private network access.
func AllowPrivateNetwork() Option {
	return func(o *options) {
		o.AllowPrivateNetwork = true
	}
}

type wildcard struct {
	prefix string
	suffix string
}

func (w wildcard) match(origin string) bool {
	return len(origin) >= len(w.prefix)+len(w.suffix) &&
		strings.HasPrefix(origin, w.prefix) &&
		strings.HasSuffix(origin, w.suffix)
}
