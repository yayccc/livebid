# AI 直播互动服务设计（2026-07-16 草案归档）

> Status: superseded。本文是 2026-07-16 的未实施草案，已被 `docs/features/live-interaction/` 与拆分后的 ai-service 文档取代，不能作为现行契约。

## 一、设计原则

1. AI 观察所有能够收到的真人弹幕，但不保证处理或回答每条消息。
2. 消息新鲜度高于可靠积压；过期任务直接丢弃，不追答旧问题。
3. 模型调用前完成窗口聚合、过滤、去重和预算控制。
4. 自动触发优先追求高精确率，无法确认时使用 `NO_REPLY`。
5. 动态事实走权威 gRPC 工具，静态非结构化知识走 RAG。
6. AI 只生成决策和内容，通过互动服务统一写入直播间。
7. 所有房间状态按 `live_session_id` 隔离，不按可复用 `room_id` 长期复用。

## 二、总体架构

```text
interaction.user_message.accepted.v1
  -> Redis Pub/Sub 固定分片
  -> Input Subscriber / Shard Lease
  -> Room Dispatcher
  -> Room Agent
       ├─ 本地环形上下文
       ├─ 300~800ms Window Aggregator
       ├─ Filter / Dedup / Semantic Cluster
       ├─ Candidate Scorer
       └─ Room Budget
  -> Fair Scheduler / Global Semaphore
  -> Decision Engine
       ├─ Tool Router -> auction/live/goods/shop gRPC
       ├─ RAG Retriever -> VectorStore
       └─ LLM Provider
  -> Output Guard
  -> gRPC PublishAssistantMessage
  -> interaction-service
  -> Redis 房间频道
  -> ws-gateway
```

低延迟接收、窗口聚合和模型生成使用独立有界队列。模型变慢时不能阻塞 Redis 订阅循环。

## 三、组件边界

| 组件 | 职责 |
| --- | --- |
| `InputSubscriber` | 订阅互动服务 AI 输入分片，校验事件并快速入队 |
| `ShardLeaseManager` | 保证一个分片同一时刻只由一个实例处理 |
| `RoomDispatcher` | 按 `live_session_id` 路由到单一 Room Agent |
| `RoomAgent` | 管理房间窗口、上下文、去重、冷却和候选 |
| `CandidateScorer` | 判断公共性、疑问强度、重复度、知识可用性和新鲜度 |
| `FairScheduler` | 在多房间之间公平选择生成任务并执行全局并发限制 |
| `KnowledgeRouter` | 选择动态工具、静态 RAG、房间上下文或不回答 |
| `DecisionEngine` | 输出结构化动作、原因、置信度和目标消息 |
| `ResponseGenerator` | 基于可信证据生成 80 字以内公开文本 |
| `OutputGuard` | 检查身份、长度、敏感内容、依据和时效 |
| `InteractionClient` | 幂等调用 `PublishAssistantMessage` |

## 四、输入事件与分片

### 4.1 共享事件

AI 只消费：

```text
event_type = interaction.user_message.accepted.v1
```

必需字段：

```text
schema_version
event_id
message_id
room_id
live_session_id
room_seq
user_id
display_name
content
content_type
created_at_ms
expires_at_ms
trace_id
```

未知 `schema_version`、缺少主键、`content_type != text`、当前时间超过 `expires_at_ms` 的事件直接丢弃并记录 `drop_reason`。

### 4.2 频道分片

频道和哈希算法必须与互动服务一致：

```text
interaction:ai:input:{b00} ... interaction:ai:input:{b63}
bucket = CRC32-IEEE(UTF-8 live_session_id) % shard_count
```

不能让所有 AI 实例订阅所有频道，否则同一消息会触发多次决策。第一阶段允许部署多个实例但只由一个逻辑 leader 持有全部启用分片；规模化后按租约分配分片。

### 4.3 分片租约

```text
Key: ai:input:shard:{bucket}:lease
Value: instance_id
TTL: 10 秒
Renew: 每 3 秒
```

