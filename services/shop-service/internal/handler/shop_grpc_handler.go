package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	shopv1 "github.com/yayccc/livebid/gen/proto/shop/v1"
	"github.com/yayccc/livebid/pkg/identity"
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
	errInvalidRegisterArgument = errors.New("注册参数无效：username、password、shopName 不能为空")
	errInvalidLoginArgument    = errors.New("登录参数无效：username 和 password 不能为空")
	errInvalidShopID           = errors.New("商铺ID无效，请传入大于0的商铺ID")
	errInvalidCredential       = errors.New("账号或密码错误")
	errMissingShopIdentity     = errors.New("未获取到商铺身份，请先登录")
	errPermissionDenied        = errors.New("无权修改其他商铺信息")
	errShopDuplicated          = errors.New("商铺账号或商铺名称已存在")
	errEmptyShopName           = errors.New("商铺名称不能为空")
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
		return nil, toGRPCError(errInvalidRegisterArgument)
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
		return nil, toGRPCError(errInvalidLoginArgument)
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
	shopID := req.GetId()
	if shopID <= 0 {
		var err error
		shopID, err = currentShopID(ctx)
		if err != nil {
			return nil, toGRPCError(err)
		}
	}
	shop, err := h.shops.FindByID(ctx, shopID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &shopv1.GetShopResponse{Shop: toProtoShop(shop)}, nil
}

func (h *ShopGRPCHandler) BatchGetPublicShops(ctx context.Context, req *shopv1.BatchGetPublicShopsRequest) (*shopv1.BatchGetPublicShopsResponse, error) {
	ids, err := normalizeIDs(req.GetIds())
	if err != nil {
		return nil, toGRPCError(err)
	}
	list, err := h.shops.BatchFindByIDs(ctx, ids)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &shopv1.BatchGetPublicShopsResponse{List: orderShopsByIDs(ids, list)}, nil
}

func (h *ShopGRPCHandler) UpdateShop(ctx context.Context, req *shopv1.UpdateShopRequest) (*shopv1.UpdateShopResponse, error) {
	shopID := req.GetId()
	if shopID <= 0 {
		return nil, toGRPCError(errInvalidShopID)
	}
	currentID, err := currentShopID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if shopID != currentID {
		return nil, toGRPCError(errPermissionDenied)
	}
	shop, err := h.shops.FindByID(ctx, shopID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	if req.ShopName != nil {
		value := strings.TrimSpace(*req.ShopName)
		if value == "" {
			return nil, toGRPCError(errEmptyShopName)
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

func currentShopID(ctx context.Context) (int64, error) {
	if shopID, ok := identity.ShopID(ctx); ok {
		return shopID, nil
	}
	principal, ok := identity.FromIncomingContext(ctx)
	if ok && principal.Kind == identity.KindShop {
		return principal.ID, nil
	}
	return 0, errMissingShopIdentity
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

func normalizeIDs(ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return nil, errInvalidShopID
	}
	seen := make(map[int64]struct{}, len(ids))
	normalized := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, errInvalidShopID
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized, nil
}

func orderShopsByIDs(ids []int64, list []*model.Shop) []*shopv1.Shop {
	byID := make(map[int64]*model.Shop, len(list))
	for _, shop := range list {
		if shop != nil && shop.ID > 0 {
			byID[shop.ID] = shop
		}
	}
	ordered := make([]*shopv1.Shop, 0, len(list))
	for _, id := range ids {
		if shop := byID[id]; shop != nil {
			ordered = append(ordered, toProtoShop(shop))
		}
	}
	return ordered
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
	case errors.Is(err, errInvalidRegisterArgument),
		errors.Is(err, errInvalidLoginArgument),
		errors.Is(err, errInvalidShopID),
		errors.Is(err, errEmptyShopName):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, errInvalidCredential):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, errMissingShopIdentity):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, errPermissionDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, repository.ErrShopNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, errShopDuplicated):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "商铺服务内部错误，请稍后重试")
	}
}
