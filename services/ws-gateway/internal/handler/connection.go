package handler

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Connection struct {
	ID          string
	InstanceID  string
	RoomID      int64
	UserID      int64
	Nickname    string
	IdentityKey string
	GuestID     string
	ConnectedAt int64

	ws          *websocket.Conn
	writeQueue  chan []byte
	onMessage   func(context.Context, *Connection, ClientMessage)
	onPong      func(context.Context, *Connection)
	onClose     func(context.Context, *Connection)
	log         *zap.Logger
	readTimeout time.Duration
	cancel      context.CancelFunc
	closeOnce   sync.Once
	closeSignal chan struct{}
}

type ConnectionOptions struct {
	ID          string
	InstanceID  string
	RoomID      int64
	UserID      int64
	Nickname    string
	IdentityKey string
	GuestID     string
	ConnectedAt int64
	WriteQueue  int
	OnMessage   func(context.Context, *Connection, ClientMessage)
	OnPong      func(context.Context, *Connection)
	OnClose     func(context.Context, *Connection)
	Log         *zap.Logger
}

func NewConnection(ws *websocket.Conn, opts ConnectionOptions) *Connection {
	if opts.WriteQueue <= 0 {
		opts.WriteQueue = 128
	}
	return &Connection{
		ID:          opts.ID,
		InstanceID:  opts.InstanceID,
		RoomID:      opts.RoomID,
		UserID:      opts.UserID,
		Nickname:    opts.Nickname,
		IdentityKey: opts.IdentityKey,
		GuestID:     opts.GuestID,
		ConnectedAt: opts.ConnectedAt,
		ws:          ws,
		writeQueue:  make(chan []byte, opts.WriteQueue),
		onMessage:   opts.OnMessage,
		onPong:      opts.OnPong,
		onClose:     opts.OnClose,
		log:         opts.Log,
		closeSignal: make(chan struct{}),
	}
}

func (c *Connection) Start(ctx context.Context, readLimit int64, heartbeatTimeout time.Duration, pingInterval time.Duration) {
	connCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	if heartbeatTimeout <= 0 {
		heartbeatTimeout = 60 * time.Second
	}
	if pingInterval <= 0 {
		pingInterval = heartbeatTimeout / 2
	}
	if pingInterval <= 0 {
		pingInterval = 30 * time.Second
	}
	c.readTimeout = heartbeatTimeout
	c.ws.SetReadLimit(readLimit)
	_ = c.ws.SetReadDeadline(time.Now().Add(heartbeatTimeout))
	c.ws.SetPongHandler(func(string) error {
		if err := c.ws.SetReadDeadline(time.Now().Add(heartbeatTimeout)); err != nil {
			return err
		}
		if c.onPong != nil {
			c.onPong(connCtx, c)
		}
		return nil
	})

	go c.writeLoop(connCtx, pingInterval)
	c.readLoop(connCtx)
}

func (c *Connection) Send(message any) bool {
	payload, err := json.Marshal(message)
	if err != nil {
		if c.log != nil {
			c.log.Warn("marshal websocket message failed", zap.Error(err))
		}
		return false
	}
	return c.SendRaw(payload)
}

func (c *Connection) SendRaw(payload []byte) bool {
	select {
	case <-c.closeSignal:
		return false
	case c.writeQueue <- payload:
		return true
	default:
		if c.log != nil {
			c.log.Warn("websocket write queue full", zap.String("connection_id", c.ID), zap.Int64("room_id", c.RoomID))
		}
		c.Close()
		return false
	}
}

func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		if c.cancel != nil {
			c.cancel()
		}
		close(c.closeSignal)
		_ = c.ws.Close()
	})
}

func (c *Connection) readLoop(ctx context.Context) {
	defer func() {
		c.Close()
		if c.onClose != nil {
			c.onClose(ctx, c)
		}
	}()

	for {
		_, payload, err := c.ws.ReadMessage()
		if err != nil {
			if c.log != nil {
				c.log.Debug("websocket read stopped", zap.String("connection_id", c.ID), zap.Error(err))
			}
			return
		}
		_ = c.ws.SetReadDeadline(time.Now().Add(c.readTimeout))
		var message ClientMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			c.Send(response("", ResponseTypeMalformed, CodeBadRequest, "消息格式错误", nil))
			continue
		}
		if c.onMessage != nil {
			c.onMessage(ctx, c, message)
		}
	}
}

func (c *Connection) writeLoop(ctx context.Context, pingInterval time.Duration) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.Close()
			return
		case <-c.closeSignal:
			return
		case payload := <-c.writeQueue:
			if err := c.ws.WriteMessage(websocket.TextMessage, payload); err != nil {
				if c.log != nil {
					c.log.Debug("websocket write stopped", zap.String("connection_id", c.ID), zap.Error(err))
				}
				c.Close()
				return
			}
		case <-ticker.C:
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				if c.log != nil {
					c.log.Debug("websocket ping failed", zap.String("connection_id", c.ID), zap.Error(err))
				}
				c.Close()
				return
			}
		}
	}
}
