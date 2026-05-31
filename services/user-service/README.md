# user-service

`user-service` 是普通用户侧底层业务服务，当前只暴露内部 gRPC 接口，不启动 Gin/HTTP REST 服务。对外 HTTP API 后续应由 `api-gateway` 承接并调用本服务的 gRPC client。

## 当前能力

- 用户注册：`RegisterUser`
- 用户登录并签发 JWT：`LoginUser`
- 查询用户公开资料：`GetUser`
- 修改当前用户资料：`UpdateCurrentUser`
- 新增、查询、修改、删除和设置默认收货地址
- gRPC Health Checking

## 本地启动

```bash
go run ./services/user-service/cmd/server
```

默认读取配置文件：

```text
services/user-service/configs/config.local.yaml
```

可通过 `USER_SERVICE_CONFIG` 指定其他配置文件：

```bash
USER_SERVICE_CONFIG=services/user-service/configs/config.local.yaml go run ./services/user-service/cmd/server
```

环境变量优先级高于配置文件。默认监听地址为 `:9003`，可通过环境变量覆盖：

```bash
USER_SERVICE_GRPC_ADDR=:9003 go run ./services/user-service/cmd/server
```

默认 MySQL 连接为：

```text
root:123456@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local
```

可通过环境变量覆盖：

```bash
USER_SERVICE_MYSQL_DSN='root:123456@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local'
USER_SERVICE_MYSQL_AUTO_MIGRATE=true
```

JWT 配置：

```bash
USER_SERVICE_JWT_SECRET='replace-me'
USER_SERVICE_JWT_ISSUER=livebid
USER_SERVICE_JWT_ACCESS_TOKEN_TTL_SECONDS=604800
```

## 接口契约

Proto 定义位于：

```text
api/proto/user/v1/user.proto
```

生成代码位于：

```text
gen/proto/user/v1
```

重新生成：

```bash
protoc --go_out=. --go_opt=module=github.com/yayccc/livebid \
  --go-grpc_out=. --go-grpc_opt=module=github.com/yayccc/livebid \
  api/proto/user/v1/user.proto
```

## 说明

当前 repository 使用 GORM 操作 MySQL。服务启动时默认执行 `AutoMigrate`，会自动创建或更新 `user`、`user_address` 表结构；生产环境可通过 `USER_SERVICE_MYSQL_AUTO_MIGRATE=false` 关闭。

受保护的当前用户资料和地址接口从 gRPC metadata `livebid-auth-subject` 读取当前 user_id。头像上传接口本阶段不在 `user-service` 中实现，仅保存头像 URL。
