# Live Service 接口与事件

## 查询需求

商家侧：

- 查询直播间详情。
- 查询当前进行中的直播间。
- 查询推流地址、WebRTC 播放地址和推流状态。
- 使用 WebRTC 播放地址在商家后台预览直播画面。

用户侧：

- 查询可进入的直播间列表。
- 查询直播间详情。
- 查询直播间当前状态。
- 查询直播间 WebRTC 播放地址。
- 查询直播间当前竞拍信息，后续由 `api-gateway` 聚合 `live-service` 和 `auction-service`。

内部服务侧：

- 根据 `live_room_id` 查询直播间详情。
- 校验直播间是否存在。
- 校验直播间是否归属某个 `shop_id`。
- 校验直播间是否允许创建竞拍。
- 校验直播间是否允许开始竞拍。

## gRPC 方法

建议方法：

```text
CreateLiveRoom
UpdateLiveRoom
StartLive
EndLive
GetLiveRoom
ListLiveRooms
GetShopCurrentLiveRoom
ValidateLiveRoomForAuction
GetLiveStreamInfo
HandleSRSPublishCallback
HandleSRSUnpublishCallback
```

第一版 `ListLiveRooms` 用于用户侧直播广场：

- 请求只传 `page` 和 `page_size`。
- 服务端固定查询 `status = living` 的直播间。
- 按 `live_room.status` 索引分页查询。
- 按 `live_room_id` 查询详情走 `GetLiveRoom`，不混入列表接口。

对外 HTTP API 由 `api-gateway` 暴露，普通业务服务不直接提供外部 HTTP 业务接口。

## api-gateway HTTP 接口

第一版 HTTP 路由：

```text
POST /api/live/rooms
GET  /api/live/rooms
GET  /api/live/rooms/{id}
POST /api/live/rooms/{id}/start
POST /api/live/rooms/{id}/end
GET  /api/live/rooms/{id}/stream

POST /api/srs/callbacks/publish
POST /api/srs/callbacks/unpublish
```

【新增说明：2026-06-08，本段根据商铺管理后台直播间管理和竞拍创建 RoomID 选择需求补充】

商家后台新增商家视角直播间列表接口：

```text
GET /api/merchant/live/rooms
```

查询参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 10，最大 100 |
| status | string | 否 | 直播间状态，支持 `not_live`、`living` |

