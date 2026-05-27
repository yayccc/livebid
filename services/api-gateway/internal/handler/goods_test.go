package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	goodsv1 "github.com/yayccc/livebid/gen/proto/goods/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockGoodsClient struct {
	create     func(ctx context.Context, in *goodsv1.CreateGoodsRequest, opts ...grpc.CallOption) (*goodsv1.CreateGoodsResponse, error)
	update     func(ctx context.Context, in *goodsv1.UpdateGoodsRequest, opts ...grpc.CallOption) (*goodsv1.UpdateGoodsResponse, error)
	delete     func(ctx context.Context, in *goodsv1.DeleteGoodsRequest, opts ...grpc.CallOption) (*goodsv1.DeleteGoodsResponse, error)
	get        func(ctx context.Context, in *goodsv1.GetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.GetGoodsResponse, error)
	list       func(ctx context.Context, in *goodsv1.ListGoodsRequest, opts ...grpc.CallOption) (*goodsv1.ListGoodsResponse, error)
	listShop   func(ctx context.Context, in *goodsv1.ListShopGoodsRequest, opts ...grpc.CallOption) (*goodsv1.ListShopGoodsResponse, error)
	batchGet   func(ctx context.Context, in *goodsv1.BatchGetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.BatchGetGoodsResponse, error)
	putOnSale  func(ctx context.Context, in *goodsv1.PutGoodsOnSaleRequest, opts ...grpc.CallOption) (*goodsv1.PutGoodsOnSaleResponse, error)
	putOffSale func(ctx context.Context, in *goodsv1.PutGoodsOffSaleRequest, opts ...grpc.CallOption) (*goodsv1.PutGoodsOffSaleResponse, error)
}

func (m mockGoodsClient) CreateGoods(ctx context.Context, in *goodsv1.CreateGoodsRequest, opts ...grpc.CallOption) (*goodsv1.CreateGoodsResponse, error) {
	return m.create(ctx, in, opts...)
}

func (m mockGoodsClient) UpdateGoods(ctx context.Context, in *goodsv1.UpdateGoodsRequest, opts ...grpc.CallOption) (*goodsv1.UpdateGoodsResponse, error) {
	return m.update(ctx, in, opts...)
}

func (m mockGoodsClient) DeleteGoods(ctx context.Context, in *goodsv1.DeleteGoodsRequest, opts ...grpc.CallOption) (*goodsv1.DeleteGoodsResponse, error) {
	return m.delete(ctx, in, opts...)
}

func (m mockGoodsClient) GetGoods(ctx context.Context, in *goodsv1.GetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.GetGoodsResponse, error) {
	return m.get(ctx, in, opts...)
}

func (m mockGoodsClient) ListGoods(ctx context.Context, in *goodsv1.ListGoodsRequest, opts ...grpc.CallOption) (*goodsv1.ListGoodsResponse, error) {
	return m.list(ctx, in, opts...)
}

func (m mockGoodsClient) ListShopGoods(ctx context.Context, in *goodsv1.ListShopGoodsRequest, opts ...grpc.CallOption) (*goodsv1.ListShopGoodsResponse, error) {
	return m.listShop(ctx, in, opts...)
}

func (m mockGoodsClient) BatchGetGoods(ctx context.Context, in *goodsv1.BatchGetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.BatchGetGoodsResponse, error) {
	return m.batchGet(ctx, in, opts...)
}

func (m mockGoodsClient) PutGoodsOnSale(ctx context.Context, in *goodsv1.PutGoodsOnSaleRequest, opts ...grpc.CallOption) (*goodsv1.PutGoodsOnSaleResponse, error) {
	return m.putOnSale(ctx, in, opts...)
}

func (m mockGoodsClient) PutGoodsOffSale(ctx context.Context, in *goodsv1.PutGoodsOffSaleRequest, opts ...grpc.CallOption) (*goodsv1.PutGoodsOffSaleResponse, error) {
	return m.putOffSale(ctx, in, opts...)
}

