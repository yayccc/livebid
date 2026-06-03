package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yayccc/livebid/pkg/auth"
	"github.com/yayccc/livebid/services/api-gateway/internal/handler"
)

func TestRegisterRoutesDoesNotPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("register routes panicked: %v", recovered)
		}
	}()

	registerTestRoutes(t, engine)
}

func TestUserAvatarUploadRequiresUserAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	registerTestRoutes(t, engine)

	req := httptest.NewRequest(http.MethodPost, "/api/users/avatar/upload", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}
}

func registerTestRoutes(t *testing.T, engine *gin.Engine) {
	t.Helper()

	jwtManager, err := auth.NewJWTManager("test-secret", auth.WithIssuer("livebid"))
	if err != nil {
		t.Fatalf("new jwt manager: %v", err)
	}

	Register(
		engine,
		handler.NewShopHandler(nil, jwtManager, time.Hour, time.Second),
		handler.NewUserHandler(nil, time.Second),
		handler.NewGoodsHandler(nil, time.Second),
		handler.NewFileHandler(nil),
		handler.NewLiveHandler(nil, time.Second),
		handler.NewAuctionHandler(nil, time.Second),
		jwtManager,
		jwtManager,
	)
}
