package jwt_example

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/soyacen/goose"
	"github.com/soyacen/goose/client"
	"github.com/soyacen/goose/example/user"
	"github.com/soyacen/goose/middleware/jwtauth"
	"github.com/soyacen/goose/middleware/recovery"
	"github.com/soyacen/goose/server"
)

// ============================================================================
// Configuration Constants
// ============================================================================

var (
	// jwtSecret is the secret key used for signing and validating JWT tokens
	// In production, use environment variables or secure secret management
	jwtSecret = []byte("your-256-bit-secret-key-change-in-production")
)

// ============================================================================
// Server-Side Implementation
// ============================================================================

// newJWTServerMiddleware creates JWT authentication middleware for the server
// Parameters:
//   - keyFunc: Function that validates JWT signatures
//
// Returns:
//   - server.Middleware: Middleware for JWT validation
func newJWTServerMiddleware() server.Middleware {
	// KeyFunc validates the JWT token signature
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		// Verify signing method to prevent algorithm switching attacks
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	}

	return jwtauth.Server(keyFunc)
}

// GetUserHandler demonstrates how to access JWT claims in a handler
// Parameters:
//   - ctx: Context containing the validated JWT token
//   - request: The incoming request
//
// Returns:
//   - *user.GetUserResponse: The response
//   - error: Any error that occurred
func GetUserHandler(ctx context.Context, request *user.GetUserRequest) (*user.GetUserResponse, error) {
	// Retrieve the validated token from context
	token, ok := jwtauth.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("JWT token not found in context")
	}

	// Extract claims from the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to extract JWT claims")
	}

	// Access specific claims
	userID := claims["sub"]
	userName := claims["name"]
	issuedAt := claims["iat"]
	expiresAt := claims["exp"]

	fmt.Printf("Authenticated request from user: %s (%s)\n", userName, userID)
	fmt.Printf("Token issued at: %v, expires at: %v\n", issuedAt, expiresAt)

	// Your business logic here
	return &user.GetUserResponse{
		Id:   request.Id,
		Name: "John Doe",
		Age:  30,
	}, nil
}

// NewServer creates and configures a new goose server with JWT authentication
// Returns:
//   - *goose.Server: Configured server instance
func NewServer() *goose.Server {
	return goose.NewServer(
		user.UserServiceDesc,
		server.WithAddress(":8080"),
		server.Middlewares(
			recovery.Server(),              // Recovery middleware for panic handling
			newJWTServerMiddleware(),       // JWT authentication middleware
		),
	)
}

// ============================================================================
// Client-Side Implementation
// ============================================================================

// JWTClaimsFunc generates JWT claims for client requests
// Parameters:
//   - ctx: Context for the request
//
// Returns:
//   - jwt.Claims: Claims to include in the JWT token
//   - error: Any error during claims generation
func JWTClaimsFunc(ctx context.Context) (jwt.Claims, error) {
	return jwt.MapClaims{
		"sub":  "user123",                    // Subject (user identifier)
		"name": "John Doe",                   // User's name
		"iat":  time.Now().Unix(),            // Issued at timestamp
		"exp":  time.Now().Add(time.Hour).Unix(), // Expiration timestamp (1 hour)
		"role": "admin",                      // Custom claim for user role
	}, nil
}

// SigningKeyFunc provides the key for signing JWT tokens
// Parameters:
//   - token: The JWT token to sign
//
// Returns:
//   - interface{}: The signing key
//   - error: Any error during key retrieval
func SigningKeyFunc(token *jwt.Token) (interface{}, error) {
	return jwtSecret, nil
}

// NewClient creates and configures a new goose client with JWT authentication
// Returns:
//   - *goose.Client: Configured client instance
func NewClient() *goose.Client {
	return goose.NewClient(
		user.UserServiceDesc,
		client.WithURL("http://localhost:8080"),
		client.Middlewares(
			jwtauth.Client(JWTClaimsFunc, SigningKeyFunc), // JWT authentication middleware
		),
	)
}

// ============================================================================
// Advanced Examples
// ============================================================================

// ExampleCustomRealm demonstrates custom realm configuration
func ExampleCustomRealm() server.Middleware {
	return jwtauth.Server(
		func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		},
		jwtauth.Realm("Protected API"), // Custom realm for WWW-Authenticate header
	)
}

// ExampleHS256 demonstrates using HS256 signing method
func ExampleHS256() (server.Middleware, client.Middleware) {
	// Server middleware with HS256
	serverMiddleware := jwtauth.Server(
		func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		},
		jwtauth.SigningMethod(jwt.SigningMethodHS256),
	)

	// Client middleware with HS256
	clientMiddleware := jwtauth.Client(
		JWTClaimsFunc,
		SigningKeyFunc,
		jwtauth.SigningMethod(jwt.SigningMethodHS256),
	)

	return serverMiddleware, clientMiddleware
}

// ExampleWithIssuerValidation demonstrates issuer and audience validation
func ExampleWithIssuerValidation() server.Middleware {
	return jwtauth.Server(
		func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		},
		jwtauth.ParserOptions(
			jwt.WithIssuer("my-api"),              // Validate issuer claim
			jwt.WithAudience("my-client-app"),     // Validate audience claim
		),
	)
}

// ============================================================================
// Test Helpers
// ============================================================================

// GenerateTestToken creates a JWT token for testing purposes
// Parameters:
//   - subject: The subject (user ID)
//   - name: The user's name
//   - expiresIn: Token expiration duration
//
// Returns:
//   - string: The generated JWT token
//   - error: Any error during token generation
func GenerateTestToken(subject, name string, expiresIn time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub": subject,
		"name": name,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(expiresIn).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString(jwtSecret)
}

// ValidateTestToken validates a JWT token for testing purposes
// Parameters:
//   - tokenString: The JWT token string to validate
//
// Returns:
//   - *jwt.Token: The validated token
//   - error: Any validation error
func ValidateTestToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})
}
