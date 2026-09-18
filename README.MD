# Goose

Goose 是一个面向 Go 的 Protobuf + HTTP/REST 代码生成与运行时库：用 `.proto` 描述接口，`protoc-gen-goose` 插件生成基于标准库 `net/http` 的服务端路由与客户端，运行时提供请求/响应编解码、错误处理、中间件、文件上传与 WebSocket 流式 RPC 支持，并可按需生成 OpenAPI 3.0.3 文档。

```proto
service User {
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (google.api.http) = { get: "/v1/user/{id}" };
  }
}
```

```go
router := http.NewServeMux()
router = user.AppendUserHttpRoute(router, &userService{})

client := user.NewUserHttpClient("http://localhost:8080")
resp, err := client.GetUser(ctx, &user.GetUserRequest{Id: 1})
```

## 特性

- **Proto 驱动的代码生成**：`protoc-gen-goose` 从 `.proto` 生成服务接口、服务端路由与处理器、客户端、请求编码器、响应解码器与方法描述符。
- **标准库路由**：生成的路由使用 Go 1.22+ `net/http.ServeMux` 的 method + pattern 语法（如 `GET /v1/user/{id}`、`{path...}`），不引入第三方 Web 框架。
- **零配置可用**：`Append{Service}HttpRoute(router, service)` 和 `New{Service}HttpClient(target)` 无需任何选项即可工作，按需再叠加配置。
- **非反射取值**：生成的代码通过类型化的 `GetInt` / `GetString` / `GetBool` 等泛型辅助函数解析 path 与 query 参数，不依赖反射遍历消息字段。
- **完整的 protojson 编解码**：请求体与响应体统一使用 `protojson`，同时支持 `google.api.HttpBody` 与 `google.rpc.HttpRequest` / `google.rpc.HttpResponse` 的原生透传。
- **统一错误编解码**：服务端通过 `goose.NewError` 返回带状态码的错误，默认编码器输出 JSON 或纯文本；客户端自动还原状态码、响应头与错误体。
- **请求校验**：`goose.ValidateRequest` 自动识别 `Validate()` / `ValidateAll()`，与 protoc-gen-validate 等生成的校验方法兼容，并支持回调与 fail-fast。
- **中间件体系**：`server` 与 `client` 各有一套中间件链，内置访问日志、错误日志、Basic Auth、JWT、超时、恢复、CORS、重定向、限流、OpenTelemetry、上下文注入等实现。
- **文件上传**：`upload` 包解析 `multipart/form-data`，支持多文件、混合表单、大小限制、扩展名推断与空文件识别。
- **WebSocket 流式 RPC**：支持客户端流、服务端流与双向流，连接层提供异步写缓冲、ping 保活、优雅关闭与 EOS 标记。
- **OpenAPI 生成**：`--goose_opt=openapi=true` 可为每个 proto 文件生成 OpenAPI 3.0.3 JSON 文档。
- **链式 HTTP 客户端**：`outgoing` 包提供 `Get/Post/Put/...` 链式调用，支持 JSON、XML、Protobuf、Gob、表单与 multipart 请求体。

## 环境要求

- Go 1.25+（根模块 `go.mod` 声明 `go 1.25.0`；`middleware/jwtauth`、`middleware/limiter`、`middleware/otel` 子模块声明 `go 1.23.0`）
- `protoc`（libprotoc 3.x 及以上）
- `protoc-gen-go`

仅需 Protobuf 的 Go 代码生成插件，不需要 `protoc-gen-go-grpc`：Goose 生成的是基于 `net/http` 的 REST 代码，示例与 Makefile 也只调用 `--go_out` 与 `--goose_out`。

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

## 安装

安装代码生成插件：

```bash
go install github.com/soyacen/goose/cmd/protoc-gen-goose@latest
```

在仓库内开发时，也可以直接安装当前代码：

```bash
go install ./cmd/protoc-gen-goose
protoc-gen-goose --version
```

