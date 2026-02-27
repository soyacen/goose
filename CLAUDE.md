# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Goose is a Go library for Protobuf + HTTP/REST code generation with runtime support. It provides:
- A `protoc-gen-goose` plugin that generates HTTP/REST server and client code from `.proto` files
- Encoder/decoder implementations for mapping between Protobuf messages and HTTP requests/responses
- A collection of reusable middleware components (access logging, auth, rate limiting, etc.)
- Built on Go 1.22+ standard library HTTP routing

## Common Commands

```bash
# Build the protoc-gen-goose plugin
make build

# Install protoc-gen-goose plugin to $GOBIN
make install

# Run all tests
go test ./...

# Run tests with verbose output
make test

# Regenerate example code from proto files
make example

# Run a single test file
go test -v ./example/body/...
```

## Architecture

### Code Generation Flow
1. Write a `.proto` file defining your service and messages
2. Run `protoc` with `--go_out`, `--go-grpc_out`, and `--goose_out` flags
3. The `protoc-gen-goose` plugin generates:
   - Server handlers with encoder/decoder for request/response mapping
   - Client stubs for calling the service
   - Route registration code

### Key Packages

- `cmd/protoc-gen-goose/` - The protoc plugin implementation
  - `parser/` - Parses proto files to extract service definitions
  - `server/` - Server-side code generation templates
  - `client/` - Client-side code generation templates
  - `constant/` - Generated constants

- `server/` - Server-side runtime support
  - `encoder.go` / `decoder.go` - Request/response encoding/decoding
  - `middleware.go` - Middleware chain execution
  - `option.go` - Configuration options

- `client/` - Client-side runtime support
  - Similar structure to server package for HTTP client middleware

- `middleware/` - Reusable middleware components
  - `accesslog/` - HTTP access logging with slog
  - `basicauth/` - HTTP Basic authentication
  - `jwtauth/` - JWT authentication
  - `recovery/` - Panic recovery returning 5xx
  - `timeout/` - Request timeout control
  - `bbr/` - BBR rate limiter (separate go.mod)
  - `cors/` - CORS handling
  - `ctxgoose/` - Context utilities
  - `otel/` - OpenTelemetry integration
  - `redirect/` - Redirect handling

- `example/` - Example proto files and generated code
  - `body/`, `path/`, `query/`, `response_body/`, `user/` - Different request pattern examples

### Middleware Pattern

Server middleware follows the signature:
```go
func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc)
```

Client middleware follows:
```go
func(cli *http.Client, request *http.Request, invoker client.Invoker) (*http.Response, error)
```

Use `server.Chain()` to combine multiple middleware functions.

### Route Information

The `goose.RouteInfo` struct holds HTTP route metadata and is injected into request context via `goose.InjectRouteInfo()`. Middleware can extract this using `goose.ExtractRouteInfo()`.

## Development Notes

- The bbr middleware has its own `go.mod` (middleware/bbr/go.mod)
- Generated code uses `paths=source_relative` for import paths
- The plugin requires protoc, protoc-gen-go, and protoc-gen-go-grpc to be installed
- Examples can be regenerated with `make example` after modifying the plugin
