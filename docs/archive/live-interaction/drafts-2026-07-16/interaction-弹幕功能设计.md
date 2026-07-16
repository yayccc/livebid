# 直播间弹幕与互动服务设计（2026-07-16 草案归档）

> Status: superseded。本文是 2026-07-16 的未实施草案，已被 `docs/features/live-interaction/` 与拆分后的服务文档取代，不能作为现行契约。

## 一、文档目的

本文统一直播间公开互动消息的业务边界、内部接口、实时分发、短历史以及与 `ai-service` 的协作方式。

本文所称“互动消息”第一期只包含：

1. 登录用户发送的文本弹幕。
2. AI 互动助手生成的公开回答或低频氛围消息。

点赞、礼物、私聊、运营指令和完整聊天历史不在本期范围内。

与 AI 侧对齐的需求和实现约束见：

- `docs/services/ai-service/需求.md`
- `docs/services/ai-service/设计.md`

## 二、核心结论

1. `ws-gateway` 只负责 WebSocket 连接、协议适配、调用内部服务以及向本实例连接广播，不再拥有弹幕业务规则。
2. `interaction-service` 负责消息准入、直播场次校验、幂等、频控、消息 ID、房间序号、最近消息和 Redis 房间 fan-out。
3. `ai-service` 负责观察、采样、聚合、是否回复的决策、知识查询和内容生成；`interaction-service` 不做 AI 策略判断。
4. 用户消息和 AI 消息都必须通过 `interaction-service` 形成统一的规范消息；AI 不得直接写 Redis 房间频道或调用 `ws-gateway`。
5. 在线分发和 AI 实时输入使用 Redis Pub/Sub，语义是低延迟、弱顺序、尽力投递、允许丢失，不使用 RocketMQ 作为实时 AI 主链路。
6. 最近消息是有界短历史，不是完整聊天记录，不提供离线可靠补发。
7. `room_id` 承载长期房间互动，`live_session_id` 只隔离单次直播的 AI 和业务上下文；离线房间消息不要求直播场次。

## 三、服务边界

| 能力 | ws-gateway | interaction-service | ai-service |
| --- | --- | --- | --- |
| WebSocket 连接与本地房间 Hub | 负责 | 不负责 | 不负责 |
| 用户身份解析 | 负责外部 JWT 校验并透传可信身份 | 校验内部调用身份 | 不负责 |
| 消息长度、直播状态、幂等、频控 | 只做协议级校验 | 负责最终判定 | 不负责 |
| message_id、event_id、room_seq | 不生成 | 统一生成 | 不生成 |
| 最近 10 条互动消息 | 读取后返回客户端 | 统一维护 | 不作为消费日志使用 |
| 跨网关实时 fan-out | 订阅并广播到本地连接 | 发布 Redis 房间频道 | 不参与 |
| 是否调用模型、是否回复 | 不负责 | 不负责 | 负责 |
| RAG、业务工具调用、模型生成 | 不负责 | 不负责 | 负责 |
| AI 公开消息身份 | 按规范展示 | 强制生成固定 AI 身份 | 只能提交内容和决策元数据 |

`interaction-service` 不参与出价、成交、订单等核心交易规则。涉及当前价、倒计时和竞拍状态时，权威事实仍以 `auction-service` 为准。

## 四、关键标识

| 标识 | 产生方 | 语义 |
| --- | --- | --- |
| `request_id` | 客户端或 AI 调用方 | 一次发布请求的幂等键；客户端重试必须复用 |
| `decision_id` | ai-service | 一次 AI 决策的幂等键，并作为 AI 发布请求的 `request_id` |
| `message_id` | interaction-service | 一条规范互动消息的唯一标识 |
| `event_id` | interaction-service | 一次实时事件投递的唯一标识 |
| `room_id` | live-service | 可复用的直播间标识，用于客户端连接和房间路由 |
| `live_session_id` | live-service | 一次业务开播的稳定标识；用户离线房间消息可以为空，AI 消息必须存在 |
| `room_seq` | interaction-service | 持久房间内单调递增序号，用于弱排序、去重和缺口识别，不代表可靠投递 |

### live_session_id 生命周期

目标契约由 `live-service` 在商家调用 `StartLive` 时生成 `current_live_session_id`，在 `EndLive` 时结束并清空。SRS `on_publish/on_unpublish` 只更新媒体状态，不能生成或切换直播场次。

