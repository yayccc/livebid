package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	auctionv1 "github.com/yayccc/livebid/gen/proto/auction/v1"
	goodsv1 "github.com/yayccc/livebid/gen/proto/goods/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockAuctionClient struct {
	create   func(ctx context.Context, in *auctionv1.CreateAuctionRequest, opts ...grpc.CallOption) (*auctionv1.CreateAuctionResponse, error)
	merchant func(ctx context.Context, in *auctionv1.ListMerchantAuctionsRequest, opts ...grpc.CallOption) (*auctionv1.ListMerchantAuctionsResponse, error)
	runtime  func(ctx context.Context, in *auctionv1.GetAuctionRuntimeRequest, opts ...grpc.CallOption) (*auctionv1.GetAuctionRuntimeResponse, error)
	summary  func(ctx context.Context, in *auctionv1.GetMerchantDashboardSummaryRequest, opts ...grpc.CallOption) (*auctionv1.GetMerchantDashboardSummaryResponse, error)
}

func (m mockAuctionClient) CreateAuction(ctx context.Context, in *auctionv1.CreateAuctionRequest, opts ...grpc.CallOption) (*auctionv1.CreateAuctionResponse, error) {
	return m.create(ctx, in, opts...)
}

func (m mockAuctionClient) GetAuction(ctx context.Context, in *auctionv1.GetAuctionRequest, opts ...grpc.CallOption) (*auctionv1.GetAuctionResponse, error) {
	panic("not implemented")
}

func (m mockAuctionClient) GetAuctionByGoods(ctx context.Context, in *auctionv1.GetAuctionByGoodsRequest, opts ...grpc.CallOption) (*auctionv1.GetAuctionByGoodsResponse, error) {
	panic("not implemented")
}

func (m mockAuctionClient) ListShopAuctions(ctx context.Context, in *auctionv1.ListShopAuctionsRequest, opts ...grpc.CallOption) (*auctionv1.ListShopAuctionsResponse, error) {
	panic("not implemented")
}

func (m mockAuctionClient) ListMerchantAuctions(ctx context.Context, in *auctionv1.ListMerchantAuctionsRequest, opts ...grpc.CallOption) (*auctionv1.ListMerchantAuctionsResponse, error) {
	return m.merchant(ctx, in, opts...)
}

func (m mockAuctionClient) GetAuctionRuntime(ctx context.Context, in *auctionv1.GetAuctionRuntimeRequest, opts ...grpc.CallOption) (*auctionv1.GetAuctionRuntimeResponse, error) {
	return m.runtime(ctx, in, opts...)
}

func (m mockAuctionClient) GetMerchantDashboardSummary(ctx context.Context, in *auctionv1.GetMerchantDashboardSummaryRequest, opts ...grpc.CallOption) (*auctionv1.GetMerchantDashboardSummaryResponse, error) {
	return m.summary(ctx, in, opts...)
}

func (m mockAuctionClient) UpdateAuction(ctx context.Context, in *auctionv1.UpdateAuctionRequest, opts ...grpc.CallOption) (*auctionv1.UpdateAuctionResponse, error) {
	panic("not implemented")
}

func (m mockAuctionClient) StartAuction(ctx context.Context, in *auctionv1.StartAuctionRequest, opts ...grpc.CallOption) (*auctionv1.StartAuctionResponse, error) {
	panic("not implemented")
}

func (m mockAuctionClient) FinishAuction(ctx context.Context, in *auctionv1.FinishAuctionRequest, opts ...grpc.CallOption) (*auctionv1.FinishAuctionResponse, error) {
	panic("not implemented")
}

func (m mockAuctionClient) CancelAuction(ctx context.Context, in *auctionv1.CancelAuctionRequest, opts ...grpc.CallOption) (*auctionv1.CancelAuctionResponse, error) {
	panic("not implemented")
}

func (m mockAuctionClient) DeleteAuction(ctx context.Context, in *auctionv1.DeleteAuctionRequest, opts ...grpc.CallOption) (*auctionv1.DeleteAuctionResponse, error) {
	panic("not implemented")
}

func (m mockAuctionClient) ListBidRecords(ctx context.Context, in *auctionv1.ListBidRecordsRequest, opts ...grpc.CallOption) (*auctionv1.ListBidRecordsResponse, error) {
	panic("not implemented")
}

type mockAuctionGoodsClient struct {
	list func(ctx context.Context, in *goodsv1.ListGoodsRequest, opts ...grpc.CallOption) (*goodsv1.ListGoodsResponse, error)
}

func (m mockAuctionGoodsClient) CreateGoods(ctx context.Context, in *goodsv1.CreateGoodsRequest, opts ...grpc.CallOption) (*goodsv1.CreateGoodsResponse, error) {
	panic("not implemented")
}

