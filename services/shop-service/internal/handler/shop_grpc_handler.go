package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	shopv1 "github.com/yayccc/livebid/gen/proto/shop/v1"
	"github.com/yayccc/livebid/pkg/idgen"
	"github.com/yayccc/livebid/services/shop-service/internal/model"
	"github.com/yayccc/livebid/services/shop-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

var (
	errInvalidArgument   = errors.New("invalid argument")
	errInvalidCredential = errors.New("invalid credential")
	errShopDuplicated    = errors.New("shop duplicated")
)

type ShopGRPCHandler struct {
	shopv1.UnimplementedShopServiceServer
	shops     repository.ShopRepository
	loginLogs repository.ShopLoginLogRepository
	ids       *idgen.Generator
}

func NewShopGRPCHandler(shops repository.ShopRepository, loginLogs repository.ShopLoginLogRepository, ids *idgen.Generator) *ShopGRPCHandler {
	if ids == nil {
		ids = idgen.New(0)
	}
	return &ShopGRPCHandler{
		shops:     shops,
		loginLogs: loginLogs,
		ids:       ids,
	}
}

func (h *ShopGRPCHandler) RegisterShop(ctx context.Context, req *shopv1.RegisterShopRequest) (*shopv1.RegisterShopResponse, error) {
	username := strings.TrimSpace(req.GetUsername())
	password := req.GetPassword()
	shopName := strings.TrimSpace(req.GetShopName())
	email := strings.TrimSpace(req.GetEmail())
	phone := strings.TrimSpace(req.GetPhone())

	if username == "" || password == "" || shopName == "" {
		return nil, toGRPCError(errInvalidArgument)
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, toGRPCError(err)
	}

	shop := &model.Shop{
		ID:           h.ids.Next(),
		Username:     username,
		PasswordHash: string(passwordHash),
		ShopName:     shopName,
		Logo:         strings.TrimSpace(req.GetLogo()),
		Description:  strings.TrimSpace(req.GetDescription()),
		Phone:        phone,
		Email:        email,
		Status:       model.ShopStatusEnabled,
		AuditStatus:  model.AuditStatusPending,
		Extra:        datatypes.JSON([]byte("{}")),
	}
	if err := h.shops.Create(ctx, shop); err != nil {
		if errors.Is(err, repository.ErrUsernameDuplicated) || errors.Is(err, repository.ErrShopNameDuplicated) {
			return nil, toGRPCError(errShopDuplicated)
		}
		return nil, toGRPCError(err)
	}
	return &shopv1.RegisterShopResponse{Shop: toProtoShop(shop)}, nil
}

func (h *ShopGRPCHandler) LoginShop(ctx context.Context, req *shopv1.LoginShopRequest) (*shopv1.LoginShopResponse, error) {
	username := strings.TrimSpace(req.GetUsername())
	password := req.GetPassword()
	if username == "" || password == "" {
		return nil, toGRPCError(errInvalidArgument)
	}

	shop, err := h.shops.FindByUsername(ctx, username)
	if err != nil {
		_ = h.recordLogin(ctx, 0, req.GetLoginIp(), req.GetUserAgent(), 2)
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, toGRPCError(errInvalidCredential)
		}
		return nil, toGRPCError(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(shop.PasswordHash), []byte(password)); err != nil {
		_ = h.recordLogin(ctx, shop.ID, req.GetLoginIp(), req.GetUserAgent(), 2)
		return nil, toGRPCError(errInvalidCredential)
	}

	token, err := newOpaqueToken()
	if err != nil {
		return nil, toGRPCError(err)
	}
	if err := h.recordLogin(ctx, shop.ID, req.GetLoginIp(), req.GetUserAgent(), 1); err != nil {
		return nil, toGRPCError(err)
	}
	return &shopv1.LoginShopResponse{
		Shop:  toProtoShop(shop),
		Token: token,
	}, nil
}

func (h *ShopGRPCHandler) GetShop(ctx context.Context, req *shopv1.GetShopRequest) (*shopv1.GetShopResponse, error) {
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	shop, err := h.shops.FindByID(ctx, req.GetId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &shopv1.GetShopResponse{Shop: toProtoShop(shop)}, nil
}

func (h *ShopGRPCHandler) UpdateShop(ctx context.Context, req *shopv1.UpdateShopRequest) (*shopv1.UpdateShopResponse, error) {
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	shop, err := h.shops.FindByID(ctx, req.GetId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	if req.ShopName != nil {
		value := strings.TrimSpace(*req.ShopName)
		if value == "" {
			return nil, toGRPCError(errInvalidArgument)
		}
		shop.ShopName = value
	}
	if req.Logo != nil {
		shop.Logo = strings.TrimSpace(*req.Logo)
	}
	if req.Description != nil {
		shop.Description = strings.TrimSpace(*req.Description)
	}
	if req.Phone != nil {
		shop.Phone = strings.TrimSpace(*req.Phone)
	}
	if req.Email != nil {
		shop.Email = strings.TrimSpace(*req.Email)
	}

	if err := h.shops.Update(ctx, shop); err != nil {
		if errors.Is(err, repository.ErrShopNameDuplicated) {
			return nil, toGRPCError(errShopDuplicated)
		}
		return nil, toGRPCError(err)
	}
	return &shopv1.UpdateShopResponse{Shop: toProtoShop(shop)}, nil
}

func (h *ShopGRPCHandler) recordLogin(ctx context.Context, shopID int64, loginIP string, userAgent string, result int8) error {
	if h.loginLogs == nil {
		return nil
	}
	return h.loginLogs.Create(ctx, &model.ShopLoginLog{
		ID:          h.ids.Next(),
		ShopID:      shopID,
		LoginIP:     loginIP,
		UserAgent:   userAgent,
		LoginTime:   time.Now(),
		LoginResult: result,
		Extra:       datatypes.JSON([]byte("{}")),
	})
}

func newOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func toProtoShop(shop *model.Shop) *shopv1.Shop {
	if shop == nil {
		return nil
	}
	return &shopv1.Shop{
		Id:          shop.ID,
		Username:    shop.Username,
		ShopName:    shop.ShopName,
		Logo:        shop.Logo,
		Description: shop.Description,
		Phone:       shop.Phone,
		Email:       shop.Email,
		Status:      int32(shop.Status),
		AuditStatus: int32(shop.AuditStatus),
		AuditReason: shop.AuditReason,
		CreatedAt:   timestamppb.New(shop.CreatedAt),
		UpdatedAt:   timestamppb.New(shop.UpdatedAt),
	}
}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, errInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, errInvalidCredential):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, repository.ErrShopNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, errShopDuplicated):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
