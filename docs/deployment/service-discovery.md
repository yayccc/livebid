# 服务发现与配置模型

编写日期：2026-05-31

## 目标

- 调用方通过服务名发现 gRPC 实例，不在业务配置中硬编码每个服务的 IP 和端口。
- 业务代码只读取 gRPC `target`，不判断当前运行在本机、Docker Compose 还是 Kubernetes。
- 配置中心、服务注册、健康检查各自独立开关，避免一个字段承担多重含义。

## 公共包

```text
pkg/
├── nacosx/
│   ├── config.go
│   ├── client.go
│   ├── registry.go
│   └── resolver.go
└── grpcx/
    └── client.go
```

- `pkg/nacosx` 封装 Nacos SDK 细节。
- `pkg/grpcx` 封装 gRPC Dial、超时、resolver 和负载均衡配置。
- 服务自己的 `internal/config` 仍负责定义业务配置结构和环境变量覆盖规则。

## 配置结构

```yaml
nacos:
  enabled: true
  servers:
    - host: "127.0.0.1"
      port: 8848
  namespaceId: "livebid-local"

configCenter:
  enabled: true
  dataId: "livebid.local.goods-service.yaml"
  group: "LIVEBID"

registry:
  enabled: true
  provider: "nacos"
  serviceName: "goods-service"
  group: "LIVEBID"
  cluster: "default"
  ip: ""
  port: 0
  weight: 1

healthCheck:
  enabled: true
  intervalSeconds: 5
  timeoutSeconds: 2
  failureThreshold: 3
```

- `nacos.enabled` 表示是否初始化 Nacos client。
- `configCenter.enabled` 表示是否从配置中心拉取业务配置。
- `registry.enabled` 表示当前进程是否主动注册服务实例。
- `registry.port: 0` 表示使用 `listener.Addr()` 得到的真实监听端口。
- 稳定 metadata 如 `protocol=grpc`、`env=<env>` 由 `pkg/nacosx` 自动补充。

## 加载优先级

```text
环境变量 > Nacos 配置中心 > 本地 YAML 文件 > 代码默认值
```

加载流程：

```text
defaultConfig()
 -> loadFromLocalFile()
 -> loadFromNacos()
 -> applyEnvOverrides()
 -> normalize()
```

## 端口策略

| 运行形态 | 推荐监听地址 | 说明 |
| --- | --- | --- |
| 本机裸跑 | `:0` | 多个服务进程共享宿主机网络命名空间，用随机端口避免冲突，启动后把真实端口注册到 Nacos。 |
| Docker Compose | `:9000` | 每个容器有独立 IP，业务 gRPC 端口不映射宿主机。 |
| Kubernetes | `:9000` | 每个 Pod 有独立 IP，Headless Service 映射到统一 gRPC 端口。 |

`api-gateway`、`ws-gateway` 等对外入口服务仍使用固定 HTTP/WebSocket 端口。

## Target 约定

调用方配置统一使用 `target`：

```yaml
goodsService:
  target: "nacos:///goods-service"
  timeoutSeconds: 3
```

| 场景 | target 示例 |
| --- | --- |
| 本机裸跑 + Nacos | `nacos:///goods-service` |
| Docker Compose + Nacos | `nacos:///goods-service` |
| Kubernetes DNS | `dns:///goods-service.livebid.svc.cluster.local:9000` |
| 本机静态直连兜底 | `127.0.0.1:9002` |

`pkg/grpcx` 统一配置 `round_robin`。Kubernetes 内部 gRPC 服务统一使用 Headless Service 配合 `dns:///...:9000`。

## 健康检查

所有内部 gRPC 服务必须注册标准 gRPC Health Checking 服务：

```text
grpc.health.v1.Health/Check
```

服务启动并完成依赖初始化后，将整体服务名 `""` 和当前业务 service name 设置为 `SERVING`。进入优雅关闭流程时，先把状态改为 `NOT_SERVING`，再注销实例或等待 Kubernetes readiness 摘除。

职责分层：

- 服务自身：暴露 gRPC health，反映进程和关键依赖是否可服务。
- Nacos 阶段：`pkg/nacosx` 定期执行 gRPC health check，失败时更新实例健康状态或注销实例。
- Kubernetes 阶段：使用 gRPC readinessProbe/livenessProbe，Pod 未 ready 时不进入 EndpointSlice。

## 验收清单

- [ ] 本机裸跑时服务可以监听 `:0` 并注册真实随机端口。
- [ ] Docker Compose 容器中多个业务服务可以同时监听 `:9000`。
- [ ] `api-gateway` 可通过 `nacos:///goods-service` 调用业务服务。
- [ ] `api-gateway` 可通过 `dns:///goods-service.livebid.svc.cluster.local:9000` 在 Kubernetes 中调用业务服务。
- [ ] 从 `nacos:///` target 切换到 `dns:///` target 不需要修改业务 handler 或 client 代码。
