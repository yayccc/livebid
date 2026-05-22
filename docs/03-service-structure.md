# 项目结构规范

## 仓库结构

推荐使用 monorepo：

```text
live-auction/
├── frontend/
├── services/
│   ├── api-gateway/
│   ├── ws-gateway/
│   ├── user-service/
│   ├── goods-service/
│   ├── live-service/
│   ├── auction-service/
│   ├── order-service/
│   ├── payment-service/
│   ├── message-service/
│   └── ai-service/
├── api/
│   ├── proto/
│   └── openapi/
├── gen/
├── pkg/
├── configs/
├── migrations/
├── docs/
├── deployments/
├── scripts/
├── Makefile
└── README.md
```

## 单个微服务结构

以 `auction-service` 为例：

```text
services/auction-service/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── handler/
│   │   ├── auction_handler.go
│   │   └── bid_handler.go
│   ├── service/
│   │   ├── auction_service.go
│   │   ├── bid_service.go
│   │   └── hammer_service.go
│   ├── repository/
│   │   ├── auction_repository.go
│   │   └── bid_record_repository.go
│   ├── model/
│   │   ├── auction.go
│   │   └── bid_record.go
│   ├── dto/
│   ├── client/
│   ├── mq/
│   ├── job/
│   ├── router/
│   ├── config/
│   └── bootstrap/
├── tests/
├── Dockerfile
├── Makefile
└── README.md
```

## 三层职责

handler 层：

- 接收 HTTP/gRPC 请求
- 参数绑定和基础校验
- 获取当前用户上下文
- 调用 service
- 统一返回响应
- 不写业务规则
- 不直接访问数据库、Redis、MQ

Service 层：

- 编排业务流程
- 控制事务
- 调用 repository
- 调用 Redis、MQ、RPC client
- 实现竞拍状态机
- 保证幂等和一致性

repository 层：

- 封装数据库访问
- 只处理本服务自己的数据表
- 不处理 HTTP、WebSocket、MQ
- 不直接返回前端 DTO
- 不调用其他服务

## 网关结构建议

`api-gateway` 一般不需要 repository：

```text
services/api-gateway/
├── cmd/server/main.go
├── internal/
│   ├── handler/
│   ├── service/
│   ├── client/
│   ├── middleware/
│   ├── router/
│   └── bootstrap/
└── README.md
```

`ws-gateway` 只管理连接和广播，不直接改竞拍数据库：

```text
services/ws-gateway/
├── cmd/server/main.go
├── internal/
│   ├── handler/
│   ├── service/
│   ├── client/
│   ├── ws/
│   ├── mq/
│   └── bootstrap/
└── README.md
```

## 公共包

`pkg` 只放真正跨服务复用的基础能力：

```text
pkg/
├── logger/
├── config/
├── response/
├── errors/
├── idgen/
├── redislock/
├── grpcx/
├── mq/
├── auth/
└── validator/
```

不要把业务逻辑放进 `pkg`。

# 命名规范
服务命名：

```text
api-gateway
ws-gateway
user-service
goods-service
live-service
auction-service
order-service
payment-service
ai-service
```

Go 文件：

```text
auction_controller.go
auction_service.go
auction_repository.go
bid_record_repository.go
```

前端文件：

```text
AuctionPanel.tsx
useAuction.ts
auctionStore.ts
```

Controller 命名：

```go
CreateAuction
StartAuction
PlaceBid
CloseAuction
GetAuctionDetail
ListBidRecords
```

Service 命名：

```go
CreateAuction
StartAuction
PlaceBid
HammerAuction
CreateAuctionOrder
```

repository 命名：

```go
FindByID
Create
UpdateStatus
UpdateHighestBid
ListByAuctionID
```
