package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	shopv1 "github.com/yayccc/livebid/gen/proto/shop/v1"
	"github.com/yayccc/livebid/pkg/auth"
	"github.com/yayccc/livebid/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockShopClient struct {
	register func(ctx context.Context, in *shopv1.RegisterShopRequest, opts ...grpc.CallOption) (*shopv1.RegisterShopResponse, error)
	login    func(ctx context.Context, in *shopv1.LoginShopRequest, opts ...grpc.CallOption) (*shopv1.LoginShopResponse, error)
	get      func(ctx context.Context, in *shopv1.GetShopRequest, opts ...grpc.CallOption) (*shopv1.GetShopResponse, error)
	update   func(ctx context.Context, in *shopv1.UpdateShopRequest, opts ...grpc.CallOption) (*shopv1.UpdateShopResponse, error)
}

func (m mockShopClient) RegisterShop(ctx context.Context, in *shopv1.RegisterShopRequest, opts ...grpc.CallOption) (*shopv1.RegisterShopResponse, error) {
	return m.register(ctx, in, opts...)
}

func (m mockShopClient) LoginShop(ctx context.Context, in *shopv1.LoginShopRequest, opts ...grpc.CallOption) (*shopv1.LoginShopResponse, error) {
	return m.login(ctx, in, opts...)
}

func (m mockShopClient) GetShop(ctx context.Context, in *shopv1.GetShopRequest, opts ...grpc.CallOption) (*shopv1.GetShopResponse, error) {
	return m.get(ctx, in, opts...)
}

func (m mockShopClient) BatchGetPublicShops(ctx context.Context, in *shopv1.BatchGetPublicShopsRequest, opts ...grpc.CallOption) (*shopv1.BatchGetPublicShopsResponse, error) {
	panic("not implemented")
}

func (m mockShopClient) UpdateShop(ctx context.Context, in *shopv1.UpdateShopRequest, opts ...grpc.CallOption) (*shopv1.UpdateShopResponse, error) {
	return m.update(ctx, in, opts...)
}

func TestShopHandlerRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := newTestJWTManager(t)
	handler := NewShopHandler(mockShopClient{
		register: func(ctx context.Context, in *shopv1.RegisterShopRequest, opts ...grpc.CallOption) (*shopv1.RegisterShopResponse, error) {
			if in.GetUsername() != "merchant" || in.GetShopName() != "merchant shop" {
				t.Fatalf("unexpected register request: %#v", in)
			}
			return &shopv1.RegisterShopResponse{
				Shop: testShop(),
			}, nil
		},
		login: func(ctx context.Context, in *shopv1.LoginShopRequest, opts ...grpc.CallOption) (*shopv1.LoginShopResponse, error) {
			t.Fatal("LoginShop should not be called")
			return nil, nil
		},
	}, jwtManager, 30*time.Minute, time.Second)

	body := bytes.NewBufferString(`{"username":"merchant","password":"secret","shopName":"merchant shop"}`)
	w := performRequest(handler.Register, http.MethodPost, "/api/shop/register", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShopHandlerRegisterMultipartForm(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := newTestJWTManager(t)
	handler := NewShopHandler(mockShopClient{
		register: func(ctx context.Context, in *shopv1.RegisterShopRequest, opts ...grpc.CallOption) (*shopv1.RegisterShopResponse, error) {
			if in.GetUsername() != "禚婷婷" || in.GetPassword() != "ALJzZsqdqY89Soq" || in.GetShopName() != "电子的软肥皂" {
				t.Fatalf("unexpected register request: %#v", in)
			}
			if in.GetLogo() == "" || in.GetPhone() != "15181102387" || in.GetEmail() != "o97scp_ngo@vip.qq.com" {
				t.Fatalf("unexpected optional fields: %#v", in)
			}
			return &shopv1.RegisterShopResponse{
				Shop: testShop(),
			}, nil
		},
		login: func(ctx context.Context, in *shopv1.LoginShopRequest, opts ...grpc.CallOption) (*shopv1.LoginShopResponse, error) {
			t.Fatal("LoginShop should not be called")
			return nil, nil
		},
	}, jwtManager, 30*time.Minute, time.Second)

	body, contentType := multipartBody(t, map[string]string{
		"username":    "禚婷婷",
		"password":    "ALJzZsqdqY89Soq",
		"shopName":    "电子的软肥皂",
		"logo":        "https://loremflickr.com/646/3915?lock=7722414263455125",
		"description": "精致的",
		"phone":       "15181102387",
		"email":       "o97scp_ngo@vip.qq.com",
	})
	w := performRequestWithContentType(handler.Register, http.MethodPost, "/api/shop/register", body, contentType)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShopHandlerLoginSignsJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := newTestJWTManager(t)
	handler := NewShopHandler(mockShopClient{
		register: func(ctx context.Context, in *shopv1.RegisterShopRequest, opts ...grpc.CallOption) (*shopv1.RegisterShopResponse, error) {
			t.Fatal("RegisterShop should not be called")
			return nil, nil
		},
		login: func(ctx context.Context, in *shopv1.LoginShopRequest, opts ...grpc.CallOption) (*shopv1.LoginShopResponse, error) {
			if in.GetUsername() != "merchant" || in.GetPassword() != "secret" {
				t.Fatalf("unexpected login request: %#v", in)
			}
			return &shopv1.LoginShopResponse{
				Shop: testShop(),
			}, nil
		},
	}, jwtManager, 30*time.Minute, time.Second)

	body := bytes.NewBufferString(`{"username":"merchant","password":"secret"}`)
	w := performRequest(handler.Login, http.MethodPost, "/api/shop/login", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 || resp.Data.Token == "" {
		t.Fatalf("unexpected response: %#v", resp)
	}

	claims, err := jwtManager.Verify(resp.Data.Token)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.Subject != "1001" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestShopHandlerLoginMapsUnauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewShopHandler(mockShopClient{
		register: func(ctx context.Context, in *shopv1.RegisterShopRequest, opts ...grpc.CallOption) (*shopv1.RegisterShopResponse, error) {
			t.Fatal("RegisterShop should not be called")
			return nil, nil
		},
		login: func(ctx context.Context, in *shopv1.LoginShopRequest, opts ...grpc.CallOption) (*shopv1.LoginShopResponse, error) {
			return nil, status.Error(codes.Unauthenticated, "invalid credential")
		},
	}, newTestJWTManager(t), 30*time.Minute, time.Second)

	body := bytes.NewBufferString(`{"username":"merchant","password":"bad"}`)
	w := performRequest(handler.Login, http.MethodPost, "/api/shop/login", body)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShopHandlerGetUsesPathID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewShopHandler(mockShopClient{
		get: func(ctx context.Context, in *shopv1.GetShopRequest, opts ...grpc.CallOption) (*shopv1.GetShopResponse, error) {
			if in.GetId() != 2002 {
				t.Fatalf("unexpected get request: %#v", in)
			}
			return &shopv1.GetShopResponse{Shop: testShop()}, nil
		},
	}, newTestJWTManager(t), 30*time.Minute, time.Second)

	w := performShopRequestWithIDParam(handler.Get, http.MethodGet, "/api/shop/2002", bytes.NewBuffer(nil), "2002", 0, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShopHandlerGetCurrentForwardsIdentityContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewShopHandler(mockShopClient{
		get: func(ctx context.Context, in *shopv1.GetShopRequest, opts ...grpc.CallOption) (*shopv1.GetShopResponse, error) {
			if in.GetId() != 0 {
				t.Fatalf("unexpected get request: %#v", in)
			}
			if shopID, ok := identity.ShopID(ctx); !ok || shopID != 1001 {
				t.Fatalf("missing identity context: shopID=%d ok=%v", shopID, ok)
			}
			return &shopv1.GetShopResponse{Shop: testShop()}, nil
		},
	}, newTestJWTManager(t), 30*time.Minute, time.Second)

	w := performRequestWithShopID(handler.GetCurrent, http.MethodGet, "/api/shop/me", bytes.NewBuffer(nil), 1001)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShopHandlerGetCurrentUsesContextShopID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewShopHandler(mockShopClient{
		get: func(ctx context.Context, in *shopv1.GetShopRequest, opts ...grpc.CallOption) (*shopv1.GetShopResponse, error) {
			if in.GetId() != 0 {
				t.Fatalf("unexpected get request: %#v", in)
			}
			return &shopv1.GetShopResponse{Shop: testShop()}, nil
		},
	}, newTestJWTManager(t), 30*time.Minute, time.Second)

	w := performRequestWithShopID(handler.GetCurrent, http.MethodGet, "/api/shop/me", bytes.NewBuffer(nil), 1001)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShopHandlerUpdateUsesPathID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewShopHandler(mockShopClient{
		update: func(ctx context.Context, in *shopv1.UpdateShopRequest, opts ...grpc.CallOption) (*shopv1.UpdateShopResponse, error) {
			if in.GetId() != 1001 || in.GetShopName() != "new shop" || in.GetPhone() != "18800000000" {
				t.Fatalf("unexpected update request: %#v", in)
			}
			if shopID, ok := identity.ShopID(ctx); !ok || shopID != 1001 {
				t.Fatalf("missing identity context: shopID=%d ok=%v", shopID, ok)
			}
			return &shopv1.UpdateShopResponse{Shop: testShop()}, nil
		},
	}, newTestJWTManager(t), 30*time.Minute, time.Second)

	body := bytes.NewBufferString(`{"shopName":"new shop","phone":"18800000000"}`)
	w := performShopRequestWithIDParam(handler.Update, http.MethodPut, "/api/shop/1001", body, "1001", 1001, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShopHandlerUpdateRejectsMismatchedShopID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewShopHandler(mockShopClient{
		update: func(ctx context.Context, in *shopv1.UpdateShopRequest, opts ...grpc.CallOption) (*shopv1.UpdateShopResponse, error) {
			if in.GetId() != 1002 {
				t.Fatalf("unexpected update request: %#v", in)
			}
			return nil, status.Error(codes.PermissionDenied, "permission denied")
		},
	}, newTestJWTManager(t), 30*time.Minute, time.Second)

	body := bytes.NewBufferString(`{"shopName":"new shop"}`)
	w := performShopRequestWithIDParam(handler.Update, http.MethodPut, "/api/shop/1002", body, "1002", 1001, "")

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

func performRequest(handlerFunc gin.HandlerFunc, method string, path string, body *bytes.Buffer) *httptest.ResponseRecorder {
	return performRequestWithContentType(handlerFunc, method, path, body, "application/json")
}

func performRequestWithContentType(handlerFunc gin.HandlerFunc, method string, path string, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, body)
	c.Request.Header.Set("Content-Type", contentType)
	handlerFunc(c)
	return w
}

func performShopRequestWithIDParam(handlerFunc gin.HandlerFunc, method string, path string, body *bytes.Buffer, idParam string, shopID int64, authorization string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: idParam}}
	c.Request = httptest.NewRequest(method, path, body)
	c.Request.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		c.Request.Header.Set("Authorization", authorization)
	}
	if shopID > 0 {
		principal := identity.Principal{
			Kind: identity.KindShop,
			ID:   shopID,
		}
		c.Request = c.Request.WithContext(identity.NewContext(c.Request.Context(), principal))
	}
	handlerFunc(c)
	return w
}

func multipartBody(t *testing.T, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write multipart field: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func testShop() *shopv1.Shop {
	return &shopv1.Shop{
		Id:       1001,
		Username: "merchant",
		ShopName: "merchant shop",
	}
}

func newTestJWTManager(t *testing.T) *auth.JWTManager {
	t.Helper()
	manager, err := auth.NewJWTManager("test-secret", auth.WithIssuer("livebid"))
	if err != nil {
		t.Fatalf("new jwt manager: %v", err)
	}
	return manager
}
