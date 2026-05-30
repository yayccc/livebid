package handler

import (
	"context"
	"errors"
	"strings"

	goodsv1 "github.com/yayccc/livebid/gen/proto/goods/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"github.com/yayccc/livebid/pkg/idgen"
	"github.com/yayccc/livebid/services/goods-service/internal/model"
	"github.com/yayccc/livebid/services/goods-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	maxTitleLength    = 255
	maxCoverURLLength = 500
)

var (
	errInvalidArgument   = errors.New("invalid argument")
	errInvalidCredential = errors.New("invalid credential")
)

type GoodsGRPCHandler struct {
	goodsv1.UnimplementedGoodsServiceServer
	goods repository.GoodsRepository
	ids   *idgen.Generator
}

func NewGoodsGRPCHandler(goods repository.GoodsRepository, ids *idgen.Generator) *GoodsGRPCHandler {
	if ids == nil {
		ids = idgen.New(0)
	}
	return &GoodsGRPCHandler{
		goods: goods,
		ids:   ids,
	}
}

func (h *GoodsGRPCHandler) CreateGoods(ctx context.Context, req *goodsv1.CreateGoodsRequest) (*goodsv1.CreateGoodsResponse, error) {
	shopID, err := currentShopID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	title := strings.TrimSpace(req.GetTitle())
	coverURL := strings.TrimSpace(req.GetCoverUrl())
	if !validTitle(title) || !validCoverURL(coverURL) {
		return nil, toGRPCError(errInvalidArgument)
	}

	goods := &model.Goods{
		ID:          h.ids.Next(),
		ShopID:      shopID,
		Title:       title,
		CoverURL:    coverURL,
		Description: strings.TrimSpace(req.GetDescription()),
		// 新建商品默认下架，避免未完善资料的商品直接进入展示或竞拍流程。
		Status: model.GoodsStatusOffSale,
	}
	if err := h.goods.Create(ctx, goods); err != nil {
		return nil, toGRPCError(err)
	}
	return &goodsv1.CreateGoodsResponse{Goods: toProtoGoods(goods)}, nil
}

