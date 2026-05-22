# 微服务架构

## 总体架构

```text
React Web / 管理后台
        |
        v
API Gateway
        |
        +------------------+
        |                  |
        v                  v
WebSocket Gateway      HTTP API
        |                  |
        v                  v
auction-service     user-service
live-service        goods-service
order-service       payment-service
message-service     ai-service
        |
        v
MySQL / Redis / MQ / Object Storage
```

## 服务拆分

| 服务 | 职责 | 是否拥有数据库 |
| --- | --- | --- |
| `api-gateway` | 对外 HTTP 入口、鉴权、路由、限流、响应聚合 | 否 |
| `ws-gateway` | WebSocket 长连接、直播间连接管理、消息收发 | 可选 |
| `user-service` | 用户、登录、身份、账号状态 | 是 |
| `goods-service` | 商品、分类、图片、鉴定报告、估价基础信息 | 是 |
| `live-service` | 直播间、主播、直播状态、在线人数 | 是 |
| `auction-service` | 竞拍、出价、排名、倒计时、落锤成交 | 是 |
| `order-service` | 成交订单、订单状态、超时关闭 | 是 |
| `payment-service` | 支付单、支付回调、退款，可先 mock | 是 |
| `message-service` | 站内通知、系统消息、广播事件消费 | 可选 |
| `ai-service` | AI 定价建议、话术生成、风险提示 | 可选 |

## 核心服务职责

`api-gateway`：

- 对外 REST API
- JWT 校验
- 用户身份透传
- 路由到内部服务
- 聚合页面需要的数据
- 统一错误码和响应格式

`ws-gateway`：

- 维护 WebSocket 连接
- 用户加入/离开直播间
- 接收用户出价消息
- 调用 `auction-service` 完成出价
- 向直播间广播竞拍变化
- 心跳检测和断线清理

`auction-service`：

- 创建竞拍
- 开始竞拍
- 校验出价
- 维护最高价和排名
- 控制倒计时延长
- 执行落锤成交
- 发布 `auction.success` 或 `auction.failed` 事件

`order-service`：

- 消费 `auction.success`
- 创建成交订单
- 管理订单状态
- 处理支付成功事件
- 处理订单超时关闭

`payment-service`：

- 创建支付单
- 模拟或对接支付
- 接收支付回调
- 发布 `payment.succeeded` 事件

## 服务间调用

同步调用适合：

- 网关查询直播间页面数据
- `auction-service` 校验商品是否存在
- `auction-service` 校验直播间是否存在
- `order-service` 查询支付单状态

异步事件适合：

```text
auction.started
bid.accepted
auction.success
auction.failed
order.created
order.closed
payment.succeeded
payment.failed
```

典型成交链路：

```text
auction-service 落锤成交
 -> 发布 auction.success
 -> order-service 消费事件并创建订单
 -> order-service 发布 order.created
 -> ws-gateway 消费事件并广播成交结果
```
