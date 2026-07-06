---
kind: logging_system
name: 基于 slog 的中间件日志系统
category: logging_system
scope:
    - '**'
source_files:
    - middleware/accesslog/middleware.go
    - middleware/errorlog/middleware.go
    - middleware/errorlog/option.go
---

Goose 框架采用 Go 标准库 `log/slog` 作为其核心日志基础设施，主要通过中间件（Middleware）的形式提供结构化的访问日志和错误日志功能。该设计遵循无侵入式原则，利用 `slog.Attr` 构建结构化字段，并通过 `sync.Pool` 优化高并发下的内存分配。

### 1. 核心架构与组件
日志功能主要分布在 `middleware` 目录下，分为两个独立但互补的模块：
- **访问日志 (`middleware/accesslog`)**: 记录所有经过的 HTTP 请求/响应元数据。支持服务端 (`Server`) 和客户端 (`Client`) 两种模式。
- **错误日志 (`middleware/errorlog`)**: 专门捕获并记录状态码 >= 400 的异常请求或调用失败，默认使用 `slog.LevelError` 级别。

### 2. 结构化日志字段规范
框架定义了统一的日志字段命名约定，便于后续接入 ELK、Loki 等日志分析系统：
- **系统标识**: `system` (值为 `http.server` 或 `client`)
- **性能指标**: `latency` (耗时), `timestamp` (RFC3339 格式)
- **请求上下文**: `method`, `uri`, `path`, `proto`, `host`, `remote_address`
- **追踪信息**: `request_id` (从 Header `X-Request-Id` 提取), `route` (从 Context 提取的路由模式)
- **状态信息**: `status` (HTTP 状态码)

### 3. 性能优化策略
在 `accesslog` 实现中，框架使用了 `sync.Pool` 来复用 `[]slog.Attr` 切片。由于每次请求都会产生大量日志属性，这种池化技术显著减少了 GC 压力，体现了对高性能网关场景的考量。

### 4. 开发者使用指南
- **集成方式**: 通过 `server.WithMiddleware(accesslog.Server())` 或 `client.WithMiddleware(errorlog.Client())` 注入。
- **敏感信息控制**: 默认不打印 Body。若需调试，可通过 `WithPrintRequest(true)` 和 `WithPrintResponse(true)` 开启，但需注意生产环境的隐私与性能影响。
- **日志路由**: 日志消息（Message）通常设置为路由模式（Pattern），方便按接口维度进行聚合统计。