package goose

import (
	"context"
	"net/http"
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