func (m mockAuctionGoodsClient) UpdateGoods(ctx context.Context, in *goodsv1.UpdateGoodsRequest, opts ...grpc.CallOption) (*goodsv1.UpdateGoodsResponse, error) {
	panic("not implemented")
}

func (m mockAuctionGoodsClient) DeleteGoods(ctx context.Context, in *goodsv1.DeleteGoodsRequest, opts ...grpc.CallOption) (*goodsv1.DeleteGoodsResponse, error) {
	panic("not implemented")
}

func (m mockAuctionGoodsClient) GetGoods(ctx context.Context, in *goodsv1.GetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.GetGoodsResponse, error) {
	panic("not implemented")
}

func (m mockAuctionGoodsClient) ListGoods(ctx context.Context, in *goodsv1.ListGoodsRequest, opts ...grpc.CallOption) (*goodsv1.ListGoodsResponse, error) {
	return m.list(ctx, in, opts...)
}

func (m mockAuctionGoodsClient) ListShopGoods(ctx context.Context, in *goodsv1.ListShopGoodsRequest, opts ...grpc.CallOption) (*goodsv1.ListShopGoodsResponse, error) {
	panic("not implemented")
}

func (m mockAuctionGoodsClient) BatchGetGoods(ctx context.Context, in *goodsv1.BatchGetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.BatchGetGoodsResponse, error) {
	panic("not implemented")
}

func (m mockAuctionGoodsClient) PutGoodsOnSale(ctx context.Context, in *goodsv1.PutGoodsOnSaleRequest, opts ...grpc.CallOption) (*goodsv1.PutGoodsOnSaleResponse, error) {
	panic("not implemented")
}

func (m mockAuctionGoodsClient) PutGoodsOffSale(ctx context.Context, in *goodsv1.PutGoodsOffSaleRequest, opts ...grpc.CallOption) (*goodsv1.PutGoodsOffSaleResponse, error) {
	panic("not implemented")
}

func TestAuctionHandlerCreateUsesContextShopID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewAuctionHandler(mockAuctionClient{
		create: func(ctx context.Context, in *auctionv1.CreateAuctionRequest, opts ...grpc.CallOption) (*auctionv1.CreateAuctionResponse, error) {
			if in.GetShopId() != 1001 || in.GetGoodsId() != 2001 {
				t.Fatalf("unexpected create request: %#v", in)
			}
			return &auctionv1.CreateAuctionResponse{Auction: testAuction()}, nil
		},
	}, time.Second)

	body := bytes.NewBufferString(`{"goods_id":2001,"shop_id":9999,"start_price":10000,"bid_increment":1000}`)
	w := performAuctionRequest(handler.Create, http.MethodPost, "/api/auctions", body, identity.Principal{Kind: identity.KindShop, ID: 1001})

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuctionHandlerListMerchantReturnsGoodsFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewAuctionHandler(mockAuctionClient{
		merchant: func(ctx context.Context, in *auctionv1.ListMerchantAuctionsRequest, opts ...grpc.CallOption) (*auctionv1.ListMerchantAuctionsResponse, error) {
			if in.GetShopId() != 1001 || in.GetPage() != 2 || in.GetPageSize() != 10 || in.GetKeyword() != "翡翠" || in.GetStatus() != 1 {
				t.Fatalf("unexpected merchant list request: %#v", in)
			}
			return &auctionv1.ListMerchantAuctionsResponse{
				Total:    1,
				Page:     2,
				PageSize: 10,
				List: []*auctionv1.MerchantAuction{{
					Id:            3001,
					GoodsId:       2001,
					ShopId:        1001,
					GoodsTitle:    "高冰翡翠手镯",
					GoodsCoverUrl: "https://example.com/cover.jpg",
					StartPrice:    10000,
					BidIncrement:  1000,
					CurrentPrice:  18000,
					BidCount:      8,
					Status:        1,
				}},
			}, nil
		},
	}, time.Second)

	w := performAuctionRequest(handler.ListMerchant, http.MethodGet, "/api/merchant/auctions?page=2&page_size=10&status=1&keyword=翡翠", bytes.NewBuffer(nil), identity.Principal{Kind: identity.KindShop, ID: 1001})

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data auctionListResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Data.List) != 1 || resp.Data.List[0].GoodsTitle != "高冰翡翠手镯" || resp.Data.List[0].GoodsCoverURL == "" {
		t.Fatalf("expected goods fields in merchant auction list, got %#v", resp.Data.List)
	}
}

