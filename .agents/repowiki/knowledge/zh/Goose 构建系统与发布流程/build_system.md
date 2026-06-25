## 1. 核心构建体系
Goose 项目采用标准的 **Go Modules** 进行依赖管理，并配合 **Makefile** 实现本地开发、测试与代码生成的自动化。其核心是一个 `protoc` 插件 (`protoc-gen-goose`)，用于将 Protobuf 定义转换为 Go HTTP 网关代码。

### 构建工具链
- **语言版本**: Go 1.23+
- **包管理**: Go Modules (主模块及多个子模块)
- **自动化脚本**: Makefile
- **CI/CD**: GitHub Actions

## 2. 关键文件与职责

| 文件路径 | 职责描述 |
| :--- | :--- |
| `Makefile` | 定义了 `build` (编译插件), `install` (安装到 GOPATH), `test` (运行全量测试) 和 `example` (生成示例代码) 等核心任务。 |
| `cmd/protoc-gen-goose/main.go` | 插件入口，包含版本号定义 (`Version`)，是发布流程中版本更新的目标文件。 |
| `.github/workflows/release.yml` | 自动化发布工作流，负责版本校验、源码版本替换、Git Tag 创建及 GitHub Release 发布。 |
| `tools/build.go` | 一个特殊的 Go 文件，通过 `_` 导入确保构建时拉取必要的间接依赖（如 `golang.org/x/exp`, `google.golang.org/protobuf` 等）。 |
| `middleware/*/go.mod` | 中间件组件（jwtauth, limiter, otel）采用独立子模块管理，通过 `replace` 指令指向根目录以支持本地开发。 |

## 3. 架构与约定

### 多模块仓库 (Multi-module Repo)
项目采用了多模块结构：
- **根模块**: `github.com/soyacen/goose`
- **子模块**: `middleware/jwtauth`, `middleware/limiter`, `middleware/otel`。
这种设计允许用户按需引入中间件，减少依赖体积。在 `go.mod` 中通过 `replace github.com/soyacen/goose => ../../` 保持子模块与主模块的同步开发。

### 版本管理策略
- **语义化版本**: 遵循 `vX.Y.Z` 格式。
- **全局同步**: 发布时，根模块、所有子模块以及 `main.go` 中的硬编码版本号必须保持一致。
- **Tag 规范**: 
  - 根模块 Tag: `v1.2.3`
  - 子模块 Tag: `middleware/jwtauth/v1.2.3`

### 代码生成流程
通过 `make example` 触发 `protoc` 命令，利用 `--goose_out` 和 `--goose_opt=openapi=true` 参数，从 `.proto` 文件同时生成 Go 业务代码和 OpenAPI JSON 文档。

## 4. 开发者须知

1. **本地构建**: 使用 `make install` 将最新的插件安装到环境中，以便在其他项目中调用。
2. **依赖同步**: 修改根模块接口后，需确保子模块的 `go.mod` 能通过 `replace` 正确引用最新代码。
3. **发布流程**: 
   - 严禁手动修改版本号。
   - 必须通过 GitHub Actions 的 `Release` 工作流（Workflow Dispatch）触发，输入目标版本号（如 `v1.8.0`）。
   - 工作流会自动完成代码修改、提交、打标签和创建 Release 的全过程。
4. **测试规范**: 提交前务必执行 `make test`，确保所有单元测试及示例代码的集成测试通过。