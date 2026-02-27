---
name: go-goose
description: |
  Build HTTP/REST services from .proto files using the goose library.
  Use this skill whenever the user wants to:
  - Create a protobuf-based HTTP API in Go
  - Generate server/client code from .proto files
  - Use goose middleware (accesslog, basicauth, jwtauth, recovery, timeout, bbr, etc.)
  - Build REST endpoints with path parameters, query strings, or request bodies
  - Work with the protoc-gen-goose plugin

  This skill covers the complete workflow: writing proto definitions, generating code, implementing services, adding middleware, and running tests.
compatibility: Go 1.23+, protoc, protoc-gen-go, protoc-gen-go-grpc
---

# Goose HTTP/REST Code Generation Guide

This skill helps you build HTTP/REST services using the goose library. Goose generates Go server and client code from `.proto` files using the `protoc-gen-goose` plugin.

## Proto File Structure

A goose proto file needs these elements:

```proto
syntax = "proto3";
package my.package.v1;
option go_package = "github.com/user/project/pkg/v1;pkg";

import "google/api/annotations.proto";
import "google/protobuf/wrappers.proto";

service MyService {
  rpc MyMethod(MyRequest) returns (MyResponse) {
    option (google.api.http) = {
      get: "/v1/resource/{id}"  // HTTP mapping
    };
  }
}

message MyRequest {
  int64 id = 1;
}

message MyResponse {
  string message = 1;
}
```

## HTTP Method Mapping

Goose supports these HTTP patterns:

### Path Parameters
```proto
rpc GetUser(GetUserRequest) returns (User) {
  option (google.api.http) = {
    get: "/v1/users/{id}"  // {id} maps to GetUserRequest.id
  };
}
message GetUserRequest { int64 id = 1; }
```

### Query Strings (GET)
```proto
rpc ListUsers(ListUsersRequest) returns (UserList) {
  option (google.api.http) = {
    get: "/v1/users"  // All fields become query params
  };
}
message ListUsersRequest {
  int64 page_num = 1;
  int64 page_size = 2;
}
// GET /v1/users?page_num=1&page_size=10
```

### Request Body
```proto
rpc CreateUser(CreateUserRequest) returns (User) {
  option (google.api.http) = {
    post: "/v1/users"
    body: "*"  // Entire request maps to body
  };
}
// POST /v1/users { "name": "Leo" }
```

### Named Body Field
```proto
rpc UpdateUser(UpdateUserRequest) returns (User) {
  option (google.api.http) = {
    put: "/v1/users/{id}"
    body: "user"  // Only 'user' field maps to body
  };
}
message UpdateUserRequest {
  int64 id = 1;
  User user = 2;
}
```

### Delete Method
```proto
rpc DeleteUser(DeleteUserRequest) returns (Empty) {
  option (google.api.http) = {
    delete: "/v1/users/{id}"
  };
}
```

### Patch Method
```proto
rpc PatchUser(PatchUserRequest) returns (User) {
  option (google.api.http) = {
    patch: "/v1/users/{id}"
    body: "data"
  };
}
```

### Response Body Patterns
```proto
// Omitted response body - response in body by default
rpc GetUser(GetUserRequest) returns (User) {
  option (google.api.http) = {
    get: "/v1/users/{id}"
  };
}

// Star response - entire response in body
rpc GetUser(GetUserRequest) returns (User) {
  option (google.api.http) = {
    get: "/v1/users/{id}"
    response_body: "*"
  };
}

// Named response body - specific field in body
rpc GetUser(GetUserRequest) returns (UserResponse) {
  option (google.api.http) = {
    get: "/v1/users/{id}"
    response_body: "data"  // Only 'data' field in response body
  };
}
message UserResponse {
  User data = 1;
  Metadata meta = 2;
}
```

### Raw HTTP Types
```proto
// Full HTTP request/response access
rpc CustomHandler(google.rpc.HttpRequest) returns (google.rpc.HttpResponse) {
  option (google.api.http) = {
    post: "/v1/custom"
    body: "*"
  };
}

// HttpBody for file downloads
rpc Download(google.api.HttpBody) returns (google.api.HttpBody) {
  option (google.api.http) = {
    get: "/v1/download/{filename}"
  };
}
```

## Generating Code

### Prerequisites
```bash
go install github.com/soyacen/goose/cmd/protoc-gen-goose@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Generate Command
```bash
protoc \
  --proto_path=. \
  --proto_path=./third_party \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  --goose_out=. \
  --goose_opt=paths=source_relative \
  your_service.proto
```

## Implementing the Service

### Server Implementation

```go
package main

import (
    "context"
    "log"
    "net/http"
    "github.com/soyacen/goose/example/user/v1"
    "github.com/soyacen/goose/server"
)

type userService struct{}

func (s *userService) CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.CreateUserResponse, error) {
    return &user.CreateUserResponse{
        Item: &user.UserItem{Id: 1, Name: req.Name},
    }, nil
}

func (s *userService) GetUser(ctx context.Context, req *user.GetUserRequest) (*user.GetUserResponse, error) {
    return &user.GetUserResponse{
        Item: &user.UserItem{Id: req.Id, Name: "Test"},
    }, nil
}