## 快速开始

### 1. 编写 proto

以下是 `example/user/user.proto` 的精简版本（完整示例还包含修改、更新、删除与列表方法）：

```proto
syntax = "proto3";
package leo.goose.example.user.v1;
option go_package = "github.com/soyacen/goose/example/user/v1;user";

import "google/api/annotations.proto";

service User {
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse) {
    option (google.api.http) = {
      post: "/v1/user"
      body: "*"
    };
  }

  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (google.api.http) = {
      get: "/v1/user/{id}"
    };
  }
}

message UserItem {
  int64 id = 1;
  string name = 2;
}

message CreateUserRequest { string name = 1; }
message CreateUserResponse { UserItem item = 1; }

message GetUserRequest { int64 id = 1; }
message GetUserResponse { UserItem item = 1; }
```

### 2. 生成代码

```bash
protoc \
  --proto_path=. \
  --proto_path=./third_party \
  --go_out=. \
  --go_opt=paths=source_relative \
  --goose_out=. \
  --goose_opt=paths=source_relative \
  example/user/user.proto
```

需要 OpenAPI 文档时追加 `--goose_opt=openapi=true`。仓库内可直接使用 `make example` 重新生成全部示例。

生成的 `*_goose.pb.go` 与 `*_goose.openapi.json` 通常包含：

| 产物 | 说明 |
| --- | --- |
| `XxxService` 接口 | 需要你实现的服务接口，非流式方法签名为 `(ctx context.Context, req *Req) (*Resp, error)` |
| `AppendXxxHttpRoute` | 非流式路由注册函数，接收 `*http.ServeMux`、服务实现与 `...server.Option` |
| `NewXxxHttpClient` | 客户端构造函数，接收目标地址与 `...client.Option`，返回 `XxxService` |
| `XxxHttpClient` | 客户端实现，请求编码与响应解码逻辑随服务一起生成 |
| `_pkg_Service_Method_Desc` | 每个方法的 `*goose.Desc`，内部包含 `*goose.RouteInfo`（HTTP 方法、pattern、完整 RPC 方法名） |
| `XxxStreamClient` / `NewXxxStreamClient` / `AppendXxxStreamRoute` | proto 中存在 `stream` 方法时生成 |
| `{prefix}_goose.openapi.json` | 开启 `openapi=true` 时生成 |

### 3. 实现并启动服务端

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/soyacen/goose/example/user"
)

type userService struct{}

func (s *userService) CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.CreateUserResponse, error) {
	return &user.CreateUserResponse{
		Item: &user.UserItem{Id: 1, Name: req.GetName()},
	}, nil
}

func (s *userService) GetUser(ctx context.Context, req *user.GetUserRequest) (*user.GetUserResponse, error) {
	return &user.GetUserResponse{
		Item: &user.UserItem{Id: req.GetId(), Name: "bob"},
	}, nil
}

func main() {
	router := http.NewServeMux()
	router = user.AppendUserHttpRoute(router, &userService{})
	log.Fatal(http.ListenAndServe(":8080", router))
}
```

### 4. 调用客户端

```go
client := user.NewUserHttpClient("http://localhost:8080")

created, err := client.CreateUser(ctx, &user.CreateUserRequest{Name: "Leo"})
if err != nil {
	return err
}

