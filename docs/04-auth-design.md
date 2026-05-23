# JWT 鉴权设计

## 当前目标

当前阶段只实现最小可用的 JWT 能力：

- 登录成功后签发 `access_token`。
- 请求受保护接口时验证 `access_token`。
- token 只表达“当前登录主体是谁”和“什么时候过期”。
- 暂不做角色、权限、refresh token、黑名单、主动踢下线、服务间身份透传。

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
- gRPC metadata 身份透传
- 服务间调用鉴权
- WebSocket 鉴权

## 基本约定

- 前端不要传 `user_id`、`shop_id` 来声明“我是谁”，接口应以 JWT 验证后的 claims 为准。
- JWT 只表示登录身份，不表示业务权限。
- 真实密钥不能提交到仓库。
- 过期时间不要设置太长，第一版建议 15-30 分钟。
