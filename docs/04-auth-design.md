# JWT 鉴权设计

## 当前结论

当前阶段只做最小可用鉴权：

- 登录成功后签发 `access_token`。
- 受保护接口校验 `Authorization: Bearer <access_token>`。
- `access_token` 有效期为 7 天。
- JWT 只表示登录身份，不表示业务权限。
- 商家端和用户端使用不同 `issuer` 隔离 token，不在 JWT 中额外放用户类型字段。
- 网关鉴权通过后，通过 gRPC metadata 向底层服务透传当前身份。

## 代码落点

```text
pkg/auth/jwt.go
pkg/auth/jwt_test.go
services/api-gateway
```

JWT 使用成熟库实现：

```text
github.com/golang-jwt/jwt/v5
```

签名算法使用 `HS256`。密钥通过环境变量或配置注入，真实密钥不能提交到仓库。

## Token 与 Claims

当前只使用一种 token：

| Token | 用途 | 有效期 |
| --- | --- | --- |
| `access_token` | 调用需要登录的接口 | 7 天 |

当前 `Claims` 只保留 4 个字段：

| Go 字段 | JWT 字段 | 说明 |
| --- | --- | --- |
| `Issuer` | `iss` | 签发方，例如商家端 `livebid-shop`、用户端 `livebid-user` |
| `Subject` | `sub` | 当前登录主体 ID |
| `ID` | `jti` | token ID，可选 |
| `ExpiresAt` | `exp` | 过期时间 |

示例：

```json
{
  "iss": "livebid-shop",
  "sub": "10001",
  "jti": "token-1",
  "exp": 1779520200
}
```

当前不在 JWT 中放：

- `roles`
- `scopes`
- `typ`
- `subject_type`
- `shop_id`
- `user_id`
- 业务权限
- 资源权限

## 签发与验证

登录校验通过后签发 token：

```go
token, err := manager.Sign(auth.Claims{
    Subject:   "10001",
    ID:        "token-1",
    ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
})
```

返回给前端：

```json
{
  "access_token": "<jwt>",
  "expires_in": 604800
}
```

受保护接口从 Header 读取 token：

```text
Authorization: Bearer <access_token>
```

验证通过后使用 `claims.Subject` 作为当前登录主体 ID。商家端接口只接受商家端 issuer，用户端接口只接受用户端 issuer；同一个 `sub` 在不同端的接口语义不同。

## HTTP 鉴权约定

受保护接口由 `api-gateway` 统一校验 JWT。

商家端接口使用商家鉴权中间件：

- 只接受 `iss = livebid-shop` 的 token。
- 将 `claims.Subject` 解析为 `shop_id` 并写入 Gin Context。

用户端接口后续使用用户鉴权中间件：

- 只接受 `iss = livebid-user` 的 token。
- 将 `claims.Subject` 解析为 `user_id` 并写入 Gin Context。

没有 token、token 无效或 token 过期时：

- 返回 `401 Unauthorized`。
- 不调用底层 gRPC 服务。
- 不继续执行业务逻辑。

前端收到 `401` 后，统一清理本地 token 并跳转登录页。后端只返回状态码和 JSON，不做页面跳转。

当前不做权限控制。后续如果出现“已登录但无权限”的场景，再返回 `403 Forbidden`。

## gRPC Metadata 约定

网关鉴权通过后，将当前身份写入 gRPC metadata：

| Metadata Key | 值 | 说明 |
| --- | --- | --- |
| `livebid-auth-subject-type` | `shop` / `user` | 当前登录主体类型 |
| `livebid-auth-subject-id` | `claims.Subject` | 当前登录主体 ID |

底层服务需要当前登录身份时，从 gRPC `context.Context` 读取公共 `identity.Principal`。不要相信前端请求参数中的 `user_id`、`shop_id` 等身份字段，也不要在业务代码中直接解析 metadata。

不同服务按接口语义解释 `Subject`：商家管理类接口解释为 `shop_id`，用户侧接口解释为 `user_id`。

下游服务读取商家身份示例：

```go
shopID, ok := identity.ShopID(ctx)
if !ok {
    return nil, status.Error(codes.Unauthenticated, "invalid credential")
}
```

服务内继续调用其他 gRPC 服务时，保持传递当前 `ctx`；公共 gRPC client interceptor 会自动把 `identity.Principal` 写回 outgoing metadata。

信任边界：

- JWT 只在 `api-gateway` 验证。
- 底层服务信任网关透传的 metadata。
- 底层服务的 gRPC 端口只应暴露在内网或服务发现网络中。
- 当前 metadata 不是服务间调用鉴权。

## 错误映射

`pkg/auth` 对外暴露统一错误：

| 错误 | HTTP 状态码 |
| --- | --- |
| `ErrInvalidToken` | `401 Unauthorized` |
| `ErrInvalidSignature` | `401 Unauthorized` |
| `ErrTokenExpired` | `401 Unauthorized` |
| `ErrMissingSecret` | `500 Internal Server Error` |
| `ErrInvalidClaims` | `500 Internal Server Error` |

## 当前不做

以下能力当前不实现：

- `refresh_token`
- token 黑名单
- 主动踢下线
- 改密码后批量失效 token
- `roles` / `scopes` 权限控制
- 服务间调用鉴权
- WebSocket 鉴权

## 实现检查清单

实现或修改鉴权逻辑时，必须满足：

- `Claims` 只包含 `Issuer`、`Subject`、`ID`、`ExpiresAt`。
- JWT payload 只使用 `iss`、`sub`、`jti`、`exp`。
- 商家端 token 使用商家 issuer，用户端 token 使用用户 issuer。
- `access_token` 有效期为 7 天。
- 受保护接口必须校验 `Authorization: Bearer <access_token>`。
- 鉴权失败返回 `401`，且不得调用底层服务。
- gRPC metadata 只透传 `livebid-auth-subject-type` 和 `livebid-auth-subject-id`。
- 不实现 `roles`、`scopes`、`refresh_token`、token 黑名单。
