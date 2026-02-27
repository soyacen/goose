# JWT Authentication Middleware for Goose Service

## Overview

The goose framework already includes a JWT authentication middleware package (`github.com/soyacen/goose/middleware/jwtauth`) that provides both server-side and client-side JWT authentication. This document explains how to configure and use it.

## How JWT Authentication Works in Goose

### Authentication Flow

1. **Client Side**:
   - Client generates JWT claims from context
   - Signs the JWT with a secret key
   - Adds the token to the Authorization header as a Bearer token
   - Sends the request to the server

2. **Server Side**:
   - Server extracts the Bearer token from the Authorization header
   - Validates the JWT signature using a key function
   - Stores the validated token in the request context
   - Passes control to the next handler or returns 401 Unauthorized

### Key Components

1. **Server Middleware**: Validates incoming JWT tokens
2. **Client Middleware**: Adds JWT tokens to outgoing requests
3. **Key Function**: Provides the secret/public key for signing/validation
4. **Claims Function**: Generates custom claims for JWT creation

## Configuration

### Server-Side Configuration

```go
import (
    "github.com/soyacen/goose/middleware/jwtauth"
    "github.com/golang-jwt/jwt/v5"
)

// Define your secret key
var jwtSecret = []byte("your-256-bit-secret-key")

// KeyFunc validates JWT signatures
func keyFunc(token *jwt.Token) (interface{}, error) {
    // Verify signing method
    if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
        return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
    }
    return jwtSecret, nil
}

// Create JWT auth middleware for server
jwtServerMiddleware := jwtauth.Server(keyFunc)

// Apply to server options
serverOpts := []server.Option{
    server.Middlewares(jwtServerMiddleware),
}
```

### Client-Side Configuration

```go
import (
    "github.com/soyacen/goose/middleware/jwtauth"
    "github.com/golang-jwt/jwt/v5"
    "context"
    "time"
)

// Define your secret key
var jwtSecret = []byte("your-256-bit-secret-key")

// ClaimsFunc generates JWT claims
func claimsFunc(ctx context.Context) (jwt.Claims, error) {
    return jwt.MapClaims{
        "sub":  "user123",
        "name": "John Doe",
        "iat":  time.Now().Unix(),
        "exp":  time.Now().Add(time.Hour).Unix(),
    }, nil
}

// KeyFunc provides the key for signing
func signingKeyFunc(token *jwt.Token) (interface{}, error) {
    return jwtSecret, nil
}

// Create JWT auth middleware for client
jwtClientMiddleware := jwtauth.Client(claimsFunc, signingKeyFunc)

// Apply to client options
clientOpts := []client.Option{
    client.Middlewares(jwtClientMiddleware),
}
```

## Accessing JWT Claims in Handlers

### Server-Side: Extract Token from Context

```go
import (
    "github.com/soyacen/goose/middleware/jwtauth"
    "github.com/golang-jwt/jwt/v5"
)

// In your handler function
func myHandler(ctx context.Context, request *MyRequest) (*MyResponse, error) {
    // Retrieve the validated token from context
    token, ok := jwtauth.FromContext(ctx)
    if !ok {
        return nil, fmt.Errorf("JWT token not found in context")
    }

    // Extract claims
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, fmt.Errorf("failed to extract claims")
    }

    // Access specific claims
    userID := claims["sub"]
    userName := claims["name"]

    // ... your business logic
}
```

## Complete Server Example

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/soyacen/goose"
    "github.com/soyacen/goose/example/user"
    "github.com/soyacen/goose/middleware/jwtauth"
    "github.com/soyacen/goose/middleware/recovery"
    "github.com/soyacen/goose/server"
    "github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("your-256-bit-secret-key")

func main() {
    // JWT key function for server validation
    keyFunc := func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return jwtSecret, nil
    }

    // Create server with JWT authentication
    srv := goose.NewServer(
        user.UserServiceDesc,
        server.WithAddress(":8080"),
        server.Middlewares(
            recovery.Server(),              // Recovery middleware
            jwtauth.Server(keyFunc),       // JWT auth middleware
        ),
    )

    if err := srv.Start(); err != nil {
        panic(err)
    }
}
```

## Complete Client Example

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/soyacen/goose"
    "github.com/soyacen/goose/client"
    "github.com/soyacen/goose/example/user"
    "github.com/soyacen/goose/middleware/jwtauth"
    "github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("your-256-bit-secret-key")

func main() {
    // Claims function for generating JWT
    claimsFunc := func(ctx context.Context) (jwt.Claims, error) {
        return jwt.MapClaims{
            "sub":  "user123",
            "name": "John Doe",
            "iat":  time.Now().Unix(),
            "exp":  time.Now().Add(time.Hour).Unix(),
        }, nil
    }

    // Key function for signing JWT
    signingKeyFunc := func(token *jwt.Token) (interface{}, error) {
        return jwtSecret, nil
    }

    // Create client with JWT authentication
    cli := goose.NewClient(
        user.UserServiceDesc,
        client.WithURL("http://localhost:8080"),
        client.Middlewares(
            jwtauth.Client(claimsFunc, signingKeyFunc),
        ),
    )

    // Make authenticated request
    resp, err := cli.GetUser(context.Background(), &user.GetUserRequest{Id: "123"})
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    fmt.Printf("Response: %v\n", resp)
}
```

## Advanced Configuration Options

### Custom Realm

```go
jwtServerMiddleware := jwtauth.Server(keyFunc,
    jwtauth.Realm("My API"),
)
```

### Custom Signing Method

```go
// Use HS256 instead of default HS512
jwtServerMiddleware := jwtauth.Server(keyFunc,
    jwtauth.SigningMethod(jwt.SigningMethodHS256),
)

jwtClientMiddleware := jwtauth.Client(claimsFunc, signingKeyFunc,
    jwtauth.SigningMethod(jwt.SigningMethodHS256),
)
```

### Custom Parser Options

```go
jwtServerMiddleware := jwtauth.Server(keyFunc,
    jwtauth.ParserOptions(
        jwt.WithIssuer("my-issuer"),
        jwt.WithAudience("my-audience"),
    ),
)
```

## Security Best Practices

1. **Use Strong Secrets**: Use at least 256-bit secret keys for HMAC algorithms
2. **Set Expiration**: Always set token expiration (`exp` claim)
3. **Validate Claims**: Use `jwt.WithIssuer()` and `jwt.WithAudience()` options
4. **Use HTTPS**: Always use HTTPS in production to protect tokens in transit
5. **Rotate Secrets**: Implement key rotation for production systems
6. **Error Handling**: Don't expose detailed error messages to clients

## Error Responses

When JWT validation fails, the server returns:
- **401 Unauthorized** status code
- **WWW-Authenticate** header with realm information
- No body content

Example error response:
```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Basic realm="Authorization Required"
```

## Summary

The goose framework's JWT middleware provides:
- Simple server-side token validation
- Simple client-side token generation
- Extensible options for customization
- Context integration for accessing claims in handlers
- Integration with the middleware chain via `server.Middlewares()` and `client.Middlewares()`
