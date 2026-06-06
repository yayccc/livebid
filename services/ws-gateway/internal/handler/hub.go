package handler

import "sync"

type Hub struct {
	mu          sync.RWMutex
	connections map[string]*Connection
	rooms       map[int64]map[string]*Connection
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[string]*Connection),
		rooms:       make(map[int64]map[string]*Connection),
	}
}

func (h *Hub) Add(conn *Connection) {
	if conn == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[conn.ID] = conn
	if _, ok := h.rooms[conn.RoomID]; !ok {
		h.rooms[conn.RoomID] = make(map[string]*Connection)
	}
	h.rooms[conn.RoomID][conn.ID] = conn
}

func (h *Hub) Remove(conn *Connection) {
	if conn == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.connections, conn.ID)
	if roomConns, ok := h.rooms[conn.RoomID]; ok {
		delete(roomConns, conn.ID)
		if len(roomConns) == 0 {
			delete(h.rooms, conn.RoomID)
		}
	}
}

func (h *Hub) BroadcastRoom(roomID int64, message any) {
	h.mu.RLock()
	conns := make([]*Connection, 0, len(h.rooms[roomID]))
	for _, conn := range h.rooms[roomID] {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		conn.Send(message)
	}
}

func (h *Hub) RoomIDs() []int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	roomIDs := make([]int64, 0, len(h.rooms))
	for roomID := range h.rooms {
		roomIDs = append(roomIDs, roomID)
	}
	return roomIDs
}

func (h *Hub) CloseAll() {
	h.mu.RLock()
	conns := make([]*Connection, 0, len(h.connections))
	for _, conn := range h.connections {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		conn.Close()
	}
}

func (h *Hub) Size() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}
