# ai-service 文档索引

> Status: proposed
> Authority: 本文只负责 ai-service 文档导航。
> Updated-at: 2026-07-17 +08:00

`ai-service` 面向直播中的公开互动，观察真人消息并选择性回答。当前仓库尚无该服务代码；目标不是逐条聊天机器人，而是在低延迟和严格安全边界内维护直播间问答体验。

## 文档

| 文档 | 职责 |
| --- | --- |
| [需求](./需求.md) | AI 产品行为、约束和验收 |
| [设计](./设计.md) | 输入、窗口、候选、调度和回写流程 |
| [知识与模型](./知识与模型.md) | Tool、动态事实、缓存、模型和 RAG 边界 |
| [安全与评测](./安全与评测.md) | 模型安全、审计、离线与在线评测 |
| [共享消息与接口](../../features/live-interaction/contracts/消息与接口-v1.md) | AI 输入和回写契约的唯一权威 |

阶段、跨服务开关和端到端 SLO 分别以 [实施路线](../../features/live-interaction/实施路线.md) 与 [运行与安全](../../features/live-interaction/运行与安全.md) 为准。本目录不复制 64 分片算法、RPC 字段表或 P0/P1/P2 清单。

旧 2026-07-16 大文档已移入 `docs/archive/live-interaction/drafts-2026-07-16/`，不能指导新实现。
