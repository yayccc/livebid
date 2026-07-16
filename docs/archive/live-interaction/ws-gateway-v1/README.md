# ws-gateway 直管弹幕 V1 归档

> Status: superseded

这些文件记录 2026-07-16 前后的实现与设计快照。代码迁移完成前，它们可帮助理解旧实现，但其中以下结论已经废弃：

- ws-gateway 持有弹幕准入、幂等、频控和 recent 的业务真相。
- Redis Pub/Sub 与 RocketMQ 可配置双发 AI 输入。
- AI 通过 RocketMQ/Redis 直接回流网关广播。
- 只有 `living` 房间才能进入或互动。

现行方案见 `docs/features/live-interaction/README.md`，当前外部兼容协议见 `docs/services/ws-gateway/互动协议-v1.md`。
