package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	rocketmq "github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/yayccc/livebid/services/auction-service/internal/config"
	"github.com/yayccc/livebid/services/auction-service/internal/model"
	"github.com/yayccc/livebid/services/auction-service/internal/repository"
	"go.uber.org/zap"
)

type EventConsumer interface {
	Start() error
	Close() error
}

type RocketMQEventConsumer struct {
	consumer  rocketmq.PushConsumer
	processor *AuctionEventProcessor
}

func NewRocketMQEventConsumer(cfg config.RocketMQConfig, processor *AuctionEventProcessor) (*RocketMQEventConsumer, error) {
	nameServers, err := resolveNameServers(cfg.NameServers)
	if err != nil {
		return nil, err
	}
	// 集群模式保证同一消费者组内单条消息只被一个实例处理；业务表版本和主键兜底重复投递。
	c, err := rocketmq.NewPushConsumer(
		consumer.WithGroupName(cfg.ConsumerGroup),
		consumer.WithNsResolver(primitive.NewPassthroughResolver(nameServers)),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithConsumeMessageBatchMaxSize(1),
		consumer.WithConsumeGoroutineNums(1),
		consumer.WithConsumerOrder(true),
		consumer.WithMaxReconsumeTimes(cfg.MaxReconsumeTimes),
	)
	if err != nil {
		return nil, err
	}
	if err := c.Subscribe(cfg.Topic, consumer.MessageSelector{Type: consumer.TAG, Expression: "*"}, processor.Consume); err != nil {
		return nil, err
	}
	return &RocketMQEventConsumer{consumer: c, processor: processor}, nil
}

func (c *RocketMQEventConsumer) Start() error {
	return c.consumer.Start()
}

func (c *RocketMQEventConsumer) Close() error {
	return c.consumer.Shutdown()
}

type AuctionEventProcessor struct {
	auctions          repository.AuctionRepository
	bids              repository.BidRecordRepository
	states            repository.AuctionStateStore
	events            EventPublisher
	log               *zap.Logger
	maxReconsumeTimes int32
}

func NewAuctionEventProcessor(
	auctions repository.AuctionRepository,
	bids repository.BidRecordRepository,
	states repository.AuctionStateStore,
	events EventPublisher,
	log *zap.Logger,
	maxReconsumeTimes int32,
) *AuctionEventProcessor {
	if maxReconsumeTimes <= 0 {
		maxReconsumeTimes = 5
	}
	return &AuctionEventProcessor{
		auctions:          auctions,
		bids:              bids,
		states:            states,
		events:            events,
		log:               log,
		maxReconsumeTimes: maxReconsumeTimes,
	}
}

func (p *AuctionEventProcessor) Consume(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for _, msg := range msgs {
		if err := p.consumeOne(ctx, msg); err != nil {
			if msg.ReconsumeTimes >= p.maxReconsumeTimes {
				// 超过最大重试后 ack，避免问题消息阻塞整个队列。
				p.log.Warn("auction event dropped after retries", zap.String("msg_id", msg.MsgId), zap.Error(err))
				return consumer.ConsumeSuccess, nil
			}
			return consumer.ConsumeRetryLater, err
		}
	}
	return consumer.ConsumeSuccess, nil
}

func (p *AuctionEventProcessor) consumeOne(ctx context.Context, msg *primitive.MessageExt) error {
	var event AuctionEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		return err
	}
	if event.EventID == "" || event.AuctionID <= 0 {
		return errors.New("竞拍事件格式无效")
	}
	return p.applyEvent(ctx, event)
}

func (p *AuctionEventProcessor) applyEvent(ctx context.Context, event AuctionEvent) error {
	switch event.EventType {
	case EventAuctionStarted, EventBidAccepted, EventAuctionFinished, EventAuctionFailed, EventAuctionCancelled:
		return p.applySnapshotEvent(ctx, event)
	case EventAuctionExpireCheck:
		return p.applyExpireCheck(ctx, event)
	default:
		return nil
	}
}

func (p *AuctionEventProcessor) applySnapshotEvent(ctx context.Context, event AuctionEvent) error {
	auction := auctionFromEvent(event)
	if event.EventType == EventBidAccepted {
		if record := bidRecordFromEvent(event); record != nil {
			// bid_record_id 由出价路径预生成，重复插入代表整条消息已经处理过，直接 ack。
			if err := p.bids.CreateIfNotExists(ctx, record); err != nil {
				if errors.Is(err, repository.ErrBidRecordDuplicated) {
					return nil
				}
				return err
			}
		}
	}
	err := p.auctions.UpdateStatusSnapshot(ctx, auction)
	if errors.Is(err, repository.ErrOutdatedAuctionVersion) {
		// 旧版本事件不能覆盖新状态，但可以认为消费成功。
		return nil
	}
	return err
}

func (p *AuctionEventProcessor) applyExpireCheck(ctx context.Context, event AuctionEvent) error {
	expireAt := int64FromData(event.Data, "expire_at")
	if expireAt <= 0 {
		return errors.New("竞拍延迟检查事件缺少 expire_at")
	}
	// 延迟消息只负责“检查”，真正结束仍由 Redis Lua 根据当前 version/expire_at 原子决定。
	state, changed, err := p.states.FinishExpiredAuction(ctx, event.AuctionID, event.Version, expireAt, time.Now())
	if err != nil {
		return err
	}
	if changed {
		return p.publishAndSnapshotFinalState(ctx, auctionFromState(state))
	}
	current, err := p.states.GetState(ctx, event.AuctionID)
	if err != nil {
		return nil
	}
	if current.Status == model.AuctionStatusDeal || current.Status == model.AuctionStatusFailed {
		return p.publishAndSnapshotFinalState(ctx, auctionFromState(current))
	}
	return nil
}

