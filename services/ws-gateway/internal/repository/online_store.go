package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yayccc/livebid/services/ws-gateway/internal/config"
)

type ConnectionState struct {
	ConnectionID string
	InstanceID   string
	RoomID       int64
	IdentityKey  string
	UserID       int64
	ConnectedAt  int64
	LastSeenAt   int64
	ExpireAt     int64
}

type OnlineStats struct {
	RoomID          int64
	OnlineUserCount int64
	ConnectionCount int64
	Changed         bool
}

type OnlineEvent struct {
	EventID             string `json:"event_id"`
	RoomID              int64  `json:"room_id"`
	ServerTime          int64  `json:"server_time"`
	OnlineUserCount     int64  `json:"online_user_count"`
	ConnectionCount     int64  `json:"connection_count"`
	ChangedIdentityKey  string `json:"changed_identity_key,omitempty"`
	ChangedConnectionID string `json:"changed_connection_id,omitempty"`
}

type OnlineStore interface {
	Enter(ctx context.Context, state ConnectionState) (OnlineStats, error)
	Heartbeat(ctx context.Context, state ConnectionState) (OnlineStats, error)
	Leave(ctx context.Context, roomID int64, identityKey string, connectionID string) (OnlineStats, error)
	CleanupRoom(ctx context.Context, roomID int64, limit int64) (OnlineStats, error)
	GetStats(ctx context.Context, roomID int64) (OnlineStats, error)
	PublishOnlineEvent(ctx context.Context, event OnlineEvent) error
	SubscribeOnlineEvents(ctx context.Context, handle func(context.Context, OnlineEvent)) error
	Close() error
}

type RedisOnlineStore struct {
	client        *redis.Client
	ttl           time.Duration
	connKeyPrefix string
}

func NewRedisOnlineStore(cfg config.RedisConfig, ttl time.Duration) *RedisOnlineStore {
	if ttl <= 0 {
		ttl = 90 * time.Second
	}
	return &RedisOnlineStore{
		client: redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		}),
		ttl:           ttl,
		connKeyPrefix: "ws:conn:",
	}
}

func (s *RedisOnlineStore) Enter(ctx context.Context, state ConnectionState) (OnlineStats, error) {
	return s.runConnectionScript(ctx, enterScript, state)
}

func (s *RedisOnlineStore) Heartbeat(ctx context.Context, state ConnectionState) (OnlineStats, error) {
	return s.runConnectionScript(ctx, heartbeatScript, state)
}

func (s *RedisOnlineStore) Leave(ctx context.Context, roomID int64, identityKey string, connectionID string) (OnlineStats, error) {
	now := time.Now().UnixMilli()
	result, err := leaveScript.Run(ctx, s.client, []string{
		connKey(connectionID),
		roomConnectionsKey(roomID),
		roomUsersKey(roomID),
		userConnectionsKey(roomID, identityKey),
	}, connectionID, strconv.FormatInt(roomID, 10), identityKey, now, s.connKeyPrefix).Result()
	if err != nil {
		return OnlineStats{}, err
	}
	return statsFromScript(roomID, result)
}

func (s *RedisOnlineStore) CleanupRoom(ctx context.Context, roomID int64, limit int64) (OnlineStats, error) {
	if limit <= 0 {
		limit = 256
	}
	now := time.Now().UnixMilli()
	result, err := cleanupRoomScript.Run(ctx, s.client, []string{
		roomConnectionsKey(roomID),
		roomUsersKey(roomID),
	}, strconv.FormatInt(roomID, 10), now, limit, s.connKeyPrefix).Result()
	if err != nil {
		return OnlineStats{}, err
	}
	return statsFromScript(roomID, result)
}

func (s *RedisOnlineStore) GetStats(ctx context.Context, roomID int64) (OnlineStats, error) {
	now := time.Now().UnixMilli()
	result, err := getStatsScript.Run(ctx, s.client, []string{
		roomConnectionsKey(roomID),
		roomUsersKey(roomID),
	}, now).Result()
	if err != nil {
		return OnlineStats{}, err
	}
	return statsFromScript(roomID, result)
}

