# Live Service 接口与事件

## 查询需求

商家侧：

- 查询直播间详情。
- 查询当前进行中的直播间。
- 查询推流地址和推流状态。

用户侧：

- 查询可进入的直播间列表。
- 查询直播间详情。
- 查询直播间当前状态。
- 查询直播间播放地址。
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
