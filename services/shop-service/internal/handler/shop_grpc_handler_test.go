package handler

import (
	"context"
	"strconv"
	"testing"

	shopv1 "github.com/yayccc/livebid/gen/proto/shop/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"github.com/yayccc/livebid/pkg/idgen"
	"github.com/yayccc/livebid/services/shop-service/internal/model"
	"github.com/yayccc/livebid/services/shop-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeShopRepository struct {
	shops     map[int64]*model.Shop
	updatedID int64
}

func (r *fakeShopRepository) Create(ctx context.Context, shop *model.Shop) error {
	if r.shops == nil {
		r.shops = make(map[int64]*model.Shop)
	}
	r.shops[shop.ID] = shop
	return nil
}

func (r *fakeShopRepository) FindByID(ctx context.Context, id int64) (*model.Shop, error) {
	shop, ok := r.shops[id]
	if !ok {
		return nil, repository.ErrShopNotFound
	}
	return shop, nil
}

func (r *fakeShopRepository) FindByUsername(ctx context.Context, username string) (*model.Shop, error) {
	for _, shop := range r.shops {
		if shop.Username == username {
			return shop, nil
		}
	}
	return nil, repository.ErrShopNotFound
}

func (r *fakeShopRepository) BatchFindByIDs(ctx context.Context, ids []int64) ([]*model.Shop, error) {
	list := make([]*model.Shop, 0, len(ids))
	for _, id := range ids {
		if shop, ok := r.shops[id]; ok {
			list = append(list, shop)
		}
	}
	return list, nil
}

func (r *fakeShopRepository) Update(ctx context.Context, shop *model.Shop) error {
	if _, ok := r.shops[shop.ID]; !ok {
		return repository.ErrShopNotFound
	}
	r.shops[shop.ID] = shop
	r.updatedID = shop.ID
	return nil
}

func TestShopGRPCHandlerGetShopUsesRequestID(t *testing.T) {
	repo := &fakeShopRepository{shops: map[int64]*model.Shop{
		1001: testModelShop(1001, "first shop"),
		2002: testModelShop(2002, "second shop"),
	}}
	handler := NewShopGRPCHandler(repo, nil, idgen.New(1))

	resp, err := handler.GetShop(context.Background(), &shopv1.GetShopRequest{Id: 2002})
	if err != nil {
		t.Fatalf("GetShop returned error: %v", err)
	}
	if resp.GetShop().GetId() != 2002 || resp.GetShop().GetShopName() != "second shop" {
		t.Fatalf("unexpected shop: %#v", resp.GetShop())
	}
}

func TestShopGRPCHandlerGetShopUsesMetadataShopID(t *testing.T) {
	repo := &fakeShopRepository{shops: map[int64]*model.Shop{
		1001: testModelShop(1001, "first shop"),
		2002: testModelShop(2002, "second shop"),
	}}
	handler := NewShopGRPCHandler(repo, nil, idgen.New(1))

	resp, err := handler.GetShop(shopMetadataContext(2002), &shopv1.GetShopRequest{})
	if err != nil {
		t.Fatalf("GetShop returned error: %v", err)
	}
	if resp.GetShop().GetId() != 2002 || resp.GetShop().GetShopName() != "second shop" {
		t.Fatalf("unexpected shop: %#v", resp.GetShop())
	}
}

func TestShopGRPCHandlerGetShopRejectsMissingIDAndMetadata(t *testing.T) {
	handler := NewShopGRPCHandler(&fakeShopRepository{}, nil, idgen.New(1))

	_, err := handler.GetShop(context.Background(), &shopv1.GetShopRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestShopGRPCHandlerBatchGetPublicShopsReturnsRequestedOrder(t *testing.T) {
	repo := &fakeShopRepository{shops: map[int64]*model.Shop{
		1001: testModelShop(1001, "first shop"),
		2002: testModelShop(2002, "second shop"),
	}}
	handler := NewShopGRPCHandler(repo, nil, idgen.New(1))

	resp, err := handler.BatchGetPublicShops(context.Background(), &shopv1.BatchGetPublicShopsRequest{
		Ids: []int64{2002, 1001},
	})
	if err != nil {
		t.Fatalf("BatchGetPublicShops returned error: %v", err)
	}
	if len(resp.GetList()) != 2 || resp.GetList()[0].GetId() != 2002 || resp.GetList()[1].GetId() != 1001 {
		t.Fatalf("unexpected shop order: %#v", resp.GetList())
	}
}

func TestShopGRPCHandlerUpdateShopUsesRequestID(t *testing.T) {
	repo := &fakeShopRepository{shops: map[int64]*model.Shop{
		1001: testModelShop(1001, "first shop"),
		2002: testModelShop(2002, "second shop"),
	}}
	handler := NewShopGRPCHandler(repo, nil, idgen.New(1))
	shopName := "updated second shop"

	resp, err := handler.UpdateShop(shopMetadataContext(2002), &shopv1.UpdateShopRequest{
		Id:       2002,
		ShopName: &shopName,
	})
	if err != nil {
		t.Fatalf("UpdateShop returned error: %v", err)
	}
	if repo.updatedID != 2002 || resp.GetShop().GetShopName() != shopName {
		t.Fatalf("unexpected update: updatedID=%d shop=%#v", repo.updatedID, resp.GetShop())
	}
	if repo.shops[1001].ShopName != "first shop" {
		t.Fatalf("unexpected first shop mutation: %#v", repo.shops[1001])
	}
}

func TestShopGRPCHandlerUpdateShopRejectsMismatchedMetadataShopID(t *testing.T) {
	repo := &fakeShopRepository{shops: map[int64]*model.Shop{
		1001: testModelShop(1001, "first shop"),
		2002: testModelShop(2002, "second shop"),
	}}
	handler := NewShopGRPCHandler(repo, nil, idgen.New(1))
	shopName := "updated second shop"

	_, err := handler.UpdateShop(shopMetadataContext(1001), &shopv1.UpdateShopRequest{
		Id:       2002,
		ShopName: &shopName,
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", err)
	}
}

func shopMetadataContext(shopID int64) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		identity.MetadataSubjectType, string(identity.KindShop),
		identity.MetadataSubjectID, strconv.FormatInt(shopID, 10),
	))
}

func testModelShop(id int64, shopName string) *model.Shop {
	return &model.Shop{
		ID:          id,
		Username:    shopName,
		ShopName:    shopName,
		Status:      model.ShopStatusEnabled,
		AuditStatus: model.AuditStatusPending,
	}
}
