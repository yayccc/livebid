package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yayccc/livebid/services/ws-gateway/internal/config"
)

const (
	DanmakuSenderTypeUser  = "user"
	DanmakuContentTypeText = "text"
	DanmakuStatusVisible   = "visible"
	RoomStatusLiving       = "living"
	RoomStatusNotLive      = "not_live"
)

type DanmakuData struct {
	MessageID   string `json:"message_id"`
	SenderType  string `json:"sender_type"`
	UserID      int64  `json:"user_id,omitempty"`
	Nickname    string `json:"nickname,omitempty"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	Status      string `json:"status,omitempty"`
}

type DanmakuBroadcast struct {
	Type       string      `json:"type"`
	EventID    string      `json:"event_id"`
	RoomID     int64       `json:"room_id"`
	ServerTime int64       `json:"server_time"`
	Data       DanmakuData `json:"data"`
}

type RecentDanmaku struct {
	MessageID   string `json:"message_id"`
	SenderType  string `json:"sender_type"`
	UserID      int64  `json:"user_id,omitempty"`
	Nickname    string `json:"nickname,omitempty"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	Status      string `json:"status,omitempty"`
	ServerTime  int64  `json:"server_time"`
}

type DanmakuIdempotentResult struct {
	MessageID string `json:"message_id"`
	RoomID    int64  `json:"room_id"`
}