got, err := client.GetUser(ctx, &user.GetUserRequest{Id: created.GetItem().GetId()})
```

完整的可运行示例（含测试）位于 `example/` 目录：`body`、`path`、`query`、`response_body`、`user`、`upload`、`websocket`。

## HTTP 映射规则

生成器解析 `google.api.http` 注解，支持 `get`、`post`、`put`、`delete`、`patch` 以及 `custom`（自定义 method 字符串）。未声明 `google.api.http` 的方法使用默认规则：`POST /{package}.{Service}/{Method}`，且 `body` 为 `*`。

### Path 参数

```proto
rpc GetUser(GetUserRequest) returns (GetUserResponse) {
  option (google.api.http) = { get: "/v1/user/{id}" };
}
```

- `{id}` 匹配单个路径段，`{path...}` 匹配剩余全部路径段。
- path 参数支持标量类型、枚举，以及 `google.protobuf.BoolValue`、`Int32Value`、`Int64Value`、`UInt32Value`、`UInt64Value`、`FloatValue`、`DoubleValue`、`StringValue` 等包装类型。
- path 参数不支持 `repeated`（list）和 `map`，也不支持普通 message 字段；违反时生成阶段直接报错。

### Query 参数

```proto
service ListUser {
  rpc ListUser(ListUserRequest) returns (ListUserResponse) {
    option (google.api.http) = { get: "/v1/users" };
  }
}

message ListUserRequest {
  int64 page_num = 1;
  int64 page_size = 2;
  repeated string tags = 3;
}
```

当方法没有整体 body 时，除 path 字段与 body 字段之外的标量、枚举、包装类型字段会自动作为 query 参数解析，`repeated` 字段支持重复 key（同名多值），`map` 字段会被跳过。

### 请求体

```proto
// body: "*" —— 整个请求消息作为 body
rpc CreateUser(CreateUserRequest) returns (CreateUserResponse) {
  option (google.api.http) = { post: "/v1/user" body: "*" };
}

// body: "item" —— 仅指定字段作为 body，其余字段走 path/query
rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse) {
  option (google.api.http) = { patch: "/v1/user/{id}" body: "item" };
}
```

### 响应体

```proto
// 省略 response_body —— 整个响应消息作为 body
rpc GetUser(GetUserRequest) returns (GetUserResponse) {
  option (google.api.http) = { get: "/v1/user/{id}" };
}

// response_body 也可以指定响应消息的顶层字段名（该字段需为 message 类型）
message GetUserItemResponse { UserItem item = 1; }

