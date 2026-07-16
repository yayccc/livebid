# 直播互动文档导航

> Status: proposed
> Authority: 本文只负责导航、文档状态和唯一信息所有者。
> Non-authority: 本文不定义消息字段、存储结构或 AI 算法。
> Updated-at: 2026-07-17 +08:00

本专题覆盖持久直播间中的公开互动、AI 自动公共问答及其迁移路线。总体原则是：**一次性固定可扩展的基础架构，分阶段增加业务能力**。

## 阅读入口

| 读者 | 最小阅读集 |
| --- | --- |
| 全链路评审 | [业务边界](./业务边界.md) → [实施路线](./实施路线.md) → 两份共享契约 |
| interaction-service 开发 | [消息与接口](./contracts/消息与接口-v1.md) → [实时分发](./contracts/实时分发-v1.md) → [interaction 设计](../../services/interaction-service/设计.md) |
| ai-service 开发 | [消息与接口](./contracts/消息与接口-v1.md) → [AI 需求](../../services/ai-service/需求.md) → [AI 设计](../../services/ai-service/设计.md) |
| ws-gateway 开发 | [互动协议](../../services/ws-gateway/互动协议-v1.md) → [实时分发](./contracts/实时分发-v1.md) |

## 唯一信息所有者

| 信息 | 唯一权威文件 |
| --- | --- |
| 产品范围、服务职责与非目标 | [业务边界](./业务边界.md) |
| 房间、场次、媒体生命周期 | [live-service 演进需求](../../services/live-service/演进需求.md) |
| `InteractionMessage`、事件信封、RPC 语义和错误码 | [消息与接口 v1](./contracts/消息与接口-v1.md) |
| interaction Redis Key、Lua 原子边界、广播频道和 AI 逻辑分片 | [实时分发 v1](./contracts/实时分发-v1.md) |
| WebSocket 外部 JSON | [ws-gateway 互动协议 v1](../../services/ws-gateway/互动协议-v1.md) |
| interaction-service 内部执行流程 | [interaction 设计](../../services/interaction-service/设计.md) |
| AI 筛选、聚合、决策和调度 | [AI 设计](../../services/ai-service/设计.md) |
| Tool、动态事实、缓存和 RAG | [知识与模型](../../services/ai-service/知识与模型.md) |
| 模型输出安全和 AI 评测 | [安全与评测](../../services/ai-service/安全与评测.md) |
| 跨服务开关、降级和端到端 SLO | [运行与安全](./运行与安全.md) |
| 当前状态、P0/P1/P2 和迁移步骤 | [实施路线](./实施路线.md) |

服务文档只引用上述权威定义，不复制完整字段表、Redis Key 表或阶段清单。正式 `interaction.proto` 创建后，字段号、枚举值和 RPC 形状以 Proto 为最终权威；本文档组继续负责解释语义。

## 状态说明

- `implemented`：与当前代码一致。
- `proposed`：已冻结的目标设计，尚未实现。
- `superseded`：只用于历史追溯，不能指导新实现。

当前 WS 直管弹幕实现的历史材料位于 `docs/archive/live-interaction/ws-gateway-v1/`。归档文档中的 Redis/RocketMQ 双链路不属于目标架构。
