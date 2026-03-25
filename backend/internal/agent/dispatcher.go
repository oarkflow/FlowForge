package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// JobRequest represents a job that needs to be dispatched to an agent.
type JobRequest struct {
	JobRunID      string            `json:"job_run_id"`
	PipelineRunID string            `json:"pipeline_run_id"`
	ExecutorType  string            `json:"executor_type"`
	RequiredLabels []string         `json:"required_labels"`
	Image         string            `json:"image"`
	CloneURL      string            `json:"clone_url"`
	CommitSHA     string            `json:"commit_sha"`
	Branch        string            `json:"branch"`
	EnvVars       map[string]string `json:"env_vars"`
	Steps         []StepConfig      `json:"steps"`
	Priority      int               `json:"priority"`
	CreatedAt     time.Time         `json:"created_at"`
}

// StepConfig defines configuration for a single step.
type StepConfig struct {
	StepRunID       string            `json:"step_run_id"`
	Name            string            `json:"name"`
	Command         string            `json:"command"`
	WorkingDir      string            `json:"working_dir"`
	Env             map[string]string `json:"env"`
	TimeoutSeconds  int64             `json:"timeout_seconds"`
	ContinueOnError bool             `json:"continue_on_error"`
	RetryCount      int32             `json:"retry_count"`
}

// DispatchResult contains the result of dispatching a job.
type DispatchResult struct {
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
	Accepted  bool   `json:"accepted"`
	Error     string `json:"error,omitempty"`
}

// JobHandler is called when a job is dispatched to an agent.
// The handler should execute the job and return when complete.
type JobHandler func(ctx context.Context, agentID string, req *JobRequest) error

// Dispatcher handles matching job requests to available agents and dispatching them.
type Dispatcher struct {
	pool       *Pool
	queue      chan *JobRequest
	handlers   map[string]JobHandler // per-agent handler
	mu         sync.RWMutex
	maxRetries int
	retryDelay time.Duration
}

// NewDispatcher creates a new job dispatcher.
func NewDispatcher(pool *Pool, queueSize int) *Dispatcher {
	if queueSize <= 0 {
		queueSize = 256
	}
	return &Dispatcher{
		pool:       pool,
		queue:      make(chan *JobRequest, queueSize),
		handlers:   make(map[string]JobHandler),
		maxRetries: 3,
		retryDelay: 5 * time.Second,
	}
}

// RegisterHandler registers a handler for a specific agent.
func (d *Dispatcher) RegisterHandler(agentID string, handler JobHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[agentID] = handler
}

// UnregisterHandler removes the handler for a specific agent.
func (d *Dispatcher) UnregisterHandler(agentID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.handlers, agentID)
}

// Enqueue adds a job request to the dispatch queue.
func (d *Dispatcher) Enqueue(req *JobRequest) error {
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now()
	}
	select {
	case d.queue <- req:
		log.Info().
			Str("job_run_id", req.JobRunID).
			Str("executor", req.ExecutorType).
			Msg("job enqueued for dispatch")
		return nil
	default:
		return fmt.Errorf("dispatch queue is full")
	}
}

// Start begins the dispatch loop. Blocks until ctx is cancelled.
func (d *Dispatcher) Start(ctx context.Context) {
	log.Info().Msg("job dispatcher started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("job dispatcher stopped")
			return
		case req := <-d.queue:
			go d.dispatch(ctx, req, 0)
		}
	}
}

// dispatch attempts to dispatch a job to a suitable agent with retries.
func (d *Dispatcher) dispatch(ctx context.Context, req *JobRequest, attempt int) {
	logger := log.With().
		Str("job_run_id", req.JobRunID).
		Int("attempt", attempt+1).
		Logger()

	// Select a suitable agent
	agent, err := d.pool.SelectAgent(req.ExecutorType, req.RequiredLabels)
	if err != nil {
		logger.Warn().Err(err).Msg("no suitable agent found")

		if attempt < d.maxRetries {
			select {
			case <-ctx.Done():
				return
			case <-time.After(d.retryDelay):
				d.dispatch(ctx, req, attempt+1)
				return
			}
		}

		logger.Error().Msg("job dispatch failed after all retries")
		return
	}

	logger.Info().
		Str("agent_id", agent.ID).
		Str("agent_name", agent.Name).
		Msg("dispatching job to agent")

	// Increment active jobs
	if err := d.pool.IncrementActiveJobs(agent.ID); err != nil {
		logger.Error().Err(err).Msg("failed to increment active jobs")
		return
	}

	// Get handler for this agent
	d.mu.RLock()
	handler, ok := d.handlers[agent.ID]
	d.mu.RUnlock()

	if !ok {
		// No specific handler; job will be handled via the generic mechanism
		logger.Warn().Msg("no handler registered for agent; using default log")
		d.pool.DecrementActiveJobs(agent.ID)
		return
	}

	// Execute the handler in a goroutine
	go func() {
		defer d.pool.DecrementActiveJobs(agent.ID)

		if err := handler(ctx, agent.ID, req); err != nil {
			logger.Error().Err(err).Msg("job execution failed on agent")

			// Retry on a different agent
			if attempt < d.maxRetries {
				select {
				case <-ctx.Done():
					return
				case <-time.After(d.retryDelay):
					d.dispatch(ctx, req, attempt+1)
				}
			}
		} else {
			logger.Info().Msg("job completed successfully on agent")
		}
	}()
}

// QueueSize returns the current number of jobs waiting in the queue.
func (d *Dispatcher) QueueSize() int {
	return len(d.queue)
}

// MarshalJobRequest serializes a job request to JSON for transmission.
func MarshalJobRequest(req *JobRequest) (string, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal job request: %w", err)
	}
	return string(data), nil
}

// UnmarshalJobRequest deserializes a job request from JSON.
func UnmarshalJobRequest(data string) (*JobRequest, error) {
	var req JobRequest
	if err := json.Unmarshal([]byte(data), &req); err != nil {
		return nil, fmt.Errorf("unmarshal job request: %w", err)
	}
	return &req, nil
}
