package goose

import (
	"context"
	"net"
	"net/http"
	"strings"
)

// CopyHeader copies all header values from the source header to the target header.
// It preserves all existing values in the target header and adds the source header values.
//
// Parameters:
//   - tgt: The target http.Header to which values will be copied
//   - src: The source http.Header from which values will be copied
func CopyHeader(tgt http.Header, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			tgt.Add(key, value)
		}
	}
}

// headerKey is the key type used to store and retrieve http.Header from a context.
// Using an unexported empty struct ensures the key is unique and collision-free.
type headerKey struct{}

// ExtractHeader retrieves the http.Header value stored in the context.
//
// Returns:
//   - The http.Header value if present
//   - true if the header was found, false otherwise
func ExtractHeader(ctx context.Context) (http.Header, bool) {
	val, ok := ctx.Value(headerKey{}).(http.Header)
	return val, ok
}

// InjectHeader stores the provided http.Header into the context and returns the updated context.
//
// Parameters:
//   - ctx: The parent context
//   - header: The http.Header to store in the context
func InjectHeader(ctx context.Context, header http.Header) context.Context {
	return context.WithValue(ctx, headerKey{}, header)
}

// ClientIP extracts the client's IP address from an HTTP request.
// It checks common proxy and CDN headers in the following order:
//   1. X-Forwarded-For (first/leftmost IP only)
//   2. X-Real-Ip
//   3. X-Client-Ip
//   4. Cf-Connecting-Ip (Cloudflare)
//   5. True-Client-Ip (Akamai / Cloudflare)
//
// If none of the above headers are present, it falls back to RemoteAddr.
// For X-Forwarded-For, only the first (leftmost) IP is returned,
// as subsequent entries typically represent proxy hops.
func ClientIP(req *http.Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		if before, _, found := strings.Cut(xff, ","); found {
			return strings.TrimSpace(before)
		}
		return xff
	}

	if xri := req.Header.Get("X-Real-Ip"); xri != "" {
		return xri
	}

	if xci := req.Header.Get("X-Client-Ip"); xci != "" {
		return xci
	}

	if cf := req.Header.Get("Cf-Connecting-Ip"); cf != "" {
		return cf
	}

	if tci := req.Header.Get("True-Client-Ip"); tci != "" {
		return tci
	}

	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return ip
}
