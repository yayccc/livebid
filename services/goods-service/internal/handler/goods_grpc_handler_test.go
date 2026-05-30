package handler

import (
	"context"
	"testing"
	"time"

	goodsv1 "github.com/yayccc/livebid/gen/proto/goods/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"github.com/yayccc/livebid/pkg/idgen"
	"github.com/yayccc/livebid/services/goods-service/internal/model"
	"github.com/yayccc/livebid/services/goods-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockGoodsRepository struct {
	create          func(ctx context.Context, goods *model.Goods) error
	findByIDForShop func(ctx context.Context, id int64, shopID int64) (*model.Goods, error)
	list            func(ctx context.Context, filter repository.ListGoodsFilter) ([]*model.Goods, int64, error)
	updateStatus    func(ctx context.Context, id int64, shopID int64, status model.GoodsStatus) error
}

func (m *mockGoodsRepository) Create(ctx context.Context, goods *model.Goods) error {
	if m.create == nil {
		return nil
	}
	return m.create(ctx, goods)
}

func (m *mockGoodsRepository) Update(ctx context.Context, goods *model.Goods) error {
	return nil
}

func (m *mockGoodsRepository) Delete(ctx context.Context, id int64, shopID int64) error {
	return nil
}

func (m *mockGoodsRepository) FindByID(ctx context.Context, id int64) (*model.Goods, error) {
	return nil, repository.ErrGoodsNotFound
}

func (m *mockGoodsRepository) FindByIDForShop(ctx context.Context, id int64, shopID int64) (*model.Goods, error) {
	if m.findByIDForShop == nil {
		return nil, repository.ErrGoodsNotFound
	}
	return m.findByIDForShop(ctx, id, shopID)
}

func (m *mockGoodsRepository) BatchFindByIDs(ctx context.Context, ids []int64) ([]*model.Goods, error) {
	return nil, nil
}

func (m *mockGoodsRepository) List(ctx context.Context, filter repository.ListGoodsFilter) ([]*model.Goods, int64, error) {
	if m.list == nil {
		return nil, 0, nil
	}
	return m.list(ctx, filter)
}

func (m *mockGoodsRepository) UpdateStatus(ctx context.Context, id int64, shopID int64, status model.GoodsStatus) error {
	if m.updateStatus == nil {
		return nil
	}
	return m.updateStatus(ctx, id, shopID, status)
}

func TestGoodsGRPCHandlerCreateGoodsUsesContextShopID(t *testing.T) {
	var created *model.Goods
	handler := newTestGoodsHandler(&mockGoodsRepository{
		create: func(ctx context.Context, goods *model.Goods) error {
			created = goods
			return nil
		},
	})

	resp, err := handler.CreateGoods(shopContext(1001), &goodsv1.CreateGoodsRequest{
		ShopId:      9999,
		Title:       "  翡翠手镯  ",
		CoverUrl:    "  https://example.com/cover.jpg  ",
		Description: "  天然翡翠  ",
	})
	if err != nil {
		t.Fatalf("CreateGoods returned error: %v", err)
	}
	if created == nil {
		t.Fatal("expected repository Create to be called")
	}
	if created.ShopID != 1001 {
		t.Fatalf("expected context shop id, got %d", created.ShopID)
	}
	if resp.GetGoods().GetShopId() != 1001 {
		t.Fatalf("unexpected response shop id: %d", resp.GetGoods().GetShopId())
	}
}

func TestGoodsGRPCHandlerCreateGoodsRequiresShopIdentity(t *testing.T) {
	handler := newTestGoodsHandler(&mockGoodsRepository{
		create: func(ctx context.Context, goods *model.Goods) error {
			t.Fatal("Create should not be called")
			return nil
		},
	})

	_, err := handler.CreateGoods(context.Background(), &goodsv1.CreateGoodsRequest{
		ShopId: 1001,
		Title:  "翡翠手镯",
	})
	assertCode(t, err, codes.Unauthenticated)
}

func TestGoodsGRPCHandlerListShopGoodsUsesContextShopID(t *testing.T) {
	handler := newTestGoodsHandler(&mockGoodsRepository{
		list: func(ctx context.Context, filter repository.ListGoodsFilter) ([]*model.Goods, int64, error) {
			if filter.ShopID == nil || *filter.ShopID != 1001 {
				t.Fatalf("expected context shop id filter, got %#v", filter.ShopID)
			}
			return []*model.Goods{testGoods(2001, 1001)}, 1, nil
		},
	})

	resp, err := handler.ListShopGoods(shopContext(1001), &goodsv1.ListShopGoodsRequest{
		ShopId: 9999,
	})
	if err != nil {
		t.Fatalf("ListShopGoods returned error: %v", err)
	}
	if resp.GetTotal() != 1 || len(resp.GetList()) != 1 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestGoodsGRPCHandlerPutOnSaleUsesContextShopID(t *testing.T) {
	handler := newTestGoodsHandler(&mockGoodsRepository{
		updateStatus: func(ctx context.Context, id int64, shopID int64, status model.GoodsStatus) error {
			if id != 2001 || shopID != 1001 || status != model.GoodsStatusOnSale {
				t.Fatalf("unexpected update status args: id=%d shopID=%d status=%d", id, shopID, status)
			}
			return nil
		},
	})

	if _, err := handler.PutGoodsOnSale(shopContext(1001), &goodsv1.PutGoodsOnSaleRequest{Id: 2001, ShopId: 9999}); err != nil {
		t.Fatalf("PutGoodsOnSale returned error: %v", err)
	}
}

func newTestGoodsHandler(repo repository.GoodsRepository) *GoodsGRPCHandler {
	return NewGoodsGRPCHandler(repo, idgen.New(9))
}

func shopContext(shopID int64) context.Context {
	return identity.NewContext(context.Background(), identity.Principal{
		Kind: identity.KindShop,
		ID:   shopID,
	})
}

func testGoods(id int64, shopID int64) *model.Goods {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	return &model.Goods{
		ID:          id,
		ShopID:      shopID,
		Title:       "翡翠手镯",
		CoverURL:    "https://example.com/cover.jpg",
		Description: "天然翡翠",
		Status:      model.GoodsStatusOffSale,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func assertCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %s, got nil", want)
	}
	if got := status.Code(err); got != want {
		t.Fatalf("unexpected error code: got %s want %s, err=%v", got, want, err)
	}
}
