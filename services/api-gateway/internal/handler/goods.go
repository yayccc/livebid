package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	goodsv1 "github.com/yayccc/livebid/gen/proto/goods/v1"
	"google.golang.org/grpc"
)

type GoodsHandler struct {
	goodsClient goodsServiceClient
	rpcTimeout  time.Duration
}

type goodsServiceClient interface {
	CreateGoods(ctx context.Context, in *goodsv1.CreateGoodsRequest, opts ...grpc.CallOption) (*goodsv1.CreateGoodsResponse, error)
	UpdateGoods(ctx context.Context, in *goodsv1.UpdateGoodsRequest, opts ...grpc.CallOption) (*goodsv1.UpdateGoodsResponse, error)
	DeleteGoods(ctx context.Context, in *goodsv1.DeleteGoodsRequest, opts ...grpc.CallOption) (*goodsv1.DeleteGoodsResponse, error)
	GetGoods(ctx context.Context, in *goodsv1.GetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.GetGoodsResponse, error)
	ListGoods(ctx context.Context, in *goodsv1.ListGoodsRequest, opts ...grpc.CallOption) (*goodsv1.ListGoodsResponse, error)
	ListShopGoods(ctx context.Context, in *goodsv1.ListShopGoodsRequest, opts ...grpc.CallOption) (*goodsv1.ListShopGoodsResponse, error)
	BatchGetGoods(ctx context.Context, in *goodsv1.BatchGetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.BatchGetGoodsResponse, error)
	PutGoodsOnSale(ctx context.Context, in *goodsv1.PutGoodsOnSaleRequest, opts ...grpc.CallOption) (*goodsv1.PutGoodsOnSaleResponse, error)
	PutGoodsOffSale(ctx context.Context, in *goodsv1.PutGoodsOffSaleRequest, opts ...grpc.CallOption) (*goodsv1.PutGoodsOffSaleResponse, error)
}

func NewGoodsHandler(goodsClient goodsServiceClient, rpcTimeout time.Duration) *GoodsHandler {
	return &GoodsHandler{
		goodsClient: goodsClient,
		rpcTimeout:  normalizeRPCTimeout(rpcTimeout),
	}
}

type createGoodsRequest struct {
	Title       string `json:"title" form:"title" binding:"required"`
	CoverURL    string `json:"cover_url" form:"cover_url"`
	Description string `json:"description" form:"description"`
}

type updateGoodsRequest struct {
	Title       *string `json:"title" form:"title"`
	CoverURL    *string `json:"cover_url" form:"cover_url"`
	Description *string `json:"description" form:"description"`
}

type batchGetGoodsRequest struct {
	IDs []int64 `json:"ids" binding:"required"`
}

