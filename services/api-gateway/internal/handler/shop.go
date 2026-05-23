package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	shopv1 "github.com/yayccc/livebid/gen/proto/shop/v1"
	"github.com/yayccc/livebid/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type shopServiceClient interface {
	RegisterShop(ctx context.Context, in *shopv1.RegisterShopRequest, opts ...grpc.CallOption) (*shopv1.RegisterShopResponse, error)
	LoginShop(ctx context.Context, in *shopv1.LoginShopRequest, opts ...grpc.CallOption) (*shopv1.LoginShopResponse, error)
}

type ShopHandler struct {
	shopClient     shopServiceClient
	jwt            *auth.JWTManager
	accessTokenTTL time.Duration
	rpcTimeout     time.Duration
}

func NewShopHandler(shopClient shopServiceClient, jwtManager *auth.JWTManager, accessTokenTTL time.Duration, rpcTimeout time.Duration) *ShopHandler {
	if accessTokenTTL <= 0 {
		accessTokenTTL = 30 * time.Minute
	}
	if rpcTimeout <= 0 {
		rpcTimeout = 3 * time.Second
	}
	return &ShopHandler{
		shopClient:     shopClient,
		jwt:            jwtManager,
		accessTokenTTL: accessTokenTTL,
		rpcTimeout:     rpcTimeout,
	}
}

type registerShopRequest struct {
	Username    string `json:"username" form:"username" binding:"required"`
	Password    string `json:"password" form:"password" binding:"required"`
	ShopName    string `json:"shop_name" form:"shop_name" binding:"required"`
	Logo        string `json:"logo" form:"logo"`
	Description string `json:"description" form:"description"`
	Phone       string `json:"phone" form:"phone"`
	Email       string `json:"email" form:"email"`
}

type loginShopRequest struct {
	Username string `json:"username" form:"username" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
}

type shopResponse struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	ShopName    string `json:"shop_name"`
	Logo        string `json:"logo,omitempty"`
	Description string `json:"description,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
	Status      int32  `json:"status"`
	AuditStatus int32  `json:"audit_status"`
	AuditReason string `json:"audit_reason,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type loginShopResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	ExpiresAt   string       `json:"expires_at"`
	Shop        shopResponse `json:"shop"`
}

type apiResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (h *ShopHandler) Register(c *gin.Context) {
	var req registerShopRequest
	if err := c.ShouldBind(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "invalid request")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.shopClient.RegisterShop(ctx, &shopv1.RegisterShopRequest{
		Username:    req.Username,
		Password:    req.Password,
		ShopName:    req.ShopName,
		Logo:        req.Logo,
		Description: req.Description,
		Phone:       req.Phone,
		Email:       req.Email,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}

	respondOK(c, gin.H{
		"shop": toShopResponse(resp.GetShop()),
	})
}

func (h *ShopHandler) Login(c *gin.Context) {
	var req loginShopRequest
	if err := c.ShouldBind(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "invalid request")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.shopClient.LoginShop(ctx, &shopv1.LoginShopRequest{
		Username:  req.Username,
		Password:  req.Password,
		LoginIp:   c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}

	shop := resp.GetShop()
	if shop == nil || shop.GetId() <= 0 {
		recordRequestError(c, errors.New("invalid shop service response"))
		respondError(c, http.StatusBadGateway, "invalid shop service response")
		return
	}

	now := time.Now().UTC()
	expiresAt := now.Add(h.accessTokenTTL)
	token, err := h.jwt.Sign(auth.Claims{
		Subject:   strconv.FormatInt(shop.GetId(), 10),
		UserType:  auth.SubjectTypeShop,
		ShopID:    strconv.FormatInt(shop.GetId(), 10),
		IssuedAt:  now,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusInternalServerError, "failed to sign token")
		return
	}

	respondOK(c, loginShopResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(h.accessTokenTTL.Seconds()),
		ExpiresAt:   expiresAt.Format(time.RFC3339),
		Shop:        toShopResponse(shop),
	})
}

func toShopResponse(shop *shopv1.Shop) shopResponse {
	if shop == nil {
		return shopResponse{}
	}
	return shopResponse{
		ID:          shop.GetId(),
		Username:    shop.GetUsername(),
		ShopName:    shop.GetShopName(),
		Logo:        shop.GetLogo(),
		Description: shop.GetDescription(),
		Phone:       shop.GetPhone(),
		Email:       shop.GetEmail(),
		Status:      shop.GetStatus(),
		AuditStatus: shop.GetAuditStatus(),
		AuditReason: shop.GetAuditReason(),
		CreatedAt:   timestampString(shop.GetCreatedAt()),
		UpdatedAt:   timestampString(shop.GetUpdatedAt()),
	}
}

func timestampString(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().UTC().Format(time.RFC3339)
}

func respondOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, apiResponse{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func respondError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, apiResponse{
		Code:    statusCode,
		Message: message,
	})
}

func respondGRPCError(c *gin.Context, err error) {
	recordRequestError(c, err)
	code := status.Code(err)
	switch code {
	case codes.InvalidArgument:
		respondError(c, http.StatusBadRequest, "invalid request")
	case codes.Unauthenticated:
		respondError(c, http.StatusUnauthorized, "invalid credential")
	case codes.AlreadyExists:
		respondError(c, http.StatusConflict, "shop already exists")
	case codes.NotFound:
		respondError(c, http.StatusNotFound, "resource not found")
	case codes.DeadlineExceeded:
		respondError(c, http.StatusGatewayTimeout, "upstream timeout")
	case codes.Unavailable:
		respondError(c, http.StatusBadGateway, "upstream unavailable")
	default:
		if errors.Is(err, context.DeadlineExceeded) {
			respondError(c, http.StatusGatewayTimeout, "upstream timeout")
			return
		}
		respondError(c, http.StatusInternalServerError, "internal error")
	}
}

func recordRequestError(c *gin.Context, err error) {
	if c == nil || err == nil {
		return
	}
	_ = c.Error(err)
}
