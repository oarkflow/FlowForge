package websocket

import (
	"sync"
)

type Client struct {
	RunID string
	Send  chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	rooms   map[string]map[*Client]bool
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan Message
}

type Message struct {
	RunID   string
	Content []byte
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan Message, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if _, ok := h.rooms[client.RunID]; !ok {
				h.rooms[client.RunID] = make(map[*Client]bool)
			}
			h.rooms[client.RunID][client] = true
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if room, ok := h.rooms[client.RunID]; ok {
				if _, ok := room[client]; ok {
					delete(room, client)
					close(client.Send)
					if len(room) == 0 {
						delete(h.rooms, client.RunID)
					}
				}
			}
			h.mu.Unlock()

		case msg := <-h.Broadcast:
			h.mu.RLock()
			if room, ok := h.rooms[msg.RunID]; ok {
				for client := range room {
					select {
					case client.Send <- msg.Content:
					default:
						close(client.Send)
						delete(room, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastToRun(runID string, content []byte) {
	h.Broadcast <- Message{RunID: runID, Content: content}
}
