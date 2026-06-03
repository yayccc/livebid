# api-gateway

`api-gateway` 是对外 HTTP 入口。当前阶段承接商铺、商品和直播 HTTP API；商铺接口通过 gRPC 调用内部 `shop-service`，商品接口调用 `goods-service`，直播接口调用 `live-service`。

## 当前接口

| 功能 | 方法 | 路径 | 说明 |
| --- | --- | --- | --- |
| 健康检查 | GET | `/health` | 返回网关进程状态 |
| 商铺注册 | POST | `/api/shop/register` | 转发到 `shop-service.RegisterShop` |
| 商铺登录 | POST | `/api/shop/login` | 转发到 `shop-service.LoginShop`，成功后由网关签发 JWT |
| 商铺信息 | GET | `/api/shop/:id` | 转发到 `shop-service.GetShop` |
| 当前商铺信息 | GET | `/api/shop/me` | 需要商家 JWT，通过 gRPC metadata 透传身份到 `shop-service.GetShop` |
| 更新商铺信息 | PUT | `/api/shop/:id` | 需要商家 JWT，通过 gRPC metadata 透传身份到 `shop-service.UpdateShop` 并校验只能更新当前商铺 |
| 文件上传 | POST | `/api/files/upload` | 接收 multipart `file`，通过 S3 协议上传到 RustFS 对象存储 |
| 创建商品 | POST | `/api/goods` | 转发到 `goods-service.CreateGoods` |
| 编辑商品 | PUT | `/api/goods/:id` | 转发到 `goods-service.UpdateGoods` |
| 删除商品 | DELETE | `/api/goods/:id` | 转发到 `goods-service.DeleteGoods` |
| 商品详情 | GET | `/api/goods/:id` | 转发到 `goods-service.GetGoods` |
| 商品列表 | GET | `/api/goods` | 转发到 `goods-service.ListGoods` |
| 当前商铺商品列表 | GET | `/api/goods/shop/list` | 转发到 `goods-service.ListShopGoods` |
| 批量查询商品 | POST | `/api/goods/batch` | 转发到 `goods-service.BatchGetGoods` |
| 商品上架 | PUT | `/api/goods/:id/on-sale` | 转发到 `goods-service.PutGoodsOnSale` |
| 商品下架 | PUT | `/api/goods/:id/off-sale` | 转发到 `goods-service.PutGoodsOffSale` |
| 直播间列表 | GET | `/api/live/rooms` | 转发到 `live-service.ListLiveRooms` |
| 直播间详情 | GET | `/api/live/rooms/:id` | 转发到 `live-service.GetLiveRoom` |
| 创建直播间 | POST | `/api/live/rooms` | 转发到 `live-service.CreateLiveRoom` |
| 开始直播 | POST | `/api/live/rooms/:id/start` | 转发到 `live-service.StartLive` |
| 结束直播 | POST | `/api/live/rooms/:id/end` | 转发到 `live-service.EndLive` |
| 推流信息 | GET | `/api/live/rooms/:id/stream` | 转发到 `live-service.GetLiveStreamInfo` |
| SRS 推流回调 | POST | `/api/srs/callbacks/publish` | 转发到 `live-service.HandleSRSPublishCallback` |
| SRS 断流回调 | POST | `/api/srs/callbacks/unpublish` | 转发到 `live-service.HandleSRSUnpublishCallback` |

## 文件上传说明

`/api/files/upload` 不需要登录认证；接口会校验请求中存在 multipart `file` 且大小不超过 10MiB，然后通过 AWS SDK for Go v2 的 S3 client 将对象写入 RustFS。

返回的 `url` 可作为商品封面、直播封面等业务字段保存，`object_key` 是 RustFS 中的对象 key。

## 商品接口说明

商品创建、编辑、删除、当前商铺列表、上下架请求中的 `shop_id` 均来自商家端 JWT 的 subject。网关中间件会校验 token，并把当前认证主体写入请求上下文，再通过 gRPC metadata 透传给下游服务。

## 本地启动

本地端到端测试至少需要 MySQL 和三个内部 gRPC 服务。默认 MySQL DSN 为：

```text
root:123456@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local
```

请先确认 `mydb` 数据库存在。各服务本地配置默认开启 `autoMigrate`，启动后会自动建表或补字段。

```bash
mysql -uroot -p123456 -e 'CREATE DATABASE IF NOT EXISTS mydb DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;'
```

分别打开 4 个终端，从仓库根目录启动：

```bash
SHOP_SERVICE_CONFIG=services/shop-service/configs/config.local.yaml \
  go run ./services/shop-service/cmd/server
```

```bash
GOODS_SERVICE_CONFIG=services/goods-service/configs/config.local.yaml \
  go run ./services/goods-service/cmd/server
```

