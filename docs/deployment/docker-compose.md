# Docker Compose 本地联调

## 定位

当前阶段优先使用 Docker Compose 启动基础设施。业务服务可以本机裸跑，也可以容器化运行。

约定：

- 基础设施：Nacos、MySQL、Redis、MQ 等用 Compose 启动。
- 业务 gRPC 服务：按 [服务发现与配置模型](service-discovery.md) 的端口策略运行。
- `api-gateway`：HTTP 端口映射到宿主机。
- Nginx：只在测试多个 `api-gateway` 实例时作为可选 overlay，不作为第一阶段默认依赖。

## Nacos 单机模式

建议新增 `deployments/docker-compose.nacos.yml`：

```yaml
services:
  nacos:
    image: nacos/nacos-server:v2.4.3
    environment:
      MODE: standalone
      NACOS_AUTH_ENABLE: "false"
    ports:
      - "8848:8848"
      - "9848:9848"
    volumes:
      - nacos-data:/home/nacos/data

volumes:
  nacos-data:
```

端口说明：

- `8848`：Nacos 控制台和 OpenAPI。
- `9848`：Nacos 2.x gRPC 通信端口，Go SDK 访问服务发现时需要可达。

本地调试可以先关闭鉴权；共享测试环境和生产环境必须开启鉴权。

## 业务服务容器端口

业务服务容器化运行时，gRPC 端口不需要映射到宿主机：

```yaml
services:
  goods-service:
    build:
      context: ..
      dockerfile: services/goods-service/Dockerfile
    environment:
      GOODS_SERVICE_GRPC_ADDR: ":9000"
      GOODS_SERVICE_NACOS_SERVERS: "nacos:8848"
    expose:
      - "9000"
    depends_on:
      - nacos
```

说明：

- `expose` 只声明容器网络内可访问端口，不占用宿主机端口。
- 多个业务服务都可以监听容器内 `:9000`，因为每个容器都有独立 IP。
- `api-gateway` 的 HTTP 端口需要映射到宿主机，例如 `58080:58080`。
- Compose 阶段统一通过 Nacos 做服务发现，调用方 target 使用 `nacos:///<service-name>`。
