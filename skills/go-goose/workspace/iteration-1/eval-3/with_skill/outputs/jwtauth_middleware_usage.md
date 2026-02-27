# JWT Authentication Middleware - Explanation and Usage Guide

## Overview

The `jwtauth` middleware in Goose provides JWT (JSON Web Token) authentication for both server and client. It validates JWT tokens on incoming requests and can generate JWT tokens for outgoing requests.

## How It Works

### Server-Side Flow

1. **Token Extraction**: The middleware extracts the JWT token from the `Authorization` header (expects `Bearer <token>` format)
2. **Token Validation**: It parses and validates the JWT using a provided `keyFunc` function
3. **Context Storage**: If valid, the token is stored in the request context for access by downstream handlers
4. **Error Response**: Returns `401 Unauthorized` with `WWW-Authenticate` header if token is missing or invalid

### Client-Side Flow

1. **Claims Generation**: Uses a `ClaimsFunc` to generate JWT claims from the context
2. **Token Creation**: Creates a new JWT with the claims and signs it using a `keyFunc`
3. **Header Injection**: Adds the signed token to the `Authorization` header as a Bearer token

## Configuration Options

The middleware supports these configuration options:

| Option | Description | Default |
|--------|-------------|---------|
| `Realm(realm string)` | Authentication realm for WWW-Authenticate header | "Authorization Required" |
| `ParserOptions(opts ...jwt.ParserOption)` | JWT parser options | None |
| `TokenOptions(opts ...jwt.TokenOption)` | JWT token creation options | None |
| `SigningMethod(method jwt.SigningMethod)` | Signing method for tokens | HS512 |

## Server-Side Usage Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/soyacen/goose/example/user/v1"
    "github.com/soyacen/goose/middleware/jwtauth"
    "github.com/soyacen/goose/server"
)

// Define custom claims
type UserClaims struct {
    jwt.RegisteredClaims
    UserID string `json:"user_id"`
    Role   string `json:"role"`
}

type userService struct{}

func (s *userService) CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.CreateUserResponse, error) {
    // You can access the JWT token from context
    token, ok := jwtauth.FromContext(ctx)
    if ok {
        fmt.Printf("Authenticated user: %v\n", token)
    }

    return &user.CreateUserResponse{
        Item: &user.UserItem{Id: 1, Name: req.Name},
    }, nil
}

func (s *userService) GetUser(ctx context.Context, req *user.GetUserRequest) (*user.GetUserResponse, error) {
    return &user.GetUserResponse{
        Item: &user.UserItem{Id: req.Id, Name: "Test User"},
    }, nil
}

func main() {
    // Secret key for signing/validating JWT tokens
    secretKey := []byte("your-secret-key-here")

    // Key function for server-side validation
    serverKeyFunc := func(token *jwt.Token) (interface{}, error) {
        // Validate signing method
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return secretKey, nil
    }

    router := http.NewServeMux()

    // Register routes with JWT middleware
    router = user.AppendUserHttpRoute(router, &userService{},
        server.WithMiddleware(jwtauth.Server(serverKeyFunc)),
    )

    log.Fatal(http.ListenAndServe(":8080", router))
}
```

## Client-Side Usage Example

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/soyacen/goose/example/user/v1"
    "github.com/soyacen/goose/middleware/jwtauth"
)

// Define custom claims
type UserClaims struct {
    jwt.RegisteredClaims
    UserID string `json:"user_id"`
    Role   string `json:"role"`
}

func main() {
    // Secret key for signing JWT tokens
    secretKey := []byte("your-secret-key-here")

    // Claims function - generates claims from context
    claimsFunc := func(ctx context.Context) (jwt.Claims, error) {
        return &UserClaims{
            RegisteredClaims: jwt.RegisteredClaims{
                ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
                IssuedAt:  jwt.NewNumericDate(time.Now()),
            },
            UserID: "user-123",
            Role:   "admin",
        }, nil
    }

    // Key function for client-side signing
    clientKeyFunc := func(token *jwt.Token) (interface{}, error) {
        return secretKey, nil
    }

    // Create client with JWT middleware
    client := user.NewUserHttpClient("http://localhost:8080",
        client.WithMiddleware(jwtauth.Client(claimsFunc, clientKeyFunc)),
    )

    // Make authenticated request
    resp, err := client.GetUser(context.Background(), &user.GetUserRequest{Id: 1})
    if err != nil {
        panic(err)
    }
    fmt.Printf("User: %v\n", resp.Item)
}
```

## Complete Example with Multiple Middleware

```go
import (
    "github.com/soyacen/goose/middleware/accesslog"
    "github.com/soyacen/goose/middleware/jwtauth"
    "github.com/soyacen/goose/middleware/recovery"
    "github.com/soyacen/goose/server"
)

router = user.AppendUserHttpRoute(router, &userService{},
    // Middleware order matters - executed in this order
    server.WithMiddleware(recovery.Server()),           // 1. Panic recovery
    server.WithMiddleware(accesslog.Server()),          // 2. Access logging
    server.WithMiddleware(jwtauth.Server(keyFunc)),     // 3. JWT validation
)
```

## Key Points

1. **Token Extraction**: The server expects the `Authorization: Bearer <token>` header
2. **Context Access**: Use `jwtauth.FromContext(ctx)` to retrieve the validated token in your handler
3. **Error Handling**: Invalid/missing tokens return `401 Unauthorized` with `WWW-Authenticate` header
4. **Signing Methods**: Supports HMAC methods (HS256, HS384, HS512) by default
5. **Custom Claims**: You can define custom claim structures to include additional data in tokens

## Dependencies

```go
import (
    "github.com/golang-jwt/jwt/v5"
    "github.com/soyacen/goose/middleware/jwtauth"
)
```

The middleware uses `github.com/golang-jwt/jwt/v5` for JWT operations.
