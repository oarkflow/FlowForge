package notifications

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
)

// EventBus is a typed, channel-based event bus for dispatching pipeline events
// to registered notification handlers.
type EventBus struct {
	mu       sync.RWMutex
	handlers []Handler
	ch       chan *Event
	done     chan struct{}
}

// Handler processes events from the bus.
type Handler func(event *Event)

// NewEventBus creates a new EventBus with a buffered channel.
func NewEventBus(bufferSize int) *EventBus {
	if bufferSize <= 0 {
		bufferSize = 256
	}
	return &EventBus{
		ch:   make(chan *Event, bufferSize),
		done: make(chan struct{}),
	}
}

// Subscribe registers a handler that will be called for every event.
func (b *EventBus) Subscribe(handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, handler)
}

// Publish sends an event to the bus. Non-blocking: drops if buffer is full.
func (b *EventBus) Publish(event *Event) {
	select {
	case b.ch <- event:
	default:
		log.Warn().Str("run_id", event.RunID).Msg("notification bus: event dropped (buffer full)")
	}
}

// Start begins processing events. Blocks until ctx is cancelled.
func (b *EventBus) Start(ctx context.Context) {
	log.Info().Msg("notification bus: started")
	defer func() {
		close(b.done)
		log.Info().Msg("notification bus: stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-b.ch:
			b.dispatch(event)
		}
	}
}

// dispatch sends the event to all registered handlers.
func (b *EventBus) dispatch(event *Event) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.RUnlock()

	for _, h := range handlers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Msg("notification handler panicked")
				}
			}()
			h(event)
		}()
	}
}
