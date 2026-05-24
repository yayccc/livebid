# JWT 鉴权设计

## 当前目标

当前阶段只实现最小可用的 JWT 能力：

- 登录成功后签发 `access_token`。
- 请求受保护接口时验证 `access_token`。
- token 只表达“当前登录主体是谁”和“什么时候过期”。
- 网关验证 token 后，把已验证身份通过 gRPC metadata 透传给底层服务。
- 暂不做角色、权限、refresh token、黑名单、主动踢下线。

代码落点：

```text
pkg/auth/jwt.go
pkg/auth/jwt_test.go
```

## 技术选型

JWT 使用成熟库实现：

```text
github.com/golang-jwt/jwt/v5
```

当前签名算法使用 `HS256`，也就是服务端用同一个密钥签发和验证 token。

密钥要求：

- 通过环境变量或配置文件注入。
- 不提交真实密钥到仓库。
- 本地开发可以使用示例密钥，生产环境必须替换成高强度随机密钥。

## Token 类型

当前只使用一种 token：

| Token | 用途 | 建议有效期 |
| --- | --- | --- |
| `access_token` | 调用需要登录的接口 | 15-30 分钟 |

暂不实现 `refresh_token`。token 过期后，用户需要重新登录。

## Claims

当前 claims 保持简单：

| 字段 | 说明 |
| --- | --- |
| `iss` | 签发方，例如 `livebid`，可选 |
| `sub` | 登录主体 ID，例如用户 ID、商铺 ID、管理员 ID |
| `typ` | 登录主体类型：`user`、`shop`、`admin` |
| `shop_id` | 商铺身份场景下的商铺 ID，可选 |
| `jti` | token 唯一 ID，可选 |
| `iat` | 签发时间 |
| `exp` | 过期时间 |

当前不在 token 中放：

- `roles`
- `scopes`
- 业务权限
- 资源权限

示例：

```json
{
  "iss": "livebid",
  "sub": "10001",
  "typ": "shop",
  "shop_id": "10001",
  "jti": "token-1",
  "iat": 1779518400,
  "exp": 1779520200
}
```

## 使用方式

### 初始化

```go
manager, err := auth.NewJWTManager(
    "your-secret",
    auth.WithIssuer("livebid"),
)
if err != nil {
    return err
}
```

### 签发 Token

登录校验通过后签发 token：

```go
token, err := manager.Sign(auth.Claims{
    Subject:   "10001",
    UserType:  auth.SubjectTypeShop,
    ShopID:    "10001",
    TokenID:   "token-1",
    ExpiresAt: time.Now().Add(30 * time.Minute),
})
if err != nil {
    return err
}
```

返回给前端：

```json
{
  "access_token": "<jwt>",
  "expires_in": 1800
}
```

### 验证 Token

请求受保护接口时，从 `Authorization` 头中取 token：

```text
Authorization: Bearer <access_token>
```

验证：

```go
claims, err := manager.Verify(token)
if err != nil {
    return err
}

shopID := claims.ShopID
```

`Verify` 会校验：

- token 格式是否合法。
- 签名是否正确。
- `exp` 是否存在。
- token 是否过期。
- 如果配置了 issuer，`iss` 是否匹配。

## gRPC Metadata 身份透传

底层服务不应该直接相信前端传入的 `user_id`、`shop_id` 等字段。当前约定是：

1. 前端请求 `api-gateway`。
2. 网关从 `Authorization: Bearer <access_token>` 取出 token。
3. 网关使用 `pkg/auth` 验证 token。
4. 验证通过后，网关把 claims 中的身份信息写入 gRPC metadata。
5. 底层服务只从 gRPC metadata 读取当前身份。

当前 metadata 只透传身份，不透传权限：

