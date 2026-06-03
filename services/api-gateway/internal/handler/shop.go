package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	shopv1 "github.com/yayccc/livebid/gen/proto/shop/v1"
	"github.com/yayccc/livebid/pkg/auth"
)

type ShopHandler struct {
	shopClient     shopv1.ShopServiceClient
	jwt            *auth.JWTManager
	accessTokenTTL time.Duration
	rpcTimeout     time.Duration
}

func NewShopHandler(shopClient shopv1.ShopServiceClient, jwtManager *auth.JWTManager, accessTokenTTL time.Duration, rpcTimeout time.Duration) *ShopHandler {
	if accessTokenTTL <= 0 {
		accessTokenTTL = 7 * 24 * time.Hour
	}
	return &ShopHandler{
		shopClient:     shopClient,
		jwt:            jwtManager,
		accessTokenTTL: accessTokenTTL,
		rpcTimeout:     normalizeRPCTimeout(rpcTimeout),
	}
}

type registerShopRequest struct {
	Username    string `json:"username" form:"username" binding:"required"`
	Password    string `json:"password" form:"password" binding:"required"`
	ShopName    string `json:"shopName" form:"shopName" binding:"required"`
	Logo        string `json:"logo" form:"logo"`
	Description string `json:"description" form:"description"`
	Phone       string `json:"phone" form:"phone"`
	Email       string `json:"email" form:"email"`
}

type loginShopRequest struct {
	Username string `json:"username" form:"username" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
}

type updateShopRequest struct {
	ShopName    *string `json:"shopName"`
	Logo        *string `json:"logo"`
	Description *string `json:"description"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
}

type shopResponse struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	ShopName    string `json:"shopName"`
	Logo        string `json:"logo"`
	Description string `json:"description"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
}

type loginShopResponse struct {
	Token string `json:"token"`
}

func (h *ShopHandler) Register(c *gin.Context) {
	var req registerShopRequest
	if err := c.ShouldBind(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "注册参数无效：username、password、shopName 为必填项")
		return
	}
	shopName := strings.TrimSpace(req.ShopName)
	if shopName == "" {
		respondError(c, http.StatusBadRequest, "注册参数无效：shopName 不能为空")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.shopClient.RegisterShop(ctx, &shopv1.RegisterShopRequest{
		Username:    req.Username,
		Password:    req.Password,
		ShopName:    shopName,
		Logo:        req.Logo,
		Description: req.Description,
		Phone:       req.Phone,
		Email:       req.Email,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}

	respondOKWithMessage(c, "注册成功", gin.H{
		"shopId": resp.GetShop().GetId(),
	})
}

func (h *ShopHandler) Login(c *gin.Context) {
	var req loginShopRequest
	if err := c.ShouldBind(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "登录参数无效：username 和 password 为必填项")
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
		recordRequestError(c, errors.New("shop service returned empty shop"))
		respondError(c, http.StatusBadGateway, "商铺服务返回数据异常，请稍后重试")
		return
	}

	now := time.Now().UTC()
	expiresAt := now.Add(h.accessTokenTTL)
	token, err := h.jwt.Sign(auth.Claims{
		Subject:   strconv.FormatInt(shop.GetId(), 10),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusInternalServerError, "登录成功但生成访问凭证失败，请稍后重试")
		return
	}

	respondOKWithMessage(c, "登录成功", loginShopResponse{
		Token: token,
	})
}

func (h *ShopHandler) Get(c *gin.Context) {
	if isCurrentShopParam(c.Param("id")) {
		h.GetCurrent(c)
		return
	}
	shopID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	h.getByID(c, shopID)
}

func (h *ShopHandler) GetCurrent(c *gin.Context) {
	h.getByID(c, 0)
}

func (h *ShopHandler) getByID(c *gin.Context, shopID int64) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.shopClient.GetShop(ctx, &shopv1.GetShopRequest{Id: shopID})
	if err != nil {
		respondGRPCError(c, err)
		return
	}

	respondOKWithMessage(c, "查询成功", toShopResponse(resp.GetShop()))
}

func (h *ShopHandler) Update(c *gin.Context) {
	shopID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	h.updateByID(c, shopID)
}

func (h *ShopHandler) updateByID(c *gin.Context, shopID int64) {
	var req updateShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "修改商铺参数无效，请提交合法的 JSON 请求体")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.shopClient.UpdateShop(ctx, &shopv1.UpdateShopRequest{
		Id:          shopID,
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

	respondOKWithMessage(c, "修改成功", toShopResponse(resp.GetShop()))
}

func isCurrentShopParam(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "me")
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
	}
}
