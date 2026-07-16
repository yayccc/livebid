# interaction-service 文档索引

> Status: proposed
> Authority: 本文只负责本服务文档导航。
> Updated-at: 2026-07-17 +08:00

`interaction-service` 是直播间公开消息的唯一业务写入口。当前服务尚未创建，运行中的弹幕仍由 `ws-gateway` 直管；迁移状态只在 [实施路线](../../features/live-interaction/实施路线.md) 维护。

## 文档

| 文档 | 职责 |
| --- | --- |
| [需求](./需求.md) | 本服务要解决什么、业务规则和验收标准 |
| [设计](./设计.md) | 本服务如何执行准入、提交、广播和迁移 |
| [共享消息与接口](../../features/live-interaction/contracts/消息与接口-v1.md) | 跨服务消息、事件、RPC 和错误码的唯一权威 |
| [共享实时分发](../../features/live-interaction/contracts/实时分发-v1.md) | Redis、Lua、频道与 AI 分片的唯一权威 |

全链路边界、运行安全和分期分别见 [业务边界](../../features/live-interaction/业务边界.md)、[运行与安全](../../features/live-interaction/运行与安全.md) 和 [实施路线](../../features/live-interaction/实施路线.md)。

## 阅读约束

- 本目录不复制共享字段表、Redis Key 表和 P0/P1/P2 清单。
- 正式 Proto 创建后，字段号和 RPC 形状以 Proto 为准。
- 旧 WS 直管方案已移至 `docs/archive/live-interaction/ws-gateway-v1/`；2026-07-16 未实施草案位于 archive 的 `drafts-2026-07-16/`。
