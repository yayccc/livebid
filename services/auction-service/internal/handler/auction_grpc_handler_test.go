package handler

import (
	"context"
	"testing"
	"time"

	auctionv1 "github.com/yayccc/livebid/gen/proto/auction/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"github.com/yayccc/livebid/services/auction-service/internal/client"
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

type fakeBidRepository struct {
	created []*model.BidRecord
}

func (r *fakeBidRepository) Create(ctx context.Context, record *model.BidRecord) error {
	r.created = append(r.created, record)
	return nil
}

func (r *fakeBidRepository) CreateIfNotExists(ctx context.Context, record *model.BidRecord) error {
	return r.Create(ctx, record)
}

func (r *fakeBidRepository) ListByAuction(ctx context.Context, filter repository.ListBidRecordFilter) ([]*model.BidRecord, int64, error) {
	return nil, 0, nil
}

func (r *fakeBidRepository) CountByShopBetween(ctx context.Context, shopID int64, start time.Time, end time.Time) (int64, error) {
	return 0, nil
}

type fakeAuctionStateStore struct {
	result repository.BidResult
}

func (s *fakeAuctionStateStore) LoadAuction(ctx context.Context, auction *model.Auction, now time.Time) error {
	return nil
}

func (s *fakeAuctionStateStore) PlaceBid(ctx context.Context, auctionID int64, roomID int64, userID int64, bidPrice int64, requestID string, bidRecordID int64, now time.Time) (repository.BidResult, error) {
	result := s.result
	result.BidRecordID = bidRecordID
	result.BidTime = now
	result.State.AuctionID = auctionID
	result.State.RoomID = roomID
	result.State.WinnerUserID = &userID
	result.State.CurrentPrice = bidPrice
	return result, nil
}

func (s *fakeAuctionStateStore) FinishAuction(ctx context.Context, auctionID int64, now time.Time) (repository.AuctionState, error) {
	return repository.AuctionState{}, nil
}

func (s *fakeAuctionStateStore) FinishExpiredAuction(ctx context.Context, auctionID int64, version int64, expireAt int64, now time.Time) (repository.AuctionState, bool, error) {
	return repository.AuctionState{}, false, nil
}

func (s *fakeAuctionStateStore) CancelAuction(ctx context.Context, auctionID int64, now time.Time) (repository.AuctionState, error) {
	return repository.AuctionState{}, nil
}

func (s *fakeAuctionStateStore) GetState(ctx context.Context, auctionID int64) (repository.AuctionState, error) {
	return s.result.State, nil
}

func (s *fakeAuctionStateStore) ListExpiredAuctionIDs(ctx context.Context, now time.Time, limit int64) ([]int64, error) {
	return nil, nil
}

func (s *fakeAuctionStateStore) IndexRunningAuction(ctx context.Context, auctionID int64, expireAt time.Time) error {
	return nil
}

func (s *fakeAuctionStateStore) RemoveRunningAuction(ctx context.Context, auctionID int64) error {
	return nil
}

func (s *fakeAuctionStateStore) Close() error { return nil }

type fakeEventPublisher struct {
	events      []client.AuctionEvent
	delayEvents []client.AuctionEvent
	delayLevels []int
}

func (p *fakeEventPublisher) Publish(ctx context.Context, event client.AuctionEvent) error {
	p.events = append(p.events, event)
	return nil
}

func (p *fakeEventPublisher) PublishDelay(ctx context.Context, event client.AuctionEvent, delayLevel int) error {
	p.delayEvents = append(p.delayEvents, event)
	p.delayLevels = append(p.delayLevels, delayLevel)
	return nil
}

func (p *fakeEventPublisher) Close() error { return nil }

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

func TestAuctionGRPCHandlerPlaceBidPublishesCountdownFields(t *testing.T) {
	expireAt := time.Now().Add(30 * time.Second).Truncate(time.Millisecond)
	publisher := &fakeEventPublisher{}
	state := repository.AuctionState{
		AuctionID:    5001,
		GoodsID:      3001,
		ShopID:       1001,
		RoomID:       2001,
		StartPrice:   10000,
		BidIncrement: 1000,
		CurrentPrice: 18000,
		BidCount:     8,
		Status:       model.AuctionStatusRunning,
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      time.Now().Add(time.Hour),
		ExpireAt:     expireAt,
		Version:      12,
	}
	handler := NewAuctionGRPCHandler(
		&fakeAuctionRepository{},
		&fakeBidRepository{},
		&fakeAuctionStateStore{result: repository.BidResult{State: state}},
		nil,
		nil,
		publisher,
		nil,
		4,
	)

	ctx := identity.NewContext(context.Background(), identity.Principal{Kind: identity.KindUser, ID: 4001})
	resp, err := handler.PlaceBid(ctx, &auctionv1.PlaceBidRequest{
		AuctionId: 5001,
		RoomId:    2001,
		BidPrice:  18000,
		RequestId: "bid-req-1",
	})
	if err != nil {
		t.Fatalf("PlaceBid returned error: %v", err)
	}
	if resp.GetExpireAt().AsTime().UnixMilli() != expireAt.UnixMilli() {
		t.Fatalf("unexpected response expire_at: %v", resp.GetExpireAt())
	}
	if len(publisher.events) != 1 {
		t.Fatalf("expected one bid_accepted event, got %d", len(publisher.events))
	}
	event := publisher.events[0]
	if event.EventType != client.EventBidAccepted {
		t.Fatalf("expected bid_accepted event, got %s", event.EventType)
	}
	if asInt64(event.Data["expire_at"]) != expireAt.UnixMilli() || asInt64(event.Data["server_time"]) <= 0 {
		t.Fatalf("expected countdown fields in event data: %#v", event.Data)
	}
	if len(publisher.delayEvents) != 1 || asInt64(publisher.delayEvents[0].Data["expire_at"]) != expireAt.UnixMilli() {
		t.Fatalf("expected delay expire_at event, got %#v", publisher.delayEvents)
	}
}

func TestAuctionGRPCHandlerStartAuctionRejectsRoomRunningAuction(t *testing.T) {
	pending := testPendingAuctionWithStartTime(5001, 2001, 3001, time.Now().Add(time.Minute))
	handler := NewAuctionGRPCHandler(&fakeAuctionRepository{
		auctionsByID: map[int64]*model.Auction{
			5001: pending,
		},
		auctionsByRoomID: map[int64][]*model.Auction{
			2001: {testRunningAuction(5002, 2001, 3002)},
		},
	}, nil, &fakeAuctionStateStore{}, nil, nil, nil, nil, 0)

	ctx := identity.NewContext(context.Background(), identity.Principal{Kind: identity.KindShop, ID: 1001})
	_, err := handler.StartAuction(ctx, &auctionv1.StartAuctionRequest{Id: 5001})
	if err == nil {
		t.Fatal("expected room running auction error")
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

func asInt64(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int32:
		return int64(typed)
	case int:
		return int64(typed)
	default:
		return 0
	}
}
