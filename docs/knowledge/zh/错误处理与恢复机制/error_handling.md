Goose 框架采用了一套基于接口契约的 HTTP 错误处理体系，结合了中间件链式调用、Panic 恢复以及标准化的错误编解码机制。

### 1. 核心错误类型与接口
- **`defaultError`**: 位于 `status.go`，是框架内置的标准 HTTP 错误实现。它封装了状态码（`statusCode`）、响应头（`headers`）和响应体（`body`）。
- **接口契约**:
  - `StatusCodeGetter/Setter`: 用于获取或设置错误的 HTTP 状态码。
  - `HeaderGetter/Setter`: 用于获取或设置错误的 HTTP 响应头。
  - `json.Marshaler/Unmarshaler`: 支持将错误体序列化为 JSON 格式。
- **自定义错误**: 如 `client/resolver.ResolverError`，通过实现标准的 `error` 接口提供特定场景的错误信息。

### 2. 错误编解码 (Encoding/Decoding)
- **服务端编码 (`DefaultEncodeError`)**: 
  - 自动识别错误是否实现了 `StatusCodeGetter` 以确定 HTTP 状态码（默认为 500）。
  - 若错误实现了 `json.Marshaler`，则优先以 JSON 格式返回错误体；否则返回纯文本。
  - 支持将错误携带的自定义 Header 写入响应。
- **客户端解码 (`DefaultDecodeError`)**: 
  - 从 HTTP 响应中提取状态码、Header 和 Body。
  - 利用 `ErrorFactory` 创建错误实例，并通过 `UnmarshalJSON` 还原错误内容。

### 3. 异常恢复 (Recovery)
- **`middleware/recovery`**: 提供了 `Server` 中间件，通过 `defer recover()` 捕获 Handler 执行过程中的 Panic。
- **默认行为**: 记录 Panic 信息和堆栈跟踪到 `slog`。
- **自定义处理**: 支持通过 `RecoveryHandler` 选项注入自定义的 Panic 处理逻辑。

### 4. 错误日志记录
- **`middleware/errorlog`**: 提供了针对 Server 和 Client 的错误日志中间件。
- **触发条件**: 当 HTTP 状态码 >= 400 或发生网络错误时，自动记录包含路由、方法、状态码、请求/响应体（可选）的结构化日志。

### 5. 错误流控制工具
- **`BreakOnError`**: 在链式操作中，如果前一步已产生错误，则立即中断并返回该错误。
- **`ContinueOnError`**: 在链式操作中继续执行后续步骤，最后使用 `errors.Join` 合并所有产生的错误，适用于需要收集多个校验错误的场景。

### 6. 开发规范
- **返回错误**: 业务逻辑中应优先使用 `goose.NewError` 或实现 `StatusCodeGetter` 接口的自定义错误，以便框架能正确映射 HTTP 状态码。
- **Panic 处理**: 避免在 Handler 中直接 Panic，应依赖 `recovery` 中间件进行兜底。
- **中间件顺序**: 建议将 `recovery` 放在中间件链的最外层，以确保能捕获后续所有中间件及 Handler 的异常。