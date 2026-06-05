# 微服务架构

## 总体架构

下图描述目标微服务架构，不代表当前仓库已经实现全部服务。当前开发以 `01-overview.md` 中的“当前阶段”为准。

```text
mobile-user / merchant-admin
        |
        +----------------------+
        |                      |
        v                      v
API Gateway              WebSocket Gateway
        |                      |
        |                      v
        |                auction-service
        |                      |
        +----------+-----------+-----------+
                   |                       |
                   v                       v
        user/goods/live service     order/payment service
                   |                       |
                   +-----------+-----------+
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

## 架构边界

- 前端只访问 `api-gateway` 和 `ws-gateway`，不直接调用具体业务微服务。
- `api-gateway` 是对外 HTTP 入口，负责鉴权、限流、路由、聚合和统一响应格式。
- `ws-gateway` 是 WebSocket 入口，负责连接管理、直播间订阅、消息收发和广播。
- 除网关外，普通业务微服务默认只暴露内部 gRPC 接口。
- JWT 签发与验证设计详见 `docs/04-auth-design.md`。
- 每个业务微服务只读写自己拥有的数据表。
- 跨服务同步调用优先使用 gRPC，跨服务状态流转优先使用 MQ 事件。
- 竞拍实时状态优先放 Redis，最终事实落 MySQL。
- AI 只提供定价、话术、风控等辅助建议，不参与出价有效性、落锤成交、订单支付等核心判定。

## 核心服务职责

`api-gateway`：

- 对外 REST API
- 鉴权、限流和用户身份透传
- 路由到内部 gRPC 服务
- 聚合页面需要的数据
- 统一错误码和响应格式

`auction-service`：

- 创建竞拍
- 开始竞拍
- 校验出价
- 维护最高价和排名
- 控制倒计时延长
- 执行落锤成交
- 发布 `auction_finished` 或 `auction_failed` 事件

`order-service`：

- 消费 `auction_finished`
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
auction_started
bid_accepted
auction_finished
auction_failed
order.created
order.closed
payment.succeeded
payment.failed
```

典型成交链路：

```text
auction-service 落锤成交
 -> 发布 auction_finished
 -> order-service 消费事件并创建订单
 -> order-service 发布 order.created
 -> ws-gateway 消费事件并广播成交结果
```