func (h *GoodsGRPCHandler) UpdateGoods(ctx context.Context, req *goodsv1.UpdateGoodsRequest) (*goodsv1.UpdateGoodsResponse, error) {
	shopID, err := currentShopID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	goods, err := h.goods.FindByIDForShop(ctx, req.GetId(), shopID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if !validTitle(title) {
			return nil, toGRPCError(errInvalidArgument)
		}
		goods.Title = title
	}
	if req.CoverUrl != nil {
		coverURL := strings.TrimSpace(*req.CoverUrl)
		if !validCoverURL(coverURL) {
			return nil, toGRPCError(errInvalidArgument)
		}
		goods.CoverURL = coverURL
	}
	if req.Description != nil {
		goods.Description = strings.TrimSpace(*req.Description)
	}

	if err := h.goods.Update(ctx, goods); err != nil {
		return nil, toGRPCError(err)
	}
	updated, err := h.goods.FindByIDForShop(ctx, req.GetId(), shopID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &goodsv1.UpdateGoodsResponse{Goods: toProtoGoods(updated)}, nil
}

func (h *GoodsGRPCHandler) DeleteGoods(ctx context.Context, req *goodsv1.DeleteGoodsRequest) (*goodsv1.DeleteGoodsResponse, error) {
	shopID, err := currentShopID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	if err := h.goods.Delete(ctx, req.GetId(), shopID); err != nil {
		return nil, toGRPCError(err)
	}
	return &goodsv1.DeleteGoodsResponse{}, nil
}

func (h *GoodsGRPCHandler) GetGoods(ctx context.Context, req *goodsv1.GetGoodsRequest) (*goodsv1.GetGoodsResponse, error) {
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	goods, err := h.goods.FindByID(ctx, req.GetId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &goodsv1.GetGoodsResponse{Goods: toProtoGoods(goods)}, nil
}

func (h *GoodsGRPCHandler) ListGoods(ctx context.Context, req *goodsv1.ListGoodsRequest) (*goodsv1.ListGoodsResponse, error) {
	filter, page, pageSize, err := listGoodsFilter(req)
	if err != nil {
		return nil, toGRPCError(err)
	}
	list, total, err := h.goods.List(ctx, filter)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &goodsv1.ListGoodsResponse{
		Total:    total,
		Page:     int32(page),
		PageSize: int32(pageSize),
		List:     toProtoGoodsList(list),
	}, nil
}

func (h *GoodsGRPCHandler) ListShopGoods(ctx context.Context, req *goodsv1.ListShopGoodsRequest) (*goodsv1.ListShopGoodsResponse, error) {
	shopID, err := currentShopID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	page, pageSize := normalizePagination(int(req.GetPage()), int(req.GetPageSize()))
	filter := repository.ListGoodsFilter{
		Page:     page,
		PageSize: pageSize,
		Keyword:  strings.TrimSpace(req.GetKeyword()),
		ShopID:   &shopID,
	}
	list, total, err := h.goods.List(ctx, filter)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &goodsv1.ListShopGoodsResponse{
		Total:    total,
		Page:     int32(page),
		PageSize: int32(pageSize),
		List:     toProtoGoodsList(list),
	}, nil
}

func (h *GoodsGRPCHandler) BatchGetGoods(ctx context.Context, req *goodsv1.BatchGetGoodsRequest) (*goodsv1.BatchGetGoodsResponse, error) {
	ids, err := normalizeIDs(req.GetIds())
	if err != nil {
		return nil, toGRPCError(err)
	}
	list, err := h.goods.BatchFindByIDs(ctx, ids)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &goodsv1.BatchGetGoodsResponse{List: orderGoodsByIDs(ids, list)}, nil
}

func (h *GoodsGRPCHandler) PutGoodsOnSale(ctx context.Context, req *goodsv1.PutGoodsOnSaleRequest) (*goodsv1.PutGoodsOnSaleResponse, error) {
	shopID, err := currentShopID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	if err := h.goods.UpdateStatus(ctx, req.GetId(), shopID, model.GoodsStatusOnSale); err != nil {
		return nil, toGRPCError(err)
	}
	return &goodsv1.PutGoodsOnSaleResponse{}, nil
}

func (h *GoodsGRPCHandler) PutGoodsOffSale(ctx context.Context, req *goodsv1.PutGoodsOffSaleRequest) (*goodsv1.PutGoodsOffSaleResponse, error) {
	shopID, err := currentShopID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	if err := h.goods.UpdateStatus(ctx, req.GetId(), shopID, model.GoodsStatusOffSale); err != nil {
		return nil, toGRPCError(err)
	}
	return &goodsv1.PutGoodsOffSaleResponse{}, nil
}

func listGoodsFilter(req *goodsv1.ListGoodsRequest) (repository.ListGoodsFilter, int, int, error) {
	page, pageSize := normalizePagination(int(req.GetPage()), int(req.GetPageSize()))
	filter := repository.ListGoodsFilter{
		Page:     page,
		PageSize: pageSize,
		Keyword:  strings.TrimSpace(req.GetKeyword()),
	}
	if req.ShopId != nil {
		if req.GetShopId() <= 0 {
			return filter, page, pageSize, errInvalidArgument
		}
		shopID := req.GetShopId()
		filter.ShopID = &shopID
	}
	if req.IsDeleted != nil {
		isDeleted, err := parseIsDeleted(req.GetIsDeleted())
		if err != nil {
			return filter, page, pageSize, err
		}
		filter.IsDeleted = &isDeleted
	}
	if req.Status != nil {
		goodsStatus, err := parseGoodsStatus(req.GetStatus())
		if err != nil {
			return filter, page, pageSize, err
		}
		filter.Status = &goodsStatus
	}
	return filter, page, pageSize, nil
}

func currentShopID(ctx context.Context) (int64, error) {
	shopID, ok := identity.ShopID(ctx)
	if !ok {
		return 0, errInvalidCredential
	}
	return shopID, nil
}

func parseIsDeleted(value int32) (bool, error) {
	switch value {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, errInvalidArgument
	}
}

func parseGoodsStatus(value int32) (model.GoodsStatus, error) {
	switch model.GoodsStatus(value) {
	case model.GoodsStatusOffSale, model.GoodsStatusOnSale:
		return model.GoodsStatus(value), nil
	default:
		return model.GoodsStatusOffSale, errInvalidArgument
	}
}

func normalizePagination(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func normalizeIDs(ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return nil, errInvalidArgument
	}
	seen := make(map[int64]struct{}, len(ids))
	normalized := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, errInvalidArgument
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized, nil
}

func orderGoodsByIDs(ids []int64, list []*model.Goods) []*goodsv1.Goods {
	byID := make(map[int64]*model.Goods, len(list))
	for _, goods := range list {
		byID[goods.ID] = goods
	}
	ordered := make([]*goodsv1.Goods, 0, len(list))
	for _, id := range ids {
		if goods := byID[id]; goods != nil {
			ordered = append(ordered, toProtoGoods(goods))
		}
	}
	return ordered
}

func validTitle(title string) bool {
	return title != "" && len([]rune(title)) <= maxTitleLength
}

func validCoverURL(coverURL string) bool {
	return len([]rune(coverURL)) <= maxCoverURLLength
}

func toProtoGoods(goods *model.Goods) *goodsv1.Goods {
	if goods == nil {
		return nil
	}
	return &goodsv1.Goods{
		Id:          goods.ID,
		ShopId:      goods.ShopID,
		Title:       goods.Title,
		CoverUrl:    goods.CoverURL,
		Description: goods.Description,
		Status:      int32(goods.Status),
		CreatedAt:   timestamppb.New(goods.CreatedAt),
		UpdatedAt:   timestamppb.New(goods.UpdatedAt),
	}
}

func toProtoGoodsList(list []*model.Goods) []*goodsv1.Goods {
	result := make([]*goodsv1.Goods, 0, len(list))
	for _, goods := range list {
		result = append(result, toProtoGoods(goods))
	}
	return result
}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, errInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, errInvalidCredential):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, repository.ErrGoodsNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
