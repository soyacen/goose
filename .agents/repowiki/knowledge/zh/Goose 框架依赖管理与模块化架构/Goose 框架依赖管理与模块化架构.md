---
kind: dependency_management
name: Goose 框架依赖管理与模块化架构
category: dependency_management
scope:
    - '**'
source_files:
    - go.mod
    - middleware/jwtauth/go.mod
    - middleware/limiter/go.mod
    - middleware/otel/go.mod
    - third_party/google/api/annotations.proto
    - Makefile
---

## 1. 核心依赖管理系统
Goose 采用标准的 **Go Modules** (`go.mod`/`go.sum`) 进行依赖管理。项目基于 Go 1.23.0 构建，严格遵循语义化版本控制。

### 关键依赖库
- **Protobuf 生态**: `google.golang.org/protobuf` (v1.36.10) 用于消息序列化与代码生成；`google.golang.org/genproto` 提供 Google API 标准定义（如 HTTP annotations）。
- **网络与工具**: `golang.org/x/net` 处理底层 HTTP 逻辑，`golang.org/x/exp` 提供实验性约束支持，`github.com/google/go-querystring` 处理 URL 查询参数编解码。

## 2. 多模块仓库 (Multi-module Repo) 架构
为了降低使用者的依赖体积并实现关注点分离，Goose 采用了**多模块仓库**策略。除了根目录的主模块外，部分功能独立的中间件被划分为子模块：

- **主模块**: `github.com/soyacen/goose` (根目录)
- **子模块**:
  - `middleware/jwtauth`: 集成 `github.com/golang-jwt/jwt/v5`。
  - `middleware/limiter`: 集成 `github.com/shirou/gopsutil/v4` 进行系统资源监控。
  - `middleware/otel`: 集成 OpenTelemetry SDK (`go.opentelemetry.io/otel`)。

### 本地开发协同
在子模块的 `go.mod` 中，通过 `replace github.com/soyacen/goose => ../../` 指令，将对外部发布版本的引用重定向到本地根目录。这确保了在开发过程中，中间件能实时同步主框架的变更，而无需频繁发布新版本。

## 3. 第三方协议定义管理
项目在 `third_party/google/` 目录下 vendoring（内嵌）了关键的 Protobuf 定义文件（如 `annotations.proto`, `http.proto`）。这种做法确保了 `protoc-gen-goose` 插件在生成代码时，能够稳定地引用 Google API 标准，而不受外部 `protoc` 包含路径配置的影响。

## 4. 开发者规范
- **依赖更新**: 修改根目录或子模块的依赖后，需运行 `go mod tidy` 保持 `go.sum` 同步。
- **代码生成**: 使用 `make example` 或手动执行 `protoc` 时，必须通过 `--proto_path` 显式包含 `./third_party` 目录，以解析 Google API 扩展选项。
- **模块引用**: 若在外部项目中使用 Goose 的特定中间件，应直接导入对应的子模块路径（如 `github.com/soyacen/goose/middleware/otel`），以避免引入不必要的间接依赖。