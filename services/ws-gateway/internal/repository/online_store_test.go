package repository

import "testing"

func TestRedisKeys(t *testing.T) {
	if got := roomConnectionsKey(4001); got != "ws:room:4001:connections" {
		t.Fatalf("unexpected room connections key: %s", got)
	}
	if got := roomUsersKey(4001); got != "ws:room:4001:users" {
		t.Fatalf("unexpected room users key: %s", got)
	}
	if got := userConnectionsKey(4001, "u:9001"); got != "ws:room:4001:user:u:9001:connections" {
		t.Fatalf("unexpected user connections key: %s", got)
	}
	if got := roomEventsChannel(4001); got != "ws:room:4001:events" {
		t.Fatalf("unexpected room events channel: %s", got)
	}
}

func TestStatsFromScript(t *testing.T) {
	stats, err := statsFromScript(4001, []any{int64(2), int64(3), int64(1)})
	if err != nil {
		t.Fatalf("statsFromScript failed: %v", err)
	}
	if stats.RoomID != 4001 || stats.OnlineUserCount != 2 || stats.ConnectionCount != 3 || !stats.Changed {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}