房间未开播时，用户消息的 `live_session_id` 为空；AI 输入和 AI 回写只接受有效的当前场次。

## 五、总体拓扑

```text
用户
  -> WebSocket send_danmaku
  -> ws-gateway
  -> gRPC PublishUserMessage
  -> interaction-service
       -> live-service 校验房间可见性、互动开关并解析可选当前场次
       -> Redis 原子准入、幂等、频控、room_seq、最近消息
       -> Redis Pub/Sub 房间 fan-out
       -> Redis Pub/Sub AI 输入分片（只发布真人消息）
            -> ai-service 选择性处理
            -> gRPC PublishAssistantMessage
            -> interaction-service
            -> 同一房间 fan-out

Redis 房间 fan-out
  -> 持有该房间本地连接的 ws-gateway 实例
  -> WebSocket 客户端
```

实时主链路不经过 RocketMQ。后续审计、统计、训练样本等可靠异步需求可以独立发布事件，但不得让 AI 同时从 Redis 和 MQ 消费同一批实时弹幕。

## 六、统一互动消息

内部统一使用 `InteractionMessage` 语义：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `message_id` | string | interaction-service 生成 |
| `event_id` | string | 当前创建事件 ID |
| `room_id` | int64 | 直播间 ID |
| `live_session_id` | optional int64 | 当前直播场次 ID；离线房间用户消息为空，AI 消息必填 |
| `room_seq` | int64 | 持久房间内递增序号 |
| `sender_type` | enum | `user` 或 `ai_assistant` |
| `sender_id` | int64 | 真人为 user_id；AI 为 0 |
| `display_name` | string | 用户展示昵称或固定 AI 名称 |
| `assistant_role` | string | AI 固定为 `ai_interaction`，真人为空 |
| `content` | string | 规范化后的文本 |
| `content_type` | string | 第一期固定 `text` |
| `action_type` | string | AI 可为 `answer`、`atmosphere` 或 `redirect`，真人为空 |
| `reply_to_message_ids` | string[] | AI 回答所依据的代表消息，可为空 |
| `status` | string | 第一期固定 `visible` |
| `created_at_ms` | int64 | 服务端 Unix 毫秒时间 |

内部 gRPC 使用枚举名 `ANSWER`、`ATMOSPHERE`、`REDIRECT`；规范消息和 WebSocket JSON 中的 `action_type` 使用对应的小写值 `answer`、`atmosphere`、`redirect`。

长度限制分开配置：

- 用户弹幕：默认 1～30 个 Unicode 字符。
- AI 公开消息：默认 1～80 个 Unicode 字符，必须适合直播间快速阅读。

AI 回复和真人弹幕使用同一最近消息列表。否则新进入直播间的用户可能只看到问题，看不到已经公开的 AI 回答。

## 七、内部 gRPC 契约草案

本节定义接口语义，不代表 Proto 已生成。正式开发时应在 `api/proto/interaction/v1/interaction.proto` 中增量落地。

```text
service InteractionService {
  rpc PublishUserMessage(PublishUserMessageRequest)
      returns (PublishMessageResponse);

  rpc PublishAssistantMessage(PublishAssistantMessageRequest)
      returns (PublishMessageResponse);

  rpc GetRecentMessages(GetRecentMessagesRequest)
      returns (GetRecentMessagesResponse);
}
```

### PublishUserMessage

请求字段：

| 字段 | 必须 | 说明 |
| --- | --- | --- |
| `request_id` | 是 | 客户端请求 ID |
| `room_id` | 是 | 连接绑定的直播间 ID |
| `user_id` | 是 | 必须与 ws-gateway 透传的可信用户身份一致 |
| `display_name_snapshot` | 否 | ws-gateway 已缓存的展示昵称；interaction-service 可规范化或使用兜底值 |
| `content` | 是 | 用户文本 |
| `trace_id` | 否 | 链路追踪 ID |

处理规则：

1. 只接受受信任的 `ws-gateway` 内部调用，不信任客户端直接传入的 `user_id`。
2. 校验直播间已发布且允许互动，解析可选的当前 `live_session_id`。
3. 按 `room_id + user_id + request_id` 幂等。
4. 幂等命中时返回首次生成的规范消息，不重复频控、广播或投递 AI。
5. 同一用户同一房间默认 1 秒最多发送 1 条。
6. 通过基础内容规则后生成 `message_id`、`event_id` 和 `room_seq`。

### PublishAssistantMessage

