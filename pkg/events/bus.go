package events

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
)

// EventType identifies the kind of event.
type EventType string

// Common event types used across the system.
const (
	PipelineQueued    EventType = "pipeline.queued"
	PipelineStarted   EventType = "pipeline.started"
	PipelineCompleted EventType = "pipeline.completed"
	PipelineFailed    EventType = "pipeline.failed"
	PipelineCancelled EventType = "pipeline.cancelled"

	StepStarted   EventType = "step.started"
	StepCompleted EventType = "step.completed"
	StepFailed    EventType = "step.failed"

	AgentConnected    EventType = "agent.connected"
	AgentDisconnected EventType = "agent.disconnected"
	AgentDraining     EventType = "agent.draining"

	SecretAccessed EventType = "secret.accessed"
	SecretCreated  EventType = "secret.created"
	SecretDeleted  EventType = "secret.deleted"

	ArtifactUploaded EventType = "artifact.uploaded"
	ArtifactExpired  EventType = "artifact.expired"

	UserLogin  EventType = "user.login"
	UserLogout EventType = "user.logout"
)

// Event represents a system event.
type Event struct {
	Type     EventType         `json:"type"`
	Data     map[string]string `json:"data,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Handler processes events.
type Handler func(event *Event)

// Bus is a typed, channel-based event bus for publishing and subscribing to system events.
type Bus struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
	allHandlers []Handler
	ch       chan *Event
	done     chan struct{}
}

// NewBus creates a new event bus with the given buffer size.
func NewBus(bufferSize int) *Bus {
	if bufferSize <= 0 {
		bufferSize = 512
	}
	return &Bus{
		handlers: make(map[EventType][]Handler),
		ch:       make(chan *Event, bufferSize),
		done:     make(chan struct{}),
	}
}

// Subscribe registers a handler for a specific event type.
func (b *Bus) Subscribe(eventType EventType, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// SubscribeAll registers a handler that receives all events.
func (b *Bus) SubscribeAll(handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.allHandlers = append(b.allHandlers, handler)
}

// Publish sends an event to the bus. Non-blocking; drops if buffer is full.
func (b *Bus) Publish(event *Event) {
	select {
	case b.ch <- event:
	default:
		log.Warn().Str("type", string(event.Type)).Msg("event bus: event dropped (buffer full)")
	}
}

// PublishSync sends an event and dispatches it synchronously (blocks until handled).
func (b *Bus) PublishSync(event *Event) {
	b.dispatch(event)
}

// Start begins processing events. Blocks until ctx is cancelled.
func (b *Bus) Start(ctx context.Context) {
	log.Info().Msg("event bus: started")
	defer func() {
		close(b.done)
		log.Info().Msg("event bus: stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			// Drain remaining events
			for {
				select {
				case event := <-b.ch:
					b.dispatch(event)
				default:
					return
				}
			}
		case event := <-b.ch:
			b.dispatch(event)
		}
	}
}

// dispatch sends the event to all matching handlers.
func (b *Bus) dispatch(event *Event) {
	b.mu.RLock()
	// Type-specific handlers
	specific := make([]Handler, len(b.handlers[event.Type]))
	copy(specific, b.handlers[event.Type])
	// All-event handlers
	all := make([]Handler, len(b.allHandlers))
	copy(all, b.allHandlers)
	b.mu.RUnlock()

	for _, h := range specific {
		safeCall(h, event)
	}
	for _, h := range all {
		safeCall(h, event)
	}
}

// safeCall invokes a handler and recovers from panics.
func safeCall(h Handler, event *Event) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Interface("panic", r).
				Str("event_type", string(event.Type)).
				Msg("event handler panicked")
		}
	}()
	h(event)
}

// Wait blocks until the bus has stopped.
func (b *Bus) Wait() {
	<-b.done
}
