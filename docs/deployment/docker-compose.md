# Docker Compose 本地联调

## 定位

当前阶段优先使用 Docker Compose 启动基础设施。业务服务可以本机裸跑，也可以容器化运行；仓库已提供一份用于端到端联调的完整 Compose 文件：

```text
deployments/docker-compose.yml
```

约定：

- 基础设施：Nacos、MySQL、Redis、MQ 等用 Compose 启动。
- 业务 gRPC 服务：按 [服务发现与配置模型](service-discovery.md) 的端口策略运行。
- `api-gateway`：HTTP 端口映射到宿主机。
- Nginx：只在测试多个 `api-gateway` 实例时作为可选 overlay，不作为第一阶段默认依赖。

## 快速启动

从仓库根目录执行：

```bash
docker compose -f deployments/docker-compose.yml up --build
```

首次启动会构建 `shop-service`、`user-service`、`goods-service`、`live-service`、`auction-service`、`api-gateway` 和 `ws-gateway` 七个镜像，并启动 MySQL、Redis、Nacos、RocketMQ、RustFS 与 SRS。

本地 WebRTC 调试时，如果浏览器需要访问宿主机/WSL 地址 `172.21.103.73`，可以使用脚本启动：

```bash
scripts/start-compose-local.sh
```

脚本默认设置：

```text
SRS_PUBLIC_HOST=172.21.103.73
SRS_RTC_CANDIDATE=172.21.103.73
```

因此 SRS 返回的 WebRTC ICE candidate，以及 `live-service` 返回给前端的 RTMP/WebRTC 地址会使用同一个本地可访问 IP。需要临时改 IP 时：

```bash
SRS_HOST=172.21.x.x scripts/start-compose-local.sh
```

HTTP 网关健康检查：

```bash
curl http://127.0.0.1:58080/health
```

WebSocket 网关健康检查：

```bash
curl http://127.0.0.1:58081/health
```

端到端接口测试：

```bash
scripts/e2e-api-gateway.sh
```

脚本默认请求 `http://127.0.0.1:58080`，会自动注册测试商家和用户，并串起商铺、用户、地址、文件上传、商品、直播、SRS 回调和拍卖接口。若网关端口不同，可通过 `BASE_URL` 覆盖：

```bash
BASE_URL=http://127.0.0.1:58080 scripts/e2e-api-gateway.sh
```

停止但保留数据卷：

```bash
docker compose -f deployments/docker-compose.yml down
```

停止并清理本地数据：

```bash
docker compose -f deployments/docker-compose.yml down -v
```

## Nacos 单机模式

Compose 中内置 Nacos 单机模式，配置为：

- `MODE=standalone`
- `NACOS_AUTH_ENABLE=true`
- 默认认证 token：`MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=`
- 服务侧默认使用 `NACOS_USERNAME=nacos`、`NACOS_PASSWORD=nacos` 连接 Nacos。

端口说明：

- `8080`：Nacos 3.x 控制台。
- `8848`：Nacos OpenAPI。
- `9848`：Nacos SDK gRPC 通信端口。
- `9849`：Nacos Raft/内部通信端口。

Compose 中保留了 `NACOS_AUTH_ENABLE`、`NACOS_AUTH_TOKEN`、`NACOS_AUTH_IDENTITY_KEY`、`NACOS_AUTH_IDENTITY_VALUE`、`NACOS_USERNAME`、`NACOS_PASSWORD` 覆盖项，便于和本机已有 Nacos 脚本保持一致。共享测试环境和生产环境必须替换默认认证信息。

注意：`NACOS_USERNAME` / `NACOS_PASSWORD` 是业务服务连接 Nacos 时使用的客户端账号密码，不负责初始化 Nacos 服务端管理员密码。首次使用新的 `nacos-data` volume 启动且开启认证时，先打开 `http://localhost:8080` 按 Nacos 控制台提示初始化或确认管理员账号密码；之后保持 Compose 中的 `NACOS_USERNAME` / `NACOS_PASSWORD` 与该账号一致。

## 业务服务容器端口

