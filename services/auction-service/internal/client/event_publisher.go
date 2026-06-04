package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	rocketmq "github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/yayccc/livebid/services/auction-service/internal/config"
)

const (
	EventAuctionStarted     = "auction_started"
	EventBidAccepted        = "bid_accepted"
	EventAuctionFinished    = "auction_finished"
	EventAuctionFailed      = "auction_failed"
	EventAuctionCancelled   = "auction_cancelled"
	EventAuctionExpireCheck = "auction_expire_check"
)

type AuctionEvent struct {
	EventID   string         `json:"event_id"`
	EventType string         `json:"event_type"`
	BizID     string         `json:"biz_id"`
	AuctionID int64          `json:"auction_id"`
	ShopID    int64          `json:"shop_id"`
	Version   int64          `json:"version"`
	Timestamp int64          `json:"timestamp"`
	Data      map[string]any `json:"data,omitempty"`
}

type EventPublisher interface {
	Publish(ctx context.Context, event AuctionEvent) error
	PublishDelay(ctx context.Context, event AuctionEvent, delayLevel int) error
	Close() error
}

type RocketMQEventPublisher struct {
	producer rocketmq.Producer
	topic    string
}

func NewRocketMQEventPublisher(cfg config.RocketMQConfig) (*RocketMQEventPublisher, error) {
	nameServers, err := resolveNameServers(cfg.NameServers)
	if err != nil {
		return nil, err
	}
	p, err := rocketmq.NewProducer(
		producer.WithNameServer(nameServers),
		producer.WithGroupName(cfg.ProducerGroup),
		producer.WithQueueSelector(producer.NewHashQueueSelector()),
	)
	if err != nil {
		return nil, err
	}
	if err := p.Start(); err != nil {
		return nil, err
	}
	return &RocketMQEventPublisher{producer: p, topic: cfg.Topic}, nil
}

func resolveNameServers(addrs []string) ([]string, error) {
	resolved := make([]string, 0, len(addrs))
	for _, addr := range addrs {
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
			return nil, fmt.Errorf("resolve rocketmq nameserver %q: no ip found", host)
		}
		resolved = append(resolved, net.JoinHostPort(ips[0].String(), port))
	}
	return resolved, nil
}

func (p *RocketMQEventPublisher) Publish(ctx context.Context, event AuctionEvent) error {
	return p.send(ctx, event, 0)
}

func (p *RocketMQEventPublisher) PublishDelay(ctx context.Context, event AuctionEvent, delayLevel int) error {
	return p.send(ctx, event, delayLevel)
}

func (p *RocketMQEventPublisher) Close() error {
	return p.producer.Shutdown()
}

func (p *RocketMQEventPublisher) send(ctx context.Context, event AuctionEvent, delayLevel int) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	msg := primitive.NewMessage(p.topic, body)
	msg.WithTag(event.EventType)
	msg.WithKeys([]string{fmt.Sprintf("auction_%d", event.AuctionID)})
	msg.WithShardingKey(fmt.Sprintf("%d", event.AuctionID))
	if delayLevel > 0 {
		// RocketMQ delayLevel 是固定延迟等级，不是秒数；具体映射取决于 broker 配置。
		msg.WithDelayTimeLevel(delayLevel)
	}
	_, err = p.producer.SendSync(ctx, msg)
	return err
}

func NewAuctionEvent(eventID string, eventType string, auctionID int64, shopID int64, version int64, data map[string]any) AuctionEvent {
	return AuctionEvent{
		EventID:   eventID,
		EventType: eventType,
		BizID:     fmt.Sprintf("%d", auctionID),
		AuctionID: auctionID,
		ShopID:    shopID,
		Version:   version,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
}
