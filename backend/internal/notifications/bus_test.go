package notifications

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewEventBus_DefaultBufferSize(t *testing.T) {
	bus := NewEventBus(0)
	if bus == nil {
		t.Fatal("bus should not be nil")
	}
	// Should default to 256 buffer
	if cap(bus.ch) != 256 {
		t.Errorf("buffer size = %d, want 256", cap(bus.ch))
	}
}

func TestNewEventBus_CustomBufferSize(t *testing.T) {
	bus := NewEventBus(100)
	if cap(bus.ch) != 100 {
		t.Errorf("buffer size = %d, want 100", cap(bus.ch))
	}
}

func TestEventBus_SubscribeAndPublish(t *testing.T) {
	bus := NewEventBus(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received []*Event
	var mu sync.Mutex

	bus.Subscribe(func(e *Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	go bus.Start(ctx)

	// Publish an event
	event := &Event{
		Type:    EventRunSuccess,
		RunID:   "run-1",
		Status:  "success",
	}
	bus.Publish(event)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	cancel()

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("received %d events, want 1", len(received))
	}
	if received[0].RunID != "run-1" {
		t.Errorf("RunID = %q, want %q", received[0].RunID, "run-1")
	}
}

func TestEventBus_MultipleHandlers(t *testing.T) {
	bus := NewEventBus(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var count1, count2 int
	var mu sync.Mutex

	bus.Subscribe(func(e *Event) {
		mu.Lock()
		count1++
		mu.Unlock()
	})
	bus.Subscribe(func(e *Event) {
		mu.Lock()
		count2++
		mu.Unlock()
	})

	go bus.Start(ctx)

	bus.Publish(&Event{Type: EventRunFailure, RunID: "run-2"})
	time.Sleep(100 * time.Millisecond)
	cancel()

	mu.Lock()
	defer mu.Unlock()
	if count1 != 1 || count2 != 1 {
		t.Errorf("both handlers should receive event: count1=%d, count2=%d", count1, count2)
	}
}

func TestEventBus_HandlerPanicRecovery(t *testing.T) {
	bus := NewEventBus(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received bool
	var mu sync.Mutex

	// First handler panics
	bus.Subscribe(func(e *Event) {
		panic("test panic")
	})

	// Second handler should still run
	bus.Subscribe(func(e *Event) {
		mu.Lock()
		received = true
		mu.Unlock()
	})

	go bus.Start(ctx)

	bus.Publish(&Event{Type: EventRunSuccess, RunID: "run-3"})
	time.Sleep(100 * time.Millisecond)
	cancel()

	mu.Lock()
	defer mu.Unlock()
	if !received {
		t.Error("second handler should still receive event after first panics")
	}
}

func TestEventBus_Publish_NonBlocking(t *testing.T) {
	// Create bus with tiny buffer
	bus := NewEventBus(1)

	// Fill the buffer
	bus.Publish(&Event{Type: EventRunSuccess, RunID: "fill"})

	// This should not block even though buffer is full
	done := make(chan bool, 1)
	go func() {
		bus.Publish(&Event{Type: EventRunFailure, RunID: "overflow"})
		done <- true
	}()

	select {
	case <-done:
		// OK - non-blocking
	case <-time.After(time.Second):
		t.Error("Publish should be non-blocking even when buffer is full")
	}
}

func TestEventBus_StopsOnContextCancel(t *testing.T) {
	bus := NewEventBus(10)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		bus.Start(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// OK - Start returned
	case <-time.After(time.Second):
		t.Error("Start should return after context cancel")
	}
}

// --- Event Types ---

func TestEventTypes(t *testing.T) {
	tests := []struct {
		eventType EventType
		want      string
	}{
		{EventRunSuccess, "run_success"},
		{EventRunFailure, "run_failure"},
		{EventRunCancelled, "run_cancelled"},
		{EventDeployment, "deployment"},
		{EventApproval, "approval_required"},
	}
	for _, tt := range tests {
		if string(tt.eventType) != tt.want {
			t.Errorf("EventType %q != %q", tt.eventType, tt.want)
		}
	}
}
