# ws-gateway

LiveBid WebSocket 网关服务，负责直播竞拍实时连接、直播间订阅、在线人数统计、出价转发和竞拍事件广播。

弹幕一期能力在 `docs/services/ws-gateway/弹幕功能需求.md` 和 `docs/services/ws-gateway/弹幕功能设计.md` 中定义；消息协议、运行机制和 AI 对接分别拆到同目录的专题设计文档。

## 启动

```bash
go run ./services/ws-gateway/cmd/server
```

默认读取：

```text
services/ws-gateway/configs/config.local.yaml
```

也可以通过环境变量指定配置文件：

```bash
WS_GATEWAY_CONFIG=services/ws-gateway/configs/config.local.yaml go run ./services/ws-gateway/cmd/server
```

## WebSocket 入口

```text
GET /ws/live?room_id={room_id}&token={access_token}
```

- `room_id` 必须大于 0。
- `token` 为空时按游客连接处理，只允许观看和接收广播。
- `token` 非空时使用用户端 JWT 校验，只接受 `iss=livebid-user`。
- 出价时使用连接绑定的 `user_id`，并通过 gRPC metadata 透传给 `auction-service`。

## 核心配置

|配置项|默认值|
|---|---|
|`WS_GATEWAY_HTTP_ADDR`|`:58081`|
|`WS_GATEWAY_REDIS_ADDR`|`127.0.0.1:6379`|
|`WS_GATEWAY_AUCTION_SERVICE_TARGET`|`127.0.0.1:9003`|
|`WS_GATEWAY_LIVE_SERVICE_TARGET`|`127.0.0.1:9007`|
|`WS_GATEWAY_JWT_SECRET`|`local-dev-jwt-secret-change-me`|
|`WS_GATEWAY_JWT_USER_ISSUER`|`livebid-user`|
|`WS_GATEWAY_HEARTBEAT_INTERVAL_SECONDS`|`30`|
|`WS_GATEWAY_HEARTBEAT_TIMEOUT_SECONDS`|`90`|
|`WS_GATEWAY_ONLINE_BROADCAST_INTERVAL_SECONDS`|`3`|
|`WS_GATEWAY_MAX_MESSAGE_BYTES`|`16384`|
|`WS_GATEWAY_ROCKETMQ_ENABLED`|`false`|
|`WS_GATEWAY_ROCKETMQ_NAME_SERVER`|空|
|`WS_GATEWAY_DANMAKU_ENABLED`|`true`|
|`WS_GATEWAY_DANMAKU_MAX_CHARS`|`30`|
|`WS_GATEWAY_DANMAKU_RECENT_LIMIT`|`10`|
|`WS_GATEWAY_AI_INPUT_MODE`|`redis_pubsub`|
|`WS_GATEWAY_USER_SERVICE_TARGET`|`127.0.0.1:9004`|

本地配置默认关闭 RocketMQ consumer。部署时设置 `WS_GATEWAY_ROCKETMQ_ENABLED=true` 并配置 `WS_GATEWAY_ROCKETMQ_NAME_SERVER` 后，会以广播消费模式消费 `auction_event`。

## 消息类型

客户端请求：

- `ping`
- `place_bid`
- `send_danmaku`
- `room_leave`

服务端广播：

- `room_online_changed`
- `auction_started`
- `bid_accepted`
- `auction_finished`
- `auction_failed`
- `auction_cancelled`
- `danmaku_created`
- `ai_interaction_created`

对外 JSON 时间字段统一使用 Unix 毫秒。

服务端 `type=response` 是通用响应信封，响应中会带 `request_type` 用于精确分发：

```json
{
  "type": "response",
  "request_id": "req_bid_1",
  "request_type": "place_bid",
  "code": 0,
  "message": "success",
  "server_time": 1780000123000,
  "data": {}
}
```

WebSocket 连接成功时，`request_type=connect` 的响应 `data` 会返回重连恢复提示：

```json
{
  "reconnect_strategy": "http_snapshot",
  "resync_on_connect": true,
  "snapshot_url": "/api/user/live/rooms/500000000001/auction-snapshot",
  "auction_records_url": "/api/user/live/rooms/500000000001/auction-records"
}
```

客户端每次首次连接或断线重连成功后，应通过上述 HTTP 接口重新拉取当前竞拍快照和本场竞拍记录，用 HTTP 权威状态覆盖断线期间可能漏掉的 WebSocket 增量事件。

用户端出价推荐优先走 WebSocket `place_bid`；api-gateway 的 HTTP 出价接口仅作为 WebSocket 不可用、重连中或旧客户端兼容时的降级入口。两条入口都需要复用同一个 `request_id` 幂等语义。

在线状态写入 Redis 后实时维护；面向客户端的 `room_online_changed` 展示消息由 `ws-gateway` 按房间聚合，默认每 3 秒最多广播一次最新人数。

## 验证

```bash
go test ./services/ws-gateway/...
```
