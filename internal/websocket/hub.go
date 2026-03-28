package websocket

import (
	"sync"
)

type Client struct {
	RunID string
	Send  chan []byte
}

type Hub struct {
	mu         sync.RWMutex
	rooms      map[string]map[*Client]bool
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
		Broadcast:  make(chan Message, 1024),
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
			room, ok := h.rooms[msg.RunID]
			if !ok {
				h.mu.RUnlock()
				continue
			}
			// Collect slow clients to evict after releasing the read lock
			var evict []*Client
			for client := range room {
				select {
				case client.Send <- msg.Content:
				default:
					evict = append(evict, client)
				}
			}
			h.mu.RUnlock()

			// Evict slow clients under write lock
			if len(evict) > 0 {
				h.mu.Lock()
				for _, client := range evict {
					if room, ok := h.rooms[client.RunID]; ok {
						if _, ok := room[client]; ok {
							delete(room, client)
							close(client.Send)
							if len(room) == 0 {
								delete(h.rooms, client.RunID)
							}
						}
					}
				}
				h.mu.Unlock()
			}
		}
	}
}

func (h *Hub) BroadcastToRun(runID string, content []byte) {
	select {
	case h.Broadcast <- Message{RunID: runID, Content: content}:
	default:
		// Broadcast channel full — drop the message rather than block the caller.
		// This prevents the pipeline runner from stalling when the hub is congested.
	}
}

// BroadcastGlobal sends a message to the global events channel so all
// connected event-socket clients receive it (used for notifications, etc.).
func (h *Hub) BroadcastGlobal(content []byte) {
	select {
	case h.Broadcast <- Message{RunID: "__events__", Content: content}:
	default:
	}
}