请求字段：

| 字段 | 必须 | 说明 |
| --- | --- | --- |
| `decision_id` | 是 | AI 决策 ID，同时作为幂等键 |
| `room_id` | 是 | 目标直播间 |
| `live_session_id` | 是 | AI 观察消息所属场次 |
| `content` | 是 | 已经过 AI 输出安全检查的公开文本 |
| `action_type` | 是 | `answer`、`atmosphere` 或 `redirect` |
| `reply_to_message_ids` | 否 | 被回答或被聚合的消息 ID |
| `source_refs` | 否 | 工具或 RAG 来源引用，只用于审计与调试，不直接信任为用户可见内容 |
| `valid_until_ms` | 是 | 超过该时间后 interaction-service 拒绝过时回复 |

处理规则：

1. 只接受通过服务身份认证的 `ai-service` 调用。
2. interaction-service 重新确认直播仍在进行且场次未变化。
3. 按 `decision_id` 幂等；重复调用返回首次生成的消息。
4. `sender_type`、`display_name`、`assistant_role`、`message_id`、`event_id` 和 `room_seq` 全部由 interaction-service 生成，不能信任 AI 请求传入。
5. AI 消息写入最近消息并发布房间 fan-out，但绝不再次发布到 AI 输入频道。

### GetRecentMessages

请求字段为 `room_id` 和可选 `limit`。返回当前 `live_session_id`、最近消息列表和最新 `room_seq`。

第一期默认返回 10 条，按 `room_seq` 正序排列。该接口用于进入直播间时的氛围展示和轻量重连补偿，不承诺补齐所有缺口。

### WebSocket 兼容映射

迁移 interaction-service 时不要求前端同步更换消息类型：

| InteractionMessage | WebSocket type | 字段映射 |
| --- | --- | --- |
| `sender_type=user` | `danmaku_created` | `display_name` 对外映射为现有 `nickname` |
| `sender_type=ai_assistant` | `ai_interaction_created` | 保留 `assistant_role` 和 AI 专属展示字段 |

广播信封新增 `live_session_id` 和 `room_seq`；客户端应忽略不认识的新增字段，从而保持向后兼容。`send_danmaku` 成功响应除 `message_id`、`room_id` 外，也返回 `live_session_id` 和 `room_seq`，便于发送端与随后广播去重。

`connect` 响应中的 `recent_danmaku` 目标上改为统一最近互动消息，允许同时包含 `user` 和 `ai_assistant`；字段名是否同步改为 `recent_interactions` 需要前端协议评审，迁移期可以保留旧字段名。

## 八、用户消息处理流程

```text
1. ws-gateway 完成 JSON 解析、连接身份检查和基础大小限制。
2. ws-gateway 调用 PublishUserMessage，不在本地生成消息或提前广播。
3. interaction-service 解析当前直播场次并执行基础内容校验。
4. 在 Redis 中按 `room_id` 原子完成幂等判断、频控、room_seq 分配、最近消息写入和幂等结果记录。
5. interaction-service 发布房间 fan-out。
6. 仅当消息带有效当前 `live_session_id` 时，将已接纳真人消息发布到 AI 输入分片。
7. interaction-service 返回规范消息；ws-gateway 返回 send_danmaku 响应。
8. 各 ws-gateway 从房间频道收到规范消息后，向本实例连接广播。
```

“发送成功”的提交点是第 4 步完成。房间 Pub/Sub 和 AI 输入发布均是尽力投递，不属于成功前置条件；发布失败记录指标但不回滚已接纳消息。

客户端不得依赖“发送响应”和“房间广播”的网络到达顺序，应使用 `message_id` 和 `room_seq` 去重、排序。

## 九、AI 输入事件

事件名称固定为：

```text
interaction.user_message.accepted.v1
```

示例：

```json
{
  "schema_version": 1,
  "event_id": "evt_interaction_10000001",
  "event_type": "interaction.user_message.accepted.v1",
  "message_id": "msg_10000001",
  "room_id": 4001,
  "live_session_id": 910000000001,
  "room_seq": 81,
  "user_id": 9001,
  "display_name": "张三",
  "content": "这个怎么加价",
  "content_type": "text",
  "created_at_ms": 1780000001000,
  "expires_at_ms": 1780000006000,
  "trace_id": "trace_10001"
}
```

约束：

