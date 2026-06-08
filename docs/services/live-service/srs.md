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

【新增说明：2026-06-06，本地 WebRTC 播放排查】

WebRTC 播放不只依赖 `rtc/v1/play` 信令成功，还要求浏览器能访问 SRS 在 SDP 中返回的 UDP ICE candidate。若播放页日志已经出现 `收到媒体轨道` 和 `SRS session`，但连接状态随后变成 `failed`，同时 SRS 日志出现 `DTLS_HANG` 或 `session destroy by timeout`，通常表示 SRS 返回的 candidate 对浏览器不可达。

本地 Docker/WSL 联调时，OBS 推流成功、SRS 回调成功、直播间状态 online，并不代表 WebRTC 播放链路已通。需要确认：

- SRS HTTP API 端口默认是 `1985`，播放测试页端口如 `18080/18081` 只是本地代理页端口。
- SRS RTC UDP 端口默认是 `8000/udp`，浏览器必须能访问 `candidate:8000/udp`。
- `SRS_RTC_CANDIDATE` 或 Compose 的 `SRS_PUBLIC_HOST` 应设置为浏览器可访问的宿主机/局域网/WSL IP，而不是只对容器或某个网络命名空间有效的地址。

如果 OBS 推流地址使用的是类似 `rtmp://172.21.x.x:1935/live`，则本地 WebRTC 测试通常也应让 SRS 返回同一个可访问 IP：

```bash
SRS_PUBLIC_HOST=172.21.x.x docker compose -f deployments/docker-compose.yml up -d --force-recreate srs live-service
scripts/test-srs-webrtc.sh 'webrtc://172.21.x.x/live/{stream_name}'
```

如果使用独立脚本启动 SRS，则设置：

```bash
SRS_RTC_CANDIDATE=172.21.x.x scripts/start-srs.sh
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
