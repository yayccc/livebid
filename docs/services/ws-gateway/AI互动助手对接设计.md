# AI 互动助手对接设计

## 一、边界

`ws-gateway` 只负责 AI 互动助手消息的输入和输出通道：

1. 将用户弹幕事件提供给 `ai-service` 消费。
2. 将 `ai-service` 生成的直播间 AI 消息广播给对应直播间。
3. 确保 AI 消息与普通用户弹幕区分，不能伪装成真实用户。

`ws-gateway` 不判断 AI 是否自动回复，也不处理商家或主播确认流程；这些策略由 `ai-service` 和商家后台负责。

## 二、弹幕输入事件

### danmaku.created

用户弹幕通过校验后，`ws-gateway` 发布事件给 `ai-service` 消费。

```json
{
  "event_id": "evt_dm_10000001",
  "event_type": "danmaku.created",
  "message_id": "dm_10000001",
  "room_id": 4001,
  "user_id": 9001,
  "nickname": "张三",
  "content": "这个怎么加价",
  "content_type": "text",
  "server_time": 1780000001000
}
```

发布规则：

1. 只有通过校验并已广播的弹幕才发布事件。
2. 事件中不包含头像。
3. 消费方按 `event_id` 或 `message_id` 幂等。
4. AI 互动助手不保证处理每条弹幕，可按问题识别、房间频控、相似问题合并和服务负载进行采样。
5. 事件投递失败不影响弹幕实时广播，但必须记录日志和指标。

## 三、AI 输入投递模式

第一版支持两种投递模式，通过配置切换，并可在联调和压测时同时开启。

### Redis Pub/Sub 实时模式

```text
Channel: ws:room:{room_id}:ai:danmaku
Payload: danmaku.created 事件 JSON
```

特点：

1. 低延迟，适合 AI 互动助手及时识别用户问题。
2. 允许丢失，`ai-service` 不承诺处理每条弹幕。
3. `ai-service` 重启或订阅断开期间错过的弹幕不补偿。
4. 适合第一版验证 AI 互动体验。

### RocketMQ 事件模式

```text
Topic: live_interaction_event
ProducerGroup: ws-gateway-interaction-producer
Tag: danmaku.created
ConsumerGroup: ai-service-interaction
Consumer: ai-service
```

特点：

1. 有消费进度和失败重试，适合更可靠的异步消费。
2. 延迟和链路复杂度高于 Redis Pub/Sub。
3. 适合后续审计、统计、风控、复盘等消费场景。
4. 如果用于 AI 实时回答，需要设置消息过期或消费侧丢弃过期事件，避免过时回复。

### 配置

| 配置项 | 说明 | 默认值 |
| --- | --- | --- |
| `WS_GATEWAY_INTERACTION_EVENT_TOPIC` | 弹幕和 AI 互动事件 Topic | `live_interaction_event` |
| `WS_GATEWAY_INTERACTION_PRODUCER_GROUP` | ws-gateway 互动事件 ProducerGroup | `ws-gateway-interaction-producer` |
| `WS_GATEWAY_AI_INPUT_MODE` | AI 输入投递模式，支持 `redis_pubsub`、`rocketmq`、`both` | `redis_pubsub` |

## 四、AI 消息回流

### ai_interaction.created

`ai-service` 生成可展示的直播间消息后发布事件，`ws-gateway` 消费后统一通过 Redis Pub/Sub fanout 给所有 `ws-gateway` 实例。

```json
{
  "event_id": "evt_ai_10000001",
  "event_type": "ai_interaction.created",
  "message_id": "ai_msg_10000001",
  "room_id": 4001,
  "assistant_role": "ai_interaction",
  "display_name": "AI互动助手",
  "content": "这件商品当前竞拍还在进行，可以留意倒计时和当前价。",
  "content_type": "text",
  "server_time": 1780000005000
}
```

处理规则：

1. `ws-gateway` 只校验事件格式和 `room_id`，不判断 AI 策略。
2. RocketMQ 使用集群消费模式，单条 `ai_interaction.created` 只由一个 `ws-gateway` 实例消费。
3. 消费到事件的 `ws-gateway` 将其转换为 WebSocket 广播 `ai_interaction_created`，并发布到 Redis Pub/Sub 房间 AI 广播频道。
4. 所有 `ws-gateway` 实例订阅 Redis Pub/Sub 后向本实例房间连接广播。
5. 同一实例按 `event_id` 做短期去重。
6. AI 消息优先级低于竞拍事件，高于普通在线人数变化。

RocketMQ 约定：

```text
Topic: live_interaction_event
ProducerGroup: ai-service-interaction-producer
ConsumerGroup: ws-gateway-ai-interaction
MessageModel: CLUSTERING
Tag: ai_interaction.created
```

Redis Pub/Sub 约定：

```text
Channel: ws:room:{room_id}:ai:events
Payload: ai_interaction_created 广播消息 JSON
```

回流流程：

```text
1. ws-gateway 以 RocketMQ 集群消费模式消费 ai_interaction.created。
2. 校验 room_id、message_id、content。
3. 构造 ai_interaction_created 广播消息。
4. 发布到 Redis Pub/Sub: ws:room:{room_id}:ai:events。
5. 所有 ws-gateway 实例收到 Pub/Sub 消息后，向本实例 room_id 连接广播。
```

AI 消息回流固定使用“RocketMQ 集群消费到一个 ws-gateway，再 Redis Pub/Sub fanout 到所有 ws-gateway”的拓扑，避免 RocketMQ 广播消费和 Redis fanout 两套语义并存。

## 五、实现落点

| 模块 | 改动 |
| --- | --- |
| `internal/client` | 新增或扩展 AI 互动事件 producer/consumer，支持 Redis Pub/Sub 和 RocketMQ 两种投递模式 |
| `internal/config` | 新增互动事件配置 |
| `internal/bootstrap` | 启动 AI 输入发布组件和 AI 消息回流 consumer |
