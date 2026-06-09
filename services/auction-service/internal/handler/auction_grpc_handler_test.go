package handler

import (
	"context"
	"testing"
	"time"

	auctionv1 "github.com/yayccc/livebid/gen/proto/auction/v1"
	"github.com/yayccc/livebid/services/auction-service/internal/model"
	"github.com/yayccc/livebid/services/auction-service/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func (r *fakeAuctionRepository) ListByRoom(ctx context.Context, filter repository.ListAuctionFilter) ([]*model.Auction, int64, error) {
	list := make([]*model.Auction, 0)
	for _, auction := range r.auctionsByRoomID[filter.RoomID] {
		if filter.StartedAfter != nil && !auctionBelongsToSession(auction, *filter.StartedAfter, filter.UpcomingFrom, filter.UpcomingBefore) {
			continue
		}
		list = append(list, auction)
	}
	return list, int64(len(list)), nil
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

func TestAuctionGRPCHandlerListRoomAuctionsFiltersBySessionStart(t *testing.T) {
	sessionStart := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	handler := NewAuctionGRPCHandler(&fakeAuctionRepository{
		auctionsByRoomID: map[int64][]*model.Auction{
			2001: {
				testAuctionWithCreatedAt(5001, 2001, 3001, sessionStart.Add(-time.Minute)),
				testAuctionWithStartedAt(5002, 2001, 3002, sessionStart.Add(time.Minute)),
			},
		},
	}, nil, nil, nil, nil, nil, nil, 0)

	resp, err := handler.ListRoomAuctions(context.Background(), &auctionv1.ListRoomAuctionsRequest{
		RoomId:       2001,
		StartedAfter: timestamppb.New(sessionStart),
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		t.Fatalf("ListRoomAuctions returned error: %v", err)
	}
	if resp.GetTotal() != 1 || len(resp.GetList()) != 1 || resp.GetList()[0].GetId() != 5002 {
		t.Fatalf("unexpected room auctions: total=%d list=%#v", resp.GetTotal(), resp.GetList())
	}
}

func TestAuctionGRPCHandlerListRoomAuctionsLimitsPendingToNextHour(t *testing.T) {
	sessionStart := time.Now().Add(-time.Hour)
	soon := time.Now().Add(30 * time.Minute)
	later := time.Now().Add(2 * time.Hour)
	handler := NewAuctionGRPCHandler(&fakeAuctionRepository{
		auctionsByRoomID: map[int64][]*model.Auction{
			2001: {
				testPendingAuctionWithStartTime(5001, 2001, 3001, soon),
				testPendingAuctionWithStartTime(5002, 2001, 3002, later),
				testAuctionWithStartedAt(5003, 2001, 3003, sessionStart.Add(10*time.Minute)),
			},
		},
	}, nil, nil, nil, nil, nil, nil, 0)

	resp, err := handler.ListRoomAuctions(context.Background(), &auctionv1.ListRoomAuctionsRequest{
		RoomId:       2001,
		StartedAfter: timestamppb.New(sessionStart),
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		t.Fatalf("ListRoomAuctions returned error: %v", err)
	}
	if len(resp.GetList()) != 2 {
		t.Fatalf("expected 2 auctions, got %#v", resp.GetList())
	}
	for _, auction := range resp.GetList() {
		if auction.GetId() == 5002 {
			t.Fatalf("future pending auction should be hidden: %#v", resp.GetList())
		}
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

func testAuctionWithCreatedAt(id int64, roomID int64, goodsID int64, createdAt time.Time) *model.Auction {
	auction := testRunningAuction(id, roomID, goodsID)
	auction.CreatedAt = createdAt
	auction.UpdatedAt = createdAt
	auction.StartTime = &createdAt
	auction.EndTime = &createdAt
	return auction
}

func testAuctionWithStartedAt(id int64, roomID int64, goodsID int64, startedAt time.Time) *model.Auction {
	auction := testRunningAuction(id, roomID, goodsID)
	createdAt := startedAt.Add(-time.Hour)
	auction.CreatedAt = createdAt
	auction.UpdatedAt = createdAt
	auction.StartTime = &startedAt
	endAt := startedAt.Add(time.Minute)
	auction.EndTime = &endAt
	return auction
}

func testPendingAuctionWithStartTime(id int64, roomID int64, goodsID int64, startTime time.Time) *model.Auction {
	auction := testRunningAuction(id, roomID, goodsID)
	auction.Status = model.AuctionStatusPending
	auction.StartTime = &startTime
	auction.EndTime = nil
	createdAt := startTime.Add(-time.Hour)
	auction.CreatedAt = createdAt
	auction.UpdatedAt = createdAt
	return auction
}

func auctionBelongsToSession(auction *model.Auction, startedAfter time.Time, upcomingFrom *time.Time, upcomingBefore *time.Time) bool {
	if auction == nil {
		return false
	}
	if auction.Status == model.AuctionStatusPending {
		if auction.StartTime == nil || upcomingFrom == nil || upcomingBefore == nil {
			return false
		}
		return !auction.StartTime.Before(*upcomingFrom) && !auction.StartTime.After(*upcomingBefore)
	}
	return (auction.StartTime != nil && !auction.StartTime.Before(startedAfter)) ||
		(auction.EndTime != nil && !auction.EndTime.Before(startedAfter)) ||
		!auction.CreatedAt.Before(startedAfter)
}