func (s *RedisOnlineStore) PublishOnlineEvent(ctx context.Context, event OnlineEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return s.client.Publish(ctx, roomEventsChannel(event.RoomID), payload).Err()
}

func (s *RedisOnlineStore) SubscribeOnlineEvents(ctx context.Context, handle func(context.Context, OnlineEvent)) error {
	pubsub := s.client.PSubscribe(ctx, "ws:room:*:events")
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
			var event OnlineEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				continue
			}
			if event.RoomID > 0 && handle != nil {
				handle(ctx, event)
			}
		}
	}
}

func (s *RedisOnlineStore) Close() error {
	return s.client.Close()
}

func (s *RedisOnlineStore) runConnectionScript(ctx context.Context, script *redis.Script, state ConnectionState) (OnlineStats, error) {
	result, err := script.Run(ctx, s.client, []string{
		connKey(state.ConnectionID),
		roomConnectionsKey(state.RoomID),
		roomUsersKey(state.RoomID),
		userConnectionsKey(state.RoomID, state.IdentityKey),
	},
		state.ConnectionID,
		state.InstanceID,
		strconv.FormatInt(state.RoomID, 10),
		state.IdentityKey,
		state.UserID,
		state.ConnectedAt,
		state.LastSeenAt,
		state.ExpireAt,
		int64(s.ttl.Seconds()),
		time.Now().UnixMilli(),
		s.connKeyPrefix,
	).Result()
	if err != nil {
		return OnlineStats{}, err
	}
	return statsFromScript(state.RoomID, result)
}

func statsFromScript(roomID int64, result any) (OnlineStats, error) {
	values, ok := result.([]any)
	if !ok || len(values) < 3 {
		return OnlineStats{}, fmt.Errorf("unexpected redis script result: %T", result)
	}
	online, err := toInt64(values[0])
	if err != nil {
		return OnlineStats{}, err
	}
	connections, err := toInt64(values[1])
	if err != nil {
		return OnlineStats{}, err
	}
	changed, err := toInt64(values[2])
	if err != nil {
		return OnlineStats{}, err
	}
	return OnlineStats{
		RoomID:          roomID,
		OnlineUserCount: online,
		ConnectionCount: connections,
		Changed:         changed == 1,
	}, nil
}

func toInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case int:
		return int64(typed), nil
	case string:
		return strconv.ParseInt(typed, 10, 64)
	case []byte:
		return strconv.ParseInt(string(typed), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected integer value: %T", value)
	}
}

func connKey(connectionID string) string {
	return "ws:conn:" + connectionID
}

func roomConnectionsKey(roomID int64) string {
	return fmt.Sprintf("ws:room:%d:connections", roomID)
}

func roomUsersKey(roomID int64) string {
	return fmt.Sprintf("ws:room:%d:users", roomID)
}

func userConnectionsKey(roomID int64, identityKey string) string {
	return fmt.Sprintf("ws:room:%d:user:%s:connections", roomID, identityKey)
}

func roomEventsChannel(roomID int64) string {
	return fmt.Sprintf("ws:room:%d:events", roomID)
}

const recomputeIdentityLua = `
local function recompute_identity(room_id, identity_key, now_ms, conn_prefix)
  local room_conn_key = 'ws:room:' .. room_id .. ':connections'
  local room_users_key = 'ws:room:' .. room_id .. ':users'
  local user_conns_key = 'ws:room:' .. room_id .. ':user:' .. identity_key .. ':connections'
  local conns = redis.call('SMEMBERS', user_conns_key)
  local max_expire_at = 0
  for _, connection_id in ipairs(conns) do
    local conn_key = conn_prefix .. connection_id
    local conn_room_id = redis.call('HGET', conn_key, 'room_id')
    local conn_identity_key = redis.call('HGET', conn_key, 'identity_key')
    local expire_at = tonumber(redis.call('HGET', conn_key, 'expire_at') or '0')
    if conn_room_id ~= room_id or conn_identity_key ~= identity_key or expire_at <= now_ms then
      redis.call('SREM', user_conns_key, connection_id)
      redis.call('ZREM', room_conn_key, connection_id)
      if expire_at <= now_ms then
        redis.call('DEL', conn_key)
      end
    else
      if expire_at > max_expire_at then
        max_expire_at = expire_at
      end
    end
  end
  if max_expire_at > 0 then
    redis.call('ZADD', room_users_key, max_expire_at, identity_key)
  else
    redis.call('ZREM', room_users_key, identity_key)
    redis.call('DEL', user_conns_key)
  end
end
`

