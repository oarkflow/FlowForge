package websocket

import (
	"testing"
	"time"
)

func TestHub_RegisterUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		RunID: "run-1",
		Send:  make(chan []byte, 10),
	}

	hub.Register <- client
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	room, ok := hub.rooms["run-1"]
	hub.mu.RUnlock()

	if !ok {
		t.Fatal("room should exist after registration")
	}
	if !room[client] {
		t.Error("client should be in the room")
	}

	hub.Unregister <- client
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	_, roomExists := hub.rooms["run-1"]
	hub.mu.RUnlock()

	if roomExists {
		t.Error("room should be deleted when last client leaves")
	}
}

func TestHub_BroadcastToRun(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := &Client{RunID: "run-1", Send: make(chan []byte, 10)}
	c2 := &Client{RunID: "run-1", Send: make(chan []byte, 10)}
	c3 := &Client{RunID: "run-2", Send: make(chan []byte, 10)}

	hub.Register <- c1
	hub.Register <- c2
	hub.Register <- c3
	time.Sleep(50 * time.Millisecond)

	hub.BroadcastToRun("run-1", []byte("log line"))
	time.Sleep(50 * time.Millisecond)

	// c1 and c2 should receive the message
	select {
	case msg := <-c1.Send:
		if string(msg) != "log line" {
			t.Errorf("c1 received %q, want %q", msg, "log line")
		}
	default:
		t.Error("c1 should receive the broadcast")
	}

	select {
	case msg := <-c2.Send:
		if string(msg) != "log line" {
			t.Errorf("c2 received %q, want %q", msg, "log line")
		}
	default:
		t.Error("c2 should receive the broadcast")
	}

	// c3 should NOT receive (different room)
	select {
	case msg := <-c3.Send:
		t.Errorf("c3 should not receive broadcast from run-1, got %q", msg)
	default:
		// OK
	}
}

func TestHub_BroadcastGlobal(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{RunID: "__events__", Send: make(chan []byte, 10)}
	hub.Register <- client
	time.Sleep(50 * time.Millisecond)

	hub.BroadcastGlobal([]byte("global event"))
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-client.Send:
		if string(msg) != "global event" {
			t.Errorf("received %q, want %q", msg, "global event")
		}
	default:
		t.Error("global event client should receive broadcast")
	}
}

func TestHub_BroadcastToNonexistentRun(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Should not panic or block
	hub.BroadcastToRun("nonexistent-run", []byte("data"))
	time.Sleep(50 * time.Millisecond)
}

func TestHub_MultipleRooms(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := &Client{RunID: "run-a", Send: make(chan []byte, 10)}
	c2 := &Client{RunID: "run-b", Send: make(chan []byte, 10)}

	hub.Register <- c1
	hub.Register <- c2
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	if len(hub.rooms) != 2 {
		t.Errorf("rooms count = %d, want 2", len(hub.rooms))
	}
	hub.mu.RUnlock()
}

func TestHub_SlowClientEviction(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Slow client has buffer size 0 (unbuffered send channel would block)
	// But we use buffered with size 0 to simulate slow client
	slow := &Client{RunID: "run-1", Send: make(chan []byte)} // unbuffered
	fast := &Client{RunID: "run-1", Send: make(chan []byte, 10)}

	hub.Register <- slow
	hub.Register <- fast
	time.Sleep(50 * time.Millisecond)

	// Send a message - slow client can't receive, should be evicted
	hub.BroadcastToRun("run-1", []byte("message"))
	time.Sleep(100 * time.Millisecond)

	// Fast client should still get the message
	select {
	case msg := <-fast.Send:
		if string(msg) != "message" {
			t.Errorf("fast client got %q, want %q", msg, "message")
		}
	default:
		t.Error("fast client should receive the message")
	}
}

func TestHub_UnregisterClosesChannel(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{RunID: "run-1", Send: make(chan []byte, 10)}
	hub.Register <- client
	time.Sleep(50 * time.Millisecond)

	hub.Unregister <- client
	time.Sleep(50 * time.Millisecond)

	// The Send channel should be closed
	_, ok := <-client.Send
	if ok {
		t.Error("Send channel should be closed after unregister")
	}
}

func TestHub_BroadcastToRun_NonBlocking(t *testing.T) {
	hub := NewHub()
	// Don't run the hub loop - broadcast channel will fill up

	// Fill the broadcast channel
	for i := 0; i < 1024; i++ {
		hub.BroadcastToRun("run", []byte("fill"))
	}

	// This should not block
	done := make(chan bool, 1)
	go func() {
		hub.BroadcastToRun("run", []byte("extra"))
		done <- true
	}()

	select {
	case <-done:
		// OK
	case <-time.After(time.Second):
		t.Error("BroadcastToRun should be non-blocking")
	}
}