rpc GetUserItem(GetUserRequest) returns (GetUserItemResponse) {
  option (google.api.http) = { get: "/v1/user/{id}/item" response_body: "item" };
}
```

## 运行时行为

### 编解码

- 请求/响应体默认使用 `protojson`，`Content-Type` 为 `application/json; charset=utf-8`。
- `google.api.HttpBody`：请求按 `data` + `content_type` 原样解码，响应按 `content_type` 与 `data` 原样输出，适用于文件上传下载。
- `google.rpc.HttpRequest`：解码 `method`、`uri`、`headers`、`body`；`google.rpc.HttpResponse`：编码 `status`、`headers`、`body`。
- 自定义解码：请求消息若实现 `UnmarshalRequest(context.Context, *http.Request) error`，生成的服务端会优先调用该方法（`server.CustomDecodeRequest`）。
- 成功响应固定写入 HTTP 200（`server.EncodeResponse` / `server.EncodeHttpBody`），`google.rpc.HttpResponse` 则使用其自带状态码。

### 错误处理

服务端返回错误时由 `ErrorEncoder` 编码。默认实现 `goose.DefaultEncodeError` 的行为：

1. 若错误实现 `StatusCode() int`，使用其状态码，否则返回 500。
2. 若错误实现 `json.Marshaler`，输出 JSON，否则输出 `text/plain`。
3. 把错误中携带的响应头写入响应，并在 `X-Goose-Error` 头中记录这些头的 key 列表。

```go
return nil, goose.NewError(http.StatusNotFound, map[string]string{"message": "user not found"})
```

客户端由 `ErrorDecoder` 解码错误。默认实现 `goose.DefaultDecodeError` 通过响应中的 `X-Goose-Error` 头判断是否为错误响应，再用 `ErrorFactory` 构造错误对象，填充状态码、响应头与 body。因此服务端与客户端需要配套使用：若服务端不是 Goose 生成的实现，请确保错误响应带上该头，或替换 `client.ErrorEncoder`。

### 请求校验

生成的服务端与客户端在调用业务方法前后会调用 `goose.ValidateRequest(ctx, req, fast, callback)`：

- `fast=true`：依次尝试 `Validate()`、`Validate(false)`。
- `fast=false`：依次尝试 `ValidateAll()`、`Validate(true)`、`Validate()`。
- 校验失败时调用 `OnValidationErrCallback`，并返回错误。

通过 `server.FailFast()` / `client.FailFast()` 开启 fast 模式，通过 `OnValidationErrCallback` 注册回调。

### 路由上下文

`server.Invoke` / `client.Invoke` 会把路由信息与请求头注入 context：

```go
routeInfo, ok := goose.ExtractRouteInfo(ctx)  // *goose.RouteInfo{HttpMethod, Pattern, FullMethod}
header, ok := goose.ExtractHeader(ctx)        // http.Header
```

中间件与业务代码都可以据此获取当前路由，`accesslog`、`otel`、`errorlog` 等中间件正是基于它工作。

## 服务端配置

```go
router = user.AppendUserHttpRoute(router, &userService{},
	server.Middlewares(
		recovery.Server(),
		accesslog.Server(),
		timeout.Server(5*time.Second),
	),
	server.MarshalOptions(protojson.MarshalOptions{UseProtoNames: true}),
	server.UnmarshalOptions(protojson.UnmarshalOptions{DiscardUnknown: true}),
	server.OnValidationErrCallback(func(ctx context.Context, err error) {
		slog.ErrorContext(ctx, "validation failed", "error", err)
	}),
)
```

| Option | 说明 |
| --- | --- |
| `server.Middlewares(...)` | 追加服务端中间件，按传入顺序执行 |
| `server.MarshalOptions(protojson.MarshalOptions)` | 响应编码选项 |
| `server.UnmarshalOptions(protojson.UnmarshalOptions)` | 请求解码选项 |
| `server.ErrorEncoder(goose.ErrorEncoder)` | 自定义错误编码器，默认 `goose.DefaultEncodeError` |
| `server.OnValidationErrCallback(goose.OnValidationErrCallback)` | 校验失败回调 |
| `server.FailFast()` | 校验使用 fast 模式 |

服务端中间件签名为：

```go
type Middleware func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc)
```

也可以用 `server.Chain(m1, m2, ...)` 手动组合中间件链。

## 客户端配置

```go
client := user.NewUserHttpClient("https://api.example.com",
	client.Client(&http.Client{Timeout: 10 * time.Second}),
	client.Middlewares(
		timeout.Client(3*time.Second),
		accesslog.Client(),
	),
	client.MarshalOptions(protojson.MarshalOptions{UseProtoNames: true}),
	client.UnmarshalOptions(protojson.UnmarshalOptions{DiscardUnknown: true}),
)
```

| Option | 说明 |
| --- | --- |
| `client.Client(*http.Client)` | 指定底层 HTTP 客户端，默认零值 `&http.Client{}` |
| `client.Middlewares(...)` | 追加客户端中间件，按传入顺序执行 |
| `client.MarshalOptions(protojson.MarshalOptions)` | 请求编码选项 |
| `client.UnmarshalOptions(protojson.UnmarshalOptions)` | 响应解码选项 |
| `client.ErrorEncoder(goose.ErrorDecoder)` | 设置错误解码器（名字历史原因，实际设置的是 decoder） |
| `client.ErrorFactory(goose.ErrorFactory)` | 设置错误对象工厂 |
| `client.Resolvers(resolver.Resolver)` | 指定 URL 解析器 |
| `client.OnValidationErrCallback(goose.OnValidationErrCallback)` | 校验失败回调 |
| `client.FailFast()` | 校验使用 fast 模式 |

客户端中间件签名为：

```go
type Middleware func(cli *http.Client, request *http.Request, invoker client.Invoker) (*http.Response, error)
```

### URL 解析

客户端目标地址由 `client/resolver` 解析，内置三类解析器：

- 空 scheme：`DefaultResolver`，默认补全为 `http`（可通过 `DefaultResolver{HttpScheme: "https"}` 调整）。
- `http` / `https`：原样保留。
- 其他 scheme：返回 `resolver: scheme xxx is not supported`。

可通过 `resolver.RegisterResolver(custom)` 注册自定义 scheme 解析器。

## 中间件

`middleware/` 下的实现分两类：随根模块发布（`accesslog`、`basicauth`、`context`、`cors`、`errorlog`、`recovery`、`redirect`、`timeout`），以及带独立 `go.mod` 的子模块（`jwtauth`、`limiter`、`otel`，需要单独引入）。

| 中间件 | 端 | 主要 API | 行为 |
| --- | --- | --- | --- |
| `accesslog` | server / client | `accesslog.Server(opts...)`、`accesslog.Client(opts...)` | 使用 `slog` 记录访问日志，包含耗时、状态码、路由、请求 ID 等字段 |
| `errorlog` | server / client | `errorlog.Server(opts...)`、`errorlog.Client(opts...)` | 仅记录错误：服务端状态码 ≥ 400，客户端错误或状态码 ≥ 400 |
| `basicauth` | server / client | `basicauth.Server(Accounts, opts...)`、`basicauth.Client(Account)`、`basicauth.FromContext(ctx)` | HTTP Basic 认证，常量时间比较；通过 context 传递用户名 |
| `jwtauth` | server / client | `jwtauth.Server(keyFunc, opts...)`、`jwtauth.Client(claimsFunc, keyFunc, opts...)`、`jwtauth.FromContext(ctx)` | 校验或签发 Bearer JWT，默认 HS512 |
| `timeout` | server / client | `timeout.Server(d)`、`timeout.Client(d)` | 为请求上下文设置超时，并通过 `X-Leo-Timeout` 头传递 |
| `recovery` | server | `recovery.Server(opts...)` | 捕获 panic 并交给 `RecoveryHandler`；默认实现只记录日志与堆栈 |
| `cors` | server | `cors.Server(opts...)` | 处理预检与实际请求的 CORS 响应头 |
| `redirect` | server | `redirect.Server()` | 非 HTTPS 请求返回 302 跳转到 HTTPS |
| `limiter` | server | `limiter.Server(opts...)` | 基于 BBR 与 CPU 负载的自适应限流，超限返回 429 |
| `context` | server / client | `context.Server(fn)`、`context.Client(fn)` | 在调用链前后注入/派生 context |
| `otel` | server / client | `otel.Server()`、`otel.Client()`、`otel.ExtractTraceId(ctx)` | OpenTelemetry HTTP 追踪，服务端自动带上 `http.route` 与 `rpc.method` |

常用配置项：

```go
accesslog.Server(
	accesslog.WithLevel(slog.LevelDebug),
	accesslog.WithSkip(func(route string) bool { return route == "/healthz" }),
	accesslog.WithPrintRequest(true),
	accesslog.WithPrintResponse(true),
)

