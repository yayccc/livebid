package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yayccc/livebid/services/auction-service/internal/config"
	"github.com/yayccc/livebid/services/auction-service/internal/model"
)

var (
	ErrDuplicateBidRequest = errors.New("重复出价请求，请勿重复提交")
	ErrBidTooLow           = errors.New("出价金额过低，必须不低于当前价加固定加价幅度")
	ErrBidOverSealPrice    = errors.New("出价金额超过封顶价")
	ErrAuctionExpired      = errors.New("竞拍已结束或不在可出价时间范围内")
)

type AuctionState struct {
	AuctionID    int64
	GoodsID      int64
	ShopID       int64
	StartPrice   int64
	BidIncrement int64
	SealPrice    *int64
	CurrentPrice int64
	BidCount     int64
	Status       model.AuctionStatus
	StartTime    time.Time
	EndTime      time.Time
	WinnerUserID *int64
	ExpireAt     time.Time
	Version      int64
}

type BidResult struct {
	State       AuctionState
	BidRecordID int64
	BidTime     time.Time
}

type AuctionStateStore interface {
	LoadAuction(ctx context.Context, auction *model.Auction, now time.Time) error
	PlaceBid(ctx context.Context, auctionID int64, userID int64, bidPrice int64, requestID string, bidRecordID int64, now time.Time) (BidResult, error)
	FinishAuction(ctx context.Context, auctionID int64, now time.Time) (AuctionState, error)
	FinishExpiredAuction(ctx context.Context, auctionID int64, version int64, expireAt int64, now time.Time) (AuctionState, bool, error)
	CancelAuction(ctx context.Context, auctionID int64, now time.Time) (AuctionState, error)
	GetState(ctx context.Context, auctionID int64) (AuctionState, error)
	Close() error
}

type RedisAuctionStateStore struct {
	client *redis.Client
}

func NewRedisAuctionStateStore(cfg config.RedisConfig) *RedisAuctionStateStore {
	return &RedisAuctionStateStore{
		client: redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		}),
	}
}

func (s *RedisAuctionStateStore) LoadAuction(ctx context.Context, auction *model.Auction, now time.Time) error {
	// 开始竞拍时把 DB 配置加载到 Redis，后续出价都以 Redis 运行态为准。
	startTime := now
	if auction.StartTime != nil {
		startTime = *auction.StartTime
	}
	endTime := startTime.Add(24 * time.Hour)
	if auction.EndTime != nil {
		endTime = *auction.EndTime
	}
	values := map[string]any{
		"auction_id":     auction.ID,
		"goods_id":       auction.GoodsID,
		"shop_id":        auction.ShopID,
		"start_price":    auction.StartPrice,
		"bid_increment":  auction.BidIncrement,
		"seal_price":     int64(0),
		"current_price":  auction.CurrentPrice,
		"bid_count":      auction.BidCount,
		"status":         int64(model.AuctionStatusRunning),
		"start_time":     startTime.UnixMilli(),
		"end_time":       endTime.UnixMilli(),
		"winner_user_id": int64(0),
		"expire_at":      endTime.UnixMilli(),
		"version":        auction.Version,
	}
	if auction.SealPrice != nil {
		values["seal_price"] = *auction.SealPrice
	}
	if auction.WinnerUserID != nil {
		values["winner_user_id"] = *auction.WinnerUserID
	}
	return s.client.HSet(ctx, stateKey(auction.ID), values).Err()
}

func (s *RedisAuctionStateStore) PlaceBid(ctx context.Context, auctionID int64, userID int64, bidPrice int64, requestID string, bidRecordID int64, now time.Time) (BidResult, error) {
	// Lua 脚本同时完成请求幂等、金额校验、最高价更新和排行榜更新。
	result, err := placeBidScript.Run(ctx, s.client, []string{
		stateKey(auctionID),
		bidRequestKey(auctionID, requestID),
		lastBidKey(auctionID, userID),
		rankKey(auctionID),
	}, now.UnixMilli(), userID, bidPrice, bidRecordID).Result()
	if err != nil {
		return BidResult{}, mapRedisScriptError(err)
	}
	values, ok := result.([]any)
	if !ok || len(values) < 15 {
		return BidResult{}, errors.New("Redis 出价结果格式异常")
	}
	state, err := stateFromScript(values[:14])
	if err != nil {
		return BidResult{}, err
	}
	return BidResult{
		State:       state,
		BidRecordID: asInt64(values[14]),
		BidTime:     now,
	}, nil
}