实例只有持有租约时才能订阅或处理对应分片。租约切换允许短暂重叠，因此房间生成锁和互动服务 `decision_id` 幂等仍是最终防重边界。

## 五、房间执行模型

### 5.1 Room Agent

每个活跃 `live_session_id` 对应一个逻辑 Room Agent，同一时间只由一个 goroutine 或等价串行执行器修改本地房间状态。

Room Agent 维护：

- 最近 20 到 50 条弹幕，或最近 30 到 60 秒窗口。
- 当前待聚合消息。
- `message_id` 本地去重集合。
- 最近候选意图和公开 AI 消息摘要。
- 当前问答、氛围冷却和生成中状态的本地镜像。
- 最近一次可用的商品、直播和竞拍上下文引用。

本地状态有界且可丢失。实例重启后从新消息重新建立上下文，不追读历史消息。

### 5.2 生命周期

```text
收到当前场次第一条有效事件 -> 创建 Room Agent
持续收到消息             -> 刷新活跃时间
直播场次变化             -> 关闭旧 Agent，拒绝旧任务发布
长期无消息且无任务         -> 回收 Agent
```

第一阶段只知道已经收到消息的活跃房间，因此只能做基础冷场辅助。要覆盖“开播后无人发言”的房间，后续必须接入直播生命周期或显式房间激活信号。

## 六、处理状态机

```text
OBSERVED
  -> DROPPED            事件非法、过期、重复或队列满
  -> WINDOWED           进入房间聚合窗口
  -> FILTERED           噪声、非公共问题或已回答
  -> CANDIDATE          形成候选意图
  -> RESERVED           获得房间生成权和全局预算
  -> ENRICHED           完成工具或 RAG 取证
  -> DECIDED            得到结构化动作
  -> GENERATED          生成可展示内容
  -> GUARDED            通过输出检查
  -> PUBLISHED          互动服务接纳
  -> DROPPED            任一阶段失败、过期或返回 NO_REPLY
```

只记录状态、`reason_code` 和有限诊断信息，不记录模型 chain-of-thought。

建议 `drop_reason` 至少包括：

```text
invalid_event
expired
duplicate_message
noise
non_public
already_answered
room_cooldown
room_inflight
global_budget
queue_full
tool_unavailable
rag_no_evidence
model_timeout
guard_rejected
session_changed
publish_failed
```

## 七、窗口、过滤和候选选择

### 7.1 房间窗口

1. 默认聚合窗口为 500 毫秒，可配置在 300 到 800 毫秒之间。
2. 窗口保留不同用户的消息数量，用于识别多人重复问题。
3. 每个窗口最多输出 Top-K 候选，默认 `K=3`，最终最多生成一条公开消息。
4. 明确高优先级问题可以提前关闭窗口，但仍需经过房间预算和知识检查。

### 7.2 模型前过滤

规则层优先过滤：

1. 纯表情、无意义重复字符和过短噪声。
2. 相同用户短时间重复内容。
3. 已在冷却期回答的标准化意图。
4. 明显个人账户、订单和隐私问题。
5. 要求 AI 执行出价、改变价格或承诺结果的指令。
6. 与当前房间无关的 Prompt 注入和系统探测文本。

过滤规则只负责减少不必要的模型调用，不能把用户文本升级为可信事实。

### 7.3 候选评分

评分使用可配置特征，不把权重硬编码在 Prompt 中：

```text
score =
  question_strength
  + repeated_user_count
  + room_relevance
  + explicit_ai_signal
  + knowledge_availability
  + freshness
  - answered_penalty
  - privacy_risk
  - interruption_penalty
```

第一阶段可使用规则和轻量分类完成候选选择；后续再增加 Embedding 聚类或小模型分类。不要为了让大模型输出 `NO_REPLY` 而对每条弹幕单独调用大模型。

## 八、调度与背压

### 8.1 房间生成锁