basicauth.Server(basicauth.Accounts{
	{User: "admin", Password: "secret"},
}, basicauth.Realm("Admin Area"))

cors.Server(
	cors.AllowedOrigins([]string{"https://*.example.com"}),
	cors.AllowedMethods([]string{http.MethodGet, http.MethodPost}),
	cors.AllowCredentials(),
	cors.MaxAge(10*time.Minute),
)

limiter.Server(
	limiter.WithWindow(10*time.Second),
	limiter.WithBuckets(100),
	limiter.WithCPUThreshold(0.8),
	limiter.WithCPUInterval(500*time.Millisecond),
)
```

`basicauth.Server` 在账号列表为空或用户名为空时会直接 panic；`recovery` 的默认 handler 不会主动写响应，如需返回 5xx 请使用 `recovery.RecoveryHandler` 自定义。

## 文件上传

`upload` 包负责解析 `multipart/form-data` 并把文件写入磁盘，也可以直接把原始 body 当作单文件保存。

```go
handler, err := upload.NewHandler(
	upload.WithUploadDir("./uploads"),   // 默认 ./uploads，目录不存在会自动创建
	upload.WithMaxFileSize(32<<20),      // 单文件上限，0 表示不限制
	upload.WithMaxTotalSize(128<<20),    // 总体积上限，0 表示不限制
)
if err != nil {
	log.Fatal(err)
}