type goodsResponse struct {
	ID          int64  `json:"id"`
	ShopID      int64  `json:"shop_id"`
	Title       string `json:"title"`
	CoverURL    string `json:"cover_url,omitempty"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type goodsListResponse struct {
	Total    int64           `json:"total"`
	Page     int32           `json:"page"`
	PageSize int32           `json:"page_size"`
	List     []goodsResponse `json:"list"`
}

func (h *GoodsHandler) Create(c *gin.Context) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}

	var req createGoodsRequest
	if err := c.ShouldBind(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "创建商品参数无效，请检查 title、cover_url 和 description")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.goodsClient.CreateGoods(ctx, &goodsv1.CreateGoodsRequest{
		ShopId:      shopID,
		Title:       req.Title,
		CoverUrl:    req.CoverURL,
		Description: req.Description,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, toGoodsResponse(resp.GetGoods()))
}

func (h *GoodsHandler) Update(c *gin.Context) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req updateGoodsRequest
	if err := c.ShouldBind(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "编辑商品参数无效，请提交合法的商品信息")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.goodsClient.UpdateGoods(ctx, &goodsv1.UpdateGoodsRequest{
		Id:          id,
		ShopId:      shopID,
		Title:       req.Title,
		CoverUrl:    req.CoverURL,
		Description: req.Description,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, toGoodsResponse(resp.GetGoods()))
}

func (h *GoodsHandler) Delete(c *gin.Context) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	_, err := h.goodsClient.DeleteGoods(ctx, &goodsv1.DeleteGoodsRequest{
		Id:     id,
		ShopId: shopID,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, "")
}

func (h *GoodsHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.goodsClient.GetGoods(ctx, &goodsv1.GetGoodsRequest{Id: id})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, toGoodsResponse(resp.GetGoods()))
}

func (h *GoodsHandler) List(c *gin.Context) {
	req, ok := listGoodsRequest(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.goodsClient.ListGoods(ctx, req)
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, goodsListResponse{
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
		List:     toGoodsResponseList(resp.GetList()),
	})
}

func (h *GoodsHandler) ListShopGoods(c *gin.Context) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}
	page, ok := parseOptionalInt32Query(c, "page")
	if !ok {
		return
	}
	pageSize, ok := parseOptionalInt32Query(c, "page_size")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.goodsClient.ListShopGoods(ctx, &goodsv1.ListShopGoodsRequest{
		ShopId:   shopID,
		Page:     int32ValueOrZero(page),
		PageSize: int32ValueOrZero(pageSize),
		Keyword:  c.Query("keyword"),
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, goodsListResponse{
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
		List:     toGoodsResponseList(resp.GetList()),
	})
}

func (h *GoodsHandler) BatchGetGoods(c *gin.Context) {
	var req batchGetGoodsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "批量查询商品参数无效，请提交非空的 ids 数组")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.goodsClient.BatchGetGoods(ctx, &goodsv1.BatchGetGoodsRequest{Ids: req.IDs})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, toGoodsResponseList(resp.GetList()))
}

func (h *GoodsHandler) PutOnSale(c *gin.Context) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	_, err := h.goodsClient.PutGoodsOnSale(ctx, &goodsv1.PutGoodsOnSaleRequest{
		Id:     id,
		ShopId: shopID,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, "")
}

func (h *GoodsHandler) PutOffSale(c *gin.Context) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	_, err := h.goodsClient.PutGoodsOffSale(ctx, &goodsv1.PutGoodsOffSaleRequest{
		Id:     id,
		ShopId: shopID,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, "")
}

func listGoodsRequest(c *gin.Context) (*goodsv1.ListGoodsRequest, bool) {
	page, ok := parseOptionalInt32Query(c, "page")
	if !ok {
		return nil, false
	}
	pageSize, ok := parseOptionalInt32Query(c, "page_size")
	if !ok {
		return nil, false
	}
	shopID, ok := parseOptionalInt64Query(c, "shop_id")
	if !ok {
		return nil, false
	}
	isDeleted, ok := parseOptionalInt32Query(c, "is_deleted")
	if !ok {
		return nil, false
	}
	status, ok := parseOptionalInt32Query(c, "status")
	if !ok {
		return nil, false
	}
	return &goodsv1.ListGoodsRequest{
		Page:      int32ValueOrZero(page),
		PageSize:  int32ValueOrZero(pageSize),
		Keyword:   c.Query("keyword"),
		ShopId:    shopID,
		IsDeleted: isDeleted,
		Status:    status,
	}, true
}

func toGoodsResponse(goods *goodsv1.Goods) goodsResponse {
	if goods == nil {
		return goodsResponse{}
	}
	return goodsResponse{
		ID:          goods.GetId(),
		ShopID:      goods.GetShopId(),
		Title:       goods.GetTitle(),
		CoverURL:    goods.GetCoverUrl(),
		Description: goods.GetDescription(),
		CreatedAt:   documentTimeString(goods.GetCreatedAt()),
		UpdatedAt:   documentTimeString(goods.GetUpdatedAt()),
	}
}

func toGoodsResponseList(list []*goodsv1.Goods) []goodsResponse {
	result := make([]goodsResponse, 0, len(list))
	for _, goods := range list {
		result = append(result, toGoodsResponse(goods))
	}
	return result
}
