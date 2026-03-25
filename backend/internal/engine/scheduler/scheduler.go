package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/engine/queue"
	"github.com/oarkflow/deploy/backend/internal/websocket"
)

// PipelineConfig describes the pipeline structure parsed from the stored config.
// It is used by the Runner to know which stages/jobs/steps to execute.
type PipelineConfig struct {
	Stages []StageConfig `json:"stages"`
}

// StageConfig describes a single stage containing one or more jobs.
type StageConfig struct {
	Name  string      `json:"name"`
	Needs []string    `json:"needs,omitempty"` // stage dependencies for DAG-based execution
	Jobs  []JobConfig `json:"jobs"`
}

// JobConfig describes a single job containing one or more steps.
type JobConfig struct {
	Name         string       `json:"name"`
	ExecutorType string       `json:"executor_type"`
	Steps        []StepConfig `json:"steps"`
}

// StepConfig describes a single step (a command to run).
type StepConfig struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	WorkDir string            `json:"work_dir,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Timeout string            `json:"timeout,omitempty"` // e.g. "5m", "30s"
}

// RunFunc is the function signature the scheduler uses to execute a dispatched job.
// It receives the context, the pipeline run ID, and the parsed pipeline config.
type RunFunc func(ctx context.Context, pipelineRunID string, config PipelineConfig) error

// OnCompleteFunc is called after a pipeline run finishes (success or failure).
// It receives the context, the run ID, and the final status.
type OnCompleteFunc func(ctx context.Context, runID string, status string)

// Scheduler polls the priority queue and dispatches jobs for execution.
type Scheduler struct {
	queue        *queue.PriorityQueue
	repos        *queries.Repositories
	hub          *websocket.Hub
	runFn        RunFunc
	onCompleteFn OnCompleteFunc
}

// New creates a new Scheduler.
func New(q *queue.PriorityQueue, db *sqlx.DB, hub *websocket.Hub) *Scheduler {
	return &Scheduler{
		queue: q,
		repos: queries.NewRepositories(db),
		hub:   hub,
	}
}

// SetRunFunc sets the function used to execute a dispatched job.
// This must be called before Start.
func (s *Scheduler) SetRunFunc(fn RunFunc) {
	s.runFn = fn
}

// SetOnCompleteFunc sets a callback invoked after each pipeline run finishes.
// This is used by the engine for pipeline composition (downstream triggers).
func (s *Scheduler) SetOnCompleteFunc(fn OnCompleteFunc) {
	s.onCompleteFn = fn
}

// Start begins the scheduling loop. It blocks until ctx is cancelled.
// Jobs are dequeued and dispatched as goroutines.
func (s *Scheduler) Start(ctx context.Context) {
	log.Info().Msg("scheduler: started")
	defer log.Info().Msg("scheduler: stopped")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Block waiting for the next job (or context cancellation)
		job, err := s.dequeueWithContext(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // Shutting down
			}
			log.Error().Err(err).Msg("scheduler: dequeue error")
			time.Sleep(time.Second)
			continue
		}
		if job == nil {
			continue
		}

		// Dispatch the job in a goroutine
		go s.dispatch(ctx, job)
	}
}

// dequeueWithContext wraps the blocking Dequeue call with context awareness.
func (s *Scheduler) dequeueWithContext(ctx context.Context) (*queue.Job, error) {
	type result struct {
		job *queue.Job
		err error
	}

	ch := make(chan result, 1)
	go func() {
		j, err := s.queue.Dequeue()
		ch <- result{j, err}
	}()

	select {
	case <-ctx.Done():
		// Close the queue to unblock the Dequeue goroutine
		s.queue.Close()
		return nil, ctx.Err()
	case r := <-ch:
		return r.job, r.err
	}
}

// dispatch executes a single job. It parses the config, updates statuses,
// and calls the registered run function.
func (s *Scheduler) dispatch(ctx context.Context, job *queue.Job) {
	runID := job.PipelineRunID
	log.Info().Str("run_id", runID).Str("job_id", job.ID).Msg("scheduler: dispatching job")

	// Broadcast a status update
	s.broadcastStatus(runID, "running")

	// Mark the pipeline run as started
	if err := s.repos.Runs.SetStarted(ctx, runID); err != nil {
		log.Error().Err(err).Str("run_id", runID).Msg("scheduler: failed to mark run as started")
	}

	// Parse the pipeline config
	var config PipelineConfig
	if err := json.Unmarshal(job.Config, &config); err != nil {
		errMsg := fmt.Sprintf("scheduler: failed to parse pipeline config: %v", err)
		log.Error().Str("run_id", runID).Msg(errMsg)
		s.finishRun(ctx, runID, "failure", 0, errMsg)
		return
	}

	// Execute the pipeline via the registered run function
	if s.runFn == nil {
		log.Error().Str("run_id", runID).Msg("scheduler: no run function registered")
		s.finishRun(ctx, runID, "failure", 0, "no run function registered")
		return
	}

	start := time.Now()
	err := s.runFn(ctx, runID, config)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		log.Error().Err(err).Str("run_id", runID).Msg("scheduler: pipeline run failed")
		s.finishRun(ctx, runID, "failure", durationMs, err.Error())
		return
	}

	s.finishRun(ctx, runID, "success", durationMs, "")
}

// finishRun updates the pipeline run status and broadcasts the result.
func (s *Scheduler) finishRun(ctx context.Context, runID, status string, durationMs int, errorSummary string) {
	if err := s.repos.Runs.SetFinished(ctx, runID, status, durationMs, errorSummary); err != nil {
		log.Error().Err(err).Str("run_id", runID).Msg("scheduler: failed to update run status")
	}
	s.broadcastStatus(runID, status)

	// Notify the engine for pipeline composition (downstream triggers)
	if s.onCompleteFn != nil {
		s.onCompleteFn(ctx, runID, status)
	}
}

// broadcastStatus sends a JSON status message to WebSocket clients subscribed to the run.
func (s *Scheduler) broadcastStatus(runID, status string) {
	msg, _ := json.Marshal(map[string]string{
		"type":   "status",
		"run_id": runID,
		"status": status,
	})
	s.hub.BroadcastToRun(runID, msg)
}
