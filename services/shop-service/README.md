# shop-service

`shop-service` 是店铺侧底层业务服务，当前只暴露内部 gRPC 接口，不启动 Gin/HTTP REST 服务。对外 HTTP API 后续应由 `api-gateway` 承接并调用本服务的 gRPC client。

## 当前能力

- 商铺注册：`RegisterShop`
- 商铺登录校验：`LoginShop`
- 查询商铺信息：`GetShop`
- 修改商铺信息：`UpdateShop`
- gRPC Health Checking

## 本地启动

```bash
go run ./services/shop-service/cmd/server
```

默认读取配置文件：

```text
services/shop-service/configs/config.local.yaml
```

可通过 `SHOP_SERVICE_CONFIG` 指定其他配置文件：

```bash
SHOP_SERVICE_CONFIG=services/shop-service/configs/config.local.yaml go run ./services/shop-service/cmd/server
```

环境变量优先级高于配置文件。默认监听地址为 `:9001`，可通过环境变量覆盖：

```bash
SHOP_SERVICE_GRPC_ADDR=:9001 go run ./services/shop-service/cmd/server
```

默认 MySQL 连接为：

```text
root:123456@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local
```

可通过环境变量覆盖：

```bash
SHOP_SERVICE_MYSQL_DSN='root:123456@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local'
SHOP_SERVICE_MYSQL_AUTO_MIGRATE=true
```

## 接口契约

Proto 定义位于：

```text
api/proto/shop/v1/shop.proto
```

生成代码位于：

```text
gen/proto/shop/v1
```

重新生成：

```bash
protoc --go_out=. --go_opt=module=github.com/yayccc/livebid \
  --go-grpc_out=. --go-grpc_opt=module=github.com/yayccc/livebid \
  api/proto/shop/v1/shop.proto
```

## 说明

当前 repository 使用 GORM 操作 MySQL。服务启动时默认执行 `AutoMigrate`，会自动创建或更新 `shop`、`shop_login_log` 表结构；生产环境可通过 `SHOP_SERVICE_MYSQL_AUTO_MIGRATE=false` 关闭。