func TestAuctionHandlerDashboardSummaryMergesGoodsCounts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	goodsCalls := 0
	handler := NewAuctionHandlerWithGoods(mockAuctionClient{
		summary: func(ctx context.Context, in *auctionv1.GetMerchantDashboardSummaryRequest, opts ...grpc.CallOption) (*auctionv1.GetMerchantDashboardSummaryResponse, error) {
			if in.GetShopId() != 1001 {
				t.Fatalf("unexpected summary request: %#v", in)
			}
			return &auctionv1.GetMerchantDashboardSummaryResponse{
				Summary: &auctionv1.MerchantDashboardSummary{
					AuctionTotal:     36,
					AuctionRunning:   6,
					AuctionPending:   12,
					AuctionDeal:      14,
					AuctionFailed:    3,
					AuctionCancelled: 1,
					TodayDealAmount:  8642000,
					TodayBidCount:    238,
				},
			}, nil
		},
	}, mockAuctionGoodsClient{
		list: func(ctx context.Context, in *goodsv1.ListGoodsRequest, opts ...grpc.CallOption) (*goodsv1.ListGoodsResponse, error) {
			goodsCalls++
			if in.GetShopId() != 1001 || in.GetPage() != 1 || in.GetPageSize() != 1 {
				t.Fatalf("unexpected goods count request: %#v", in)
			}
			switch {
			case in.Status == nil:
				return &goodsv1.ListGoodsResponse{Total: 128}, nil
			case in.GetStatus() == 1:
				return &goodsv1.ListGoodsResponse{Total: 86}, nil
			case in.GetStatus() == 0:
				return &goodsv1.ListGoodsResponse{Total: 42}, nil
			default:
				t.Fatalf("unexpected goods status: %#v", in.Status)
				return nil, nil
			}
		},
	}, time.Second)

	w := performAuctionRequest(handler.DashboardSummary, http.MethodGet, "/api/merchant/dashboard/summary", bytes.NewBuffer(nil), identity.Principal{Kind: identity.KindShop, ID: 1001})

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if goodsCalls != 3 {
		t.Fatalf("expected 3 goods count calls, got %d", goodsCalls)
	}
	var resp struct {
		Data merchantDashboardSummaryResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Data.GoodsTotal != 128 || resp.Data.GoodsOnSale != 86 || resp.Data.AuctionRunning != 6 || resp.Data.TodayBidCount != 238 {
		t.Fatalf("unexpected dashboard summary: %#v", resp.Data)
	}
}

func TestAuctionHandlerGetRuntimeReturnsCountdownFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	serverTime := timestamppb.New(time.Date(2026, 5, 24, 20, 15, 0, 0, time.UTC))
	expireAt := timestamppb.New(time.Date(2026, 5, 24, 20, 15, 14, 0, time.UTC))
	winner := int64(30001)
	handler := NewAuctionHandler(mockAuctionClient{
		runtime: func(ctx context.Context, in *auctionv1.GetAuctionRuntimeRequest, opts ...grpc.CallOption) (*auctionv1.GetAuctionRuntimeResponse, error) {
			if in.GetAuctionId() != 3001 {
				t.Fatalf("unexpected runtime request: %#v", in)
			}
			return &auctionv1.GetAuctionRuntimeResponse{
				Runtime: &auctionv1.AuctionRuntime{
					AuctionId:    3001,
					Status:       1,
					CurrentPrice: 18000,
					BidCount:     8,
					WinnerUserId: &winner,
					ServerTime:   serverTime,
					ExpireAt:     expireAt,
					Version:      8,
				},
			}, nil
		},
	}, time.Second)

	w := performAuctionRequestWithParams(handler.GetRuntime, http.MethodGet, "/api/auctions/3001/runtime", bytes.NewBuffer(nil), identity.Principal{Kind: identity.KindShop, ID: 1001}, gin.Params{{Key: "id", Value: "3001"}})

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data auctionRuntimeResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Data.Version != 8 || resp.Data.ExpireAt == "" || resp.Data.ServerTime == "" || resp.Data.WinnerUserID == nil {
		t.Fatalf("unexpected runtime response: %#v", resp.Data)
	}
}

func testAuction() *auctionv1.Auction {
	now := timestamppb.New(time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC))
	return &auctionv1.Auction{
		Id:           3001,
		GoodsId:      2001,
		ShopId:       1001,
		StartPrice:   10000,
		BidIncrement: 1000,
		CurrentPrice: 10000,
		Status:       0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func performAuctionRequest(handlerFunc gin.HandlerFunc, method string, path string, body *bytes.Buffer, principal identity.Principal) *httptest.ResponseRecorder {
	return performAuctionRequestWithParams(handlerFunc, method, path, body, principal, nil)
}

func performAuctionRequestWithParams(handlerFunc gin.HandlerFunc, method string, path string, body *bytes.Buffer, principal identity.Principal, params gin.Params) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, body)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = params
	c.Request = c.Request.WithContext(identity.NewContext(c.Request.Context(), principal))
	handlerFunc(c)
	return w
}
