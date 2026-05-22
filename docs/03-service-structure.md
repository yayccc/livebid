# 项目结构规范

## 仓库结构

当前仓库使用 monorepo，把前端应用、后端微服务、接口定义、公共 Go 包、部署资源和项目文档放在同一个仓库中统一管理：

```text
livebid/
├── frontend/
│   ├── mobile-user/
│   └── merchant-admin/
├── services/
│   └── shop-service/
├── api/
│   ├── proto/
│   └── openapi/
├── gen/
├── pkg/
├── configs/
├── migrations/
├── docs/
├── deployments/
├── scripts/
├── Makefile
└── README.md
```

各目录职责如下：

| 目录/项目 | 功能说明 |
| --- | --- |
| `frontend/mobile-user` | 用户端 H5 应用，面向参与直播竞拍的普通用户，承载直播间浏览、竞拍出价、订单支付等移动端页面。 |
| `frontend/merchant-admin` | 商家/运营管理后台，面向主播、商家或运营人员，承载商品管理、直播间管理、竞拍配置、订单查看等后台页面。 |
| `services/shop-service` | 当前已落地的 Go 微服务骨架，用于承载店铺侧业务能力；后续可以按同样结构继续拆分 `user-service`、`goods-service`、`live-service`、`auction-service`、`order-service`、`payment-service` 等独立服务。 |
| `api/proto` | gRPC/Protobuf 接口定义目录，内部服务之间的 RPC 契约放在这里。 |
| `api/openapi` | OpenAPI/Swagger 接口定义目录，对外 HTTP API 文档和网关接口契约放在这里。 |
| `gen` | 由 `proto`、`openapi` 或其他代码生成工具产出的代码目录，原则上不手写核心业务逻辑。 |
| `pkg` | Go 公共包目录，放置多个服务可复用的基础能力，如鉴权、配置、错误处理、gRPC 封装、ID 生成、日志、消息队列、Redis 锁、统一响应和参数校验。 |
| `configs` | 仓库级配置目录，适合放本地联调、部署编排、共享中间件等配置示例；不强制集中存放每个服务自己的配置文件。 |
| `migrations` | 数据库迁移脚本目录，用于管理表结构变更和初始化数据。 |
| `docs` | 项目文档目录，包含项目总览、架构设计、服务结构规范等说明。 |
| `deployments` | 部署相关文件目录，例如 Docker Compose、Kubernetes、Nginx 或 CI/CD 部署配置。 |
| `scripts` | 开发、构建、测试、生成代码、部署等辅助脚本目录。 |
| `templates/go-service` | Go 微服务模板目录，新建服务时可以复制该模板，保持服务结构一致。 |
| `Makefile` | 项目级常用命令入口，用于封装构建、测试、代码生成等重复操作。 |
| `README.md` | 仓库入口说明，提供项目简介和快速开始信息。 |

## 单个微服务结构

以 `auction-service` 为例：

```text
services/auction-service/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── handler/
│   │   ├── auction_handler.go
│   │   └── bid_handler.go
│   ├── service/
│   │   ├── auction_service.go
│   │   ├── bid_service.go
│   │   └── hammer_service.go
│   ├── repository/
│   │   ├── auction_repository.go
│   │   └── bid_record_repository.go
│   ├── model/
│   │   ├── auction.go
│   │   └── bid_record.go
│   ├── dto/
│   ├── client/
│   ├── mq/
│   ├── job/
│   ├── router/
│   ├── config/
│   └── bootstrap/
├── tests/
├── configs/
├── Dockerfile
├── Makefile
└── README.md
```

各目录职责如下：

| 目录/文件 | 功能说明 |
| --- | --- |
| `cmd/server/main.go` | 服务启动入口，负责加载配置、初始化日志、数据库、Redis、MQ、RPC client、gRPC/HTTP server 等基础组件，并启动服务。 |
| `internal/handler` | 接口处理层，接收 gRPC 或网关 HTTP 请求，完成参数绑定、基础校验、用户上下文读取和响应返回，不直接写业务规则。 |
| `internal/service` | 业务逻辑层，负责编排核心业务流程，例如创建竞拍、出价校验、倒计时延长、落锤成交、事件发布等。 |
| `internal/repository` | 数据访问层，封装本服务数据库表的增删改查和查询组合，不处理 HTTP、WebSocket、MQ 等外部协议。 |
| `internal/model` | 数据模型目录，定义数据库实体、领域模型或持久化对象，例如 `Auction`、`BidRecord`。 |
| `internal/dto` | 数据传输对象目录，定义请求参数、响应结构、RPC 入参出参等，不直接等同于数据库模型。 |
| `internal/client` | 外部依赖客户端目录，封装对其他微服务、第三方 API、对象存储等外部系统的调用。 |
| `internal/mq` | 消息队列相关目录，放置事件生产者、消费者、消息结构和订阅处理逻辑。 |
| `internal/job` | 定时任务或异步后台任务目录，例如订单超时关闭、竞拍状态补偿、数据清理等。 |
| `internal/router` | 路由或服务注册目录，按服务类型集中管理 gRPC handler、HTTP 路由或中间件挂载。 |
| `internal/config` | 当前服务的配置结构和配置加载逻辑，例如数据库、Redis、MQ、端口、超时时间等。 |
| `internal/bootstrap` | 启动装配目录，负责把配置、基础组件、repository、service、handler、router 等对象组装起来。 |
| `tests` | 服务级测试目录，放置集成测试、接口测试或较完整的业务流程测试。 |
| `configs` | 当前服务自己的配置文件或配置模板，例如 `config.local.yaml`、`config.example.yaml`；真实敏感配置通过环境变量或部署系统注入。 |
| `Dockerfile` | 当前服务的容器构建文件。 |
| `Makefile` | 当前服务的常用命令入口，例如构建、测试、生成代码、运行本服务。 |
| `README.md` | 当前服务说明文档，记录服务职责、启动方式、依赖资源、接口说明和本地调试方法。 |

