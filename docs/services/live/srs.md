# Live Service SRS 接入

## 媒体链路

第一版链路：

```text
主播端 OBS / 推流工具
 -> RTMP 推流
 -> SRS
 -> WebRTC 播放
 -> H5 用户端
```

兜底播放：

```text
SRS -> HTTP-FLV
SRS -> HLS
```

## 协议选择

| 场景 | 协议 | 说明 |
| --- | --- | --- |
| 主播推流 | RTMP | OBS 支持成熟，第一版接入成本低 |
| H5 主播放 | WebRTC | 延迟低，适合实时竞拍 |
| H5 兜底播放 | HTTP-FLV | 延迟较低，需要前端播放器支持 |
| 兼容兜底/回放 | HLS | 兼容性好，但延迟较高，不作为竞拍主播放 |

## 推拉流地址

直播间创建后，`live-service` 生成：

- `stream_name`，媒体流名称。
- `stream_code`，推流码。
- RTMP 推流地址。
- WebRTC 播放地址，第一版可同时用于商家后台预览和用户端播放。
- HTTP-FLV 播放地址，可选。
- HLS 播放地址，可选。

推流地址和推流码只返回给商家/主播端，不返回给普通用户。WebRTC 播放地址是播放侧地址，第一版商家预览和用户播放可以共用；后续如引入 CDN、边缘节点、播放鉴权或多码率，再按场景拆分不同播放地址。

地址结构示例：

```text
rtmp://srs.example.com/live/{stream_name}?token={stream_code}
webrtc://srs.example.com/live/{stream_name}
https://srs.example.com/live/{stream_name}.flv
https://srs.example.com/live/{stream_name}.m3u8
```

具体域名、协议和 token 形式以部署环境为准。

浏览器不能直接播放 `webrtc://` URL，需要通过 SRS WebRTC 播放信令接口完成 SDP 交换。当前仓库提供 `scripts/test-srs-webrtc.sh` 作为本地播放验证工具。

本地可以使用脚本启动 SRS：

```bash
scripts/start-srs.sh
```

常用可覆盖参数：

```bash
SRS_CALLBACK_BASE_URL=http://host.docker.internal:58080 \
SRS_RTMP_PORT=1935 \
SRS_HTTP_API_PORT=1985 \
SRS_HTTP_SERVER_PORT=8088 \
SRS_RTC_PORT=8000 \
SRS_RTC_CANDIDATE=127.0.0.1 \
scripts/start-srs.sh
```

## SRS 回调

第一版建议接入：

```text
on_publish     推流开始
on_unpublish   推流结束
on_play        用户开始播放，可选
on_stop        用户停止播放，可选
```

`on_publish` 处理：

- 校验 `stream_name` 是否存在。
- 校验 `stream_code` 是否有效。
- 校验直播间状态是否允许推流。
- 记录媒体流在线状态。
- 必要时广播直播流已就绪。

`on_unpublish` 处理：

- 记录媒体流离线状态。
- 必要时广播直播流中断。
- 不直接结束直播间。

第一版不依赖 `on_play` 和 `on_stop` 做精确在线人数。在线人数后续由 `ws-gateway` 或专门统计链路处理。
