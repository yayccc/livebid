package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	auctionv1 "github.com/yayccc/livebid/gen/proto/auction/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockAuctionClient struct {
	create func(ctx context.Context, in *auctionv1.CreateAuctionRequest, opts ...grpc.CallOption) (*auctionv1.CreateAuctionResponse, error)
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