```bash
LIVE_SERVICE_CONFIG=services/live-service/configs/config.local.yaml \
  go run ./services/live-service/cmd/server
```

```bash
API_GATEWAY_CONFIG=services/api-gateway/configs/config.local.yaml \
  go run ./services/api-gateway/cmd/server
```

默认监听：

- api-gateway HTTP：`:58080`
- shop-service gRPC：`127.0.0.1:9001`
- goods-service gRPC：`127.0.0.1:9002`
- live-service gRPC：`127.0.0.1:9007`

## 端到端测试

健康检查：

```bash
curl http://127.0.0.1:58080/health
```

注册商铺：

```bash
curl -X POST http://127.0.0.1:58080/api/shop/register \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "demo_shop",
    "password": "123456",
    "shopName": "端到端测试店铺"
  }'
```

登录并保存 token：

```bash
TOKEN=$(curl -s -X POST http://127.0.0.1:58080/api/shop/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo_shop","password":"123456"}' \
  | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')

echo "$TOKEN"
```

创建商品，验证商家身份从 JWT 透传到 `goods-service`：

```bash
curl -X POST http://127.0.0.1:58080/api/goods \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "翡翠手镯",
    "cover_url": "https://example.com/cover.jpg",
    "description": "端到端测试商品"
  }'
```

查询当前商铺商品：

```bash
curl "http://127.0.0.1:58080/api/goods/shop/list?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN"
```

创建直播间，验证商家身份从 JWT 透传到 `live-service`：

```bash
curl -X POST http://127.0.0.1:58080/api/live/rooms \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "端到端测试直播间",
    "cover": "https://example.com/live-cover.jpg",
    "description": "测试直播间"
  }'
```

未携带 token 访问商家接口应返回 `401`：

```bash
curl -i -X POST http://127.0.0.1:58080/api/goods \
  -H 'Content-Type: application/json' \
  -d '{"title":"未授权商品"}'
```

## 配置

配置文件：

```text
services/api-gateway/configs/config.local.yaml
```

常用环境变量：

| 环境变量 | 说明 |
| --- | --- |
| `API_GATEWAY_HTTP_ADDR` | 网关 HTTP 监听地址 |
| `API_GATEWAY_SHOP_SERVICE_TARGET` | `shop-service` gRPC target，例如 `127.0.0.1:9001` 或 `nacos:///shop-service` |
| `API_GATEWAY_GOODS_SERVICE_TARGET` | `goods-service` gRPC target，例如 `127.0.0.1:9002` 或 `nacos:///goods-service` |
| `API_GATEWAY_LIVE_SERVICE_TARGET` | `live-service` gRPC target，例如 `127.0.0.1:9007` 或 `nacos:///live-service` |
| `API_GATEWAY_SHOP_SERVICE_ADDR` | 兼容旧配置，未设置 target 时作为 target 兜底 |
| `API_GATEWAY_GOODS_SERVICE_ADDR` | 兼容旧配置，未设置 target 时作为 target 兜底 |
| `API_GATEWAY_LIVE_SERVICE_ADDR` | 兼容旧配置，未设置 target 时作为 target 兜底 |
| `API_GATEWAY_STORAGE_ENDPOINT` | RustFS S3 endpoint，例如 `http://127.0.0.1:9000` |
| `API_GATEWAY_STORAGE_PUBLIC_BASE_URL` | 对象公开访问基准地址，例如 `http://127.0.0.1:9000/livebid` |
| `API_GATEWAY_STORAGE_BUCKET` | RustFS bucket 名称 |
| `API_GATEWAY_STORAGE_REGION` | S3 签名 region，RustFS 本地默认可使用 `us-east-1` |
| `API_GATEWAY_STORAGE_ACCESS_KEY_ID` | RustFS access key |
| `API_GATEWAY_STORAGE_SECRET_ACCESS_KEY` | RustFS secret key |
| `API_GATEWAY_STORAGE_PATH_STYLE` | 是否使用 path-style endpoint，本地 RustFS 默认 `true` |
| `API_GATEWAY_JWT_SECRET` | JWT HS256 签名密钥 |
| `API_GATEWAY_JWT_SHOP_ISSUER` | 商家端 JWT 签发方 |
| `API_GATEWAY_JWT_USER_ISSUER` | 用户端 JWT 签发方 |
| `API_GATEWAY_JWT_ACCESS_TOKEN_TTL_SECONDS` | access token 有效期 |
| `API_GATEWAY_RPC_TIMEOUT_SECONDS` | 调用底层 gRPC 服务超时时间 |
| `API_GATEWAY_NACOS_ENABLED` | 是否初始化 Nacos client |
| `API_GATEWAY_CONFIG_CENTER_ENABLED` | 是否从 Nacos 配置中心拉取配置 |

本地配置中的 JWT secret 只是开发示例值，生产环境必须通过环境变量或部署系统注入。
