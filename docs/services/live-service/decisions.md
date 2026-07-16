# Live Service 决策记录

## D001: 第一版接入 SRS

第一版使用 SRS 作为独立媒体服务器，`live-service` 不自研音视频推流、播放和协议转换能力。

## D002: 主播端使用 RTMP 推流

第一版主播端使用 OBS 或推流工具通过 RTMP 推流到 SRS。浏览器 WebRTC 推流暂缓。

## D003: 用户端主播放使用 WebRTC

H5 用户端优先使用 WebRTC 播放，降低直播竞拍场景下的音视频延迟。

## D004: HTTP-FLV 和 HLS 作为兜底

HTTP-FLV 和 HLS 可预留播放地址。第一版主链路不依赖 HLS，因为 HLS 延迟较高。

## D005: SRS 不承载业务消息

SRS 只处理媒体流。竞拍出价、倒计时、成交、订单、系统消息等业务消息由 `ws-gateway` 推送。

## D006: live-service 不管理 WebSocket 连接

`live-service` 只管理直播间业务事实和媒体流配置。用户连接、直播间订阅和房间广播由 `ws-gateway` 管理。

## D007: 开播状态先保持简单

第一版只保留 `not_live` 和 `living` 两个主状态。商家点击开始直播后，直播间状态变为 `living`。SRS 推流回调用 `media_stream_status` 表达媒体流是否在线。

## D008: 直播间可以提前创建竞拍

`not_live` 状态可以提前创建竞拍。只有 `living` 状态可以开始竞拍。

## D009: 直播间可访问性与开播状态分离

【决策日期：2026-07-17】

`LiveRoom` 是长期存在的用户入口。普通用户能否发现和进入房间由 `visibility` 决定，不再由 `status=living` 间接表达。

- `published + not_live` 的房间可以进入，并根据 `chat_enabled` 决定是否允许公开互动。
- 公开“正在直播”推荐流仍显式筛选 `published + living`。
- `draft/disabled` 房间不对普通用户开放。

## D010: 业务直播场次不由媒体回调创建

【决策日期：2026-07-17】

商家调用 `StartLive` 时生成新的 `live_session_id`，调用 `EndLive` 时结束该场次。SRS `on_publish/on_unpublish` 只维护 `media_stream_status`。

同一次业务直播中的 OBS 断线重连继续使用原 `live_session_id`，避免 AI 上下文、互动状态和竞拍事实被错误切分。

## D011: 房间互动与场次上下文使用不同隔离维度

【决策日期：2026-07-17】

- 房间连接、在线广播、`room_seq` 和有界短历史按持久 `room_id` 维护。
- `live_session_id` 作为互动消息的可选上下文，只在业务直播进行中存在。
- AI 上下文、冷却、已回答意图和迟到回写校验按 `live_session_id` 隔离。

## D012: 媒体连接使用 client_id 防止迟到回调覆盖

【决策日期：2026-07-17】

每次成功 `on_publish` 都替换活动 `client_id` 并递增媒体连接 generation。`on_unpublish` 只有匹配当前活动 `client_id` 时才能置 `media_stream_status=offline`；旧连接的迟到回调不能覆盖重连后的新媒体状态。

该 generation 只属于媒体连接，不创建新的 `live_session_id`，也不结束业务直播场次。
