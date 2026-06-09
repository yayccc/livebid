package client

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/yayccc/livebid/services/auction-service/internal/config"
	"github.com/yayccc/livebid/services/auction-service/internal/model"
	"github.com/yayccc/livebid/services/auction-service/internal/repository"
	"go.uber.org/zap"
)

type AuctionExpireScanner struct {
	states    repository.AuctionStateStore
	processor *AuctionEventProcessor
	log       *zap.Logger
	interval  time.Duration
	batchSize int64
	stop      chan struct{}
	done      chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	mu        sync.Mutex
	started   bool
}

func NewAuctionExpireScanner(states repository.AuctionStateStore, processor *AuctionEventProcessor, cfg config.AuctionExpireScannerConfig, log *zap.Logger) *AuctionExpireScanner {
	if !cfg.Enabled || states == nil || processor == nil {
		return nil
	}
	interval := time.Duration(cfg.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	batchSize := int64(cfg.BatchSize)
	if batchSize <= 0 {
		batchSize = 100
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &AuctionExpireScanner{
		states:    states,
		processor: processor,
		log:       log,
		interval:  interval,
		batchSize: batchSize,
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
}

func (s *AuctionExpireScanner) Start(ctx context.Context) {
	s.startOnce.Do(func() {
		s.mu.Lock()
		s.started = true
		s.mu.Unlock()
		go s.run(ctx)
	})
}

func (s *AuctionExpireScanner) Close() {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		started := s.started
		s.mu.Unlock()
		if !started {
			return
		}
		close(s.stop)
		<-s.done
	})
}

func (s *AuctionExpireScanner) run(ctx context.Context) {
	defer close(s.done)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.scan(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stop:
			return
		case <-ticker.C:
			s.scan(ctx)
		}
	}
}

func (s *AuctionExpireScanner) scan(ctx context.Context) {
	now := time.Now()
	ids, err := s.states.ListExpiredAuctionIDs(ctx, now, s.batchSize)
	if err != nil {
		s.log.Warn("scan expired auctions failed", zap.Error(err))
		return
	}
	for _, auctionID := range ids {
		if err := s.handleExpiredCandidate(ctx, auctionID, now); err != nil {
			s.log.Warn("handle expired auction candidate failed", zap.Int64("auction_id", auctionID), zap.Error(err))
		}
	}
}

func (s *AuctionExpireScanner) handleExpiredCandidate(ctx context.Context, auctionID int64, now time.Time) error {
	state, err := s.states.GetState(ctx, auctionID)
	if err != nil {
		if errors.Is(err, repository.ErrAuctionNotFound) {
			return s.states.RemoveRunningAuction(ctx, auctionID)
		}
		return err
	}
	if state.Status != model.AuctionStatusRunning {
		return s.states.RemoveRunningAuction(ctx, auctionID)
	}
	if state.ExpireAt.After(now) {
		return s.states.IndexRunningAuction(ctx, auctionID, state.ExpireAt)
	}

	finalState, changed, err := s.states.FinishExpiredAuction(ctx, auctionID, state.Version, state.ExpireAt.UnixMilli(), now)
	if err != nil {
		return err
	}
	if !changed {
		current, err := s.states.GetState(ctx, auctionID)
		if err != nil {
			if errors.Is(err, repository.ErrAuctionNotFound) {
				return s.states.RemoveRunningAuction(ctx, auctionID)
			}
			return err
		}
		if current.Status == model.AuctionStatusRunning {
			return s.states.IndexRunningAuction(ctx, auctionID, current.ExpireAt)
		}
		return s.states.RemoveRunningAuction(ctx, auctionID)
	}
	return s.processor.publishAndSnapshotFinalState(ctx, auctionFromState(finalState))
}
