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
