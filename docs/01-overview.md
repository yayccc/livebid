# 项目总览

## 项目定位

系统目标是支持直播间内高价值非标商品的实时竞拍，覆盖：

- 主播/运营创建竞拍商品
- 配置竞拍规则
- 用户进入直播间
- WebSocket 实时出价
- 实时最高价和排名广播
- 倒计时和延时规则
- 落锤成交
- 订单生成
- 支付和超时关闭
- AI 辅助定价、话术和氛围营造

## 核心难点

这个项目不是普通电商 CRUD，真正难点在实时交易链路：

- 高并发出价
- 毫秒级实时通知
- 出价顺序一致性
- 竞拍状态机
- 防重复出价、防刷、防作弊
- 直播间消息广播
- 成交、订单、支付之间的最终一致性

# 技术选型表
```text
前端：React + TypeScript + Vite + H5 + Zustand
后端：Go + Gin + GORM + zap + validator
微服务通信：gRPC + Protobuf
注册配置：Nacos
实时通信：WebSocket Gateway
存储：MySQL + Redis
消息：Kafka，或 RabbitMQ/NATS 简化
容器：Docker + Docker Compose
仓库与 CI：GitHub + GitHub Actions
AI：独立 ai-service，调用模型 API 或本地开源模型，只做辅助建议，不参与核心成交判定
```

## 第一版范围

第一版建议实现完整闭环，而不是追求所有工程能力：

```text
登录
 -> 创建直播间
 -> 创建商品
 -> 创建竞拍
 -> WebSocket 进入直播间
 -> 开始竞拍
 -> 实时出价
 -> 广播最高价
 -> 落锤成交
 -> 生成订单
 -> 支付 mock
 -> AI 定价/话术建议
```

## 核心原则

1. 前端只访问 `api-gateway` 和 `ws-gateway`。
2. 每个微服务只访问自己的数据库。
3. 竞拍实时态放 Redis，最终事实落 MySQL。
4. 出价和成交必须做幂等。
5. 成交到订单使用事件驱动。
6. AI 只辅助，不接管交易决策。
