# 部署与运行时说明

本目录记录 LiveBid 在本机、Docker Compose 和 Kubernetes 中的运行约定。

## 文档索引

- [服务发现与配置模型](service-discovery.md)：统一说明配置结构、端口策略、gRPC target 和健康检查。
- [Nacos 接入方案](nacos.md)：说明 Nacos 作为服务发现和配置中心时的命名、注册、配置规范。
- [Docker Compose 本地联调](docker-compose.md)：说明本地基础设施、容器内端口和最小联调方式。
- [Kubernetes 迁移兼容](kubernetes.md)：说明 K8s DNS、Service、Headless Service 和向后兼容策略。

## 推荐路线

当前阶段优先使用 Docker Compose 启动基础设施，业务服务可以本机裸跑或容器化运行。具体端口、target 和健康检查约定以 [服务发现与配置模型](service-discovery.md) 为准。

```text
本机/Compose：Nacos 服务发现
Kubernetes：Headless Service + DNS
```
