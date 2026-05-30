package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yayccc/livebid/pkg/auth"
	"github.com/yayccc/livebid/pkg/identity"
)

func TestRequireShopAuthAcceptsShopIssuer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := newTestJWTManager(t, "livebid-shop")
	token := signTestToken(t, manager, "1001")
	w := performAuthRequest(RequireShopAuth(manager), "Bearer "+token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRequireShopAuthRejectsUserIssuer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	shopManager := newTestJWTManager(t, "livebid-shop")
	userManager := newTestJWTManager(t, "livebid-user")
	token := signTestToken(t, userManager, "1001")
	w := performAuthRequest(RequireShopAuth(shopManager), "Bearer "+token)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}
}

func performAuthRequest(middleware gin.HandlerFunc, authorization string) *httptest.ResponseRecorder {
	engine := gin.New()
	engine.Use(middleware)
	engine.GET("/protected", func(c *gin.Context) {
		shopID, ok := identity.ShopID(c.Request.Context())
		if !ok || shopID != 1001 {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "missing shop_id"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"shop_id": shopID})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func newTestJWTManager(t *testing.T, issuer string) *auth.JWTManager {
	t.Helper()
	manager, err := auth.NewJWTManager("test-secret", auth.WithIssuer(issuer))
	if err != nil {
		t.Fatalf("new jwt manager: %v", err)
	}
	return manager
}

func signTestToken(t *testing.T, jwtManager *auth.JWTManager, subject string) string {
	t.Helper()
	token, err := jwtManager.Sign(auth.Claims{
		Subject:   subject,
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}
