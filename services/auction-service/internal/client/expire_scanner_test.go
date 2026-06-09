package client

import (
	"context"
	"testing"
	"time"

	"github.com/yayccc/livebid/services/auction-service/internal/config"
	"github.com/yayccc/livebid/services/auction-service/internal/model"
	"github.com/yayccc/livebid/services/auction-service/internal/repository"
	"go.uber.org/zap"
)

type scannerStateStore struct {
	ids       []int64
	state     repository.AuctionState
	changed   bool
	indexed   []int64
	removed   []int64
	finished  []int64
	lastLimit int64
}

func (s *scannerStateStore) LoadAuction(ctx context.Context, auction *model.Auction, now time.Time) error {
	return nil
}

func (s *scannerStateStore) PlaceBid(ctx context.Context, auctionID int64, roomID int64, userID int64, bidPrice int64, requestID string, bidRecordID int64, now time.Time) (repository.BidResult, error) {
	return repository.BidResult{}, nil
}

func (s *scannerStateStore) FinishAuction(ctx context.Context, auctionID int64, now time.Time) (repository.AuctionState, error) {
	return repository.AuctionState{}, nil
}

func (s *scannerStateStore) FinishExpiredAuction(ctx context.Context, auctionID int64, version int64, expireAt int64, now time.Time) (repository.AuctionState, bool, error) {
	s.finished = append(s.finished, auctionID)
	state := s.state
	if s.changed && state.Status == model.AuctionStatusRunning {
		if state.BidCount > 0 {
			state.Status = model.AuctionStatusDeal
		} else {
			state.Status = model.AuctionStatusFailed
		}
		state.EndTime = now
		state.ExpireAt = now
		state.Version++
	}
	return state, s.changed, nil
}

func (s *scannerStateStore) CancelAuction(ctx context.Context, auctionID int64, now time.Time) (repository.AuctionState, error) {
	return repository.AuctionState{}, nil
}

func (s *scannerStateStore) GetState(ctx context.Context, auctionID int64) (repository.AuctionState, error) {
	return s.state, nil
}

func (s *scannerStateStore) ListExpiredAuctionIDs(ctx context.Context, now time.Time, limit int64) ([]int64, error) {
	s.lastLimit = limit
	return s.ids, nil
}

func (s *scannerStateStore) IndexRunningAuction(ctx context.Context, auctionID int64, expireAt time.Time) error {
	s.indexed = append(s.indexed, auctionID)
	return nil
}

func (s *scannerStateStore) RemoveRunningAuction(ctx context.Context, auctionID int64) error {
	s.removed = append(s.removed, auctionID)
	return nil
}

func (s *scannerStateStore) Close() error { return nil }

func TestAuctionExpireScannerFinishesExpiredAuction(t *testing.T) {
	state := scannerExpiredState()
	store := &scannerStateStore{
		ids:     []int64{state.AuctionID},
		state:   state,
		changed: true,
	}
	repo := &fakeAuctionRepository{}
	publisher := &fakeEventPublisher{}
	processor := NewAuctionEventProcessor(repo, &fakeBidRepository{}, store, publisher, zap.NewNop(), 5)
	scanner := NewAuctionExpireScanner(store, processor, config.AuctionExpireScannerConfig{
		Enabled:         true,
		IntervalSeconds: 5,
		BatchSize:       20,
	}, zap.NewNop())

	scanner.scan(context.Background())

	if store.lastLimit != 20 || len(store.finished) != 1 || store.finished[0] != state.AuctionID {
		t.Fatalf("expected expired auction to be finished, store=%#v", store)
	}
	if len(repo.updated) != 1 || repo.updated[0].Status != model.AuctionStatusDeal {
		t.Fatalf("expected final snapshot update, got %#v", repo.updated)
	}
	if len(publisher.events) != 1 || publisher.events[0].EventType != EventAuctionFinished {
		t.Fatalf("expected auction_finished event, got %#v", publisher.events)
	}
}

func TestAuctionExpireScannerReindexesFreshCandidate(t *testing.T) {
	state := scannerExpiredState()
	state.ExpireAt = time.Now().Add(time.Minute)
	store := &scannerStateStore{
		ids:   []int64{state.AuctionID},
		state: state,
	}
	processor := NewAuctionEventProcessor(&fakeAuctionRepository{}, &fakeBidRepository{}, store, &fakeEventPublisher{}, zap.NewNop(), 5)
	scanner := NewAuctionExpireScanner(store, processor, config.AuctionExpireScannerConfig{
		Enabled:         true,
		IntervalSeconds: 5,
		BatchSize:       100,
	}, zap.NewNop())

	scanner.scan(context.Background())

	if len(store.finished) != 0 || len(store.indexed) != 1 || store.indexed[0] != state.AuctionID {
		t.Fatalf("expected fresh candidate reindexed, store=%#v", store)
	}
}

func scannerExpiredState() repository.AuctionState {
	winner := int64(4001)
	now := time.Now()
	return repository.AuctionState{
		AuctionID:    5001,
		GoodsID:      3001,
		ShopID:       1001,
		RoomID:       2001,
		StartPrice:   10000,
		BidIncrement: 1000,
		CurrentPrice: 18000,
		BidCount:     3,
		Status:       model.AuctionStatusRunning,
		StartTime:    now.Add(-time.Hour),
		EndTime:      now,
		WinnerUserID: &winner,
		ExpireAt:     now.Add(-time.Second),
		Version:      12,
	}
}