func (s *RedisAuctionStateStore) FinishAuction(ctx context.Context, auctionID int64, now time.Time) (AuctionState, error) {
	result, err := finishAuctionScript.Run(ctx, s.client, []string{stateKey(auctionID)}, now.UnixMilli()).Result()
	if err != nil {
		return AuctionState{}, mapRedisScriptError(err)
	}
	values, ok := result.([]any)
	if !ok || len(values) < 14 {
		return AuctionState{}, errors.New("Redis 结束竞拍结果格式异常")
	}
	return stateFromScript(values[:14])
}

func (s *RedisAuctionStateStore) FinishExpiredAuction(ctx context.Context, auctionID int64, version int64, expireAt int64, now time.Time) (AuctionState, bool, error) {
	// 延迟检查消息可能过期；只有 Redis 当前 version/expire_at 仍匹配时才真正结束。
	result, err := finishExpiredAuctionScript.Run(ctx, s.client, []string{stateKey(auctionID)}, now.UnixMilli(), version, expireAt).Result()
	if err != nil {
		return AuctionState{}, false, mapRedisScriptError(err)
	}
	values, ok := result.([]any)
	if !ok || len(values) == 0 {
		return AuctionState{}, false, errors.New("Redis 延迟结束检查结果格式异常")
	}
	if asInt64(values[0]) == 0 {
		return AuctionState{}, false, nil
	}
	if len(values) < 15 {
		return AuctionState{}, false, errors.New("Redis 延迟结束状态格式异常")
	}
	state, err := stateFromScript(values[1:15])
	return state, true, err
}

func (s *RedisAuctionStateStore) CancelAuction(ctx context.Context, auctionID int64, now time.Time) (AuctionState, error) {
	result, err := cancelAuctionScript.Run(ctx, s.client, []string{stateKey(auctionID)}, now.UnixMilli()).Result()
	if err != nil {
		return AuctionState{}, mapRedisScriptError(err)
	}
	values, ok := result.([]any)
	if !ok || len(values) < 14 {
		return AuctionState{}, errors.New("Redis 取消竞拍结果格式异常")
	}
	return stateFromScript(values[:14])
}

func (s *RedisAuctionStateStore) GetState(ctx context.Context, auctionID int64) (AuctionState, error) {
	values, err := s.client.HGetAll(ctx, stateKey(auctionID)).Result()
	if err != nil {
		return AuctionState{}, err
	}
	if len(values) == 0 {
		return AuctionState{}, ErrAuctionNotFound
	}
	return stateFromMap(values)
}

func (s *RedisAuctionStateStore) Close() error {
	return s.client.Close()
}

// placeBidScript 是出价路径的并发控制核心，所有校验和状态更新必须保持原子性。
var placeBidScript = redis.NewScript(`
local state_key = KEYS[1]
local req_key = KEYS[2]
local last_bid_key = KEYS[3]
local rank_key = KEYS[4]
local now = tonumber(ARGV[1])
local user_id = tonumber(ARGV[2])
local bid_price = tonumber(ARGV[3])
local bid_record_id = tonumber(ARGV[4])

if redis.call("EXISTS", state_key) == 0 then return redis.error_reply("auction not found") end
if redis.call("SET", req_key, "1", "NX", "EX", 600) == false then return redis.error_reply("duplicate bid request") end

local status = tonumber(redis.call("HGET", state_key, "status"))
local start_time = tonumber(redis.call("HGET", state_key, "start_time"))
local end_time = tonumber(redis.call("HGET", state_key, "end_time"))
if status ~= 1 then return redis.error_reply("invalid auction state") end
if now < start_time or now > end_time then return redis.error_reply("auction expired") end

local current_price = tonumber(redis.call("HGET", state_key, "current_price"))
local bid_increment = tonumber(redis.call("HGET", state_key, "bid_increment"))
local seal_price = tonumber(redis.call("HGET", state_key, "seal_price") or "0")
if bid_price < current_price + bid_increment then return redis.error_reply("bid price too low") end
if seal_price > 0 and bid_price > seal_price then return redis.error_reply("bid price over seal price") end

local bid_count = tonumber(redis.call("HINCRBY", state_key, "bid_count", 1))
local version = tonumber(redis.call("HINCRBY", state_key, "version", 1))
redis.call("HSET", state_key, "current_price", bid_price, "winner_user_id", user_id, "expire_at", now + 15000)
redis.call("SET", last_bid_key, bid_price, "EX", 86400)
redis.call("ZADD", rank_key, bid_price, user_id)

return {
  redis.call("HGET", state_key, "auction_id"),
  redis.call("HGET", state_key, "goods_id"),
  redis.call("HGET", state_key, "shop_id"),
  redis.call("HGET", state_key, "start_price"),
  redis.call("HGET", state_key, "bid_increment"),
  redis.call("HGET", state_key, "seal_price"),
  bid_price,
  bid_count,
  status,
  start_time,
  end_time,
  user_id,
  version,
  redis.call("HGET", state_key, "expire_at"),
  bid_record_id
}
`)

