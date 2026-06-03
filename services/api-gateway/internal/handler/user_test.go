package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	userv1 "github.com/yayccc/livebid/gen/proto/user/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockUserClient struct {
	register          func(ctx context.Context, in *userv1.RegisterUserRequest, opts ...grpc.CallOption) (*userv1.RegisterUserResponse, error)
	login             func(ctx context.Context, in *userv1.LoginUserRequest, opts ...grpc.CallOption) (*userv1.LoginUserResponse, error)
	get               func(ctx context.Context, in *userv1.GetUserRequest, opts ...grpc.CallOption) (*userv1.GetUserResponse, error)
	updateCurrent     func(ctx context.Context, in *userv1.UpdateCurrentUserRequest, opts ...grpc.CallOption) (*userv1.UpdateCurrentUserResponse, error)
	createAddress     func(ctx context.Context, in *userv1.CreateAddressRequest, opts ...grpc.CallOption) (*userv1.CreateAddressResponse, error)
	listAddresses     func(ctx context.Context, in *userv1.ListAddressesRequest, opts ...grpc.CallOption) (*userv1.ListAddressesResponse, error)
	getAddress        func(ctx context.Context, in *userv1.GetAddressRequest, opts ...grpc.CallOption) (*userv1.GetAddressResponse, error)
	updateAddress     func(ctx context.Context, in *userv1.UpdateAddressRequest, opts ...grpc.CallOption) (*userv1.UpdateAddressResponse, error)
	deleteAddress     func(ctx context.Context, in *userv1.DeleteAddressRequest, opts ...grpc.CallOption) (*userv1.DeleteAddressResponse, error)
	setDefaultAddress func(ctx context.Context, in *userv1.SetDefaultAddressRequest, opts ...grpc.CallOption) (*userv1.SetDefaultAddressResponse, error)
}

func (m mockUserClient) RegisterUser(ctx context.Context, in *userv1.RegisterUserRequest, opts ...grpc.CallOption) (*userv1.RegisterUserResponse, error) {
	return m.register(ctx, in, opts...)
}

func (m mockUserClient) LoginUser(ctx context.Context, in *userv1.LoginUserRequest, opts ...grpc.CallOption) (*userv1.LoginUserResponse, error) {
	return m.login(ctx, in, opts...)
}

func (m mockUserClient) GetUser(ctx context.Context, in *userv1.GetUserRequest, opts ...grpc.CallOption) (*userv1.GetUserResponse, error) {
	return m.get(ctx, in, opts...)
}

func (m mockUserClient) UpdateCurrentUser(ctx context.Context, in *userv1.UpdateCurrentUserRequest, opts ...grpc.CallOption) (*userv1.UpdateCurrentUserResponse, error) {
	return m.updateCurrent(ctx, in, opts...)
}

func (m mockUserClient) CreateAddress(ctx context.Context, in *userv1.CreateAddressRequest, opts ...grpc.CallOption) (*userv1.CreateAddressResponse, error) {
	return m.createAddress(ctx, in, opts...)
}

func (m mockUserClient) ListAddresses(ctx context.Context, in *userv1.ListAddressesRequest, opts ...grpc.CallOption) (*userv1.ListAddressesResponse, error) {
	return m.listAddresses(ctx, in, opts...)
}

func (m mockUserClient) GetAddress(ctx context.Context, in *userv1.GetAddressRequest, opts ...grpc.CallOption) (*userv1.GetAddressResponse, error) {
	return m.getAddress(ctx, in, opts...)
}

func (m mockUserClient) UpdateAddress(ctx context.Context, in *userv1.UpdateAddressRequest, opts ...grpc.CallOption) (*userv1.UpdateAddressResponse, error) {
	return m.updateAddress(ctx, in, opts...)
}

func (m mockUserClient) DeleteAddress(ctx context.Context, in *userv1.DeleteAddressRequest, opts ...grpc.CallOption) (*userv1.DeleteAddressResponse, error) {
	return m.deleteAddress(ctx, in, opts...)
}

