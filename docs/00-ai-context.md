# AI 协作上下文

这份文档记录当前阶段的稳定结论，供 AI 和新协作者快速建立上下文。详细设计仍以各服务文档为准。

## 当前阶段

- 项目目标是支持直播间内高价值非标商品的实时竞拍。
- 第一版优先打通业务闭环，不一次性补齐所有目标服务。
- 普通业务服务默认只暴露内部 gRPC，对外 HTTP 由 `api-gateway` 承接。
- WebSocket 长连接和直播间订阅由 `ws-gateway` 承接。

## 已确定方向

- 第一版直播接入 SRS，不自研音视频服务。
- 主播端第一版使用 RTMP 推流。
- 用户端第一版主播放使用 WebRTC。
- HTTP-FLV 和 HLS 只作为兜底播放能力预留。
- SRS 只处理媒体流，不承载竞拍、订单、系统消息。
- 竞拍、订单、系统消息通过业务 WebSocket 推送。
- `live-service` 管直播间业务状态和媒体流配置。
- `auction-service` 管竞拍、出价、倒计时、落锤和竞拍事件。
- `order-service` 后续消费成交事件并创建订单。

## 鉴权约束

- JWT 只在 `api-gateway` 校验。
- 网关通过 gRPC metadata 透传 `livebid-auth-subject-type` 和 `livebid-auth-subject-id`。
- 底层服务不信任请求参数中的 `user_id`、`shop_id`。
- 当前 JWT 不承载 roles、scopes、shop_id 等业务权限字段。

## 文档阅读建议

- 项目总览先看 [01-overview.md](01-overview.md)。
- 架构边界先看 [02-architecture.md](02-architecture.md)。
- 服务结构先看 [03-service-structure.md](03-service-structure.md)。
- 直播服务先看 [services/live/README.md](services/live/README.md) 和 [services/live/decisions.md](services/live/decisions.md)。