```text
Key: ai:session:{live_session_id}:inflight
Value: decision_id
TTL: 模型硬超时 + 发布缓冲
```

使用 `SET NX` 或 Lua 获取。一个场次同一时刻最多一个生成任务。获得锁后模型调用失败或输出 `NO_REPLY` 时释放；进程崩溃依靠 TTL 自动释放。

### 8.2 公平调度

调度器按房间维护有界候选队列并采用轮转或加权公平队列：

1. 同一房间不能连续占用全部全局槽位。
2. 热门房间可以获得更高但有上限的权重。
3. 候选超过 `candidate_max_age` 后直接丢弃。
4. 队列满时优先丢弃旧、低分和普通闲聊候选。

### 8.3 冷却

```text
ai:session:{live_session_id}:cooldown:answer
ai:session:{live_session_id}:cooldown:atmosphere
ai:session:{live_session_id}:answered:{intent_hash}
```

默认建议：

- 问答冷却 8 到 15 秒。
- 氛围冷却 30 到 60 秒。
- 已回答意图 30 到 120 秒。

生成前先写短期候选预留。发布成功后将其转换为正式冷却和已回答状态；发布结果不确定时保留预留 TTL，避免立刻生成第二条相似回答。

## 九、结构化决策

模型或决策引擎必须输出可校验结构，不直接返回自由文本控制业务动作：

```json
{
  "action": "NO_REPLY | ANSWER | ATMOSPHERE | REDIRECT",
  "route": "NONE | AUCTION_CURRENT | AUCTION_RUNTIME | GOODS | LIVE | SHOP | RAG",
  "target_message_ids": ["dm_10000001"],
  "intent_key": "auction.bid_increment",
  "confidence": 0.94,
  "reason_code": "PUBLIC_FACTUAL_QUESTION",
  "valid_until_ms": 1780000010000
}
```

规则：

1. `NO_REPLY` 不生成公开消息。
2. `ANSWER` 必须有工具事实、RAG 证据或明确允许的静态平台知识。
3. `ATMOSPHERE` 不得编造价格、库存、优惠和倒计时。
4. `REDIRECT` 只能给出通用客服引导，不公开用户隐私信息。
5. 置信度阈值按动作和知识路由分别配置，不使用单一全局阈值。

## 十、知识路由与工具

### 10.1 路由顺序

```text
候选问题
  -> 是否涉及实时业务事实？ 是 -> gRPC Tool
  -> 是否涉及静态非结构化知识？ 是 -> RAG
  -> 是否仅需房间近期上下文？   是 -> Room Context
  -> 无可靠来源                    -> NO_REPLY / REDIRECT
```

RAG 不能作为动态工具失败后的兜底来源。

### 10.2 工具映射

| Route | RPC | 用途 |
| --- | --- | --- |
| `AUCTION_CURRENT` | `AuctionService.GetCurrentAuctionByRoom` | 当前竞拍、价格、加价幅度、商品和状态 |
| `AUCTION_RUNTIME` | `AuctionService.GetAuctionRuntime` | 当前价、次数、领先用户、倒计时和版本 |
| `GOODS` | `GoodsService.GetGoods` | 商品标题、描述和状态 |
| `LIVE` | `LiveService.GetLiveRoom` | 直播状态、商铺和场次校验 |
| `SHOP` | `ShopService.GetShop` | 商铺公开名称和描述 |

工具调用要求：

1. 使用短超时、并发上限和熔断。
2. 同一房间同一窗口对相同 RPC 做 singleflight 合并。
3. 当前价和倒计时等高波动事实在发布前检查数据版本和时效；生成耗时过长时重新查询或放弃回答。
4. 不把包含个人账户、订单和支付数据的工具开放给公开场控模型。

## 十一、RAG 设计

### 11.1 适用知识

- 平台竞拍规则。
- 商品说明和使用资料。
- 商家 FAQ。
- 物流、售后和服务政策。

商家 FAQ 当前没有明确的上游管理接口，第二阶段实施 RAG 前必须补充知识导入来源和商家权限边界。

