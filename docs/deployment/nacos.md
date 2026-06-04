# Nacos 接入方案

## 定位

Nacos 在当前阶段承担两个职责：

- 服务发现：本机裸跑和 Docker Compose 阶段，业务服务主动注册 gRPC 实例。
- 配置中心：各服务从 Nacos 拉取 YAML 配置，并保留本地 YAML 和环境变量兜底。

Kubernetes 环境中，Nacos 可以继续作为配置中心使用；服务发现切到 K8s Headless Service / DNS。

## 依赖

```bash
go get github.com/nacos-group/nacos-sdk-go/v2
```

SDK 调用集中封装在 `pkg/nacosx`，业务服务不要直接散落 Nacos SDK 代码。

## 引导配置

Nacos 引导配置必须来自本地 YAML 或环境变量，因为不能先从 Nacos 读取 Nacos 地址。

```yaml
nacos:
  enabled: true
  servers:
    - host: "127.0.0.1"
      port: 8848
  namespaceId: "livebid-local"
  username: ""
  password: ""
  timeoutMs: 5000
```

这段只负责初始化 Nacos client。配置中心和服务注册分别由 `configCenter.enabled`、`registry.enabled` 控制。

环境变量只作为覆盖项，不要求每个服务全部配置。

常用覆盖项：

```text
<SERVICE_PREFIX>_NACOS_ENABLED
<SERVICE_PREFIX>_NACOS_SERVERS
<SERVICE_PREFIX>_NACOS_NAMESPACE_ID
<SERVICE_PREFIX>_NACOS_USERNAME
<SERVICE_PREFIX>_NACOS_PASSWORD
```

Docker Compose 本地联调默认启用 Nacos 认证，各服务默认使用 `nacos` / `nacos` 作为客户端账号密码连接 Nacos。各服务通过自己的前缀注入，例如：

```text
SHOP_SERVICE_NACOS_USERNAME=nacos
SHOP_SERVICE_NACOS_PASSWORD=nacos
API_GATEWAY_NACOS_USERNAME=nacos
API_GATEWAY_NACOS_PASSWORD=nacos
```

共享测试环境和生产环境必须替换默认账号密码和 `NACOS_AUTH_TOKEN`。

注意：`<SERVICE_PREFIX>_NACOS_USERNAME` / `<SERVICE_PREFIX>_NACOS_PASSWORD` 只影响服务连接 Nacos 的 SDK 客户端账号，不会初始化 Nacos 服务端管理员密码。首次使用新的 Nacos 数据卷启动时，先访问 `http://localhost:8080` 按控制台提示初始化或确认管理员账号密码，再让 Compose 中的 `NACOS_USERNAME` / `NACOS_PASSWORD` 与该账号保持一致。

按需覆盖项：

```text
<SERVICE_PREFIX>_CONFIG_CENTER_ENABLED
<SERVICE_PREFIX>_CONFIG_CENTER_DATA_ID
<SERVICE_PREFIX>_REGISTRY_ENABLED
<SERVICE_PREFIX>_REGISTRY_IP
<SERVICE_PREFIX>_REGISTRY_PORT
<SERVICE_PREFIX>_HEALTH_CHECK_ENABLED
```

高级覆盖项：

```text
<SERVICE_PREFIX>_NACOS_TIMEOUT_MS
<SERVICE_PREFIX>_CONFIG_CENTER_GROUP
<SERVICE_PREFIX>_REGISTRY_PROVIDER
<SERVICE_PREFIX>_REGISTRY_GROUP
<SERVICE_PREFIX>_REGISTRY_CLUSTER
<SERVICE_PREFIX>_HEALTH_CHECK_INTERVAL_SECONDS
<SERVICE_PREFIX>_HEALTH_CHECK_TIMEOUT_SECONDS
<SERVICE_PREFIX>_HEALTH_CHECK_FAILURE_THRESHOLD
```

`<SERVICE_PREFIX>` 沿用现有服务约定，例如 `SHOP_SERVICE`、`GOODS_SERVICE`、`AUCTION_SERVICE`、`API_GATEWAY`。

## 配置中心

Nacos 配置使用 YAML 格式：

```text
Data ID: livebid.<env>.<service>.yaml
Group:   LIVEBID
```

示例：

```text
livebid.local.goods-service.yaml
livebid.dev.auction-service.yaml
```

`configCenter.dataId` 默认按 `livebid.<env>.<service>.yaml` 生成，只有特殊场景才需要显式覆盖。

## 服务注册

服务名使用完整服务目录名：

```text
shop-service
goods-service
user-service
auction-service
live-service
```

实例字段约定：

| 字段 | 建议值 |
| --- | --- |
| ServiceName | `goods-service` |
| GroupName | `LIVEBID` |
| ClusterName | `default` |
| IP | 容器或主机可被调用方访问的 IP |
| Port | gRPC 实际监听端口，从 `listener.Addr()` 解析得到 |
| Weight | 默认 `1` |
| Enable | `true` |
| Healthy | `true` |
| Ephemeral | `true` |

`protocol=grpc`、`env=<env>` 等稳定 metadata 由 `pkg/nacosx` 自动补充，不要求每个服务重复配置。

注册时机放在 `Run()` 中成功监听端口之后，避免端口占用或启动失败时产生不可用实例。

## 健康检查

业务服务必须注册标准 gRPC Health Checking 服务：

```text
grpc.health.v1.Health/Check
```

Nacos 注册侧由 `pkg/nacosx` 管理主动探活：

1. 服务成功监听端口后注册实例，初始 `Healthy=true`。
2. 按 `healthCheck.intervalSeconds` 调用本实例的 gRPC health check。
3. 连续失败达到 `failureThreshold` 后，更新实例为不健康或注销实例。
4. 恢复成功后，再更新为健康或重新注册实例。
5. 收到退出信号时，先把 gRPC health 状态改为 `NOT_SERVING`，再注销实例。

## 安全与故障处理

- `prod` 必须开启 Nacos 鉴权，账号密码通过环境变量注入。
- 数据库密码、JWT 密钥、支付密钥、模型 API Key 不提交到仓库。
- 不同环境使用不同 namespace，例如 `livebid-local`、`livebid-dev`、`livebid-test`、`livebid-prod`。
- Nacos 不可用但服务已经启动：本地 gRPC 服务继续运行，调用方依赖本地缓存和已有连接。
- 服务启动时无法连接 Nacos：如果 `nacos.enabled=true` 且无本地兜底配置，应启动失败并输出明确日志。
- 配置中心拉取失败：本地 YAML 存在时使用本地 YAML 并打印 warn 日志；生产环境建议视为启动失败。
- 服务发现无实例：调用方返回明确的上游不可用错误，由 `api-gateway` 映射为统一 HTTP 错误响应。
