# api-gateway

`api-gateway` 是对外 HTTP 入口。当前阶段只承接商铺注册、商铺登录两个接口，并通过 gRPC 调用内部 `shop-service`。

## 当前接口

| 功能 | 方法 | 路径 | 说明 |
| --- | --- | --- | --- |
| 健康检查 | GET | `/health` | 返回网关进程状态 |
| 商铺注册 | POST | `/api/shop/register` | 转发到 `shop-service.RegisterShop` |
| 商铺登录 | POST | `/api/shop/login` | 转发到 `shop-service.LoginShop`，成功后由网关签发 JWT |

## 本地启动

先启动 `shop-service`：

```bash
SHOP_SERVICE_CONFIG=services/shop-service/configs/config.local.yaml go run ./services/shop-service/cmd/server
```

再启动网关：

```bash
API_GATEWAY_CONFIG=services/api-gateway/configs/config.local.yaml go run ./services/api-gateway/cmd/server
```

默认监听：

- HTTP：`:8080`
- shop-service gRPC：`127.0.0.1:9001`

## 配置

配置文件：

```text
services/api-gateway/configs/config.local.yaml
```

常用环境变量：

| 环境变量 | 说明 |
| --- | --- |
| `API_GATEWAY_HTTP_ADDR` | 网关 HTTP 监听地址 |
| `API_GATEWAY_SHOP_SERVICE_ADDR` | `shop-service` gRPC 地址 |
| `API_GATEWAY_JWT_SECRET` | JWT HS256 签名密钥 |
| `API_GATEWAY_JWT_ISSUER` | JWT 签发方 |
| `API_GATEWAY_JWT_ACCESS_TOKEN_TTL_SECONDS` | access token 有效期 |
| `API_GATEWAY_RPC_TIMEOUT_SECONDS` | 调用底层 gRPC 服务超时时间 |

本地配置中的 JWT secret 只是开发示例值，生产环境必须通过环境变量或部署系统注入。
