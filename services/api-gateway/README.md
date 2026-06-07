# api-gateway

`api-gateway` 是对外 HTTP 入口。当前阶段承接商铺、用户、商品和直播 HTTP API；商铺接口通过 gRPC 调用内部 `shop-service`，用户接口调用 `user-service`，商品接口调用 `goods-service`，直播接口调用 `live-service`。

## 当前接口

| 功能 | 方法 | 路径 | 说明 |
| --- | --- | --- | --- |
| 健康检查 | GET | `/health` | 返回网关进程状态 |
| 商铺注册 | POST | `/api/shop/register` | 转发到 `shop-service.RegisterShop` |
| 商铺登录 | POST | `/api/shop/login` | 转发到 `shop-service.LoginShop`，成功后由网关签发 JWT |
| 商铺信息 | GET | `/api/shop/:id` | 转发到 `shop-service.GetShop` |
| 当前商铺信息 | GET | `/api/shop/me` | 需要商家 JWT，通过 gRPC metadata 透传身份到 `shop-service.GetShop` |
| 更新商铺信息 | PUT | `/api/shop/:id` | 需要商家 JWT，通过 gRPC metadata 透传身份到 `shop-service.UpdateShop` 并校验只能更新当前商铺 |
| 用户注册 | POST | `/api/users/register` | 转发到 `user-service.RegisterUser` |
| 用户登录 | POST | `/api/users/login` | 转发到 `user-service.LoginUser`，透传 `access_token` |
| 用户信息 | GET | `/api/users/:id` | 转发到 `user-service.GetUser` |
| 更新当前用户 | PUT | `/api/users/:id` | 需要用户 JWT，通过 gRPC metadata 透传身份到 `user-service.UpdateCurrentUser` |
| 上传头像 | POST | `/api/users/avatar/upload` | 需要用户 JWT，复用文件上传能力，返回头像 URL |
| 新增收货地址 | POST | `/api/users/address` | 需要用户 JWT，转发到 `user-service.CreateAddress` |
| 收货地址列表 | GET | `/api/users/address/list` | 需要用户 JWT，转发到 `user-service.ListAddresses` |
| 收货地址详情 | GET | `/api/users/address/:id` | 需要用户 JWT，转发到 `user-service.GetAddress` |
| 修改收货地址 | PUT | `/api/users/address/:id` | 需要用户 JWT，转发到 `user-service.UpdateAddress` |
| 删除收货地址 | DELETE | `/api/users/address/:id` | 需要用户 JWT，转发到 `user-service.DeleteAddress` |
| 设置默认地址 | PUT | `/api/users/address/:id/default` | 需要用户 JWT，转发到 `user-service.SetDefaultAddress` |
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
| 竞拍详情 | GET | `/api/auctions/:id` | 转发到 `auction-service.GetAuction` |
| 竞拍运行态 | GET | `/api/auctions/:id/runtime` | 转发到 `auction-service.GetAuctionRuntime` |
| 商品竞拍信息 | GET | `/api/auctions/goods/:goods_id` | 转发到 `auction-service.GetAuctionByGoods` |
| 竞拍出价记录 | GET | `/api/auctions/:id/bids` | 转发到 `auction-service.ListBidRecords` |
| 商家竞拍列表 | GET | `/api/merchant/auctions` | 需要商家 JWT，返回商品标题和封面等聚合字段 |
| 商家工作台统计 | GET | `/api/merchant/dashboard/summary` | 需要商家 JWT，聚合商品统计和竞拍统计 |
| 创建竞拍 | POST | `/api/auctions` | 需要商家 JWT，转发到 `auction-service.CreateAuction` |
| 商铺竞拍列表 | GET | `/api/auctions/shop` | 需要商家 JWT，兼容旧路径并补充商品展示字段 |
| 修改竞拍 | PUT | `/api/auctions/:id` | 需要商家 JWT，转发到 `auction-service.UpdateAuction` |
| 开始竞拍 | POST | `/api/auctions/:id/start` | 需要商家 JWT，转发到 `auction-service.StartAuction` |
| 结束竞拍 | POST | `/api/auctions/:id/finish` | 需要商家 JWT，转发到 `auction-service.FinishAuction` |
| 取消竞拍 | POST | `/api/auctions/:id/cancel` | 需要商家 JWT，转发到 `auction-service.CancelAuction` |
| 删除竞拍 | DELETE | `/api/auctions/:id` | 需要商家 JWT，转发到 `auction-service.DeleteAuction` |
| 直播间列表 | GET | `/api/live/rooms` | 转发到 `live-service.ListLiveRooms` |
| 直播间详情 | GET | `/api/live/rooms/:id` | 转发到 `live-service.GetLiveRoom` |
| 创建直播间 | POST | `/api/live/rooms` | 转发到 `live-service.CreateLiveRoom` |
| 开始直播 | POST | `/api/live/rooms/:id/start` | 转发到 `live-service.StartLive` |
| 结束直播 | POST | `/api/live/rooms/:id/end` | 转发到 `live-service.EndLive` |
| 商家推流信息 | GET | `/api/live/rooms/:id/stream` | 需要商家 JWT，网关校验直播间归属后转发到 `live-service.GetLiveStreamInfo` 并返回完整推流字段 |
| 用户直播间流 | GET | `/api/user/live/feed` | mobile-user 首页聚合接口，返回直播间、公开商铺、预览播放和当前竞拍摘要 |
| 用户预览播放信息 | GET | `/api/user/live/rooms/:id/preview` | 转发到 `live-service.GetLiveStreamInfo`，只返回用户播放字段 |
| 用户进入直播间详情 | GET | `/api/user/live/rooms/:id/entry` | 可选用户 JWT，聚合直播间、公开商铺、播放、当前竞拍、商品、运行态和 WebSocket 配置 |
| 用户竞拍快照 | GET | `/api/user/live/rooms/:id/auction-snapshot` | 可选用户 JWT，返回当前运行中竞拍快照；无竞拍时返回 null |
| SRS 推流回调 | POST | `/api/srs/callbacks/publish` | 转发到 `live-service.HandleSRSPublishCallback` |
| SRS 断流回调 | POST | `/api/srs/callbacks/unpublish` | 转发到 `live-service.HandleSRSUnpublishCallback` |