var finishAuctionScript = redis.NewScript(`
local state_key = KEYS[1]
local now = tonumber(ARGV[1])
if redis.call("EXISTS", state_key) == 0 then return redis.error_reply("auction not found") end
local status = tonumber(redis.call("HGET", state_key, "status"))
if status ~= 1 then return redis.error_reply("invalid auction state") end
local bid_count = tonumber(redis.call("HGET", state_key, "bid_count"))
local current_price = tonumber(redis.call("HGET", state_key, "current_price"))
local winner_user_id = tonumber(redis.call("HGET", state_key, "winner_user_id") or "0")
local next_status = 3
if bid_count > 0 then next_status = 2 end
local version = tonumber(redis.call("HINCRBY", state_key, "version", 1))
redis.call("HSET", state_key, "status", next_status, "end_time", now, "expire_at", now)
return {
  redis.call("HGET", state_key, "auction_id"),
  redis.call("HGET", state_key, "goods_id"),
  redis.call("HGET", state_key, "shop_id"),
  redis.call("HGET", state_key, "start_price"),
  redis.call("HGET", state_key, "bid_increment"),
  redis.call("HGET", state_key, "seal_price"),
  current_price,
  bid_count,
  next_status,
  redis.call("HGET", state_key, "start_time"),
  now,
  winner_user_id,
  version,
  now
}
`)

// finishExpiredAuctionScript 让延迟消息具备“检查而非强制结束”的语义。
var finishExpiredAuctionScript = redis.NewScript(`
local state_key = KEYS[1]
local now = tonumber(ARGV[1])
local expected_version = tonumber(ARGV[2])
local expected_expire_at = tonumber(ARGV[3])
if redis.call("EXISTS", state_key) == 0 then return redis.error_reply("auction not found") end
local status = tonumber(redis.call("HGET", state_key, "status"))
local version_now = tonumber(redis.call("HGET", state_key, "version"))
local expire_at_now = tonumber(redis.call("HGET", state_key, "expire_at"))
if status ~= 1 or version_now ~= expected_version or expire_at_now ~= expected_expire_at or expire_at_now > now then
  return {0}
end
local bid_count = tonumber(redis.call("HGET", state_key, "bid_count"))
local current_price = tonumber(redis.call("HGET", state_key, "current_price"))
local winner_user_id = tonumber(redis.call("HGET", state_key, "winner_user_id") or "0")
local next_status = 3
if bid_count > 0 then next_status = 2 end
local next_version = tonumber(redis.call("HINCRBY", state_key, "version", 1))
redis.call("HSET", state_key, "status", next_status, "end_time", now, "expire_at", now)
return {
  1,
  redis.call("HGET", state_key, "auction_id"),
  redis.call("HGET", state_key, "goods_id"),
  redis.call("HGET", state_key, "shop_id"),
  redis.call("HGET", state_key, "start_price"),
  redis.call("HGET", state_key, "bid_increment"),
  redis.call("HGET", state_key, "seal_price"),
  current_price,
  bid_count,
  next_status,
  redis.call("HGET", state_key, "start_time"),
  now,
  winner_user_id,
  next_version,
  now
}
`)