func (m mockUserClient) SetDefaultAddress(ctx context.Context, in *userv1.SetDefaultAddressRequest, opts ...grpc.CallOption) (*userv1.SetDefaultAddressResponse, error) {
	return m.setDefaultAddress(ctx, in, opts...)
}

func TestUserHandlerRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewUserHandler(mockUserClient{
		register: func(ctx context.Context, in *userv1.RegisterUserRequest, opts ...grpc.CallOption) (*userv1.RegisterUserResponse, error) {
			if in.GetUsername() != "user001" || in.GetPassword() != "123456" || in.GetNickname() != "tester" {
				t.Fatalf("unexpected register request: %#v", in)
			}
			return &userv1.RegisterUserResponse{User: testUser()}, nil
		},
	}, time.Second)

	body := bytes.NewBufferString(`{"username":"user001","password":"123456","nickname":"tester"}`)
	w := performRequest(handler.Register, http.MethodPost, "/api/users/register", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			UserID int64 `json:"userId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.UserID != 2001 {
		t.Fatalf("unexpected user id: %d", resp.Data.UserID)
	}
}

func TestUserHandlerLoginPassesThroughToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewUserHandler(mockUserClient{
		login: func(ctx context.Context, in *userv1.LoginUserRequest, opts ...grpc.CallOption) (*userv1.LoginUserResponse, error) {
			if in.GetUsername() != "user001" || in.GetPassword() != "123456" {
				t.Fatalf("unexpected login request: %#v", in)
			}
			return &userv1.LoginUserResponse{
				User:        testUser(),
				AccessToken: "service-token",
				ExpiresIn:   3600,
			}, nil
		},
	}, time.Second)

	body := bytes.NewBufferString(`{"username":"user001","password":"123456"}`)
	w := performRequest(handler.Login, http.MethodPost, "/api/users/login", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Token     string `json:"token"`
			ExpiresIn int64  `json:"expires_in"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.Token != "service-token" || resp.Data.ExpiresIn != 3600 {
		t.Fatalf("unexpected login response: %#v", resp.Data)
	}
}

func TestUserHandlerLoginMapsUnauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewUserHandler(mockUserClient{
		login: func(ctx context.Context, in *userv1.LoginUserRequest, opts ...grpc.CallOption) (*userv1.LoginUserResponse, error) {
			return nil, status.Error(codes.Unauthenticated, "invalid credential")
		},
	}, time.Second)

	body := bytes.NewBufferString(`{"username":"user001","password":"bad"}`)
	w := performRequest(handler.Login, http.MethodPost, "/api/users/login", body)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandlerUpdateCurrentForwardsIdentityContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewUserHandler(mockUserClient{
		updateCurrent: func(ctx context.Context, in *userv1.UpdateCurrentUserRequest, opts ...grpc.CallOption) (*userv1.UpdateCurrentUserResponse, error) {
			if userID, ok := identity.UserID(ctx); !ok || userID != 2001 {
				t.Fatalf("missing identity context: userID=%d ok=%v", userID, ok)
			}
			if in.GetNickname() != "new name" || in.GetGender() != 1 {
				t.Fatalf("unexpected update request: %#v", in)
			}
			return &userv1.UpdateCurrentUserResponse{User: testUser()}, nil
		},
	}, time.Second)

	body := bytes.NewBufferString(`{"nickname":"new name","gender":1}`)
	w := performRequestWithUserID(handler.UpdateCurrent, http.MethodPut, "/api/users/2001", body, 2001)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandlerUpdateCurrentRejectsMismatchedPathID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewUserHandler(mockUserClient{
		updateCurrent: func(ctx context.Context, in *userv1.UpdateCurrentUserRequest, opts ...grpc.CallOption) (*userv1.UpdateCurrentUserResponse, error) {
			t.Fatal("UpdateCurrentUser should not be called")
			return nil, nil
		},
	}, time.Second)

	body := bytes.NewBufferString(`{"nickname":"new name"}`)
	w := performRequestWithIDParamAndUserID(handler.UpdateCurrent, http.MethodPut, "/api/users/9999", body, "9999", 2001)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandlerCreateAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewUserHandler(mockUserClient{
		createAddress: func(ctx context.Context, in *userv1.CreateAddressRequest, opts ...grpc.CallOption) (*userv1.CreateAddressResponse, error) {
			if userID, ok := identity.UserID(ctx); !ok || userID != 2001 {
				t.Fatalf("missing identity context: userID=%d ok=%v", userID, ok)
			}
			if in.GetReceiverName() != "张三" || in.GetDetailAddress() != "科技园1001室" || in.GetIsDefault() != 1 {
				t.Fatalf("unexpected create address request: %#v", in)
			}
			return &userv1.CreateAddressResponse{Address: testAddress()}, nil
		},
	}, time.Second)

	body := bytes.NewBufferString(`{"receiverName":"张三","receiverPhone":"13800000000","province":"广东省","city":"深圳市","district":"南山区","detailAddress":"科技园1001室","isDefault":1}`)
	w := performRequestWithUserID(handler.CreateAddress, http.MethodPost, "/api/users/address", body, 2001)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandlerListAddresses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewUserHandler(mockUserClient{
		listAddresses: func(ctx context.Context, in *userv1.ListAddressesRequest, opts ...grpc.CallOption) (*userv1.ListAddressesResponse, error) {
			if in.GetPage() != 2 || in.GetPageSize() != 20 {
				t.Fatalf("unexpected list addresses request: %#v", in)
			}
			return &userv1.ListAddressesResponse{
				Total:    1,
				Page:     2,
				PageSize: 20,
				List:     []*userv1.UserAddress{testAddress()},
			}, nil
		},
	}, time.Second)

	w := performRequestWithUserID(handler.ListAddresses, http.MethodGet, "/api/users/address/list?page=2&page_size=20", bytes.NewBuffer(nil), 2001)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func performRequestWithUserID(handlerFunc gin.HandlerFunc, method string, target string, body *bytes.Buffer, userID int64) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, body)
	c.Request.Header.Set("Content-Type", "application/json")
	if idParam := trailingIDParam(target); idParam != "" {
		c.Params = gin.Params{{Key: "id", Value: idParam}}
	}
	if userID > 0 {
		principal := identity.Principal{
			Kind: identity.KindUser,
			ID:   userID,
		}
		c.Request = c.Request.WithContext(identity.NewContext(c.Request.Context(), principal))
	}
	handlerFunc(c)
	return w
}

func trailingIDParam(target string) string {
	path, _, _ := strings.Cut(target, "?")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	for _, ch := range last {
		if ch < '0' || ch > '9' {
			return ""
		}
	}
	return last
}

func performRequestWithIDParamAndUserID(handlerFunc gin.HandlerFunc, method string, target string, body *bytes.Buffer, idParam string, userID int64) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: idParam}}
	c.Request = httptest.NewRequest(method, target, body)
	c.Request.Header.Set("Content-Type", "application/json")
	if userID > 0 {
		principal := identity.Principal{
			Kind: identity.KindUser,
			ID:   userID,
		}
		c.Request = c.Request.WithContext(identity.NewContext(c.Request.Context(), principal))
	}
	handlerFunc(c)
	return w
}

func testUser() *userv1.User {
	return &userv1.User{
		Id:       2001,
		Username: "user001",
		Nickname: "tester",
		Status:   1,
	}
}

func testAddress() *userv1.UserAddress {
	return &userv1.UserAddress{
		Id:            3001,
		UserId:        2001,
		ReceiverName:  "张三",
		ReceiverPhone: "13800000000",
		Province:      "广东省",
		City:          "深圳市",
		District:      "南山区",
		DetailAddress: "科技园1001室",
		IsDefault:     1,
	}
}