// heartbeatScript is a Redis script for handling a connection heartbeat,
// which includes updating the connection's last seen time and expire time,
// and recomputing the online stats if necessary.
var enterScript = redis.NewScript(recomputeIdentityLua + `
local conn_key = KEYS[1]
local room_conn_key = KEYS[2]
local room_users_key = KEYS[3]
local user_conns_key = KEYS[4]
local connection_id = ARGV[1]
local instance_id = ARGV[2]
local room_id = ARGV[3]
local identity_key = ARGV[4]
local user_id = ARGV[5]
local connected_at = ARGV[6]
local last_seen_at = ARGV[7]
local expire_at = tonumber(ARGV[8])
local ttl_seconds = tonumber(ARGV[9])
local now_ms = tonumber(ARGV[10])
local conn_prefix = ARGV[11]

local old_online = redis.call('ZCARD', room_users_key)
local old_connections = redis.call('ZCARD', room_conn_key)
redis.call('ZREMRANGEBYSCORE', room_conn_key, '-inf', now_ms)
redis.call('ZREMRANGEBYSCORE', room_users_key, '-inf', now_ms)

redis.call('HSET', conn_key,
  'connection_id', connection_id,
  'instance_id', instance_id,
  'room_id', room_id,
  'identity_key', identity_key,
  'user_id', user_id,
  'connected_at', connected_at,
  'last_seen_at', last_seen_at,
  'expire_at', expire_at
)
redis.call('EXPIRE', conn_key, ttl_seconds)
redis.call('ZADD', room_conn_key, expire_at, connection_id)
redis.call('SADD', user_conns_key, connection_id)
redis.call('EXPIRE', user_conns_key, ttl_seconds)
recompute_identity(room_id, identity_key, now_ms, conn_prefix)

local new_online = redis.call('ZCARD', room_users_key)
local new_connections = redis.call('ZCARD', room_conn_key)
local changed = 0
if old_online ~= new_online or old_connections ~= new_connections then
  changed = 1
end
return {new_online, new_connections, changed}
`)

// heartbeatScript is a Redis script for handling a connection heartbeat,
// which updates the connection's last seen time and expiration,
// and recomputes the online stats if necessary.
var heartbeatScript = redis.NewScript(recomputeIdentityLua + `
local conn_key = KEYS[1]
local room_conn_key = KEYS[2]
local room_users_key = KEYS[3]
local user_conns_key = KEYS[4]
local connection_id = ARGV[1]
local instance_id = ARGV[2]
local room_id = ARGV[3]
local identity_key = ARGV[4]
local user_id = ARGV[5]
local connected_at = ARGV[6]
local last_seen_at = ARGV[7]
local expire_at = tonumber(ARGV[8])
local ttl_seconds = tonumber(ARGV[9])
local now_ms = tonumber(ARGV[10])
local conn_prefix = ARGV[11]

local old_online = redis.call('ZCARD', room_users_key)
local old_connections = redis.call('ZCARD', room_conn_key)
redis.call('ZREMRANGEBYSCORE', room_conn_key, '-inf', now_ms)
redis.call('ZREMRANGEBYSCORE', room_users_key, '-inf', now_ms)

local current_room_id = redis.call('HGET', conn_key, 'room_id')
local current_identity_key = redis.call('HGET', conn_key, 'identity_key')
if current_room_id == room_id and current_identity_key == identity_key then
  redis.call('HSET', conn_key,
    'instance_id', instance_id,
    'user_id', user_id,
    'connected_at', connected_at,
    'last_seen_at', last_seen_at,
    'expire_at', expire_at
  )
  redis.call('EXPIRE', conn_key, ttl_seconds)
  redis.call('ZADD', room_conn_key, expire_at, connection_id)
  redis.call('SADD', user_conns_key, connection_id)
  redis.call('EXPIRE', user_conns_key, ttl_seconds)
end
recompute_identity(room_id, identity_key, now_ms, conn_prefix)

local new_online = redis.call('ZCARD', room_users_key)
local new_connections = redis.call('ZCARD', room_conn_key)
local changed = 0
if old_online ~= new_online or old_connections ~= new_connections then
  changed = 1
end
return {new_online, new_connections, changed}
`)