func main() {
    router := http.NewServeMux()
    // Generated function: Append{ServiceName}HttpRoute
    router = user.AppendUserHttpRoute(router, &userService{})

    log.Fatal(http.ListenAndServe(":8080", router))
}
```

### Client Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/soyacen/goose/example/user/v1"
)

func main() {
    // Generated client constructor: New{ServiceName}HttpClient
    client := user.NewUserHttpClient("http://localhost:8080")

    resp, err := client.CreateUser(context.Background(), &user.CreateUserRequest{
        Name: "Leo",
    })
    if err != nil {
        panic(err)
    }
    fmt.Printf("Created user: %v\n", resp.Item)
}
```

## Using Middleware

### Server Middleware Chain

```go
import (
    "github.com/soyacen/goose/middleware/accesslog"
    "github.com/soyacen/goose/middleware/recovery"
    "github.com/soyacen/goose/middleware/basicauth"
    "github.com/soyacen/goose/server"
)

router := http.NewServeMux()
router = user.AppendUserHttpRoute(router, &userService{},
    server.WithMiddleware(accesslog.Server()),
    server.WithMiddleware(recovery.Server()),
    server.WithMiddleware(basicauth.Server(
        basicauth.WithCredential("admin", "secret"),
    )),
)
```

### Available Middleware

| Middleware | Description |
|------------|-------------|
| `accesslog` | HTTP access logging with slog |
| `basicauth` | HTTP Basic authentication |
| `jwtauth` | JWT token validation |
| `recovery` | Panic recovery returning 5xx |
| `timeout` | Request timeout control |
| `bbr` | BBR rate limiting |
| `cors` | CORS headers |
| `otel` | OpenTelemetry tracing |

### Client Middleware

```go
import (
    "github.com/soyacen/goose/middleware/accesslog"
    "github.com/soyacen/goose/client"
)

client := user.NewUserHttpClient("http://localhost:8080",
    client.WithMiddleware(accesslog.Client()),
)
```

## Configuration Options

### Server Options
```go
import "github.com/soyacen/goose/server"

router = user.AppendUserHttpRoute(router, service,
    server.WithMiddleware(middleware),
    server.WithMarshalOptions(&server.MarshalOptions{
        UseJSONNames: true,
    }),
    server.WithUnmarshalOptions(&server.UnmarshalOptions{
        DiscardUnknown: true,
    }),
    server.WithErrorEncoder(myErrorEncoder),
    server.WithShouldFailFast(true),
)
```

### Client Options
```go
import "github.com/soyacen/goose/client"

client := user.NewUserHttpClient("http://localhost:8080",
    client.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
    client.WithMiddleware(middleware),
    client.WithResolver(resolver),
)
```

## Validation

Goose uses protobuf validation. Add validation rules in your proto:

```proto
message CreateUserRequest {
  string name = 1 [(validate.rules).string.min_len = 1];
  int32 age = 2 [(validate.rules).int32.gte = 0];
}
```

Enable validation:
```go
server.WithShouldFailFast(true),
```

## Testing

Example test pattern from the codebase:

```go
func TestCreateUser(t *testing.T) {
    // Start server
    server := &http.Server{Addr: ":38080"}
    defer server.Shutdown(context.Background())

    router := http.NewServeMux()
    router = user.AppendUserHttpRoute(router, &mockService{})
    server.Handler = router

    go server.ListenAndServe()
    time.Sleep(100 * time.Millisecond) // Wait for server

    // Use client
    client := user.NewUserHttpClient("http://localhost:38080")
    resp, err := client.CreateUser(context.Background(), &user.CreateUserRequest{
        Name: "Leo",
    })
    if err != nil {
        t.Fatal(err)
    }
    if resp.Item.Name != "Leo" {
        t.Fatal("name mismatch")
    }
}
```

## Type Support

Goose handles these protobuf types:
- **Basic**: bool, int32, int64, uint32, uint64, float, double, string, bytes
- **Wrapper**: google.protobuf.BoolValue, Int32Value, StringValue, etc.
- **Optional**: `optional` keyword (Go 1.23+)
- **Repeated**: slices and lists
- **Enum**: enum types with custom values

## Project Structure

A typical goose project:

```
myproject/
├── proto/
│   └── service.proto
├── third_party/
│   ├── google/api/httpbody.proto
│   └── google/api/annotations.proto
├── cmd/
│   └── server/main.go
├── pkg/
│   └── service/
│       └── service.go
└── go.mod
```

## Key Patterns

1. **Service Interface**: Goose generates a `{Service}Service` interface you must implement
2. **Route Registration**: Use `Append{Service}HttpRoute(router, service, opts...)` to register routes
3. **Client Creation**: Use `New{Service}HttpClient(baseURL, opts...)` to create clients
4. **Middleware Chain**: Use `server.Chain(m1, m2, ...)` or `server.WithMiddleware()` options
5. **Error Handling**: Custom error encoder via `server.WithErrorEncoder()`

## References

- Repository: https://github.com/soyacen/goose
- Examples: `example/` directory in the repository contains working examples for body, path, query, and response patterns
