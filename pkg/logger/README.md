# logger

`pkg/logger` 是各 Go 服务共用的 zap 日志包，默认输出 JSON，支持服务名、环境、文件切割、context 字段透传和 Gin 访问日志/恢复中间件。

## 基础用法

```go
log := logger.MustInit(logger.ProductionConfig("auction-service"))
defer logger.Sync()

log.Info("service started")
logger.L().Info("bid accepted", logger.RequestID("req-1"), logger.AuctionID(1001))
```

## 开发环境

```go
logger.MustInit(logger.DevelopmentConfig("auction-service"))
```

## 文件日志

```go
logger.MustInit(logger.ProductionConfig("auction-service"), logger.WithFile("./logs/auction-service.log"))
```

默认切割策略：

- 单文件最大 `100MB`
- 最多保留 `10` 个备份
- 最长保留 `30` 天
- 默认压缩历史文件

## 环境变量

```text
LIVEBID_LOG_SERVICE=auction-service
LIVEBID_LOG_ENV=prod
LIVEBID_LOG_LEVEL=info
LIVEBID_LOG_ENCODING=json
LIVEBID_LOG_OUTPUTS=stdout
LIVEBID_LOG_FILE=./logs/auction-service.log
LIVEBID_LOG_FILE_MAX_SIZE_MB=100
LIVEBID_LOG_FILE_MAX_BACKUPS=10
LIVEBID_LOG_FILE_MAX_AGE_DAYS=30
LIVEBID_LOG_FILE_COMPRESS=true
```

```go
logger.MustInit(logger.LoadConfigFromEnv(""))
```

## Gin

```go
r := gin.New()
r.Use(logger.GinRecovery(nil))
r.Use(logger.GinMiddleware(nil, logger.WithGinSkipPaths("/healthz")))

r.GET("/ping", func(c *gin.Context) {
	logger.GinLogger(c).Info("pong")
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
})
```

中间件会自动记录 method、path、query、client_ip、user_agent、status、body_size、latency，并透传 `X-Request-Id`、`X-Trace-Id` 到请求 context。
