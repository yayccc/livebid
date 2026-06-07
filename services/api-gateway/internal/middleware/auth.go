package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yayccc/livebid/pkg/auth"
	"github.com/yayccc/livebid/pkg/identity"
)

func RequireShopAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return RequireAuth(jwtManager, identity.KindShop)
}

func RequireUserAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return RequireAuth(jwtManager, identity.KindUser)
}

func OptionalUserAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return OptionalAuth(jwtManager, identity.KindUser)
}

func RequireAuth(jwtManager *auth.JWTManager, kind identity.Kind) gin.HandlerFunc {
	return func(c *gin.Context) {
		if jwtManager == nil {
			abortUnauthorized(c, "认证服务未配置，无法校验登录状态")
			return
		}

		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			abortUnauthorized(c, "请先登录并在 Authorization 请求头中携带 Bearer Token")
			return
		}

		claims, err := jwtManager.Verify(token)
		if err != nil {
			_ = c.Error(err)
			abortUnauthorized(c, "登录凭证无效或已过期，请重新登录")
			return
		}

		principal, ok := identity.NewPrincipal(kind, claims.Subject)
		if !ok {
			abortUnauthorized(c, "登录凭证中的身份信息无效")
			return
		}

		c.Request = c.Request.WithContext(identity.NewContext(c.Request.Context(), principal))
		c.Next()
	}
}

func OptionalAuth(jwtManager *auth.JWTManager, kind identity.Kind) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			c.Next()
			return
		}
		if jwtManager == nil {
			abortUnauthorized(c, "认证服务未配置，无法校验登录状态")
			return
		}
		claims, err := jwtManager.Verify(token)
		if err != nil {
			_ = c.Error(err)
			abortUnauthorized(c, "登录凭证无效或已过期，请重新登录")
			return
		}
		principal, ok := identity.NewPrincipal(kind, claims.Subject)
		if !ok {
			abortUnauthorized(c, "登录凭证中的身份信息无效")
			return
		}
		c.Request = c.Request.WithContext(identity.NewContext(c.Request.Context(), principal))
		c.Next()
	}
}

func bearerToken(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	if authHeader == "" {
		return ""
	}
	scheme, token, ok := strings.Cut(authHeader, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

func abortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"code":    http.StatusUnauthorized,
		"message": message,
		"data":    nil,
	})
}