func TestGoodsHandlerCreateForwardsToGoodsService(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewGoodsHandler(mockGoodsClient{
		create: func(ctx context.Context, in *goodsv1.CreateGoodsRequest, opts ...grpc.CallOption) (*goodsv1.CreateGoodsResponse, error) {
			if in.GetShopId() != defaultFakeShopID || in.GetTitle() != "翡翠手镯" || in.GetCoverUrl() != "https://example.com/cover.jpg" {
				t.Fatalf("unexpected create request: %#v", in)
			}
			return &goodsv1.CreateGoodsResponse{Goods: testGoods()}, nil
		},
	}, time.Second)

	body := bytes.NewBufferString(`{"title":"翡翠手镯","cover_url":"https://example.com/cover.jpg","description":"天然翡翠"}`)
	w := performRequest(handler.Create, http.MethodPost, "/api/goods", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGoodsHandlerListShopGoodsUsesFakeShopID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewGoodsHandler(mockGoodsClient{
		listShop: func(ctx context.Context, in *goodsv1.ListShopGoodsRequest, opts ...grpc.CallOption) (*goodsv1.ListShopGoodsResponse, error) {
			if in.GetShopId() != defaultFakeShopID || in.GetPage() != 2 || in.GetPageSize() != 20 || in.GetKeyword() != "玉" {
				t.Fatalf("unexpected list shop goods request: %#v", in)
			}
			return &goodsv1.ListShopGoodsResponse{
				Total:    1,
				Page:     2,
				PageSize: 20,
				List:     []*goodsv1.Goods{testGoods()},
			}, nil
		},
	}, time.Second)

	w := performRequest(handler.ListShopGoods, http.MethodGet, "/api/goods/shop/list?page=2&page_size=20&keyword=玉", bytes.NewBuffer(nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGoodsHandlerBatchGetForwardsIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewGoodsHandler(mockGoodsClient{
		batchGet: func(ctx context.Context, in *goodsv1.BatchGetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.BatchGetGoodsResponse, error) {
			if len(in.GetIds()) != 2 || in.GetIds()[0] != 1001 || in.GetIds()[1] != 1002 {
				t.Fatalf("unexpected batch request: %#v", in)
			}
			return &goodsv1.BatchGetGoodsResponse{List: []*goodsv1.Goods{testGoods()}}, nil
		},
	}, time.Second)

	body := bytes.NewBufferString(`{"ids":[1001,1002]}`)
	w := performRequest(handler.BatchGetGoods, http.MethodPost, "/api/goods/batch", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGoodsHandlerUploadCoverReturnsFakeURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewGoodsHandler(mockGoodsClient{}, time.Second)
	body, contentType := multipartFileBody(t, "file", "cover.JPG", []byte("fake image"))
	w := performRequestWithContentType(handler.UploadCover, http.MethodPost, "/api/goods/cover/upload", body, contentType)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			CoverURL string `json:"cover_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected code: %d", resp.Code)
	}
	if !strings.HasPrefix(resp.Data.CoverURL, "https://static.livebid.local/goods/cover/10001/") {
		t.Fatalf("unexpected cover url: %s", resp.Data.CoverURL)
	}
	if !strings.HasSuffix(resp.Data.CoverURL, ".jpg") {
		t.Fatalf("expected lower-case extension, got %s", resp.Data.CoverURL)
	}
}

func TestGoodsHandlerUploadCoverRequiresFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewGoodsHandler(mockGoodsClient{}, time.Second)
	body, contentType := multipartBody(t, map[string]string{"name": "cover"})
	w := performRequestWithContentType(handler.UploadCover, http.MethodPost, "/api/goods/cover/upload", body, contentType)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func multipartFileBody(t *testing.T, fieldName string, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func testGoods() *goodsv1.Goods {
	now := timestamppb.New(time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC))
	return &goodsv1.Goods{
		Id:          1001,
		ShopId:      defaultFakeShopID,
		Title:       "翡翠手镯",
		CoverUrl:    "https://example.com/cover.jpg",
		Description: "天然翡翠",
		Status:      0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
