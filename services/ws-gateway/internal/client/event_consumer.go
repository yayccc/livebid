package client

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	rocketmq "github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/yayccc/livebid/services/ws-gateway/internal/config"
	"github.com/yayccc/livebid/services/ws-gateway/internal/handler"
	"go.uber.org/zap"
)

type EventConsumer interface {
	Start() error
	Close() error
}

type AuctionEventSink interface {
	BroadcastAuctionEvent(ctx context.Context, event handler.AuctionEvent)
}

type RocketMQAuctionEventConsumer struct {
	consumer rocketmq.PushConsumer
	log      *zap.Logger
}

func NewRocketMQAuctionEventConsumer(cfg config.RocketMQConfig, sink AuctionEventSink, log *zap.Logger) (*RocketMQAuctionEventConsumer, error) {
	nameServers, err := resolveNameServers(cfg.NameServers)
	if err != nil {
		return nil, err
	}
	processor := newAuctionEventProcessor(sink, log, cfg.MaxReconsumeTimes)
	c, err := rocketmq.NewPushConsumer(
		consumer.WithGroupName(cfg.ConsumerGroup),
		consumer.WithNsResolver(primitive.NewPassthroughResolver(nameServers)),
		consumer.WithConsumerModel(consumer.BroadCasting),
		consumer.WithConsumeMessageBatchMaxSize(1),
		consumer.WithConsumeGoroutineNums(2),
		consumer.WithMaxReconsumeTimes(cfg.MaxReconsumeTimes),
	)
	if err != nil {
		return nil, err
	}
	selector := consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: strings.Join([]string{handler.EventAuctionStarted, handler.EventBidAccepted, handler.EventAuctionFinished, handler.EventAuctionFailed, handler.EventAuctionCancelled}, " || "),
	}
	if err := c.Subscribe(cfg.Topic, selector, processor.Consume); err != nil {
		return nil, err
	}
	return &RocketMQAuctionEventConsumer{consumer: c, log: log}, nil
}

func (c *RocketMQAuctionEventConsumer) Start() error {
	return c.consumer.Start()
}

func (c *RocketMQAuctionEventConsumer) Close() error {
	return c.consumer.Shutdown()
}

type auctionEventProcessor struct {
	sink              AuctionEventSink
	log               *zap.Logger
	maxReconsumeTimes int32
	seen              *eventDeduper
}

func newAuctionEventProcessor(sink AuctionEventSink, log *zap.Logger, maxReconsumeTimes int32) *auctionEventProcessor {
	if maxReconsumeTimes <= 0 {
		maxReconsumeTimes = 3
	}
	return &auctionEventProcessor{
		sink:              sink,
		log:               log,
		maxReconsumeTimes: maxReconsumeTimes,
		seen:              newEventDeduper(10 * time.Minute),
	}
}

func (p *auctionEventProcessor) Consume(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for _, msg := range msgs {
		if err := p.consumeOne(ctx, msg); err != nil {
			if msg.ReconsumeTimes >= p.maxReconsumeTimes {
				if p.log != nil {
					p.log.Warn("auction websocket event dropped after retries", zap.String("msg_id", msg.MsgId), zap.Error(err))
				}
				return consumer.ConsumeSuccess, nil
			}
			return consumer.ConsumeRetryLater, err
		}
	}
	return consumer.ConsumeSuccess, nil
}

func (p *auctionEventProcessor) consumeOne(ctx context.Context, msg *primitive.MessageExt) error {
	event, err := handler.DecodeAuctionEvent(msg.Body)
	if err != nil {
		return err
	}
	if !event.Broadcastable() {
		return nil
	}
	if p.seen.Seen(event.EventID) {
		return nil
	}
	if p.sink != nil {
		p.sink.BroadcastAuctionEvent(ctx, event)
	}
	return nil
}

type eventDeduper struct {
	mu   sync.Mutex
	ttl  time.Duration
	seen map[string]time.Time
}

func newEventDeduper(ttl time.Duration) *eventDeduper {
	return &eventDeduper{
		ttl:  ttl,
		seen: make(map[string]time.Time),
	}
}

func (d *eventDeduper) Seen(eventID string) bool {
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	for id, expiresAt := range d.seen {
		if !expiresAt.After(now) {
			delete(d.seen, id)
		}
	}
	if expiresAt, ok := d.seen[eventID]; ok && expiresAt.After(now) {
		return true
	}
	d.seen[eventID] = now.Add(d.ttl)
	return false
}

func resolveNameServers(addrs []string) ([]string, error) {
	resolved := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		if net.ParseIP(host) != nil {
			resolved = append(resolved, addr)
			continue
		}
		ips, err := net.LookupIP(host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			resolved = append(resolved, addr)
			continue
		}
		resolved = append(resolved, net.JoinHostPort(ips[0].String(), port))
	}
	return resolved, nil
}