## 文件上传说明

`/api/files/upload` 不需要登录认证；接口会校验请求中存在 multipart `file` 且大小不超过 10MiB，然后通过 AWS SDK for Go v2 的 S3 client 将对象写入 RustFS。

返回的 `url` 可作为商品封面、直播封面等业务字段保存，`object_key` 是 RustFS 中的对象 key。

## 商品接口说明

商品创建、编辑、删除、当前商铺列表、上下架请求中的 `shop_id` 均来自商家端 JWT 的 subject。网关中间件会校验 token，并把当前认证主体写入请求上下文，再通过 gRPC metadata 透传给下游服务。

## 用户接口说明

用户资料修改、头像上传和收货地址接口中的 `user_id` 均来自用户端 JWT 的 subject。用户登录 token 当前由 `user-service` 签发，网关只做 HTTP 到 gRPC 转发与响应格式统一。

## 本地启动

本地端到端测试至少需要 MySQL 和四个内部 gRPC 服务。默认 MySQL DSN 为：

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
USER_SERVICE_CONFIG=services/user-service/configs/config.local.yaml \
  go run ./services/user-service/cmd/server
```

```bash
LIVE_SERVICE_CONFIG=services/live-service/configs/config.local.yaml \
  go run ./services/live-service/cmd/server
```

```bash
AUCTION_SERVICE_CONFIG=services/auction-service/configs/config.local.yaml \
  go run ./services/auction-service/cmd/server
```

```bash
API_GATEWAY_CONFIG=services/api-gateway/configs/config.local.yaml \
  go run ./services/api-gateway/cmd/server
```

默认监听：

- api-gateway HTTP：`:58080`
- shop-service gRPC：`127.0.0.1:9001`
- user-service gRPC：`127.0.0.1:9004`
- goods-service gRPC：`127.0.0.1:9002`
- auction-service gRPC：`127.0.0.1:9003`
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
| `API_GATEWAY_SHOP_SERVICE_TARGET` | `shop-service` gRPC target，例如 `127.0.0.1:9001` 或 `nacosx:///shop-service` |
| `API_GATEWAY_USER_SERVICE_TARGET` | `user-service` gRPC target，例如 `127.0.0.1:9004` 或 `nacosx:///user-service` |
| `API_GATEWAY_GOODS_SERVICE_TARGET` | `goods-service` gRPC target，例如 `127.0.0.1:9002` 或 `nacosx:///goods-service` |
| `API_GATEWAY_LIVE_SERVICE_TARGET` | `live-service` gRPC target，例如 `127.0.0.1:9007` 或 `nacosx:///live-service` |
| `API_GATEWAY_USER_LIVE_WS_URL` | 用户端进入直播间后返回给前端的 WebSocket 地址，默认 `/ws/live` |
| `API_GATEWAY_USER_LIVE_HEARTBEAT_INTERVAL_SECONDS` | 用户端 WebSocket 心跳间隔秒数，默认 15 |
| `API_GATEWAY_SHOP_SERVICE_ADDR` | 兼容旧配置，未设置 target 时作为 target 兜底 |
| `API_GATEWAY_USER_SERVICE_ADDR` | 兼容旧配置，未设置 target 时作为 target 兜底 |
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