type DanmakuCreatedEvent struct {
	EventID     string `json:"event_id"`
	EventType   string `json:"event_type"`
	MessageID   string `json:"message_id"`
	RoomID      int64  `json:"room_id"`
	UserID      int64  `json:"user_id"`
	Nickname    string `json:"nickname"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	ServerTime  int64  `json:"server_time"`
}

type DanmakuStore interface {
	GetRoomStatus(ctx context.Context, roomID int64) (string, bool, error)
	SetRoomStatus(ctx context.Context, roomID int64, status string, ttl time.Duration) error
	GetIdempotent(ctx context.Context, roomID int64, userID int64, requestID string) (DanmakuIdempotentResult, bool, error)
	SetIdempotent(ctx context.Context, roomID int64, userID int64, requestID string, result DanmakuIdempotentResult, ttl time.Duration) error
	AllowSend(ctx context.Context, roomID int64, userID int64, interval time.Duration) (bool, error)
	GetNickname(ctx context.Context, userID int64) (string, bool, error)
	SetNickname(ctx context.Context, userID int64, nickname string, ttl time.Duration) error
	AppendRecent(ctx context.Context, roomID int64, message DanmakuBroadcast, limit int, ttl time.Duration) error
	GetRecent(ctx context.Context, roomID int64, limit int) ([]RecentDanmaku, error)
	PublishDanmaku(ctx context.Context, message DanmakuBroadcast) error
	SubscribeDanmaku(ctx context.Context, handle func(context.Context, DanmakuBroadcast)) error
	PublishAIInput(ctx context.Context, event DanmakuCreatedEvent) error
	Close() error
}

type RedisDanmakuStore struct {
	client      *redis.Client
	aiInputMode string
}

func NewRedisDanmakuStore(cfg config.RedisConfig, aiInputMode string) *RedisDanmakuStore {
	aiInputMode = strings.ToLower(strings.TrimSpace(aiInputMode))
	if aiInputMode != "" && aiInputMode != "redis_pubsub" {
		aiInputMode = "redis_pubsub"
	}
	return &RedisDanmakuStore{
		client: redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		}),
		aiInputMode: aiInputMode,
	}
}

func (s *RedisDanmakuStore) GetRoomStatus(ctx context.Context, roomID int64) (string, bool, error) {
	value, err := s.client.Get(ctx, roomStatusKey(roomID)).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (s *RedisDanmakuStore) SetRoomStatus(ctx context.Context, roomID int64, status string, ttl time.Duration) error {
	return s.client.Set(ctx, roomStatusKey(roomID), status, ttl).Err()
}

func (s *RedisDanmakuStore) GetIdempotent(ctx context.Context, roomID int64, userID int64, requestID string) (DanmakuIdempotentResult, bool, error) {
	value, err := s.client.Get(ctx, danmakuRequestKey(roomID, userID, requestID)).Bytes()
	if err == redis.Nil {
		return DanmakuIdempotentResult{}, false, nil
	}
	if err != nil {
		return DanmakuIdempotentResult{}, false, err
	}
	var result DanmakuIdempotentResult
	if err := json.Unmarshal(value, &result); err != nil {
		return DanmakuIdempotentResult{}, false, err
	}
	return result, true, nil
}

func (s *RedisDanmakuStore) SetIdempotent(ctx context.Context, roomID int64, userID int64, requestID string, result DanmakuIdempotentResult, ttl time.Duration) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, danmakuRequestKey(roomID, userID, requestID), payload, ttl).Err()
}

func (s *RedisDanmakuStore) AllowSend(ctx context.Context, roomID int64, userID int64, interval time.Duration) (bool, error) {
	if interval <= 0 {
		interval = time.Second
	}
	return s.client.SetNX(ctx, danmakuRateKey(roomID, userID), "1", interval).Result()
}

func (s *RedisDanmakuStore) GetNickname(ctx context.Context, userID int64) (string, bool, error) {
	value, err := s.client.Get(ctx, nicknameKey(userID)).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return strings.TrimSpace(value), true, nil
}

func (s *RedisDanmakuStore) SetNickname(ctx context.Context, userID int64, nickname string, ttl time.Duration) error {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		return nil
	}
	return s.client.Set(ctx, nicknameKey(userID), nickname, ttl).Err()
}

func (s *RedisDanmakuStore) AppendRecent(ctx context.Context, roomID int64, message DanmakuBroadcast, limit int, ttl time.Duration) error {
	if limit <= 0 {
		limit = 10
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	key := recentDanmakuKey(roomID)
	pipe := s.client.TxPipeline()
	pipe.LPush(ctx, key, payload)
	pipe.LTrim(ctx, key, 0, int64(limit-1))
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisDanmakuStore) GetRecent(ctx context.Context, roomID int64, limit int) ([]RecentDanmaku, error) {
	if limit <= 0 {
		limit = 10
	}
	values, err := s.client.LRange(ctx, recentDanmakuKey(roomID), 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	recent := make([]RecentDanmaku, 0, len(values))
	for _, value := range values {
		var message DanmakuBroadcast
		if err := json.Unmarshal([]byte(value), &message); err != nil {
			continue
		}
		if message.RoomID <= 0 || message.Data.MessageID == "" {
			continue
		}
		recent = append(recent, RecentDanmaku{
			MessageID:   message.Data.MessageID,
			SenderType:  message.Data.SenderType,
			UserID:      message.Data.UserID,
			Nickname:    message.Data.Nickname,
			Content:     message.Data.Content,
			ContentType: message.Data.ContentType,
			Status:      message.Data.Status,
			ServerTime:  message.ServerTime,
		})
	}
	sort.SliceStable(recent, func(i, j int) bool {
		if recent[i].ServerTime == recent[j].ServerTime {
			return recent[i].MessageID < recent[j].MessageID
		}
		return recent[i].ServerTime < recent[j].ServerTime
	})
	return recent, nil
}

func (s *RedisDanmakuStore) PublishDanmaku(ctx context.Context, message DanmakuBroadcast) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return s.client.Publish(ctx, danmakuChannel(message.RoomID), payload).Err()
}

func (s *RedisDanmakuStore) SubscribeDanmaku(ctx context.Context, handle func(context.Context, DanmakuBroadcast)) error {
	pubsub := s.client.PSubscribe(ctx, "ws:room:*:danmaku")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			var event DanmakuBroadcast
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				continue
			}
			if event.RoomID > 0 && event.Data.MessageID != "" && handle != nil {
				handle(ctx, event)
			}
		}
	}
}

func (s *RedisDanmakuStore) PublishAIInput(ctx context.Context, event DanmakuCreatedEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return s.client.Publish(ctx, aiDanmakuChannel(event.RoomID), payload).Err()
}

func (s *RedisDanmakuStore) Close() error {
	return s.client.Close()
}

func roomStatusKey(roomID int64) string {
	return fmt.Sprintf("ws:room:%d:live_status", roomID)
}

func danmakuRequestKey(roomID int64, userID int64, requestID string) string {
	return fmt.Sprintf("ws:room:%d:danmaku:req:%d:%s", roomID, userID, requestID)
}

func danmakuRateKey(roomID int64, userID int64) string {
	return fmt.Sprintf("ws:room:%d:danmaku:rate:user:%d", roomID, userID)
}

func nicknameKey(userID int64) string {
	return fmt.Sprintf("ws:user:nickname:%d", userID)
}

func recentDanmakuKey(roomID int64) string {
	return fmt.Sprintf("ws:room:%d:danmaku:recent", roomID)
}

func danmakuChannel(roomID int64) string {
	return fmt.Sprintf("ws:room:%d:danmaku", roomID)
}

func aiDanmakuChannel(roomID int64) string {
	return fmt.Sprintf("ws:room:%d:ai:danmaku", roomID)
}