result, err := handler.Handle(data, contentType)
if err != nil {
	log.Printf("upload failed: %v", err)
	return
}
log.Printf("files=%d fields=%d json=%s", result.FileCount, result.FieldCount, result.JSON())
```

主要 API：

| API | 说明 |
| --- | --- |
| `NewHandler(opts...) (*Handler, error)` | 创建处理器并确保上传目录存在 |
| `(*Handler).Handle(data []byte, contentType string)` | 自动区分 `multipart/*` 与原始 body |
| `(*Handler).ParseMultipart(data []byte, boundary string)` | 解析 multipart，保存文件并聚合表单字段 |
| `(*Handler).SaveSingleFile(data []byte, ext, fieldName, origName, contentType string)` | 保存单个文件 |
| `ExtensionFromContentType` / `ExtensionFromFilename` | 推断扩展名 |
| `FormatBytes(int64)` | 人类可读的体积格式 |

解析结果类型：

| 类型 | 字段 |
| --- | --- |
| `Result` | `Files`、`Fields`、`TotalSize`、`FileCount`、`FieldCount`，并提供 `JSON()` |
| `SavedFile` | `FieldName`、`OrigName`、`ContentType`、`SavedAs`、`Size`、`IsEmpty` |
| `FormField` | `Name`、`Values`（同名多值聚合为切片） |

导出的错误：`ErrFileTooLarge`、`ErrTotalTooLarge`、`ErrMissBoundary`。

与生成代码集成时，proto 请求类型可以有三种写法（见 `example/upload/upload.proto`）：`google.api.HttpBody` 作为整个请求、`google.api.HttpBody` 作为命名字段、`google.rpc.HttpRequest`。前两者把 `req.GetData()` 与 `req.GetContentType()` 传给 `Handle`，后者从 `req.GetBody()` 与 `Content-Type` 头取值。

## WebSocket 流式 RPC

proto 中使用 `stream` 声明流式方法：

```proto
service Websocket {
  rpc ClientStream(stream Request) returns (Response) {
    option (google.api.http) = { post: "/ws/client-stream" };
  }
  rpc ServerStream(Request) returns (stream Response) {
    option (google.api.http) = { post: "/ws/server-stream" };
  }
  rpc BidStream(stream Request) returns (stream Response) {
    option (google.api.http) = { post: "/ws/bidi-stream" };
  }
}
```

生成器会为服务接口生成流式方法签名（`ws.ClientStreamingServer`、`ws.ServerStreamingServer`、`ws.BidiStreamingServer`），并额外生成 `XxxStreamClient` 接口、`NewXxxStreamClient` 构造函数与 `AppendXxxStreamRoute` 路由注册函数。

服务端：

```go
mux := http.NewServeMux()
mux = websocket.AppendWebsocketStreamRoute(
	mux,
	&streamService{},
	server.Chain(),                  // server.Middleware
	protojson.MarshalOptions{},      // 出站编码
	protojson.UnmarshalOptions{},    // 入站解码
	ws.AcceptOptions(),              // *websocket.AcceptOptions
	ws.DefaultConnConfig(),          // *ws.ConnConfig
	slog.Default(),                  // *slog.Logger
)
```

服务实现：

```go
func (s *streamService) BidStream(stream ws.BidiStreamingServer[*Request, *Response]) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.Send(&Response{Message: "ack: " + req.GetName()}); err != nil {
			return err
		}
	}
}
```

客户端：

```go
cli := websocket.NewWebsocketStreamClient(
	"ws://localhost:8080",
	slog.Default(),
	protojson.MarshalOptions{},
	protojson.UnmarshalOptions{},
	ws.DialOptions(),
)

stream, err := cli.BidStream(ctx)
if err != nil {
	return err
}
if err := stream.Send(&websocket.Request{Name: "alice"}); err != nil {
	return err
}
resp, err := stream.Recv()
```

连接层由 `ws.Conn` 提供，`ws.DefaultConnConfig()` 的默认值为：单条消息最大 1 MB、写缓冲 256、ping 间隔 30s、写超时 10s。消息使用 WebSocket 文本帧承载 protojson，流结束时发送空文本帧作为 EOS 标记；`ws.IsNormalClose(err)` 用于判断正常关闭（含 `StatusNormalClosure`、`StatusGoingAway` 与 `context.Canceled`）。

需要注意：`Conn.Send` 是非阻塞的，写缓冲满时会丢弃该消息并记录一条 warn 日志。

## OpenAPI 文档生成

在生成命令中加入 `--goose_opt=openapi=true`：

```bash
protoc \
  --proto_path=. --proto_path=./third_party \
  --go_opt=paths=source_relative \
  --goose_out=. --goose_opt=paths=source_relative --goose_opt=openapi=true \
  example/user/user.proto
```

会为每个 proto 文件生成 `{前缀}_goose.openapi.json`：

- OpenAPI 版本 3.0.3，`info.title` 为 `{proto package} API`，`info.version` 为 `1.0.0`。
- `paths` 来自各服务的 `google.api.http` 注解，path 参数、query 参数、请求体与响应体都会映射为对应的 parameter / requestBody / response。
- message 会转换为 `components.schemas`，并以完整 proto 名称作为 key。
- `operationId` 形如 `CreateUserRequest_CreateUser`，`summary` 为方法名。

示例文档见 `example/*/*_goose.openapi.json`，各示例目录下的 `openapi_test.go` 会用生成的文档校验接口行为。

## outgoing 链式 HTTP 客户端

`outgoing` 提供独立于代码生成的链式 HTTP 调用方式，适合调用第三方接口：

```go
result, err := outgoing.Post(outgoing.Client(&http.Client{Timeout: 10 * time.Second})).
	URL(outgoing.URLString("https://httpbin.org/post")).
	Query(outgoing.QueryObject(params)).
	Header(outgoing.BearerAuth("token"), outgoing.SetHeader("X-Trace-Id", traceID)).
	Body(outgoing.JSONBody(req)).
	Send(ctx)
