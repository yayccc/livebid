---
name: livebid-microservice
description: 用于 LiveBid 仓库内 services/*-service 的 Go 后端微服务工作：新建微服务、修改已有服务、更新 api/proto 契约、生成 gen/proto 代码、实现 gRPC handler/model/repository/config，或维护 docs/services/<service-name> 开发进度。不要用于纯前端任务、普通文档整理，或与 LiveBid 微服务无关的一次性 shell 操作。
---

# LiveBid 微服务

这个 skill 用于本仓库的 Go 微服务开发。保持实用：先读当前代码，判断任务类型，做最小安全改动，再验证。

## 信息来源

改服务行为前，优先读取当前项目文件，不要把这个 skill 当完整业务规格：

- 服务文档：优先读 `docs/services/<service-name>/`。
- 服务代码：`services/<service-name>/`。
- Proto 契约：`api/proto/<domain>/v1/<domain>.proto`。
- 生成代码：`gen/proto/<domain>/v1/`。
- 参考实现：`services/shop-service/` 和最接近的已有服务。
- 新服务骨架：`templates/go-service/`。

不要把具体服务的业务规则写进这个 skill。表结构、状态流转、归属关系、校验规则和 RPC 能力，都从当前文档、proto、测试和实现中确认。

## 通用规则

- 保护已有工作。禁止把 `templates/go-service` 复制覆盖到已有服务目录。
- 除非现有代码或用户明确要求，否则服务保持 gRPC-first，不在业务服务里直接扩 HTTP。
- `api/proto` 是手写契约；`gen/proto` 是生成结果。
- 不要手写或手改生成的 `*.pb.go`、`*_grpc.pb.go`。
- 修改已有服务 proto 时保持兼容：不要复用字段号、删除在用字段、随意改 package 或 service 名，除非用户明确确认破坏性变更。
- 遵守本地 Go 风格：轻量 handler、GORM repository、`internal/model`、`internal/config`、`internal/router`、`pkg/logger`、gRPC health check、`gofmt`。
- 修改用户写过的服务文档时只追加或标记废弃，不直接删除原文。
- 涉及核心契约且文档不清楚时先问用户：proto 形状、数据库 schema、状态流转、权限边界、跨服务调用、不可逆行为。
- 对很小且明确的修复，不要强制进入完整需求澄清流程；采用保守假设时，在最终回复或开发进度里说明。

## 按任务类型选择流程

### 修改已有服务

当 `services/<service-name>/` 已存在时走这条路径。

1. 读取服务文档、proto、生成包引用、服务 README/Makefile，以及本次最可能影响的代码文件。
2. 先确认当前行为和兼容性约束，再编辑。
3. 如果需要改 proto，在已有 proto 上增量修改，并用服务 Makefile 或相同风格的根目录 `protoc` 命令重新生成代码。
4. 在 handler/model/repository/config/router/client 中做最小范围改动。
5. 更新靠近改动点的测试。涉及校验、repository 行为或错误映射时，补聚焦测试。
6. 只有当行为、命令、配置、契约或已知缺口变化时，才更新 README 或 `docs/services/<service-name>/开发进度.md`。

### 新建服务

只有当 `services/<service-name>/` 不存在，且用户明确要新建微服务时走这条路径。

1. 从 `<service-name>` 去掉 `-service` 得到 `<domain>`。
2. 读取 `templates/go-service/README.md`、`templates/go-service/`、`services/shop-service/` 和该服务文档。
3. 复制 `templates/go-service` 到 `services/<service-name>`。
4. 补齐模板可能没有的必要文件，尤其是 `configs/config.local.yaml`、`cmd/server/main.go`、`internal/config/config.go`、`internal/bootstrap/app.go`、`internal/router/grpc.go`、handler、model、repository、`README.md`、`Makefile`。
5. 新建 `api/proto/<domain>/v1/<domain>.proto`，使用 package `livebid.<domain>.v1`，并设置 `go_package = "github.com/yayccc/livebid/gen/proto/<domain>/v1;<domain>v1";`。
6. 生成代码到 `gen/proto/<domain>/v1`。如果缺少 `protoc` 或插件，保持 proto 完整，跳过生成文件，并报告阻塞点。
7. 只有当同目录已有真实文件替代占位作用时，才删除 `.gitkeep`。

### 只更新 Proto 或生成代码

当用户只要求契约相关工作时走这条路径。

1. 修改前读取已有 proto 和调用方。
2. 除非用户明确确认破坏性变更，否则保留字段号、service 名和 package 名。
3. 局部更新优先使用 `optional` 标量字段；时间使用 `google.protobuf.Timestamp`；列表接口定义分页 request/response。
4. 将 HTTP 网关细节转换为内部 gRPC 语义；除非服务确实负责上传或 HTTP 入口，不要把 multipart、HTTP method、HTTP path 写进业务服务 proto。
5. 用现有服务 Makefile 的同类命令风格重新生成 `gen/proto`。

### 只做文档或方案

当用户要求 review、设计、规划、文档，不要求实现时走这条路径。

1. 读取相关服务文档和当前代码。
2. 具体指出不一致、缺失决策和实现影响。
3. 编辑文档时追加有日期的说明或小节。除非用户明确要求，不要重写旧决策。

## 服务文档

服务文档优先放在：

```text
docs/services/<service-name>/
```

本仓库文件名并不完全统一，常见有 `需求.md`、`服务设计.md`、`设计.md`、`需求分析.md`、`README.md`、`开发进度.md`。做较大改动前，读取该服务文档目录下全部 Markdown 文件。

追加开发进度时包含：

- 本地时区日期时间。
- 用户本轮目标。
- 已修改文件或模块。
- 验证命令和结果。
- 剩余工作、阻塞点和假设。

## 收尾检查

把这部分当最终检查清单，不要当死板流水线：

- 已读取相关文档和现有代码。
- 没有覆盖已有服务目录。
- Proto 改动兼容，或破坏性变更已获得明确确认。
- 生成文件与 proto 匹配，且没有手工编辑生成文件。
- 配置路径、环境变量前缀、端口和服务名与目标服务一致。
- Handler 将校验错误和领域错误映射为合适的 gRPC status code。
- Repository 处理未找到、唯一冲突、软删除、分页和必要事务。
- 改过的 Go 文件已运行 `gofmt`。
- 已运行聚焦测试；可行时运行更大范围的 `go test ./...`。
- README、Makefile、服务开发进度文档反映了有意义的行为或流程变化。
