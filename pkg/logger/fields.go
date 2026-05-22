package logger

import "go.uber.org/zap"

const (
	FieldService   = "service"
	FieldEnv       = "env"
	FieldRequestID = "request_id"
	FieldTraceID   = "trace_id"
	FieldUserID    = "user_id"
	FieldRoomID    = "room_id"
	FieldAuctionID = "auction_id"
)

func String(key, value string) zap.Field {
	return zap.String(key, value)
}

func Int64(key string, value int64) zap.Field {
	return zap.Int64(key, value)
}

func Uint64(key string, value uint64) zap.Field {
	return zap.Uint64(key, value)
}

func Error(err error) zap.Field {
	return zap.Error(err)
}

func RequestID(value string) zap.Field {
	return zap.String(FieldRequestID, value)
}

func TraceID(value string) zap.Field {
	return zap.String(FieldTraceID, value)
}

func UserID(value int64) zap.Field {
	return zap.Int64(FieldUserID, value)
}

func RoomID(value int64) zap.Field {
	return zap.Int64(FieldRoomID, value)
}

func AuctionID(value int64) zap.Field {
	return zap.Int64(FieldAuctionID, value)
}
