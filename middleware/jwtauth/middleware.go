// Package jwtauth provides JWT (JSON Web Token) authentication middleware for both server and client
package jwtauth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/soyacen/goose/client"
	"github.com/soyacen/goose/server"
	"github.com/golang-jwt/jwt/v5"
)

// ClaimsFunc is a function type that generates JWT claims from a context
// Parameters:
//   - ctx: Context containing information for generating claims
//
// Returns:
//   - jwt.Claims: Claims to be included in the JWT
//   - error: Error if claims generation fails
type ClaimsFunc func(ctx context.Context) (jwt.Claims, error)

// ctxKey is a private type used as a key for storing JWT tokens in context
type ctxKey struct{}

// FromContext retrieves a JWT token from the context
// Parameters:
//   - ctx: Context that may contain a JWT token
//
// Returns:
//   - *jwt.Token: Pointer to the JWT token if found
//   - bool: True if token was found, false otherwise
func FromContext(ctx context.Context) (*jwt.Token, bool) {
	v, ok := ctx.Value(ctxKey{}).(*jwt.Token)
	return v, ok
}

// options holds configuration options for the JWT middleware
type options struct {
	realm         string             // Authentication realm for WWW-Authenticate header
	parserOptions []jwt.ParserOption // Options for JWT parsing
	tokenOptions  []jwt.TokenOption  // Options for JWT token creation
	signingMethod jwt.SigningMethod  // Method used for signing JWT tokens
}

// defaultOptions returns the default configuration options
// Returns:
//   - *options: Default options with "Authorization Required" realm and HS512 signing method
func defaultOptions() *options {
	return &options{
		realm:         "Authorization Required",
		signingMethod: jwt.SigningMethodHS512,
	}
}

// apply applies the given options to the options struct
// Parameters:
//   - opts: Variable number of Option functions
//
// Returns:
//   - *options: Pointer to the updated options struct
func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Option is a function type for configuring JWT middleware options
type Option func(o *options)

// Realm sets the authentication realm
// Parameters:
//   - realm: Realm string for WWW-Authenticate header
//
// Returns:
//   - Option: Function to set the realm option
func Realm(realm string) Option {
	return func(o *options) {
		o.realm = realm
	}
}

// ParserOptions sets the JWT parser options
// Parameters:
//   - opts: Variable number of jwt.ParserOption
//
// Returns:
//   - Option: Function to set the parser options
func ParserOptions(opts ...jwt.ParserOption) Option {
	return func(o *options) {
		o.parserOptions = append(o.parserOptions, opts...)
	}
}

// TokenOptions sets the JWT token options
// Parameters:
//   - opts: Variable number of jwt.TokenOption
//
// Returns:
//   - Option: Function to set the token options
func TokenOptions(opts ...jwt.TokenOption) Option {
	return func(o *options) {
		o.tokenOptions = append(o.tokenOptions, opts...)
	}
}

// SigningMethod sets the JWT signing method
// Parameters:
//   - method: JWT signing method
//
// Returns:
//   - Option: Function to set the signing method option
func SigningMethod(method jwt.SigningMethod) Option {
	return func(o *options) {
		o.signingMethod = method
	}
}

// Server creates a server-side JWT authentication middleware
// Parameters:
//   - keyFunc: Function to provide the key for validating JWT signatures
//   - opts: Variable number of Option functions for configuration
//
// Returns:
//   - server.Middleware: Server middleware function
//
// Behavior:
//  1. Parses the Authorization header for a Bearer token
//  2. Validates the JWT using the provided key function
//  3. Stores the validated token in the request context
//  4. Returns 401 Unauthorized for invalid or missing tokens
func Server(keyFunc jwt.Keyfunc, opts ...Option) server.Middleware {
	opt := defaultOptions().apply(opts...)
	realm := "Basic realm=" + strconv.Quote(opt.realm)
	return func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
		// Parse the Authorization header for a Bearer token
		tokenString, found := parseAuthorization(request.Header.Get("Authorization"))
		if !found {
			// Return 401 if no valid Bearer token is found
			response.Header().Set("WWW-Authenticate", realm)
			response.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Parse and validate the JWT token
		token, err := jwt.Parse(tokenString, keyFunc, opt.parserOptions...)
		if err != nil {
			// Return 401 if token parsing fails
			response.Header().Set("WWW-Authenticate", realm)
			response.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Check if token is valid
		if !token.Valid {
			// Return 401 if token is invalid
			response.Header().Set("WWW-Authenticate", realm)
			response.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Store the validated token in the request context
		request = request.WithContext(context.WithValue(request.Context(), ctxKey{}, token))

		// Invoke the next handler
		invoker(response, request)
	}
}

// Client creates a client-side JWT authentication middleware
// Parameters:
//   - claimsFunc: Function to generate claims for the JWT
//   - keyFunc: Function to provide the key for signing the JWT
//   - opts: Variable number of Option functions for configuration
//
// Returns:
//   - client.Middleware: Client middleware function
//
// Behavior:
//  1. Generates claims using the provided claims function
//  2. Creates and signs a new JWT with the claims
//  3. Adds the JWT to the Authorization header as a Bearer token
func Client(claimsFunc ClaimsFunc, keyFunc jwt.Keyfunc, opts ...Option) client.Middleware {
	opt := defaultOptions().apply(opts...)
	return func(cli *http.Client, request *http.Request, invoker client.Invoker) (*http.Response, error) {
		// Get context from request
		ctx := request.Context()

		// Generate claims from the context
		claims, err := claimsFunc(ctx)
		if err != nil {
			return nil, err
		}

		// Create a new JWT with the claims
		token := jwt.NewWithClaims(opt.signingMethod, claims, opt.tokenOptions...)

		// Get the key for signing the token
		key, err := keyFunc(token)
		if err != nil {
			return nil, err
		}

		// Sign the token with the key
		tokenString, err := token.SignedString(key)
		if err != nil {
			return nil, err
		}

		// Add the signed token to the Authorization header
		request.Header.Set("Authorization", generateAuthorization(tokenString))

		// Invoke the next handler
		return invoker(cli, request)
	}
}

// parseAuthorization parses the Authorization header to extract a Bearer token
// Parameters:
//   - authorization: Authorization header value
//
// Returns:
//   - string: Extracted token string
//   - bool: True if a valid Bearer token was found, false otherwise
func parseAuthorization(authorization string) (string, bool) {
	// Check if the header starts with "Bearer "
	if !strings.HasPrefix(authorization, "Bearer ") {
		return "", false
	}
	// Extract and return the token part
	return authorization[len("Bearer "):], true
}

// generateAuthorization generates an Authorization header value for a Bearer token
// Parameters:
//   - tokenString: JWT token string
//
// Returns:
//   - string: Authorization header value in "Bearer <token>" format
func generateAuthorization(tokenString string) string {
	return fmt.Sprintf("Bearer %s", tokenString)
}
