package handler

import (
	"context"
	"testing"
	"time"

	auctionv1 "github.com/yayccc/livebid/gen/proto/auction/v1"
	"github.com/yayccc/livebid/services/auction-service/internal/model"
	"github.com/yayccc/livebid/services/auction-service/internal/repository"
)

type fakeAuctionRepository struct {
	auctionsByID     map[int64]*model.Auction
	auctionsByRoomID map[int64][]*model.Auction
}

func (r *fakeAuctionRepository) Create(ctx context.Context, auction *model.Auction) error {
	return nil
}

func (r *fakeAuctionRepository) FindByID(ctx context.Context, id int64) (*model.Auction, error) {
	if auction := r.auctionsByID[id]; auction != nil {
		return auction, nil
	}
	return nil, repository.ErrAuctionNotFound
}

func (r *fakeAuctionRepository) FindByIDForShop(ctx context.Context, id int64, shopID int64) (*model.Auction, error) {
	return r.FindByID(ctx, id)
}

func (r *fakeAuctionRepository) FindByGoodsID(ctx context.Context, goodsID int64) (*model.Auction, error) {
	return nil, repository.ErrAuctionNotFound
}

func (r *fakeAuctionRepository) FindCurrentByRoomID(ctx context.Context, roomID int64) (*model.Auction, error) {
	for _, auction := range r.auctionsByRoomID[roomID] {
		if auction.Status == model.AuctionStatusRunning {
			return auction, nil
		}
	}
	return nil, repository.ErrAuctionNotFound
}

func (r *fakeAuctionRepository) BatchFindCurrentByRoomIDs(ctx context.Context, roomIDs []int64) ([]*model.Auction, error) {
	list := make([]*model.Auction, 0, len(roomIDs))
	for _, roomID := range roomIDs {
		auction, err := r.FindCurrentByRoomID(ctx, roomID)
		if err == nil {
			list = append(list, auction)
		}
	}
	return list, nil
}

func (r *fakeAuctionRepository) ListByShop(ctx context.Context, filter repository.ListAuctionFilter) ([]*model.Auction, int64, error) {
	return nil, 0, nil
}

func (r *fakeAuctionRepository) SummarizeByShop(ctx context.Context, shopID int64, todayStart time.Time, todayEnd time.Time) (repository.MerchantAuctionSummary, error) {
	return repository.MerchantAuctionSummary{}, nil
}

func (r *fakeAuctionRepository) UpdatePendingConfig(ctx context.Context, auction *model.Auction) error {
	return nil
}

func (r *fakeAuctionRepository) UpdateStatusSnapshot(ctx context.Context, auction *model.Auction) error {
	return nil
}

func (r *fakeAuctionRepository) Delete(ctx context.Context, id int64, shopID int64) error {
	return nil
}

func TestAuctionGRPCHandlerGetCurrentAuctionByRoom(t *testing.T) {
	handler := NewAuctionGRPCHandler(&fakeAuctionRepository{
		auctionsByRoomID: map[int64][]*model.Auction{
			2001: {testRunningAuction(5001, 2001, 3001)},
		},
	}, nil, nil, nil, nil, nil, nil, 0)

	resp, err := handler.GetCurrentAuctionByRoom(context.Background(), &auctionv1.GetCurrentAuctionByRoomRequest{RoomId: 2001})
	if err != nil {
		t.Fatalf("GetCurrentAuctionByRoom returned error: %v", err)
	}
	if resp.GetAuction().GetId() != 5001 || resp.GetAuction().GetRoomId() != 2001 {
		t.Fatalf("unexpected auction: %#v", resp.GetAuction())
	}
}

func TestAuctionGRPCHandlerBatchGetCurrentAuctionsByRoomKeepsRoomOrder(t *testing.T) {
	handler := NewAuctionGRPCHandler(&fakeAuctionRepository{
		auctionsByRoomID: map[int64][]*model.Auction{
			2001: {testRunningAuction(5001, 2001, 3001)},
			2002: {testRunningAuction(5002, 2002, 3002)},
		},
	}, nil, nil, nil, nil, nil, nil, 0)

	resp, err := handler.BatchGetCurrentAuctionsByRoom(context.Background(), &auctionv1.BatchGetCurrentAuctionsByRoomRequest{
		RoomIds: []int64{2002, 2001},
	})
	if err != nil {
		t.Fatalf("BatchGetCurrentAuctionsByRoom returned error: %v", err)
	}
	if len(resp.GetList()) != 2 || resp.GetList()[0].GetRoomId() != 2002 || resp.GetList()[1].GetRoomId() != 2001 {
		t.Fatalf("unexpected auction order: %#v", resp.GetList())
	}
}

func testRunningAuction(id int64, roomID int64, goodsID int64) *model.Auction {
	now := time.Now()
	return &model.Auction{
		ID:           id,
		GoodsID:      goodsID,
		ShopID:       1001,
		RoomID:       roomID,
		StartPrice:   10000,
		BidIncrement: 1000,
		CurrentPrice: 10000,
		Status:       model.AuctionStatusRunning,
		StartTime:    &now,
		EndTime:      &now,
	}
}