响应数据：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "total": 1,
    "page": 1,
    "page_size": 10,
    "list": [
      {
        "id": 700000000001,
        "shop_id": 10001,
        "title": "翡翠专场直播",
        "cover": "https://example.com/live-cover.jpg",
        "description": "晚场专拍",
        "status": "not_live",
        "media_stream_status": "offline",
        "actual_start_time": "",
        "actual_end_time": "",
        "created_at": "2026-06-08T12:00:00Z",
        "updated_at": "2026-06-08T12:00:00Z"
      }
    ]
  }
}
```

约定：

- 商家身份从 JWT 注入，不允许客户端传 `shop_id`。
- 与公开 `GET /api/live/rooms` 不同，本接口返回当前商铺未删除的直播间，可包含 `not_live` 和 `living`，用于直播间管理和创建竞拍时选择 `room_id`。
- 本接口不返回 `stream_name`、`rtmp_push_url`、`webrtc_play_url`；商家查看推流信息继续调用 `GET /api/live/rooms/{id}/stream`。
- `live-service.ListLiveRooms` 内部请求保留用户侧默认行为：未传 `shop_id/status` 时固定查询 `living`；传入 `shop_id` 时按商家维度查询，`status` 可选。

约定：

- 商家创建、开播、关播需要 `Authorization: Bearer <token>`。
- `api-gateway` 从 JWT subject 解析 `shop_id`，不信任请求体里的 `shop_id`。
- 用户侧直播间列表 `GET /api/live/rooms` 只查 `living` 直播间。
- `CreateLiveRoom` 返回 `live_room`、`initial_stream_code`、`rtmp_push_url` 和 `webrtc_play_url`。其中 `initial_stream_code` 只在创建时返回，`webrtc_play_url` 是 WebRTC 播放地址，第一版可同时用于商家预览和用户播放。
- `live-service.GetLiveStreamInfo` 第一版不做商家认证和归属校验，只按直播间 ID 返回内部流信息。
- `api-gateway` 负责区分对外字段边界：商家侧 `GET /api/live/rooms/{id}/stream` 必须校验商家 JWT 和直播间归属后，才返回 `stream_name`、`rtmp_push_url`、`webrtc_play_url` 和 `media_stream_status`；用户侧播放/预览接口只能返回 `webrtc_play_url` 和 `media_stream_status` 等播放字段。
- 后续如需要更清晰的服务契约，再将内部流信息拆为商家推流接口和用户播放接口。
- SRS callback 不走 JWT，由 SRS 调用并携带 `stream` 和 `param`。
- `on_publish` 成功返回 HTTP 200 且响应体 `0`；认证失败返回非 0，SRS 拒绝推流。

## WebSocket 关系

`live-service` 不管理 WebSocket 连接。

`ws-gateway` 负责：

- 用户连接管理。
- 用户进入直播间后的订阅关系。
- 根据 `live_room_id` 广播竞拍、直播状态和系统消息。
- 用户断线重连后的订阅恢复。

## MQ 事件

建议事件：

```text
live.started
live.ended
live.stream.published
live.stream.unpublished
```

事件要求：

- 包含全局唯一事件 ID。
- 包含直播间 ID。
- 包含商铺 ID。
- 包含事件发生时间。
- 包含直播间状态。
- 支持消费者幂等。

## 跨服务关系

创建竞拍时，`auction-service` 需要校验：

- 直播间存在。
- 直播间归属当前商铺。
- 直播间未被删除。

开始竞拍时，`auction-service` 需要校验：

- 直播间存在。
- 直播间状态是 `living`。

第一版规则：

- `not_live` 状态可以提前创建竞拍。
- 只有 `living` 状态可以开始竞拍。

## 2026-07-17 目标契约增量

本节描述持久直播间演进后的目标契约。当前服务行为仍按上文第一版规则运行，实施步骤见 [演进需求.md](演进需求.md)。

`LiveRoom` 增量字段：

| 字段 | 类型 | 语义 |
| --- | --- | --- |
| `visibility` | `LiveRoomVisibility` | `draft/published/disabled`，决定普通用户是否可以访问房间 |
| `chat_enabled` | `optional bool` | 是否允许公开互动，不依赖业务是否开播；缺失表示尚未迁移的旧数据 |
| `current_live_session_id` | `optional int64` | 当前业务直播场次；`not_live` 时不存在 |

`UpdateLiveRoom` 是 L0 必须冻结、当前 Proto 尚未补齐的目标 RPC。请求语义固定为：

| 字段 | 类型 | 语义 |
| --- | --- | --- |
| `request_id` | string | 商家、房间作用域的幂等键 |
| `id` | int64 | 直播间 ID |
| `shop_id` | int64 | 过渡期调用身份一致性校验；最终权限以认证上下文为准 |
| `title/cover/description` | optional string | 可更新的展示字段 |
| `visibility` | optional enum | 只能更新房间可见性 |
| `chat_enabled` | optional bool | 开启或关闭公开互动 |
| `update_mask` | `google.protobuf.FieldMask` | 必填，明确本次更新字段 |

响应返回最新 `LiveRoom` 和幂等重放标识。`status`、`media_stream_status`、`current_live_session_id`、推流凭证和归属字段禁止通过该 RPC 修改；它们分别只能由 `StartLive/EndLive`、SRS 回调或内部安全流程维护。相同 `request_id` 重试返回首次结果，字段掩码为空、包含只读字段或调用者无房间归属时拒绝。

`ListLiveRoomsRequest` 增加可选 `visibility` 过滤。目标调用约定：

1. 用户直播推荐流显式传 `visibility=published`、`status=living`。
2. 店铺公开房间目录传 `visibility=published`，不强制传直播状态。
3. 商家管理列表按商家身份查询，并可组合可见性和直播状态筛选。
4. 按 ID 进入房间时校验 `visibility=published`，不再要求 `status=living`。

`StartLive` 的目标语义：

- 从 `not_live` 进入 `living` 时生成新的 `current_live_session_id`。
- 幂等重试返回相同场次 ID。
- 不等待 SRS 推流上线。

`EndLive` 的目标语义：

- 结束当前直播场次并清空 `current_live_session_id`。
- 房间保持 `published`，可以继续访问和互动。

SRS callback 继续只维护 `media_stream_status` 和活动媒体连接身份，不得创建或切换直播场次。`on_unpublish` 只有在 `client_id` 与当前活动连接匹配时才能置离线。
