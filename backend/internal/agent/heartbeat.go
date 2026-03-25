package agent

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// HeartbeatMonitor monitors agent health by checking last-seen timestamps.
type HeartbeatMonitor struct {
	pool     *Pool
	timeout  time.Duration
	interval time.Duration
	onEvict  func(agentID string) // Callback when an agent is evicted
}

// NewHeartbeatMonitor creates a new heartbeat monitor.
func NewHeartbeatMonitor(pool *Pool, timeout, interval time.Duration) *HeartbeatMonitor {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return &HeartbeatMonitor{
		pool:     pool,
		timeout:  timeout,
		interval: interval,
	}
}

// OnEvict sets the callback invoked when an agent is marked offline.
func (h *HeartbeatMonitor) OnEvict(fn func(agentID string)) {
	h.onEvict = fn
}

// Start begins the heartbeat monitoring loop. Blocks until ctx is cancelled.
func (h *HeartbeatMonitor) Start(ctx context.Context) {
	log.Info().
		Dur("timeout", h.timeout).
		Dur("interval", h.interval).
		Msg("heartbeat monitor started")

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("heartbeat monitor stopped")
			return
		case <-ticker.C:
			h.check()
		}
	}
}

// check runs a single heartbeat check cycle.
func (h *HeartbeatMonitor) check() {
	evicted := h.pool.MarkOffline(h.timeout)
	for _, agentID := range evicted {
		log.Warn().
			Str("agent_id", agentID).
			Dur("timeout", h.timeout).
			Msg("agent marked offline: heartbeat timeout")

		if h.onEvict != nil {
			h.onEvict(agentID)
		}
	}
}
