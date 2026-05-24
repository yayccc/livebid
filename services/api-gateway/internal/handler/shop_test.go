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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockShopClient struct {
	register func(ctx context.Context, in *shopv1.RegisterShopRequest, opts ...grpc.CallOption) (*shopv1.RegisterShopResponse, error)
	login    func(ctx context.Context, in *shopv1.LoginShopRequest, opts ...grpc.CallOption) (*shopv1.LoginShopResponse, error)
}

func (m mockShopClient) RegisterShop(ctx context.Context, in *shopv1.RegisterShopRequest, opts ...grpc.CallOption) (*shopv1.RegisterShopResponse, error) {
	return m.register(ctx, in, opts...)
}

func (m mockShopClient) LoginShop(ctx context.Context, in *shopv1.LoginShopRequest, opts ...grpc.CallOption) (*shopv1.LoginShopResponse, error) {
	return m.login(ctx, in, opts...)
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

	body := bytes.NewBufferString(`{"username":"merchant","password":"secret","shop_name":"merchant shop"}`)
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
		"shop_name":   "电子的软肥皂",
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
			AccessToken string `json:"access_token"`
			Shop        struct {
				ID int64 `json:"id"`
			} `json:"shop"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 || resp.Data.Shop.ID != 1001 || resp.Data.AccessToken == "" {
		t.Fatalf("unexpected response: %#v", resp)
	}

	claims, err := jwtManager.Verify(resp.Data.AccessToken)
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
	now := timestamppb.New(time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC))
	return &shopv1.Shop{
		Id:          1001,
		Username:    "merchant",
		ShopName:    "merchant shop",
		Status:      1,
		AuditStatus: 0,
		CreatedAt:   now,
		UpdatedAt:   now,
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
