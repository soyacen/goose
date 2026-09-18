// Package cors provides HTTP Cross-Origin Resource Sharing (CORS) middleware.
//
// The middleware handles both preflight (OPTIONS) and actual CORS requests
// according to the W3C CORS specification. It supports configurable allowed
// origins, methods, headers, credentials, and private network access.
//
// Basic usage with default options (allows all origins, GET/POST/HEAD methods):
//
//	mdw := cors.Server()
//
// Restrict to specific origins and methods:
//
//	mdw := cors.Server(
//	    cors.AllowedOrigins([]string{"https://example.com"}),
//	    cors.AllowedMethods([]string{http.MethodGet, http.MethodPost}),
//	)
//
// Use wildcard patterns for subdomains:
//
//	mdw := cors.Server(
//	    cors.AllowedOrigins([]string{"https://*.example.com"}),
//	)
package cors

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/soyacen/goose/server"
)

// Server creates a CORS middleware with the given options.
//
// The middleware inspects incoming requests and sets appropriate CORS headers.
// For preflight requests (OPTIONS with Access-Control-Request-Method), it
// validates the origin, method, and headers, then returns the allowed values.
// For actual requests, it sets Access-Control-Allow-Origin and other headers
// if the origin and method are permitted.
//
// Default configuration allows all origins ("*"), methods GET/POST/HEAD,
// and headers Accept, Content-Type, X-Requested-With.
func Server(opts ...Option) server.Middleware {
	opt := defaultOptions().apply(opts...)

	// Process allowed origins
	allowedOriginsAll := false
	var allowedOrigins []string
	var allowedWOrigins []wildcard

	if opt.AllowOriginFunc == nil {
		if len(opt.AllowedOrigins) == 0 {
			allowedOriginsAll = true
		} else {
			for _, origin := range opt.AllowedOrigins {
				origin = strings.ToLower(origin)
				if origin == "*" {
					allowedOriginsAll = true
					allowedOrigins = nil
					allowedWOrigins = nil
					break
				}
				if prefix, suffix, ok := strings.Cut(origin, "*"); ok {
					allowedWOrigins = append(allowedWOrigins, wildcard{prefix, suffix})
				} else {
					allowedOrigins = append(allowedOrigins, origin)
				}
			}
		}
	}

	// Process allowed methods
	allowedMethods := opt.AllowedMethods
	if len(allowedMethods) == 0 {
		allowedMethods = []string{http.MethodGet, http.MethodPost, http.MethodHead}
	}

	// Process allowed headers
	allowedHeadersAll := false
	var allowedHeaders []string
	if len(opt.AllowedHeaders) == 0 {
		allowedHeaders = []string{"accept", "content-type", "x-requested-with"}
	} else {
		allowedHeaders = make([]string, len(opt.AllowedHeaders))
		for i, h := range opt.AllowedHeaders {
			allowedHeaders[i] = strings.ToLower(h)
		}
		if slices.Contains(opt.AllowedHeaders, "*") {
			allowedHeadersAll = true
		}
	}

	// Exposed headers
	var exposedHeaders []string
	if len(opt.ExposedHeaders) > 0 {
		exposedHeaders = make([]string, len(opt.ExposedHeaders))
		for i, h := range opt.ExposedHeaders {
			exposedHeaders[i] = http.CanonicalHeaderKey(h)
		}
	}

	// Max age
	var maxAge string
	if opt.MaxAge > 0 {
		maxAge = strconv.Itoa(int(opt.MaxAge / time.Second))
	}

	// Preflight vary
	preflightVary := "Origin, Access-Control-Request-Method, Access-Control-Request-Headers"
	if opt.AllowPrivateNetwork {
		preflightVary += ", Access-Control-Request-Private-Network"
	}

	// Determine origin function
	var isOriginAllowed func(r *http.Request, origin string) bool
	if opt.AllowOriginFunc != nil {
		isOriginAllowed = func(r *http.Request, origin string) bool {
			return opt.AllowOriginFunc(r, origin)
		}
	} else {
		isOriginAllowed = func(r *http.Request, origin string) bool {
			if allowedOriginsAll {
				return true
			}
			origin = strings.ToLower(origin)
			if slices.Contains(allowedOrigins, origin) {
				return true
			}
			for _, w := range allowedWOrigins {
				if w.match(origin) {
					return true
				}
			}
			return false
		}
	}

	isMethodAllowed := func(method string) bool {
		if method == http.MethodOptions {
			return true
		}
		return slices.Contains(allowedMethods, method)
	}

	return func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
		origin := request.Header.Get("Origin")

		if request.Method == http.MethodOptions && request.Header.Get("Access-Control-Request-Method") != "" {
			// Preflight request
			handlePreflight(response, request, origin, isOriginAllowed, isMethodAllowed, allowedHeadersAll, allowedHeaders, opt.AllowCredentials, opt.AllowPrivateNetwork, maxAge, preflightVary, allowedOriginsAll)
			return
		}

		// Actual request
		handleActualRequest(response, request, origin, isOriginAllowed, isMethodAllowed, opt.AllowCredentials, exposedHeaders, allowedOriginsAll)
		invoker(response, request)
	}
}

