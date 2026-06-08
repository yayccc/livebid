# Live Service

`live-service` 是直播间业务容器服务，负责直播间创建、开播、关播、查询、商家归属校验，以及 SRS 推拉流配置管理。

第一版直播方案：

```text
主播端 OBS / 推流工具
 -> RTMP 推流
 -> SRS
 -> WebRTC 播放
 -> H5 用户端
```

`live-service` 不直接处理音视频数据；SRS 处理媒体流；`ws-gateway` 处理竞拍、订单、系统消息等业务 WebSocket。

## 文档

- [decisions.md](decisions.md)：已确定的设计决策。
- [lifecycle.md](lifecycle.md)：直播间状态机和业务流程。
- [srs.md](srs.md)：SRS 接入、推拉流地址和回调。
- [api.md](api.md)：查询需求、gRPC 方法、MQ 事件和跨服务关系。
- [data.md](data.md)：数据模型和字段说明。

## 第一版范围

优先实现：

- 创建、修改、开始和结束直播间。
- 查询直播间详情和直播中的直播间列表。
- 生成 RTMP 推流地址和推流码。
- 返回 WebRTC 播放地址，第一版可同时用于商家预览和用户播放。
- 接入 SRS RTMP 推流和 WebRTC 播放。
- 接收 SRS 推流开始和结束回调。
- 校验直播间是否可用于竞拍。

暂缓：

- 自研音视频服务。
- 浏览器 WebRTC 推流。
- CDN 调度。
- 直播回放、连麦、弹幕、礼物。
- 精确在线人数。
- 直播间商品橱窗和当前讲解商品。