### 11.2 知识对象

```text
KnowledgeDocument
  document_id
  owner_type: platform | shop | goods
  shop_id
  goods_id
  knowledge_type
  title
  content
  knowledge_version
  status
  updated_at

KnowledgeChunk
  chunk_id
  document_id
  chunk_index
  content
  embedding
  metadata
  content_hash
```

### 11.3 导入与更新

1. 拉取或接收知识源变更。
2. 规范化文本并按标题、段落和语义边界切分。
3. 以 `content_hash + embedding_model_version` 缓存 Embedding。
4. 写入向量存储，并携带 `shop_id`、`goods_id`、`knowledge_version` 和状态元数据。
5. 新版本可用后再切换活动版本，旧版本延迟清理。

向量存储通过 `VectorStore` 接口抽象，具体产品在实施阶段选择；当前文档不把尚未部署的向量数据库描述为既有依赖。

### 11.4 检索

1. 先按平台、商家和当前商品做强元数据过滤，再执行向量召回。
2. 第一阶段 RAG 默认 Top-K 和相似度阈值可配置。
3. 检索结果携带 `source_ref`、版本和片段，不只返回拼接文本。
4. 没有足够证据时返回 `rag_no_evidence`，不让模型依靠常识补写商家政策。
5. 所有缓存键包含知识版本，版本切换后旧缓存自然失效。

## 十二、缓存设计

### 12.1 本地缓存

| 缓存 | 上限/淘汰 |
| --- | --- |
| 房间弹幕环形缓冲 | 20 到 50 条或 30 到 60 秒 |
| `message_id` 去重 | 有界 LRU，TTL 大于订阅切换窗口 |
| 工具结果 | 仅单决策窗口或 singleflight 生命周期 |
| RAG 检索结果 | 有界 LRU，键包含知识版本 |

### 12.2 Redis 状态

```text
ai:input:shard:{bucket}:lease
ai:session:{live_session_id}:inflight
ai:session:{live_session_id}:cooldown:answer
ai:session:{live_session_id}:cooldown:atmosphere
ai:session:{live_session_id}:answered:{intent_hash}
ai:rag:{shop_id}:{goods_id}:{knowledge_version}:query:{query_hash}
ai:faq:{shop_id}:{goods_id}:{knowledge_version}:{prompt_version}:{intent_hash}
```

最后一个 Key 只缓存静态 FAQ 答案。以下内容禁止进入自然语言答案缓存：

- 当前价格和倒计时。
- 当前领先用户和出价结果。
- 竞拍、直播、库存或订单当前状态。
- 任何用户个人数据。

## 十三、Prompt 与模型设计

### 13.1 输入分区

模型输入明确分区：

1. 系统行为规则。
2. 允许的动作和结构化输出 Schema。
3. 权威工具事实。
4. RAG 证据与来源。
5. 房间近期上下文。
6. 使用明确分隔符包裹的不可信用户弹幕。

用户弹幕中的“忽略系统规则”“调用其他工具”等内容只能作为被分析文本，不能改变工具白名单和系统策略。

### 13.2 模型策略

1. 模型客户端通过统一 `ModelProvider` 接口封装，业务逻辑不绑定特定厂商。
2. 决策结果和生成结果都执行 JSON Schema 或等价结构校验。
3. 设置连接、首 token、总时长和最大 token 限制。
4. 模型超时默认 `NO_REPLY`；只有明确可重试错误且仍在新鲜度内才短重试一次。
5. 记录模型名、Prompt 版本、token、延迟和结束原因，不记录 chain-of-thought。

第一阶段可使用规则选候选并由一次模型调用完成工具选择和回答；流量增大后再引入小模型分类器和独立生成模型，避免过早形成两次昂贵模型调用。

## 十四、输出检查与发布

### 14.1 Output Guard

发布前检查：