func handlePreflight(w http.ResponseWriter, r *http.Request, origin string, isOriginAllowed func(*http.Request, string) bool, isMethodAllowed func(string) bool, allowedHeadersAll bool, allowedHeaders []string, allowCredentials, allowPrivateNetwork bool, maxAge, preflightVary string, allowedOriginsAll bool) {
	headers := w.Header()

	if vary := headers["Vary"]; vary == nil {
		headers["Vary"] = []string{preflightVary}
	} else {
		headers["Vary"] = append(vary, preflightVary)
	}

	if origin == "" {
		return
	}
	if !isOriginAllowed(r, origin) {
		return
	}

	reqMethod := r.Header.Get("Access-Control-Request-Method")
	if !isMethodAllowed(reqMethod) {
		return
	}

	reqHeaders, found := r.Header["Access-Control-Request-Headers"]
	if found && !allowedHeadersAll {
		for _, rh := range reqHeaders {
			for _, h := range strings.Split(rh, ",") {
				h = strings.ToLower(strings.TrimSpace(h))
				if h == "" {
					continue
				}
				if !slices.Contains(allowedHeaders, h) {
					return
				}
			}
		}
	}

	if allowedOriginsAll {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		headers.Set("Access-Control-Allow-Origin", origin)
	}
	headers.Set("Access-Control-Allow-Methods", reqMethod)
	if found && len(reqHeaders) > 0 && reqHeaders[0] != "" {
		headers.Set("Access-Control-Allow-Headers", strings.Join(reqHeaders, ", "))
	}
	if allowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
	if allowPrivateNetwork && r.Header.Get("Access-Control-Request-Private-Network") == "true" {
		headers.Set("Access-Control-Allow-Private-Network", "true")
	}
	if maxAge != "" {
		headers.Set("Access-Control-Max-Age", maxAge)
	}
}

func handleActualRequest(w http.ResponseWriter, r *http.Request, origin string, isOriginAllowed func(*http.Request, string) bool, isMethodAllowed func(string) bool, allowCredentials bool, exposedHeaders []string, allowedOriginsAll bool) {
	headers := w.Header()

	if vary := headers["Vary"]; vary == nil {
		headers["Vary"] = []string{"Origin"}
	} else {
		headers["Vary"] = append(vary, "Origin")
	}

	if origin == "" {
		return
	}
	if !isOriginAllowed(r, origin) {
		return
	}
	if !isMethodAllowed(r.Method) {
		return
	}

	if allowedOriginsAll {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		headers.Set("Access-Control-Allow-Origin", origin)
	}
	if len(exposedHeaders) > 0 {
		headers.Set("Access-Control-Expose-Headers", strings.Join(exposedHeaders, ", "))
	}
	if allowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
}
