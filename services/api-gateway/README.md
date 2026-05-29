# api-gateway

`api-gateway` 是对外 HTTP 入口。当前阶段承接商铺注册、商铺登录和商品管理 HTTP API；商铺接口通过 gRPC 调用内部 `shop-service`，商品管理接口通过 gRPC 调用内部 `goods-service`。

## 当前接口

| 功能 | 方法 | 路径 | 说明 |
| --- | --- | --- | --- |
| 健康检查 | GET | `/health` | 返回网关进程状态 |
| 商铺注册 | POST | `/api/shop/register` | 转发到 `shop-service.RegisterShop` |
| 商铺登录 | POST | `/api/shop/login` | 转发到 `shop-service.LoginShop`，成功后由网关签发 JWT |
| 商品封面上传 | POST | `/api/goods/cover/upload` | 接收 multipart `file`，当前仅返回假 URL，不进行图片存储 |
| 创建商品 | POST | `/api/goods` | 转发到 `goods-service.CreateGoods` |
| 编辑商品 | PUT | `/api/goods/:id` | 转发到 `goods-service.UpdateGoods` |
| 删除商品 | DELETE | `/api/goods/:id` | 转发到 `goods-service.DeleteGoods` |
| 商品详情 | GET | `/api/goods/:id` | 转发到 `goods-service.GetGoods` |
| 商品列表 | GET | `/api/goods` | 转发到 `goods-service.ListGoods` |
| 当前商铺商品列表 | GET | `/api/goods/shop/list` | 转发到 `goods-service.ListShopGoods` |
| 批量查询商品 | POST | `/api/goods/batch` | 转发到 `goods-service.BatchGetGoods` |
| 商品上架 | PUT | `/api/goods/:id/on-sale` | 转发到 `goods-service.PutGoodsOnSale` |
| 商品下架 | PUT | `/api/goods/:id/off-sale` | 转发到 `goods-service.PutGoodsOffSale` |

## 商品封面上传说明

当前 `/api/goods/cover/upload` 是占位实现：只校验请求中存在 `file` 且大小不超过 10MiB，然后返回形如 `https://static.livebid.local/goods/cover/{shop_id}/{timestamp}.jpg` 的假 URL。

`shop_id` 由商家端 JWT 鉴权中间件注入，请求必须携带 `Authorization: Bearer <token>`。

## 商品接口说明

商品创建、编辑、删除、当前商铺列表、上下架请求中的 `shop_id` 均来自商家端 JWT 的 subject。网关中间件会校验 token，并把当前认证主体写入请求 identity。

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
- goods-service gRPC：`127.0.0.1:9002`

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
| `API_GATEWAY_GOODS_SERVICE_ADDR` | `goods-service` gRPC 地址 |
| `API_GATEWAY_JWT_SECRET` | JWT HS256 签名密钥 |
| `API_GATEWAY_JWT_SHOP_ISSUER` | 商家端 JWT 签发方 |
| `API_GATEWAY_JWT_USER_ISSUER` | 用户端 JWT 签发方 |
| `API_GATEWAY_JWT_ACCESS_TOKEN_TTL_SECONDS` | access token 有效期 |
| `API_GATEWAY_RPC_TIMEOUT_SECONDS` | 调用底层 gRPC 服务超时时间 |

本地配置中的 JWT secret 只是开发示例值，生产环境必须通过环境变量或部署系统注入。