1. 动作是否为 `ANSWER`、`ATMOSPHERE` 或 `REDIRECT`。
2. 文本是否为空、超过 80 字或包含禁止内容。
3. `ANSWER` 是否满足对应 route 的证据要求。
4. 动态事实版本和时间是否仍有效。
5. 当前场次、房间冷却和 `valid_until_ms` 是否有效。
6. 文本是否泄露内部 Prompt、工具字段、用户隐私或调试内容。

### 14.2 gRPC 回流

```text
InteractionService.PublishAssistantMessage
  decision_id
  room_id
  live_session_id
  action_type
  content
  reply_to_message_ids
  source_refs
  valid_until_ms
  trace_id
```

`decision_id` 在决定生成任务时创建，并在所有重试中保持不变。互动服务返回成功后，AI 才写入正式回答冷却和已回答意图；如果本地提交失败，预留 TTL 继续抑制短时间重复回答。

gRPC 请求使用 `ANSWER`、`ATMOSPHERE`、`REDIRECT` 枚举值；互动服务落入规范消息后映射为小写 `action_type`。

## 十五、失败与降级

| 场景 | 行为 |
| --- | --- |
| Redis Pub/Sub 断开 | 重连后只消费新消息，不补偿旧消息 |
| 分片租约丢失 | 停止接收新任务，取消尚未进入模型的候选 |
| 房间队列满 | 丢弃旧和低优先级候选 |
| 工具调用失败 | 动态问题 `NO_REPLY`，不降级到 RAG 编造 |
| RAG 不可用或无证据 | 静态知识问题 `NO_REPLY` 或 `REDIRECT` |
| 模型超时或限流 | 静默丢弃并记录原因 |
| 输出检查失败 | 不发布 |
| 互动服务暂时不可用 | 在 `valid_until_ms` 内使用同一 `decision_id` 短重试 |
| 场次变化或结果过期 | 取消任务且不重试 |

依赖故障不能向直播间发送“AI 服务繁忙”等高频错误消息。

## 十六、配置建议

| 配置 | 默认值 | 说明 |
| --- | --- | --- |
| `AI_REDIS_ADDR` | `127.0.0.1:6379` | Redis 地址 |
| `AI_INPUT_SHARDS` | `64` | 必须与互动服务一致 |
| `AI_INPUT_MAX_AGE_MS` | `5000` | 必须与互动服务一致 |
| `AI_ROOM_WINDOW_MS` | `500` | 房间聚合窗口 |
| `AI_ROOM_CONTEXT_MESSAGES` | `50` | 本地上下文消息上限 |
| `AI_ROOM_QUEUE_SIZE` | `32` | 每房间候选队列上限 |
| `AI_GLOBAL_MODEL_CONCURRENCY` | 待压测 | 全局模型并发 |
| `AI_ANSWER_COOLDOWN_SECONDS` | `10` | 问答冷却初始值 |
| `AI_ATMOSPHERE_COOLDOWN_SECONDS` | `45` | 氛围冷却初始值 |
| `AI_CANDIDATE_MAX_AGE_MS` | `5000` | 候选最大等待时间 |
| `AI_ASSISTANT_MAX_CHARS` | `80` | 必须与互动服务一致 |
| `AI_INTERACTION_SERVICE_TARGET` | `127.0.0.1:9008` | 建议地址，实施前确认 |
| `AI_AUCTION_SERVICE_TARGET` | `127.0.0.1:9003` | 动态竞拍工具 |
| `AI_GOODS_SERVICE_TARGET` | `127.0.0.1:9002` | 商品工具，实施时按实际端口确认 |
| `AI_LIVE_SERVICE_TARGET` | `127.0.0.1:9007` | 直播工具 |
| `AI_SHOP_SERVICE_TARGET` | `127.0.0.1:9001` | 商铺工具，实施时按实际端口确认 |

模型、Embedding 和向量存储配置在选定供应方后补充，不在设计阶段写入真实密钥。

## 十七、可观测性与审计

### 17.1 指标

