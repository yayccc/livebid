# auction-service

`auction-service` provides gRPC-only auction management and bidding APIs for LiveBid.

## Paths

- Proto: `api/proto/auction/v1/auction.proto`
- Generated code: `gen/proto/auction/v1`
- Local config: `services/auction-service/configs/config.local.yaml`

## Run

```bash
go run ./cmd/server
```

From the repository root:

```bash
go run ./services/auction-service/cmd/server
```

## Configuration

Environment variable prefix: `AUCTION_SERVICE_`.

Common overrides:

- `AUCTION_SERVICE_GRPC_ADDR`
- `AUCTION_SERVICE_MYSQL_DSN`
- `AUCTION_SERVICE_REDIS_ADDR`
- `AUCTION_SERVICE_ROCKETMQ_NAME_SERVER`
- `AUCTION_SERVICE_GOODS_ADDR`
- `AUCTION_SERVICE_WORKER_ID`

## Proto

Regenerate auction proto code from the repository root:

```bash
protoc --go_out=. --go_opt=module=github.com/yayccc/livebid \
  --go-grpc_out=. --go-grpc_opt=module=github.com/yayccc/livebid \
  api/proto/auction/v1/auction.proto
```

## Notes

The service keeps HTTP concerns in `api-gateway`. Runtime bidding state is stored in Redis and auction events are published to RocketMQ topic `auction_event`.