业务服务容器化运行时，gRPC 端口不需要映射到宿主机：

```yaml
services:
  goods-service:
    build:
      context: ..
      dockerfile: Dockerfile
      args:
        SERVICE: goods-service
    environment:
      GOODS_SERVICE_GRPC_ADDR: ":9000"
      GOODS_SERVICE_NACOS_SERVERS: "nacos:8848"
      GOODS_SERVICE_NACOS_USERNAME: "nacos"
      GOODS_SERVICE_NACOS_PASSWORD: "nacos"
      GOODS_SERVICE_REGISTRY_ENABLED: "true"
      GOODS_SERVICE_REGISTRY_PORT: "9000"
    expose:
      - "9000"
    depends_on:
      - nacos
```

说明：

- `expose` 只声明容器网络内可访问端口，不占用宿主机端口。
- 多个业务服务都可以监听容器内 `:9000`，因为每个容器都有独立 IP。
- `api-gateway` 的 HTTP 端口和 `ws-gateway` 的 WebSocket 端口需要映射到宿主机，例如 `58080:58080`、`58081:58081`。
- Compose 阶段统一通过 Nacos 做服务发现，调用方 target 使用 `nacosx:///<service-name>`。

## 配置策略

本地 Compose 阶段不启用 Nacos 配置中心，只用环境变量覆盖本地默认配置：

- MySQL DSN 使用 `mysql:3306`。
- Redis 地址使用 `redis:6379`。
- Nacos 地址使用 `nacos:8848`。
- Nacos 客户端账号默认使用 `nacos` / `nacos`，服务侧通过各自的 `<SERVICE_PREFIX>_NACOS_USERNAME` 和 `<SERVICE_PREFIX>_NACOS_PASSWORD` 注入。
- RocketMQ nameserver 使用 `rocketmq-namesrv:9876`。
- api-gateway 下游 target 使用 `nacosx:///shop-service`、`nacosx:///user-service` 等。
- ws-gateway 下游 target 使用 `nacosx:///live-service` 和 `nacosx:///auction-service`，并通过广播消费组 `ws-gateway-auction-broadcast` 消费 `auction_event`。
- RustFS S3 endpoint 使用容器内地址 `http://rustfs:9000`，对外返回 URL 使用宿主机地址 `http://127.0.0.1:9000/livebid`。
- SRS 使用 `deployments/srs/livebid.conf`，RTMP 推流端口默认 `1935`，HTTP API 默认 `1985`，HTTP 文件服务默认映射到宿主机 `8088`，WebRTC UDP 默认 `8000`。SRS 回调在 Compose 网络内请求 `http://api-gateway:58080/api/srs/callbacks/{publish,unpublish}`。

## 对外端口

| 组件 | 宿主机端口 | 说明 |
| --- | --- | --- |
| api-gateway | `58080` | 对外 HTTP 入口 |
| ws-gateway | `58081` | 对外 WebSocket 入口 |
| SRS | `1935` / `1985` / `8088` / `8000/udp` | RTMP、HTTP API、HTTP 文件服务、WebRTC UDP |
| MySQL | `3306` | 本地调试数据库 |
| Redis | `6379` | 本地调试缓存 |
| Nacos | `8080` / `8848` / `9848` / `9849` | 控制台、OpenAPI、SDK gRPC 与内部通信 |
| RocketMQ | `9876` / `10909` / `10911` | nameserver 与 broker |
| RustFS | `9000` / `9001` | S3 API 与控制台 |

## 注意事项

- 业务服务镜像共用根目录 `Dockerfile`，通过 build arg `SERVICE` 选择要构建的服务。
- `api-gateway` 依赖 Nacos resolver，业务服务注册完成前短时间请求可能返回下游不可用；等待几秒或查看容器日志即可。
- 当前 `auction-service` 启动依赖 RocketMQ broker，若本机资源较紧张，RocketMQ 启动可能需要更长时间。
- 如果本机已经运行 MySQL、Nacos、Redis、RocketMQ 或 RustFS，启动前需要先释放对应宿主机端口，或在 `deployments/docker-compose.yml` 中调整端口映射。
