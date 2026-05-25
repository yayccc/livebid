---
name: livebid-microservice
description: LiveBid 项目内用于开发 Go 微服务的本地工作流。用于新增或修改 services/*-service 下的微服务，尤其是用户用中文要求“开发微服务”“新增服务”“按照某个服务文档实现”“阅读 docs/services/{service-name} 文档”“先澄清需求再编码”“补充设计文档”“记录开发进度”“仿照 shop-service”“从 templates/go-service 复制模板”“先写 proto”“生成 gen/proto 代码”“实现 gRPC 服务、模型、仓储、配置、路由、测试和 README”等场景。
---

# LiveBid 微服务开发

## 目标

使用此项目局部 Skill 开发 LiveBid 后端微服务。开发时必须以 `services/shop-service` 为主要代码模板，以 `api/proto` 中的 proto 文件作为服务契约源头，并让现有项目结构决定实现方式。

## 必读上下文

动手修改前先阅读当前项目代码，不要凭记忆猜测：

- `templates/go-service/README.md` 与 `templates/go-service` 目录结构
- `services/shop-service/README.md`
- `api/proto/shop/v1/shop.proto`
- `services/shop-service/cmd/server/main.go`
- `services/shop-service/internal/config/config.go`
- `services/shop-service/internal/bootstrap/app.go`
- `services/shop-service/internal/router/grpc.go`
- `services/shop-service/internal/handler/*_grpc_handler.go`
- `services/shop-service/internal/model/*.go`
- `services/shop-service/internal/repository/*.go`
- 当前目标微服务的需求文档与设计文档

## 服务文档目录规范

每个新微服务的需求与设计文档优先放在：

```text
docs/services/<service-name>/
```

其中 `<service-name>` 使用完整服务目录名，例如 `goods-service`、`order-service`、`auction-service`。建议文件命名：

- `需求.md`：描述业务目标、核心功能、业务规则、接口能力。
- `设计.md`：描述数据库表、字段、状态枚举、接口文档、边界条件。
- `补充.md`：可选，用于记录临时补充、待确认问题、跨服务依赖。
- `开发进度.md`：记录每次开发的完成内容、未完成内容、阻塞点、验证结果和后续待办。

开发前按以下顺序查找文档：

1. 优先读取用户在提示词中明确给出的文档路径。
2. 如果用户只给出服务名，读取 `docs/services/<service-name>/` 下的全部 Markdown 文档。
3. 如果上述目录不存在，再在项目根目录和 `docs/` 中查找与服务中文名或英文名相关的文档，并在最终回复中建议沉淀到规范目录。

不要把某个具体服务的业务规则写死在 Skill 中。具体表结构、接口、状态、校验规则都必须从当前服务文档中提炼。

## 澄清与文档更新

阅读当前服务文档后，先做需求澄清，不要急着编码。

必须主动向用户提问的情况：

- 设计文档中的字段、枚举、接口、状态流转、权限边界或数据归属关系不明确。
- 需求和设计之间互相矛盾，或与 `shop-service` 的项目约定冲突。
- 文档中出现不合理设计，例如 SQL 语法错误、字段缺失、HTTP 接口无法直接映射为内部 gRPC、上传/支付/通知等能力缺少依赖服务。
- 当前服务需要调用其他服务，但对方服务、proto、鉴权方式、数据来源或失败处理没有定义。
- 用户要求的“完整开发”依赖尚未存在的基础设施，导致当前服务无法 100% 闭环。

提问方式：

- 先给出已理解的实现范围，再列出需要确认的问题。
- 问题要具体，最好带出可选方案和推荐方案。
- 若问题会影响 proto、数据库表、跨服务调用或核心业务规则，必须等待用户确认后再编码。
- 若问题只影响非核心细节，可采用保守默认值继续开发，但要把假设写入设计文档和开发进度。

文档更新规则：

- 不要删除用户原有文档中的任何文字。
- 对废弃内容，只能在原段落前后添加明显标识，例如：
  ```text
  【废弃说明：YYYY-MM-DD，本段因 xxx 被废弃，保留原文供追溯】
  ```
- 对新增内容，使用明显标识，例如：
  ```text
  【新增说明：YYYY-MM-DD，本段根据需求澄清补充】
  ```
- 对不明确但暂未确认的问题，追加到文档末尾的“待确认问题”区域。
- 对已确认的设计调整，追加到对应文档的“设计补充”或“需求补充”区域。
- 若没有合适区域，直接在文档末尾追加新段落，不移动、不重排、不覆盖原文。

## 开发流程

1. 先完成需求澄清和设计补充。
   - 阅读服务文档目录下的全部 Markdown 文档。
   - 按“从文档提炼实现清单”整理实现范围。
   - 主动指出不明确、不合理、跨服务依赖和无法 100% 完成的地方。
   - 根据用户回复，将结论追加到原设计文档或需求文档，严格遵守“只追加/标记，不删除原文”的规则。
   - 如果存在未解决的核心问题，先停止编码，等待用户确认。

2. 优先定义 proto 契约。
   - 新建 `api/proto/<domain>/v1/<domain>.proto`。
   - 使用 package `livebid.<domain>.v1`。
   - 使用 `option go_package = "github.com/yayccc/livebid/gen/proto/<domain>/v1;<domain>v1";`。
   - 将设计文档中的用户侧 HTTP API 抽象为内部 gRPC 方法。除非用户明确要求服务自身处理 HTTP 上传，否则 HTTP、multipart、网关注入等细节应留给 `api-gateway` 或后续接入层。
   - 部分更新接口使用 `optional` 标量字段，风格参考 `UpdateShopRequest`。
   - 列表查询需要定义分页 request/response。
   - 创建时间、更新时间使用 `google.protobuf.Timestamp`。
   - proto 方法应覆盖文档中的核心业务能力，但要将 HTTP 路径、HTTP 动词、multipart 上传等接入层细节转换为适合内部服务调用的 gRPC 语义。

3. 先生成 proto 代码，再编写依赖生成代码的 Go 实现。
   - 执行：
     ```bash
     protoc --go_out=. --go_opt=module=github.com/yayccc/livebid \
       --go-grpc_out=. --go-grpc_opt=module=github.com/yayccc/livebid \
       api/proto/<domain>/v1/<domain>.proto
     ```
   - 生成代码必须位于 `gen/proto/<domain>/v1`。
   - 如果本地缺少 `protoc` 或 Go 插件，保持 proto 文件完整，向用户说明阻塞点，不要手写或手改生成文件。

4. 从空微服务模板复制服务骨架。
   - 将 `templates/go-service` 复制为 `services/<domain>-service`。
   - 只有在真实文件替代占位文件时才删除 `.gitkeep`。
   - 维持与 `shop-service` 一致的目录布局：`cmd/server`、`configs`、`internal/bootstrap`、`internal/config`、`internal/handler`、`internal/model`、`internal/repository`、`internal/router`、`tests`。

5. 严格模仿 `shop-service` 的代码风格实现。
   - 除非现有文档或用户明确要求，否则服务保持 gRPC-only，不直接启动 Gin/HTTP REST。
   - 使用 `pkg/logger`、`pkg/idgen`、GORM repository、gRPC health check 等现有模式。
   - handler 保持轻量：参数校验、字符串 trim、调用 repository、model 转 proto、领域错误转 gRPC status code。
   - 数据库模型、状态常量、`TableName()` 放在 `internal/model`。
   - repository 接口、GORM 实现、软删除过滤、重复/未找到错误、列表过滤、必要事务放在 `internal/repository`。
   - 配置放在 `internal/config`，环境变量前缀使用大写服务名，例如 `<DOMAIN>_SERVICE_GRPC_ADDR`。
   - 只在不明显的业务规则、数据约束或复杂逻辑处添加简洁注释。

6. 验证与文档收尾。
   - 对改动过的 Go 文件运行 `gofmt`。
   - 先跑聚焦测试，再在可行时运行 `go test ./...`。
   - 新增或更新 `services/<domain>-service/README.md`，说明启动命令、配置文件路径、proto 路径、生成代码路径、proto 重新生成命令。
   - 新增或更新服务目录下的 `Makefile`，提供 `run`、`test`、`proto` 目标，风格参考 `shop-service`。
   - 新增或更新 `docs/services/<service-name>/开发进度.md`，记录本次开发结果。

## 从文档提炼实现清单

阅读当前服务文档后，先在心里形成实现清单，再开始写代码：

- 服务名：完整服务名 `<service-name>` 与领域名 `<domain>`，例如 `goods-service` 对应 `goods`。
- 数据模型：表名、字段、索引、软删除字段、状态枚举、时间字段。
- gRPC 方法：创建、更新、删除、详情、列表、批量查询、状态流转等能力，按文档实际内容取舍。
- 请求校验：必填字段、ID 合法性、分页默认值、字符串长度、状态流转限制。
- repository 行为：唯一冲突、未找到、软删除过滤、按归属方隔离、分页和排序。
- handler 行为：输入清洗、错误映射、model 与 proto 转换。
- 配置与端口：默认端口、环境变量前缀、MySQL 自动迁移、日志服务名。
- 测试重点：配置加载、repository 关键路径、handler 校验和错误映射。

## 开发进度记录

每次完成或阶段性停止开发时，都要在服务文档目录中维护：

```text
docs/services/<service-name>/开发进度.md
```

如果文件不存在则新建；如果已存在则追加新记录，不覆盖历史。每次记录建议包含：

- 日期时间：使用当前日期和本地时间。
- 本次目标：用户本轮要求完成什么。
- 已完成：proto、生成代码、服务结构、具体模块、文档更新等。
- 未完成：尚未实现的接口、测试、网关接入、跨服务调用等。
- 阻塞点：依赖其他服务、基础设施、用户确认、工具缺失等。
- 设计变更：本次追加到需求/设计文档的内容摘要。
- 验证结果：执行过的命令、通过/失败情况、失败原因。
- 后续建议：下一步最应该处理的事项。

如果当前服务因为跨服务依赖或设计缺口无法 100% 完成，也要提交可独立编译/测试的部分，并在 `开发进度.md` 中明确剩余工作和依赖条件。

## 质量要求

不要为单个新服务发明一套新架构。若需求文档与 `shop-service` 约定冲突，先向用户说明冲突并请求确认；用户确认后再把取舍追加到设计文档，并在最终回复中说明。