- `ai_input_received_total`
- `ai_input_dropped_total{reason}`
- `ai_room_window_total`
- `ai_candidate_total{result}`
- `ai_decision_total{action,route,reason_code}`
- `ai_room_inflight_total`
- `ai_model_request_total{model,result}`
- `ai_model_duration_ms{model}`
- `ai_model_tokens_total{model,type}`
- `ai_tool_request_total{tool,result}`
- `ai_rag_request_total{result}`
- `ai_publish_total{result}`
- `ai_end_to_end_duration_ms{action}`

### 17.2 追踪

使用 `trace_id` 串联：

```text
用户弹幕 event_id/message_id
  -> AI 输入分片
  -> room window / intent_key
  -> decision_id
  -> tool/RAG/model span
  -> AI message_id/room_seq
```

### 17.3 决策审计

第一阶段可以仅保存有界、脱敏的决策元数据：

```text
decision_id
live_session_id
target_message_ids
action
route
reason_code
confidence
source_refs
model_version
prompt_version
knowledge_version
latency_ms
token_usage
result
```

不保存 chain-of-thought。是否长期保留输入和输出正文需要单独的数据合规和保留期决策，不能默认开启。

## 十八、离线评测

建立带标签的直播弹幕样本集，至少包含：

1. 是否应该公开回答。
2. 预期动作和知识路由。
3. 相似问题聚类标识。
4. 所需权威事实或知识来源。
5. 是否涉及隐私、提示词注入或高风险内容。

持续评估：

- 自动触发 Precision / Recall。
- 路由准确率。
- RAG Recall@K 和有依据回答比例。
- 重复问题压缩率和重复公开回答率。
- P50/P95 延迟和单次回答成本。

上线阈值优先约束错误公开回答率，再评估漏答率。

## 十九、实施阶段

### P0：自动公共问答闭环

Redis 输入、房间窗口、规则过滤、基础候选评分、动态 gRPC 工具、单模型结构化决策、Redis 冷却、gRPC 回互动服务和基础评测。

### P1：语义聚类与 RAG

Embedding 聚类、知识导入、向量检索、版本失效、静态答案缓存、多实例分片租约和公平调度。

### P2：事件驱动场控

接入直播和竞拍生命周期、商品切换、临近结束和冷场信号；增加商家 AI 模式、主播副驾和高频问题卡片。

## 二十、待审查决策

1. P0 是否必须包含 RAG；当前建议 P0 先完成动态工具和自动回复闭环，P1 再加入完整知识生命周期。
2. 第一版模型是否同时承担结构化决策和生成，还是使用小模型选择器加生成模型。
3. VectorStore 的具体实现和商家 FAQ 的管理入口。
4. 完整事件驱动场控所需的直播、竞拍和商品切换事件来源。
5. 是否需要持久化脱敏决策审计，以及保留期限。

## 二十一、需求映射

| `需求.md` 要求 | 本文对应章节 |
| --- | --- |
| 唯一实时输入、允许丢失和多实例分片 | 四、输入事件与分片 |
| 房间上下文、窗口聚合和选择性处理 | 五、房间执行模型；六、处理状态机；七、窗口、过滤和候选选择 |
| 房间互斥、冷却、全局预算和公平性 | 八、调度与背压 |
| `NO_REPLY/ANSWER/ATMOSPHERE/REDIRECT` | 九、结构化决策 |
| 动态事实必须查询权威服务 | 十、知识路由与工具 |
| 静态规则、商品资料和 FAQ RAG | 十一、RAG 设计 |
| 本地上下文、Redis 冷却和静态缓存 | 十二、缓存设计 |
| Prompt 注入防护和结构化模型输出 | 十三、Prompt 与模型设计 |
| gRPC 回流、`decision_id` 幂等和 80 字限制 | 十四、输出检查与发布 |
| 依赖失败静默降级和过期丢弃 | 十五、失败与降级 |
| 延迟、成本、触发准确率和知识评测 | 十七、可观测性与审计；十八、离线评测 |
| P0/P1/P2 范围 | 十九、实施阶段 |