## AI 协作边界

这部分用于帮助 AI 或新协作者快速理解项目边界，改代码前优先遵守这些约束。具体业务规则和实体设计可以在开发对应功能时再补充，不需要在当前阶段提前定死。

- 只处理用户明确要求的功能，不主动拆分服务、不主动重构架构、不主动补齐目标架构中的全部模块。
- 目标架构中的服务如果当前仓库尚不存在，不要假设它已经可调用；需要时先基于现有模块实现，或在需求明确时再创建。
- 没有明确要求时，不新增数据库表、不新增微服务、不引入新的中间件或框架。
- 优先复用现有目录结构、公共包、命名方式和模板；新增代码尽量放在离功能最近的模块内。
- 根目录 `configs` 只放仓库级、本地联调或部署编排相关配置；单个服务自己的配置文件或模板优先放在 `services/<service>/configs`。
- `internal/config` 只定义配置结构和加载逻辑，不承载环境配置文件。
- 服务之间的同步调用契约放在 `api/proto`，生成代码放在 `gen`；不要手写或随意修改生成目录里的代码。
- 对外 HTTP 契约放在 `api/openapi`，由 `api-gateway` 统一承接；普通业务服务的 HTTP 路由只用于内部调试或健康检查。
- `pkg` 只放跨服务通用能力，例如日志、配置加载、错误码、响应封装、鉴权、gRPC 工具、MQ 工具、Redis 锁、参数校验；不要把某个业务域的规则下沉到 `pkg`。
- `internal/service` 可以编排 Redis、MQ、RPC client 和 repository；`internal/repository` 只能处理本服务数据库访问，不调用其他服务。
- 新增微服务时优先复制 `templates/go-service` 或参考现有服务结构，保持 `cmd/server`、`internal/*`、`tests`、`Dockerfile`、`Makefile`、`README.md` 的基本形态一致。
- 不要把密钥、真实数据库密码、支付密钥、模型 API Key 等敏感信息提交到仓库；配置文件只保留示例值或通过环境变量注入。

## 三层职责

handler 层：

- 接收 HTTP/gRPC 请求
- 参数绑定和基础校验
- 获取当前用户上下文
- 调用 service
- 统一返回响应
- 不写业务规则
- 不直接访问数据库、Redis、MQ

Service 层：

- 编排业务流程
- 控制事务
- 调用 repository
- 调用 Redis、MQ、RPC client
- 实现竞拍状态机
- 保证幂等和一致性

repository 层：

- 封装数据库访问
- 只处理本服务自己的数据表
- 不处理 HTTP、WebSocket、MQ
- 不直接返回前端 DTO
- 不调用其他服务

## 网关结构建议

`api-gateway` 一般不需要 repository：

```text
services/api-gateway/
├── cmd/server/main.go
├── internal/
│   ├── handler/
│   ├── service/
│   ├── client/
│   ├── middleware/
│   ├── router/
│   └── bootstrap/
└── README.md
```

`ws-gateway` 只管理连接和广播，不直接改竞拍数据库：

```text
services/ws-gateway/
├── cmd/server/main.go
├── internal/
│   ├── handler/
│   ├── service/
│   ├── client/
│   ├── ws/
│   ├── mq/
│   └── bootstrap/
└── README.md
```

## 公共包

`pkg` 只放真正跨服务复用的基础能力：

```text
pkg/
├── logger/
├── config/
├── response/
├── errors/
├── idgen/
├── redislock/
├── grpcx/
├── mq/
├── auth/
└── validator/
```

不要把业务逻辑放进 `pkg`。

## 命名规范
服务命名：

```text
api-gateway
ws-gateway
user-service
goods-service
live-service
auction-service
order-service
payment-service
ai-service
```

Go 文件：

```text
auction_handler.go
auction_service.go
auction_repository.go
bid_record_repository.go
```

前端文件：

```text
AuctionPanel.tsx
useAuction.ts
auctionStore.ts
```

Handler 命名：

```go
CreateAuction
StartAuction
PlaceBid
CloseAuction
GetAuctionDetail
ListBidRecords
```

Service 命名：

```go
CreateAuction
StartAuction
PlaceBid
HammerAuction
CreateAuctionOrder
```

repository 命名：

```go
FindByID
Create
UpdateStatus
UpdateHighestBid
ListByAuctionID
```
