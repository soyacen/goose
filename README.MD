# Goose

Goose 是一个面向 Go 的 Protobuf + HTTP/REST 代码生成与运行时支持库，包含用于生成客户端/服务器代码的 `protoc-gen-goose` 插件、服务端/客户端的编码解码器、以及一组常用中间件（访问日志、基本认证、JWT 验证、恢复、请求日志、超时等）。本仓库还包含示例 proto 文件、工具和测试用例，便于快速上手与二次开发。

# 优点
1. 基于proto文件，快速生成Restful服务路由和客户端
2. 零配置
3. 低依赖
4. 非反射取值，效率非常高
5. 稳定，所有方法经过单元测试
6. 代码简单，良好的代码设计，减少出错
7. 基于Go1.22标准库Http路由语法

## 主要特性

- protoc 插件：`cmd/protoc-gen-goose`，用于从 .proto 生成服务端/客户端样板代码。
- 服务端/客户端编码器与解码器：位于 `server` 与 `client` 包，支持常见的请求/响应体映射。
- 常用中间件：`middleware` 目录下包含访问日志、基本认证、JWT、恢复、请求日志、超时等实现，便于集成到生成的服务中。
- 丰富的示例：`example` 目录包含多个基于 proto 的示例（body、path、query、response_body、user），包含生成后的 Go 文件与测试。
- 辅助工具：验证、路径处理、pprof、格式化辅助等工具函数。

## 目录概览

- `cmd/protoc-gen-goose/`：protoc 插件源码（生成器、请求/响应模版等）。
- `client/`：客户端相关的编码/解码与中间件选项。
- `server/`：服务端相关的编码/解码与中间件选项。
- `middleware/`：一组可复用的中间件实现（accesslog、basicauth、jwtauth、recovery、requestlog、timeout 等）。
- `example/`：示例 proto 与生成的 Go 文件，演示如何使用插件和运行生成代码。
- `internal/`、`tools/`：库内部工具与构建脚本。

## 快速开始

下面展示一个最小的使用流程：安装插件、用 protoc 生成代码、运行示例。

### 安装（编译插件）

推荐使用 go install 来安装插件（在支持 Go Modules 的环境下）：

```bash
go install github.com/soyacen/goose/cmd/protoc-gen-goose@latest
```

这会在 `$GOBIN` 或 `$GOPATH/bin` 下生成 `protoc-gen-goose` 可执行文件，protoc 能自动找到并使用它。

> 也可以在仓库根目录下直接 `go build ./cmd/protoc-gen-goose`，得到本地可执行文件用于开发或调试。

### 生成代码

仓库内 `example/protoc.sh` 提供了示例生成命令。下面给出一个典型的 protoc 调用示例：

```bash
# 假设当前在 example 目录，且 protoc-gen-goose 在 PATH
protoc --go_out=. --go-grpc_out=. --goose_out=. \
  --proto_path=. your_service.proto
```

生成后的文件通常包含：
- Protobuf 消息类型的 Go 实现（由 `protoc-gen-go` 生成）
- 基于 Goose 的服务端与客户端样板（由 `protoc-gen-goose` 生成）

具体的插件选项和生成路径请参考 `cmd/protoc-gen-goose` 的源码与 `example/protoc.sh` 脚本。

## 快速示例（运行生成的服务）

仓库的 `example` 目录包含多个示例服务。一般步骤：

1. 进入对应示例目录（例如 `example/body`）。
2. 运行 `example/protoc.sh`（或手动执行 `protoc`）生成 Go 代码。
3. 在生成后的代码中，使用生成的服务端构建器并挂载中间件（如需要），然后启动 HTTP 服务器。
4. 使用生成的客户端（或 curl/postman）调用接口并查看响应。

示例中也包含对应的测试文件（`*_test.go`），可直接运行 `go test` 来查看行为。

## 中间件

`middleware` 目录下包含若干实现：

- `accesslog`：记录访问日志
- `basicauth`：HTTP 基本认证
- `jwtauth`：JWT 验证（示例与实现位于子模块中）
- `recovery`：捕获 panic 并返回 5xx
- `requestlog`：请求级别详细日志
- `timeout`：请求超时控制

这些中间件可以与生成的服务端代码组合使用，或在自定义的 HTTP/框架中复用。

## 本地开发与测试

- 运行所有包的测试：

```bash
go test ./...
```

- 在更改 `cmd/protoc-gen-goose` 后，重新编译并在 `example` 下运行 `protoc` 生成最新代码进行联调。

## 贡献

欢迎贡献：提交 issue、PR 或在 `cmd/protoc-gen-goose` 中添加更多生成选项与模板。贡献指南：

1. Fork 仓库并在 feature 分支开发。
2. 增加/更新测试以覆盖你的改动。
3. 提交 PR，并在描述中说明变更目的与影响范围。

## 相关链接与参考

- `example/`：查看如何使用插件生成代码并运行示例。
- `cmd/protoc-gen-goose`：插件实现，查看如何扩展生成模板。

## 许可证

本项目遵循仓库中的 LICENSE 文件（请参阅 `LICENSE`）。