func (p *AuctionEventProcessor) publishAndSnapshotFinalState(ctx context.Context, auction *model.Auction) error {
	if auction == nil {
		return nil
	}
	if p.events != nil {
		eventType := EventAuctionFailed
		if auction.Status == model.AuctionStatusDeal {
			eventType = EventAuctionFinished
		}
		auctionEvent := NewAuctionEvent(
			fmt.Sprintf("auction_%d_%d", auction.ID, auction.Version),
			eventType,
			auction.ID,
			auction.ShopID,
			auction.RoomID,
			auction.Version,
			auctionEventDataFromState(auction),
		)
		if err := p.events.Publish(ctx, auctionEvent); err != nil {
			return err
		}
	}
	if err := p.auctions.UpdateStatusSnapshot(ctx, auction); errors.Is(err, repository.ErrOutdatedAuctionVersion) {
		// DB 已经被更新到更高版本时，延迟检查结果无需再次应用。
		return nil
	} else {
		return err
	}
}

func auctionFromEvent(event AuctionEvent) *model.Auction {
	auction := &model.Auction{
		ID:           event.AuctionID,
		GoodsID:      int64FromData(event.Data, "goods_id"),
		ShopID:       event.ShopID,
		RoomID:       event.RoomID,
		StartPrice:   int64FromData(event.Data, "start_price"),
		BidIncrement: int64FromData(event.Data, "bid_increment"),
		CurrentPrice: int64FromData(event.Data, "current_price"),
		BidCount:     int64FromData(event.Data, "bid_count"),
		Status:       model.AuctionStatus(int64FromData(event.Data, "status")),
		StartTime:    timePtrFromUnixMilli(event.Data, "start_time"),
		EndTime:      timePtrFromUnixMilli(event.Data, "end_time"),
		Version:      event.Version,
	}
	if value := int64FromData(event.Data, "seal_price"); value > 0 {
		auction.SealPrice = &value
	}
	if value := int64FromData(event.Data, "deal_price"); value > 0 {
		auction.DealPrice = &value
	}
	if value := int64FromData(event.Data, "winner_user_id"); value > 0 {
		auction.WinnerUserID = &value
	}
	return auction
}

func auctionFromState(state repository.AuctionState) *model.Auction {
	auction := &model.Auction{
		ID:           state.AuctionID,
		GoodsID:      state.GoodsID,
		ShopID:       state.ShopID,
		RoomID:       state.RoomID,
		StartPrice:   state.StartPrice,
		BidIncrement: state.BidIncrement,
		SealPrice:    state.SealPrice,
		CurrentPrice: state.CurrentPrice,
		BidCount:     state.BidCount,
		Status:       state.Status,
		StartTime:    &state.StartTime,
		EndTime:      &state.EndTime,
		WinnerUserID: state.WinnerUserID,
		Version:      state.Version,
	}
	if state.Status == model.AuctionStatusDeal {
		dealPrice := state.CurrentPrice
		auction.DealPrice = &dealPrice
	}
	return auction
}

func auctionEventDataFromState(auction *model.Auction) map[string]any {
	if auction == nil {
		return nil
	}
	data := map[string]any{
		"goods_id":       auction.GoodsID,
		"shop_id":        auction.ShopID,
		"room_id":        auction.RoomID,
		"start_price":    auction.StartPrice,
		"bid_increment":  auction.BidIncrement,
		"current_price":  auction.CurrentPrice,
		"bid_count":      auction.BidCount,
		"status":         int32(auction.Status),
		"start_time":     auctionTimeMillis(auction.StartTime),
		"end_time":       auctionTimeMillis(auction.EndTime),
		"winner_user_id": int64(0),
		"deal_price":     int64(0),
		"seal_price":     int64(0),
	}
	if auction.WinnerUserID != nil {
		data["winner_user_id"] = *auction.WinnerUserID
	}
	if auction.DealPrice != nil {
		data["deal_price"] = *auction.DealPrice
	}
	if auction.SealPrice != nil {
		data["seal_price"] = *auction.SealPrice
	}
	return data
}

func bidRecordFromEvent(event AuctionEvent) *model.BidRecord {
	id := int64FromData(event.Data, "bid_record_id")
	if id <= 0 {
		return nil
	}
	return &model.BidRecord{
		ID:        id,
		AuctionID: event.AuctionID,
		GoodsID:   int64FromData(event.Data, "goods_id"),
		ShopID:    event.ShopID,
		RoomID:    event.RoomID,
		UserID:    int64FromData(event.Data, "user_id"),
		BidPrice:  int64FromData(event.Data, "bid_price"),
		BidTime:   timeFromUnix(event.Data, "bid_time"),
	}
}

func int64FromData(data map[string]any, key string) int64 {
	value, ok := data[key]
	if !ok || value == nil {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return parsed
	default:
		return 0
	}
}

func timePtrFromUnixMilli(data map[string]any, key string) *time.Time {
	value := int64FromData(data, key)
	if value <= 0 {
		return nil
	}
	t := time.UnixMilli(value)
	return &t
}

func auctionTimeMillis(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.UnixMilli()
}

func timeFromUnix(data map[string]any, key string) time.Time {
	value := int64FromData(data, key)
	if value <= 0 {
		return time.Now()
	}
	return time.Unix(value, 0)
}