if err != nil {
	return err
}

var resp Response
if err := result.JSONBody(&resp); err != nil {
	return err
}
```

支持的能力：

- 方法：`Get`、`Head`、`Post`、`Put`、`Patch`、`Delete`、`Connect`、`Options`、`Trace`、`Method`。
- Query：`SetQuery`、`AddQuery`、`DelQuery`、`QueryString`、`Queries`、`QueryObject`（结构体按 `url` tag 编码）。
- Header：`SetHeader`、`AddHeader`、`DelHeader`、`Headers`、`UserAgent`、`BasicAuth`、`BearerAuth`、`CustomAuth`、`CacheControl`、条件请求头与 Cookie 系列。
- Body：`Body`、`BytesBody`、`TextBody`、`ObjectBody`、`JSONBody`、`XMLBody`、`ProtobufBody`、`GobBody`、`FormBody`、`FormObjectBody`、`MultipartBody`。
- 响应读取：`Status`、`StatusCode`、`Headers`、`Cookies`、`Body`、`BytesBody`、`TextBody`、`ObjectBody`、`JSONBody`、`XMLBody`、`ProtobufBody`、`GobBody`。
- 中间件：`outgoing.Middleware(...)` 接受与生成客户端一致的 `client.Middleware`。

更多可运行示例见 `outgoing/outgoing_example_test.go`。

## 类型支持

| 类别 | 支持情况 |
| --- | --- |
| 标量 | `bool`、`int32`、`sint32`、`sfixed32`、`int64`、`sint64`、`sfixed64`、`uint32`、`fixed32`、`uint64`、`fixed64`、`float`、`double`、`string`、`bytes` |
| 枚举 | 支持 path 与 query |
| 包装类型 | `google.protobuf.*Value` 支持 path 与 query |
| `optional` | 支持，映射为指针语义 |
| `repeated` | query 支持同名多值或重复 key；path 不支持 |
| `map` | 不支持作为 path/query 参数，会被跳过 |

生成代码通过 `goose.FormatInt`、`goose.GetBool`、`goose.GetIntSlice`、`goose.WrapStringSlice`、`goose.UnwrapInt64Slice` 等类型化辅助函数完成转换，这些函数也在根包中导出，可在自定义处理器中直接复用。

## 项目结构

| 目录 | 说明 |
| --- | --- |
| `cmd/protoc-gen-goose/` | protoc 插件：`parser` 解析 proto，`server` / `client` / `stream` 生成代码，`openapi` 生成文档 |
| `server/` | 服务端运行时：编解码、中间件链、路由信息注入 |
| `client/` | 客户端运行时：请求编码、响应解码、错误解码、中间件链 |
| `client/resolver/` | URL scheme 解析器与注册表 |
| `middleware/` | 内置中间件，其中 `jwtauth`、`limiter`、`otel` 为独立模块 |
| `upload/` | multipart 解析与文件保存 |
| `ws/` | WebSocket 连接管理与流式 RPC 抽象 |
| `outgoing/` | 链式 HTTP 客户端 |
| `type_*.go`、`form.go`、`path.go`、`header.go`、`status.go`、`validate.go` | 根包辅助能力：类型转换、表单与 path 取值、请求头/客户端 IP、错误编解码、请求校验 |
| `example/` | 各场景示例 proto、生成代码、OpenAPI 文档与测试 |
| `third_party/` | `google/api`、`google/rpc` 等 proto 依赖 |
| `skills/go-goose/` | 面向 AI 编码助手的 Goose 使用指南 |

## 开发与测试

```bash
# 安装当前插件到 $GOBIN
make install

