# LiveBid

LiveBid 是一个面向直播间实时竞拍场景的电商系统。项目目标是支持主播或运营创建直播间和竞拍商品，用户通过 H5 进入直播间后实时出价，系统完成最高价广播、倒计时延时、落锤成交闭环。

当前仓库采用 monorepo 管理前端应用、Go 微服务、接口定义、公共包、部署资源和项目文档。现阶段已经包含用户端 H5、商家后台、HTTP 网关、WebSocket 网关，以及商铺、用户、商品、直播、竞拍等 Go 服务。

## 依赖环境

基础开发环境：

| 依赖 | 说明 |
| --- | --- |
| Go `1.24+` | 后端微服务开发与测试，版本以 `go.mod` 为准。 |
| Docker + Docker Compose | 本地启动 MySQL、Redis、Nacos、RocketMQ、RustFS、SRS 和后端服务。 |
| Node.js + npm | 前端 Vite 应用开发与构建。 |
| curl | 健康检查和本地接口联调。 |

按需安装：

| 依赖 | 说明 |
| --- | --- |
| protoc | 修改 `api/proto` 后重新生成 gRPC 代码时需要。 |
| protoc-gen-go / protoc-gen-go-grpc | Go protobuf 代码生成插件。 |
| MySQL 客户端 | 裸跑服务或手动检查本地数据库时使用。 |

本地 Compose 会启动的主要中间件：

| 组件 | 默认版本或镜像 | 用途 |
| --- | --- | --- |
| MySQL | `mysql:8.4` | 业务数据存储。 |
| Redis | `redis:7.2-alpine` | 竞拍运行态、在线状态和缓存。 |
| Nacos | `nacos/nacos-server:v3.0.3` | 服务发现，后续可作为配置中心。 |
| RocketMQ | `apache/rocketmq:5.3.2` | 竞拍事件发布与消费。 |
| RustFS | `rustfs/rustfs:latest` | 本地 S3 兼容对象存储。 |
| SRS | `ossrs/srs:5` | RTMP/WebRTC 直播调试。 |

## 启动步骤

### 1. 启动后端和基础设施

从仓库根目录执行：

```bash
docker compose -f deployments/docker-compose.yml up --build
```

首次启动会构建并运行：

- `shop-service`
- `user-service`
- `goods-service`
- `live-service`
- `auction-service`
- `api-gateway`
- `ws-gateway`

同时会启动 MySQL、Redis、Nacos、RocketMQ、RustFS 和 SRS。

常用健康检查：

```bash
curl http://127.0.0.1:58080/health
curl http://127.0.0.1:58081/health
```

端到端接口测试：

```bash
scripts/e2e-api-gateway.sh
```

停止服务但保留数据卷：

```bash
docker compose -f deployments/docker-compose.yml down
```

停止服务并清理本地数据：

```bash
docker compose -f deployments/docker-compose.yml down -v
```

如果本地 WebRTC 调试需要指定 WSL 或宿主机可访问 IP，可以使用：

```bash
SRS_HOST=172.21.x.x scripts/start-compose-local.sh
```

更完整的 Compose 说明见 [docs/deployment/docker-compose.md](docs/deployment/docker-compose.md)。

### 2. 启动用户端 H5

```bash
cd frontend/mobile-user
npm install
npm run dev
```

用户端默认通过 Vite 代理访问本地后端：

| 路径 | 默认代理目标 |
| --- | --- |
| `/api` | `http://127.0.0.1:58080` |
| `/ws` | `ws://127.0.0.1:58081` |
| `/rtc` | `http://127.0.0.1:1985` |

端口不同时可覆盖代理目标：

```bash
VITE_API_PROXY_TARGET=http://127.0.0.1:58080 \
VITE_WS_PROXY_TARGET=ws://127.0.0.1:58081 \
VITE_SRS_PROXY_TARGET=http://127.0.0.1:1985 \
npm run dev
```

### 3. 启动商家后台

```bash
cd frontend/merchant-admin
npm install
npm run dev
```

### 4. 裸跑单个 Go 服务

Compose 是当前最完整的端到端启动方式。需要本机裸跑单个服务时，优先查看对应服务 README。

## 目录结构

```text
livebid/
├── api/                    # gRPC proto 和 OpenAPI 契约
│   ├── openapi/
│   └── proto/
├── configs/                # 仓库级配置示例
├── deployments/            # Docker Compose、Kubernetes、SRS、RocketMQ 等部署资源
├── docs/                   # 项目、架构、服务和部署文档
├── frontend/
│   ├── merchant-admin/     # 商家/运营后台
│   └── mobile-user/        # 用户端移动 H5
├── gen/                    # 生成代码，主要是 proto 生成物
├── migrations/             # 数据库迁移脚本
├── pkg/                    # Go 跨服务公共包
├── scripts/                # 本地联调、测试、部署辅助脚本
├── services/
│   ├── api-gateway/        # 对外 HTTP API 网关
│   ├── auction-service/    # 竞拍、出价、倒计时和成交
│   ├── goods-service/      # 商品、图片、鉴定和估价基础信息
│   ├── live-service/       # 直播间、推流和直播状态
│   ├── shop-service/       # 商铺、商家登录和商铺资料
│   ├── user-service/       # 用户、登录、资料和收货地址
│   └── ws-gateway/         # WebSocket 长连接和直播间消息
├── templates/              # 服务模板
├── tools/                  # 项目工具
├── Dockerfile              # Go 服务统一镜像构建入口
├── go.mod
└── README.md
```