1. 只发布已经被 interaction-service 接纳的真人消息。
2. AI 消息、系统消息和被拒绝消息不得进入该事件流。
3. 投递语义为 at-most-once；AI 断线或重启期间的消息不补偿。
4. AI 收到时若 `now > expires_at_ms` 必须直接丢弃，不允许追赶旧问题。
5. 不提供 `redis_pubsub | rocketmq | both` 运行时切换，避免同一消息触发两次 AI 决策。

## 十、Redis 设计

### 业务状态 Key

```text
interaction:room:{room_id}:seq
  -> 当前 room_seq

interaction:room:{room_id}:recent
  -> 最近 InteractionMessage，默认 10 条

interaction:room:{room_id}:req:user:{user_id}:{request_id}
  -> 幂等结果，默认 TTL 60 秒

interaction:room:{room_id}:rate:user:{user_id}
  -> 用户发送频控，默认 TTL 1 秒

interaction:session:{live_session_id}:ai:req:{decision_id}
  -> AI 发布幂等结果，TTL 应覆盖最大生成和短重试窗口
```

最近消息使用 Redis List 即可满足“最近 10 条”需求：`LPUSH + LTRIM + EXPIRE`。它属于持久房间氛围，不作为 AI 消费日志；如果后续需要按游标补洞、撤回或更长窗口，再评估 bounded Redis Stream。

### 房间 fan-out

```text
Channel: interaction:room:{room_id}:fanout
Payload: InteractionMessage JSON
```

网关订阅规则：

1. 每个 `ws-gateway` 只为本实例存在连接的房间维护一份订阅，不按每条 WebSocket 连接重复订阅。
2. 本实例第一个连接进入房间时订阅，最后一个连接离开后延迟取消订阅，避免用户抖动造成频繁重订阅。
3. 不使用全房间模式订阅让每个网关接收全站互动消息。
4. 第一阶段使用普通 Redis Pub/Sub；采用 Redis Cluster 后评估 Redis 7 Sharded Pub/Sub，并保持同样的房间订阅语义。

### AI 输入分片

```text
Channel: interaction:ai:input:{bNN}
bucket = CRC32-IEEE(UTF-8 live_session_id) % shard_count
默认 shard_count = 64
```

`shard_count` 和散列算法是 interaction-service 与 ai-service 的共享契约，不允许两边独立配置成不同值。第一阶段可以只启用一个逻辑订阅者；多实例方案见 `ai-service/设计.md`。

## 十一、顺序、一致性与投递语义

1. 只保证 interaction-service 成功接纳后分配的 `room_seq` 在同一持久房间内单调递增。
2. Redis Pub/Sub、网络和多网关本地队列可能导致丢失、重复或到达顺序变化。
3. 客户端按 `message_id` 去重，并以 `room_seq` 防止旧消息覆盖新状态；不等待缺失序号无限补洞。
4. 最近消息只提供轻量恢复，不作为可靠日志。
5. AI 输入丢失不会影响用户弹幕成功；AI 没有回复也是正常业务结果。
6. 第一阶段不在 Redis 与数据库之间做分布式事务，也不为了审计阻塞实时发送链路。

## 十二、失败策略

| 场景 | 行为 |
| --- | --- |
| interaction-service 不可用 | ws-gateway 返回服务不可用，不得本地先广播 |
| live-service 与房间访问状态缓存均不可用 | 拒绝新消息，避免向未发布或已禁用房间发送 |
| 幂等或频控 Redis 操作失败 | 拒绝本次发布 |
| 最近消息写入失败 | 原子准入失败则拒绝；准入成功后不得出现“只有幂等记录、没有规范消息”的中间状态 |
| 房间 Pub/Sub 失败 | 保留成功结果，记录指标，不回滚 |
| AI 输入 Pub/Sub 失败 | 用户消息仍成功，记录指标，不补偿 |
| AI 发布的场次已变化或已过期 | 返回非重试错误，不广播 |
| AI 发布 gRPC 超时但结果不确定 | ai-service 在新鲜度内使用同一 `decision_id` 短重试 |
| 重复 AI 发布 | 返回首次结果，不重复广播 |
| 单连接写队列满 | 优先保留竞拍事件；互动消息允许丢弃或关闭慢连接 |

## 十三、安全与内容边界