# 运行根模块全部测试
go test ./...

# 带详细输出
make test

# 重新生成 example 下的代码与 OpenAPI 文档
make example
```

`make all` 等价于先 `make install` 再 `make example`。

`middleware/jwtauth`、`middleware/limiter`、`middleware/otel` 是独立 Go 模块，根模块的 `go test ./...` 不会覆盖它们，需要单独执行：

```bash
cd middleware/limiter && go mod tidy && go test ./...
```

当前根模块 `go test ./...` 全部通过。

## 注意事项

- 生成的路由依赖 Go 1.22+ 的 `http.ServeMux` pattern 语法，注册函数接收的是 `*http.ServeMux`。
- `body` 与 `response_body` 只接受顶层字段名，不支持 `a.b` 形式的嵌套路径；`response_body` 指定的字段需要是 message 类型。
- 客户端错误识别依赖 `X-Goose-Error` 响应头；与手写服务端对接时需要替换错误解码器或补齐该头。
- `recovery` 的默认 handler 只记录日志，不会写入状态码，需要自定义 `RecoveryHandler` 才能返回 5xx。
- `accesslog` 在 context 中缺少 `RouteInfo` 时会通过 unsafe 反射读取请求内部字段推断路由；正常经由生成代码注册的处理器都会注入 `RouteInfo`。
- `ws.Conn.Send` 在写缓冲满时丢弃消息，只记录 warn 日志，不返回错误。

## 许可证

本项目使用 MIT License，详见 `LICENSE`。