// leaveScript is a Redis script for handling a connection leaving a room,
// which includes cleaning up the connection and user data,
// and recomputing the online stats.
var leaveScript = redis.NewScript(recomputeIdentityLua + `
local conn_key = KEYS[1]
local room_conn_key = KEYS[2]
local room_users_key = KEYS[3]
local user_conns_key = KEYS[4]
local connection_id = ARGV[1]
local room_id = ARGV[2]
local identity_key = ARGV[3]
local now_ms = tonumber(ARGV[4])
local conn_prefix = ARGV[5]

local old_online = redis.call('ZCARD', room_users_key)
local old_connections = redis.call('ZCARD', room_conn_key)
redis.call('ZREMRANGEBYSCORE', room_conn_key, '-inf', now_ms)
redis.call('ZREMRANGEBYSCORE', room_users_key, '-inf', now_ms)

redis.call('DEL', conn_key)
redis.call('ZREM', room_conn_key, connection_id)
redis.call('SREM', user_conns_key, connection_id)
recompute_identity(room_id, identity_key, now_ms, conn_prefix)

local new_online = redis.call('ZCARD', room_users_key)
local new_connections = redis.call('ZCARD', room_conn_key)
local changed = 0
if old_online ~= new_online or old_connections ~= new_connections then
  changed = 1
end
return {new_online, new_connections, changed}
`)

// cleanupRoomScript is a Redis script for cleaning up a room's connections and users.
var cleanupRoomScript = redis.NewScript(recomputeIdentityLua + `
local room_conn_key = KEYS[1]
local room_users_key = KEYS[2]
local room_id = ARGV[1]
local now_ms = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local conn_prefix = ARGV[4]

local old_online = redis.call('ZCARD', room_users_key)
local old_connections = redis.call('ZCARD', room_conn_key)
local expired = redis.call('ZRANGEBYSCORE', room_conn_key, '-inf', now_ms, 'LIMIT', 0, limit)
local affected = {}
local affected_seen = {}

for _, connection_id in ipairs(expired) do
  local conn_key = conn_prefix .. connection_id
  local identity_key = redis.call('HGET', conn_key, 'identity_key')
  if identity_key ~= false and affected_seen[identity_key] == nil then
    table.insert(affected, identity_key)
    affected_seen[identity_key] = true
  end
  redis.call('DEL', conn_key)
  redis.call('ZREM', room_conn_key, connection_id)
  if identity_key ~= false then
    local user_conns_key = 'ws:room:' .. room_id .. ':user:' .. identity_key .. ':connections'
    redis.call('SREM', user_conns_key, connection_id)
  end
end

redis.call('ZREMRANGEBYSCORE', room_users_key, '-inf', now_ms)
for _, identity_key in ipairs(affected) do
  recompute_identity(room_id, identity_key, now_ms, conn_prefix)
end

local new_online = redis.call('ZCARD', room_users_key)
local new_connections = redis.call('ZCARD', room_conn_key)
local changed = 0
if old_online ~= new_online or old_connections ~= new_connections then
  changed = 1
end
return {new_online, new_connections, changed}
`)

// getStatsScript is a Redis script for getting the current online stats of a room,
// while also cleaning up expired connections and users.
var getStatsScript = redis.NewScript(`
local room_conn_key = KEYS[1]
local room_users_key = KEYS[2]
local now_ms = tonumber(ARGV[1])
redis.call('ZREMRANGEBYSCORE', room_conn_key, '-inf', now_ms)
redis.call('ZREMRANGEBYSCORE', room_users_key, '-inf', now_ms)
return {redis.call('ZCARD', room_users_key), redis.call('ZCARD', room_conn_key), 0}
`)