1. `interaction-service` 必须区分用户调用身份和服务调用身份，普通用户不能调用 AI 发布接口。
2. `ws-gateway` 不能仅凭客户端 JSON 中的 `user_id` 发布消息，必须透传连接绑定身份。
3. AI 身份由 interaction-service 固定生成，禁止 AI 请求自定义真人 `sender_type`、`user_id` 或昵称。
4. 用户文本进行 trim、字符数限制、控制字符过滤和基础敏感词拦截。
5. AI 输出即使已在 ai-service 完成 guard，interaction-service 仍执行长度、空内容和最终阻断词检查。
6. 日志默认不打印完整弹幕、Prompt、RAG 文档或模型回答；使用 message_id、intent hash、reason code 和长度等字段排障。

## 十四、可观测性

至少记录以下指标：

- `interaction_publish_total{sender_type,result}`
- `interaction_rejected_total{reason}`
- `interaction_idempotent_hit_total{sender_type}`
- `interaction_rate_limited_total`
- `interaction_fanout_publish_total{result}`
- `interaction_ai_input_publish_total{result,bucket}`
- `interaction_publish_latency_ms`
- `interaction_recent_read_latency_ms`

链路追踪应能通过 `trace_id -> request_id/decision_id -> message_id -> event_id` 关联用户发送、AI 决策和最终房间广播。

## 十五、目标配置

以下名称是目标语义，正式实现时再确定 YAML 和环境变量映射：

| 配置 | 默认建议 | 说明 |
| --- | --- | --- |
| `USER_MESSAGE_MAX_CHARS` | 30 | 用户文本上限 |
| `ASSISTANT_MESSAGE_MAX_CHARS` | 80 | AI 公开文本上限 |
| `RECENT_MESSAGE_LIMIT` | 10 | 最近消息条数 |
| `RECENT_MESSAGE_TTL` | 24h | 持久房间短历史 TTL；不作为 AI 场次上下文 |
| `USER_RATE_LIMIT_INTERVAL` | 1s | 用户发送间隔 |
| `REQUEST_IDEMPOTENCY_TTL` | 60s | 用户请求幂等 |
| `AI_DECISION_IDEMPOTENCY_TTL` | 10m | AI 短重试幂等 |
| `ROOM_STATUS_CACHE_TTL` | 5s | 直播状态缓存 |
| `AI_INPUT_MAX_AGE` | 5s | AI 输入新鲜度 |
| `AI_INPUT_SHARD_COUNT` | 64 | AI 输入固定分片数 |

## 十六、从当前实现迁移

当前代码已经在 `ws-gateway` 中实现弹幕 Redis repository、幂等、频控、最近消息、直播状态校验和 AI Pub/Sub 发布。迁移应分阶段进行，避免同时存在两个消息产生者：

1. 先定义并评审 interaction Proto，不修改对外 WebSocket JSON。
2. 实现 interaction-service 的 Redis 准入、最近消息和房间 fan-out。
3. ws-gateway 改为调用 `PublishUserMessage`、`GetRecentMessages`，并订阅新的房间频道。
4. 关闭 ws-gateway 本地弹幕生成、幂等、频控、recent 写入和 AI 输入发布。
5. 接入 ai-service Redis 输入和 `PublishAssistantMessage` 回流。
6. 确认没有旧频道生产者后，再删除旧配置和旧 Redis Key。

迁移期间禁止同时让 ws-gateway 和 interaction-service 为同一用户请求生成消息，否则会出现双 message_id、双广播和 AI 重复决策。

## 十七、第一期范围与后续能力

### 第一期

1. 登录用户文本弹幕。
2. 公开 AI 回答和低频氛围消息。
3. 用户与 AI 消息统一短历史。
4. Redis Pub/Sub 在线 fan-out 和 AI 实时输入。
5. 请求幂等、用户频控、场次隔离和基础内容规则。

### 后续

1. 显式 `@AI` 目标字段和优先级。
2. 管理员撤回、禁言和异步审核。
3. 高频问题卡片和主播副驾。
4. 可靠审计、统计和训练数据事件。
5. 私人问答及独立会话历史。

## 十八、待审查决策

1. AI 回复最大长度默认 80 字是否适合前端展示。
2. AI 输入固定分片数是否从 64 起步，还是第一版先使用单逻辑订阅者。
3. 最近 10 条是否同时展示用户和 AI 消息；本文推荐统一展示。
4. interaction-service 的最终提交是否采用 Redis Lua 脚本一次完成幂等、频控、room_seq 和 recent 写入。
5. 基础敏感词来源和更新机制由 interaction-service 自管，还是接入后续审核服务。
