## 1. 核心系统与模式

Goose 框架**不包含**传统的集中式配置文件加载机制（如 YAML、TOML 或 `.env` 文件解析）。其配置系统完全基于 **Go 语言惯用的函数选项模式 (Functional Options Pattern)**。

- **配置载体**：通过 `Option` 函数类型在运行时动态构建和修改内部 `options` 结构体。
- **配置层级**：配置分为两个主要维度：**服务端 (Server)** 和 **客户端 (Client)**，各自拥有独立的配置上下文。
- **中间件配置**：各个中间件（如 JWT、限流器）也独立采用相同的函数选项模式进行局部配置。

## 2. 关键文件与包

### 核心运行时配置
- `server/option.go`: 定义服务端配置项，包括 JSON 编解码选项 (`protojson`)、错误编码器、中间件链、快速失败模式及验证回调。
- `client/option.go`: 定义客户端配置项，包括 HTTP 客户端实例、URL 解析器 (`resolver.Resolver`)、编解码选项及错误处理工厂。

### 中间件配置示例
- `middleware/jwtauth/middleware.go`: 展示如何通过 `Realm`, `SigningMethod` 等选项配置 JWT 认证行为。
- `middleware/limiter/bbr.go`: 展示如何通过 `WithWindow`, `WithCPUThreshold` 等选项配置 BBR 自适应限流算法的参数。

### 代码生成器配置
- `cmd/protoc-gen-goose/main.go`: 使用标准库 `flag` 包处理编译期参数（如 `--goose_opt=openapi=true`），用于控制代码生成的行为。

## 3. 架构设计与约定

### 3.1 函数选项模式实现
每个可配置的组件都遵循以下结构：
1.  **Options 接口**：提供对配置值的只读访问（如 `server.Options`）。
2.  **内部结构体**：存储实际配置状态（如 `server.options`）。
3.  **Option 函数类型**：`type Option func(o *options)`，用于闭包修改内部状态。
4.  **构造函数**：如 `server.NewOptions(opts ...Option)`，负责初始化默认值并应用用户提供的选项。

### 3.2 默认值与修正
- **服务端**：在 `NewOptions` 中直接初始化默认值（如默认的 `protojson` 选项和错误编码器）。
- **客户端**：引入了 `Correct()` 方法，在应用选项后对空值进行兜底处理（如自动创建 `http.Client` 实例）。

### 3.3 模块化隔离
配置高度模块化。例如，限流器的 CPU 阈值配置仅存在于 `limiter` 包内，不会污染全局命名空间。这种设计使得 Goose 作为一个库（Library）而非独立服务（Service）时，能够灵活地嵌入到任何宿主应用中。

## 4. 开发者规范

1.  **禁止硬编码配置**：在扩展中间件或核心功能时，必须通过 `Option` 函数暴露可配置项，严禁在代码中写死业务逻辑相关的参数。
2.  **默认值安全**：所有配置项必须有合理的默认值。如果某个配置项缺失会导致 panic，必须在 `NewOptions` 或 `Correct` 中进行初始化。
3.  **不可变性原则**：一旦通过 `NewOptions` 创建了配置实例，建议在运行时保持其不可变。如果需要动态变更（如动态限流阈值），应通过中间件内部的状态管理（如 `atomic` 变量）实现，而非重新构建整个 Server/Client 实例。
4.  **环境变量处理**：由于 Goose 是底层框架，它不直接读取环境变量。开发者应在调用 `NewOptions` 的上层应用代码中读取环境变量，并将其转换为对应的 `Option` 传入。