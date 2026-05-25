# goods-service

`goods-service` 是商品侧底层业务服务，当前只暴露内部 gRPC 接口，不启动 Gin/HTTP REST 服务。对外 HTTP API 后续应由 `api-gateway` 承接并调用本服务的 gRPC client。

## 当前能力

- 创建商品：`CreateGoods`
- 编辑商品：`UpdateGoods`
- 删除商品：`DeleteGoods`
- 查询商品详情：`GetGoods`
- 分页查询商品列表：`ListGoods`
- 查询店铺商品列表：`ListShopGoods`
- 批量查询商品信息：`BatchGetGoods`
- 商品上架：`PutGoodsOnSale`
- 商品下架：`PutGoodsOffSale`
- gRPC Health Checking

## 本地启动

```bash
go run ./services/goods-service/cmd/server
```

默认读取配置文件：

```text
services/goods-service/configs/config.local.yaml
```

可通过 `GOODS_SERVICE_CONFIG` 指定其他配置文件：

```bash
GOODS_SERVICE_CONFIG=services/goods-service/configs/config.local.yaml go run ./services/goods-service/cmd/server
```

环境变量优先级高于配置文件。默认监听地址为 `:9002`，可通过环境变量覆盖：

```bash
GOODS_SERVICE_GRPC_ADDR=:9002 go run ./services/goods-service/cmd/server
```

默认 MySQL 连接为：

```text
root:123456@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local
```

可通过环境变量覆盖：

```bash
GOODS_SERVICE_MYSQL_DSN='root:123456@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local'
GOODS_SERVICE_MYSQL_AUTO_MIGRATE=true
```

## 接口契约

Proto 定义位于：

```text
api/proto/goods/v1/goods.proto
```

生成代码位于：

```text
gen/proto/goods/v1
```

重新生成：

```bash
protoc --go_out=. --go_opt=module=github.com/yayccc/livebid \
  --go-grpc_out=. --go-grpc_opt=module=github.com/yayccc/livebid \
  api/proto/goods/v1/goods.proto
```

## 说明

当前 repository 使用 GORM 操作 MySQL。服务启动时默认执行 `AutoMigrate`，会自动创建或更新 `goods` 表结构；生产环境可通过 `GOODS_SERVICE_MYSQL_AUTO_MIGRATE=false` 关闭。
