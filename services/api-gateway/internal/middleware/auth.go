package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yayccc/livebid/pkg/auth"
	"github.com/yayccc/livebid/services/api-gateway/internal/identity"
)

func RequireShopAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return RequireAuth(jwtManager, identity.KindShop)
}

func RequireUserAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return RequireAuth(jwtManager, identity.KindUser)
}

func RequireAuth(jwtManager *auth.JWTManager, kind identity.Kind) gin.HandlerFunc {
	return func(c *gin.Context) {
		if jwtManager == nil {
			abortUnauthorized(c, "missing auth")
			return
		}

		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			abortUnauthorized(c, "missing auth")
			return
		}

		claims, err := jwtManager.Verify(token)
		if err != nil {
			_ = c.Error(err)
			abortUnauthorized(c, "invalid token")
			return
		}

		id, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil || id <= 0 {
			if err != nil {
				_ = c.Error(err)
			}
			abortUnauthorized(c, "invalid token")
			return
		}

		identity.Set(c, identity.Principal{
			Kind:    kind,
			ID:      id,
			Subject: claims.Subject,
		})
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
	})
}
