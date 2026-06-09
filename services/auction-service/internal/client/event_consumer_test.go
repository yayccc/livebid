package client

import (
	"context"
	"testing"
	"time"

	"github.com/yayccc/livebid/services/auction-service/internal/model"
	"github.com/yayccc/livebid/services/auction-service/internal/repository"
	"go.uber.org/zap"
)

type fakeAuctionStateStore struct {
	state   repository.AuctionState
	changed bool
}

func (s *fakeAuctionStateStore) LoadAuction(ctx context.Context, auction *model.Auction, now time.Time) error {
	return nil
}

func (s *fakeAuctionStateStore) PlaceBid(ctx context.Context, auctionID int64, roomID int64, userID int64, bidPrice int64, requestID string, bidRecordID int64, now time.Time) (repository.BidResult, error) {
	return repository.BidResult{}, nil
}

func (s *fakeAuctionStateStore) FinishAuction(ctx context.Context, auctionID int64, now time.Time) (repository.AuctionState, error) {
	return repository.AuctionState{}, nil
}

func (s *fakeAuctionStateStore) FinishExpiredAuction(ctx context.Context, auctionID int64, version int64, expireAt int64, now time.Time) (repository.AuctionState, bool, error) {
	return s.state, s.changed, nil
}

func (s *fakeAuctionStateStore) CancelAuction(ctx context.Context, auctionID int64, now time.Time) (repository.AuctionState, error) {
	return repository.AuctionState{}, nil
}

func (s *fakeAuctionStateStore) GetState(ctx context.Context, auctionID int64) (repository.AuctionState, error) {
	return s.state, nil
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

type fakeAuctionRepository struct {
	updated []*model.Auction
}

func (r *fakeAuctionRepository) Create(ctx context.Context, auction *model.Auction) error { return nil }
func (r *fakeAuctionRepository) FindByID(ctx context.Context, id int64) (*model.Auction, error) {
	return nil, repository.ErrAuctionNotFound
}
func (r *fakeAuctionRepository) FindByIDForShop(ctx context.Context, id int64, shopID int64) (*model.Auction, error) {
	return nil, repository.ErrAuctionNotFound
}
func (r *fakeAuctionRepository) FindByGoodsID(ctx context.Context, goodsID int64) (*model.Auction, error) {
	return nil, repository.ErrAuctionNotFound
}
func (r *fakeAuctionRepository) FindCurrentByRoomID(ctx context.Context, roomID int64) (*model.Auction, error) {
	return nil, repository.ErrAuctionNotFound
}
func (r *fakeAuctionRepository) BatchFindCurrentByRoomIDs(ctx context.Context, roomIDs []int64) ([]*model.Auction, error) {
	return nil, nil
}
func (r *fakeAuctionRepository) ListByRoom(ctx context.Context, filter repository.ListAuctionFilter) ([]*model.Auction, int64, error) {
	return nil, 0, nil
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
	r.updated = append(r.updated, auction)
	return nil
}
func (r *fakeAuctionRepository) Delete(ctx context.Context, id int64, shopID int64) error { return nil }

type fakeBidRepository struct{}

func (r *fakeBidRepository) Create(ctx context.Context, record *model.BidRecord) error { return nil }
func (r *fakeBidRepository) CreateIfNotExists(ctx context.Context, record *model.BidRecord) error {
	return nil
}
func (r *fakeBidRepository) ListByAuction(ctx context.Context, filter repository.ListBidRecordFilter) ([]*model.BidRecord, int64, error) {
	return nil, 0, nil
}
func (r *fakeBidRepository) CountByShopBetween(ctx context.Context, shopID int64, start time.Time, end time.Time) (int64, error) {
	return 0, nil
}

type fakeEventPublisher struct {
	events []AuctionEvent
}

func (p *fakeEventPublisher) Publish(ctx context.Context, event AuctionEvent) error {
	p.events = append(p.events, event)
	return nil
}

func (p *fakeEventPublisher) PublishDelay(ctx context.Context, event AuctionEvent, delayLevel int) error {
	return nil
}

func (p *fakeEventPublisher) Close() error { return nil }

func TestAuctionEventProcessorPublishesFinalEventOnExpireCheck(t *testing.T) {
	state := repository.AuctionState{
		AuctionID:    1001,
		GoodsID:      2001,
		ShopID:       3001,
		RoomID:       4001,
		StartPrice:   10000,
		BidIncrement: 1000,
		CurrentPrice: 15000,
		BidCount:     3,
		Status:       model.AuctionStatusDeal,
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      time.Now(),
		WinnerUserID: int64Ptr(9001),
		ExpireAt:     time.Now(),
		Version:      8,
	}
	auctionRepo := &fakeAuctionRepository{}
	bidRepo := &fakeBidRepository{}
	stateStore := &fakeAuctionStateStore{state: state, changed: false}
	publisher := &fakeEventPublisher{}
	processor := NewAuctionEventProcessor(auctionRepo, bidRepo, stateStore, publisher, zap.NewNop(), 5)

	err := processor.applyExpireCheck(context.Background(), AuctionEvent{
		EventID:   "evt_1",
		EventType: EventAuctionExpireCheck,
		AuctionID: 1001,
		ShopID:    3001,
		RoomID:    4001,
		Version:   8,
		Data: map[string]any{
			"expire_at": state.ExpireAt.UnixMilli(),
		},
	})
	if err != nil {
		t.Fatalf("applyExpireCheck returned error: %v", err)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(publisher.events))
	}
	if publisher.events[0].EventType != EventAuctionFinished {
		t.Fatalf("expected auction_finished, got %s", publisher.events[0].EventType)
	}
	if len(auctionRepo.updated) != 1 {
		t.Fatalf("expected 1 snapshot update, got %d", len(auctionRepo.updated))
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