var cancelAuctionScript = redis.NewScript(`
local state_key = KEYS[1]
local now = tonumber(ARGV[1])
if redis.call("EXISTS", state_key) == 0 then return redis.error_reply("auction not found") end
local status = tonumber(redis.call("HGET", state_key, "status"))
if status ~= 0 and status ~= 1 then return redis.error_reply("invalid auction state") end
local version = tonumber(redis.call("HINCRBY", state_key, "version", 1))
redis.call("HSET", state_key, "status", 4, "end_time", now, "expire_at", now)
return {
  redis.call("HGET", state_key, "auction_id"),
  redis.call("HGET", state_key, "goods_id"),
  redis.call("HGET", state_key, "shop_id"),
  redis.call("HGET", state_key, "start_price"),
  redis.call("HGET", state_key, "bid_increment"),
  redis.call("HGET", state_key, "seal_price"),
  redis.call("HGET", state_key, "current_price"),
  redis.call("HGET", state_key, "bid_count"),
  4,
  redis.call("HGET", state_key, "start_time"),
  now,
  redis.call("HGET", state_key, "winner_user_id"),
  version,
  now
}
`)

func stateFromScript(values []any) (AuctionState, error) {
	state := AuctionState{
		AuctionID:    asInt64(values[0]),
		GoodsID:      asInt64(values[1]),
		ShopID:       asInt64(values[2]),
		StartPrice:   asInt64(values[3]),
		BidIncrement: asInt64(values[4]),
		CurrentPrice: asInt64(values[6]),
		BidCount:     asInt64(values[7]),
		Status:       model.AuctionStatus(asInt64(values[8])),
		StartTime:    time.UnixMilli(asInt64(values[9])),
		EndTime:      time.UnixMilli(asInt64(values[10])),
		Version:      asInt64(values[12]),
		ExpireAt:     time.UnixMilli(asInt64(values[13])),
	}
	if seal := asInt64(values[5]); seal > 0 {
		state.SealPrice = &seal
	}
	if winner := asInt64(values[11]); winner > 0 {
		state.WinnerUserID = &winner
	}
	return state, nil
}

func stateFromMap(values map[string]string) (AuctionState, error) {
	state := AuctionState{
		AuctionID:    parseInt(values["auction_id"]),
		GoodsID:      parseInt(values["goods_id"]),
		ShopID:       parseInt(values["shop_id"]),
		StartPrice:   parseInt(values["start_price"]),
		BidIncrement: parseInt(values["bid_increment"]),
		CurrentPrice: parseInt(values["current_price"]),
		BidCount:     parseInt(values["bid_count"]),
		Status:       model.AuctionStatus(parseInt(values["status"])),
		StartTime:    time.UnixMilli(parseInt(values["start_time"])),
		EndTime:      time.UnixMilli(parseInt(values["end_time"])),
		ExpireAt:     time.UnixMilli(parseInt(values["expire_at"])),
		Version:      parseInt(values["version"]),
	}
	if seal := parseInt(values["seal_price"]); seal > 0 {
		state.SealPrice = &seal
	}
	if winner := parseInt(values["winner_user_id"]); winner > 0 {
		state.WinnerUserID = &winner
	}
	return state, nil
}

func mapRedisScriptError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case contains(err, "auction not found"):
		return ErrAuctionNotFound
	case contains(err, "duplicate bid request"):
		return ErrDuplicateBidRequest
	case contains(err, "invalid auction state"):
		return ErrInvalidAuctionState
	case contains(err, "bid price too low"):
		return ErrBidTooLow
	case contains(err, "bid price over seal price"):
		return ErrBidOverSealPrice
	case contains(err, "auction expired"):
		return ErrAuctionExpired
	default:
		return err
	}
}

func contains(err error, text string) bool {
	return err != nil && strings.Contains(err.Error(), text)
}

func asInt64(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case string:
		return parseInt(typed)
	case []byte:
		return parseInt(string(typed))
	default:
		return 0
	}
}

func parseInt(value string) int64 {
	parsed, _ := strconv.ParseInt(value, 10, 64)
	return parsed
}

func stateKey(auctionID int64) string {
	return fmt.Sprintf("auction:%d:state", auctionID)
}

func bidRequestKey(auctionID int64, requestID string) string {
	return fmt.Sprintf("auction:%d:bid:req:%s", auctionID, requestID)
}

func lastBidKey(auctionID int64, userID int64) string {
	return fmt.Sprintf("auction:%d:user:%d:last_bid", auctionID, userID)
}

func rankKey(auctionID int64) string {
	return fmt.Sprintf("auction:%d:rank", auctionID)
}