| Metadata Key | 来源 | 说明 |
| --- | --- | --- |
| `livebid-auth-subject` | `claims.Subject` | 当前登录主体 ID |
| `livebid-auth-subject-type` | `claims.UserType` | 当前登录主体类型：`user`、`shop`、`admin` |
| `livebid-auth-shop-id` | `claims.ShopID` | 商铺身份场景下的商铺 ID，可选 |
| `livebid-auth-token-id` | `claims.TokenID` | token 唯一 ID，可选 |

### 网关写入 Metadata

网关调用底层 gRPC 服务前，把验证后的 claims 放进 outgoing context：

```go
ctx := c.Request.Context()

claims, err := manager.Verify(accessToken)
if err != nil {
    return err
}

ctx = metadata.AppendToOutgoingContext(
    ctx,
    "livebid-auth-subject", claims.Subject,
    "livebid-auth-subject-type", string(claims.UserType),
    "livebid-auth-shop-id", claims.ShopID,
    "livebid-auth-token-id", claims.TokenID,
)

resp, err := shopClient.GetShop(ctx, &shopv1.GetShopRequest{
    Id: shopID,
})
```

如果某个字段为空，可以不写入。第一版可以先在需要登录身份的 handler 中显式添加，后续接口多了再抽成网关中间件或 gRPC client helper。

### 底层服务读取 Metadata

底层服务从 incoming context 读取身份：

```go
md, ok := metadata.FromIncomingContext(ctx)
if !ok {
    return nil, status.Error(codes.Unauthenticated, "missing auth metadata")
}

subject := firstMetadataValue(md, "livebid-auth-subject")
subjectType := firstMetadataValue(md, "livebid-auth-subject-type")
shopID := firstMetadataValue(md, "livebid-auth-shop-id")

if subject == "" || subjectType == "" {
    return nil, status.Error(codes.Unauthenticated, "missing auth identity")
}
```

辅助函数示例：

```go
func firstMetadataValue(md metadata.MD, key string) string {
    values := md.Get(key)
    if len(values) == 0 {
        return ""
    }
    return values[0]
}
```

### 信任边界

- JWT 只在 `api-gateway` 验证，底层服务当前不重复验证 JWT。
- 底层服务信任来自网关的 gRPC metadata，不信任前端请求参数中的身份字段。
- 底层服务的 gRPC 端口只应暴露在内网或服务发现网络中，不应直接暴露给公网。
- 当前 metadata 是“已验证身份”的传递方式，不是服务间调用鉴权。服务间调用鉴权后续需要时再单独设计。

## 错误类型

`pkg/auth` 对外暴露统一错误：

| 错误 | 说明 |
| --- | --- |
| `ErrMissingSecret` | JWT 密钥为空 |
| `ErrInvalidClaims` | 签发时 claims 不合法 |
| `ErrInvalidToken` | token 格式、issuer 或 claims 不合法 |
| `ErrInvalidSignature` | 签名不匹配 |
| `ErrTokenExpired` | token 已过期 |

接口层可以把这些错误映射成 HTTP 状态码：

| 错误 | HTTP 状态码 |
| --- | --- |
| `ErrInvalidToken` | `401 Unauthorized` |
| `ErrInvalidSignature` | `401 Unauthorized` |
| `ErrTokenExpired` | `401 Unauthorized` |
| `ErrMissingSecret` | `500 Internal Server Error` |
| `ErrInvalidClaims` | `500 Internal Server Error` |

## 当前不做

以下能力当前先不设计、不实现，等业务需要时再补文档和代码：

- `refresh_token`
- token 黑名单
- 主动踢下线
- 改密码后批量失效 token
- `roles` / `scopes` 权限控制
- 网关统一鉴权中间件
- 服务间调用鉴权
- WebSocket 鉴权

## 基本约定

- 前端不要传 `user_id`、`shop_id` 来声明“我是谁”，接口应以 JWT 验证后的 claims 为准。
- 底层服务需要当前登录身份时，从 gRPC metadata 读取网关透传的信息。
- JWT 只表示登录身份，不表示业务权限。
- 真实密钥不能提交到仓库。
- 过期时间不要设置太长，第一版建议 15-30 分钟。