单个 Go 服务遵循以下结构：

```text
services/<service-name>/
├── cmd/server/             # 服务启动入口
├── configs/                # 服务本地配置
├── internal/
│   ├── bootstrap/          # 依赖装配和启动流程
│   ├── client/             # 外部服务、MQ、存储等客户端
│   ├── config/             # 配置结构和加载逻辑
│   ├── handler/            # gRPC、HTTP 或 WebSocket 入口处理
│   ├── model/              # 数据模型
│   ├── repository/         # 数据访问
│   └── router/             # 路由或服务注册
├── Makefile
└── README.md
```

更详细的结构规范见 [docs/03-service-structure.md](docs/03-service-structure.md)。

## 配置说明

后端服务配置加载优先级：

```text
环境变量 > Nacos 配置中心 > 本地 YAML 文件 > 代码默认值
```

当前 Docker Compose 本地联调默认不启用 Nacos 配置中心，主要通过环境变量覆盖服务默认配置。每个服务都有自己的环境变量前缀，例如：

| 服务 | 配置文件 | 环境变量前缀 |
| --- | --- | --- |
| `api-gateway` | `services/api-gateway/configs/config.local.yaml` | `API_GATEWAY_` |
| `ws-gateway` | `services/ws-gateway/configs/config.local.yaml` | `WS_GATEWAY_` |
| `shop-service` | `services/shop-service/configs/config.local.yaml` | `SHOP_SERVICE_` |
| `user-service` | `services/user-service/configs/config.local.yaml` | `USER_SERVICE_` |
| `goods-service` | `services/goods-service/configs/config.local.yaml` | `GOODS_SERVICE_` |
| `live-service` | `services/live-service/configs/config.local.yaml` | `LIVE_SERVICE_` |
| `auction-service` | `services/auction-service/configs/config.local.yaml` | `AUCTION_SERVICE_` |

常见配置项：

| 配置类别 | 示例变量 | 说明 |
| --- | --- | --- |
| 监听地址 | `API_GATEWAY_HTTP_ADDR`、`WS_GATEWAY_HTTP_ADDR`、`SHOP_SERVICE_GRPC_ADDR` | HTTP、WebSocket 或 gRPC 监听地址。 |
| 数据库 | `<SERVICE_PREFIX>_MYSQL_DSN`、`<SERVICE_PREFIX>_MYSQL_AUTO_MIGRATE` | MySQL 连接和自动迁移开关。 |
| Redis | `AUCTION_SERVICE_REDIS_ADDR`、`WS_GATEWAY_REDIS_ADDR` | 竞拍运行态和 WebSocket 在线状态依赖。 |
| Nacos | `<SERVICE_PREFIX>_NACOS_SERVERS`、`<SERVICE_PREFIX>_NACOS_USERNAME`、`<SERVICE_PREFIX>_NACOS_PASSWORD` | 服务发现连接配置。 |
| 注册与配置中心 | `<SERVICE_PREFIX>_REGISTRY_ENABLED`、`<SERVICE_PREFIX>_CONFIG_CENTER_ENABLED` | 是否注册服务实例、是否从 Nacos 拉取配置。 |
| JWT | `LIVEBID_JWT_SECRET`、`API_GATEWAY_JWT_SECRET`、`USER_SERVICE_JWT_SECRET`、`WS_GATEWAY_JWT_SECRET` | 登录 token 签名和校验。 |
| RocketMQ | `AUCTION_SERVICE_ROCKETMQ_NAME_SERVER`、`WS_GATEWAY_ROCKETMQ_NAME_SERVER` | 竞拍事件消息队列。 |
| 对象存储 | `API_GATEWAY_STORAGE_ENDPOINT`、`API_GATEWAY_STORAGE_PUBLIC_BASE_URL` | RustFS/S3 上传和公开访问地址。 |
| 直播 | `SRS_PUBLIC_HOST`、`SRS_RTC_CANDIDATE`、`SRS_CALLBACK_BASE_URL` | SRS 推流、播放和回调地址。 |
| 前端代理 | `VITE_API_PROXY_TARGET`、`VITE_WS_PROXY_TARGET`、`VITE_SRS_PROXY_TARGET` | Vite 开发服务代理目标。 |

本地示例配置中的账号、密码和 JWT secret 只用于开发。共享测试环境和生产环境必须通过环境变量或部署系统注入真实密钥，不要把数据库密码、JWT 密钥、支付密钥、模型 API Key 等敏感信息提交到仓库。

## 文档索引

- [docs/01-overview.md](docs/01-overview.md)：项目定位、当前阶段和技术选型。
- [docs/02-architecture.md](docs/02-architecture.md)：目标微服务架构和服务边界。
- [docs/03-service-structure.md](docs/03-service-structure.md)：仓库结构和单服务结构规范。
- [docs/04-auth-design.md](docs/04-auth-design.md)：JWT 鉴权设计。
- [docs/deployment/README.md](docs/deployment/README.md)：部署与运行时说明入口。
- [docs/deployment/service-discovery.md](docs/deployment/service-discovery.md)：服务发现、配置加载和端口策略。
