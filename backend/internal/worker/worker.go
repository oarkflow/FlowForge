package worker

import (
	"context"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// TaskFunc is the function signature for background tasks.
type TaskFunc func(ctx context.Context) error

// Task represents a registered background task.
type Task struct {
	Name     string
	Interval time.Duration
	Fn       TaskFunc
}

// PipelineEngine is the interface used to re-enqueue orphaned runs.
type PipelineEngine interface {
	ReenqueueRun(ctx context.Context, runID string) error
}

// Pool manages background worker goroutines for periodic tasks.
type Pool struct {
	mu     sync.Mutex
	tasks  []Task
	db     *sqlx.DB
	engine PipelineEngine
}

// NewPool creates a new worker pool.
func NewPool(db *sqlx.DB) *Pool {
	return &Pool{db: db}
}

// SetEngine sets the pipeline engine for stale run recovery.
func (p *Pool) SetEngine(engine PipelineEngine) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.engine = engine
}

// Register adds a new periodic task to the pool.
func (p *Pool) Register(name string, interval time.Duration, fn TaskFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tasks = append(p.tasks, Task{
		Name:     name,
		Interval: interval,
		Fn:       fn,
	})
}

// RegisterDefaults registers all built-in background tasks.
func (p *Pool) RegisterDefaults() {
	p.Register("artifact_expiry", 1*time.Hour, p.artifactExpiryTask)
	p.Register("agent_health", 30*time.Second, p.agentHealthTask)
	p.Register("log_cleanup", 6*time.Hour, p.logCleanupTask)
	p.Register("metrics_collector", 1*time.Minute, p.metricsCollectorTask)
	p.Register("stale_run_recovery", 30*time.Second, p.staleRunRecoveryTask)
}

// Start launches all registered tasks as goroutines. Blocks until ctx is cancelled.
func (p *Pool) Start(ctx context.Context) {
	p.mu.Lock()
	tasks := make([]Task, len(p.tasks))
	copy(tasks, p.tasks)
	p.mu.Unlock()

	var wg sync.WaitGroup

	for _, task := range tasks {
		wg.Add(1)
		go func(t Task) {
			defer wg.Done()
			p.runTask(ctx, t)
		}(task)
	}

	log.Info().Int("tasks", len(tasks)).Msg("worker pool started")

	<-ctx.Done()
	wg.Wait()
	log.Info().Msg("worker pool stopped")
}

// runTask executes a task periodically until ctx is cancelled.
func (p *Pool) runTask(ctx context.Context, task Task) {
	logger := log.With().Str("task", task.Name).Logger()
	logger.Info().Dur("interval", task.Interval).Msg("background task started")

	// Run immediately on start
	if err := task.Fn(ctx); err != nil {
		logger.Error().Err(err).Msg("task execution failed")
	}

	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("background task stopped")
			return
		case <-ticker.C:
			if err := task.Fn(ctx); err != nil {
				logger.Error().Err(err).Msg("task execution failed")
			}
		}
	}
}

// artifactExpiryTask deletes expired artifacts.
func (p *Pool) artifactExpiryTask(ctx context.Context) error {
	if p.db == nil {
		return nil
	}

	result, err := p.db.ExecContext(ctx,
		"DELETE FROM artifacts WHERE expire_at IS NOT NULL AND expire_at < CURRENT_TIMESTAMP")
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Info().Int64("deleted", rows).Msg("expired artifacts cleaned up")
	}

	return nil
}

// agentHealthTask checks agent heartbeats and marks stale agents as offline.
func (p *Pool) agentHealthTask(ctx context.Context) error {
	if p.db == nil {
		return nil
	}

	// Mark agents as offline if no heartbeat for 60 seconds
	result, err := p.db.ExecContext(ctx,
		`UPDATE agents SET status = 'offline'
		 WHERE status IN ('online', 'busy')
		 AND datetime(last_seen_at) < datetime('now', '-60 seconds')`)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Warn().Int64("agents", rows).Msg("agents marked offline (heartbeat timeout)")
	}

	return nil
}

// logCleanupTask enforces log retention policies.
func (p *Pool) logCleanupTask(ctx context.Context) error {
	if p.db == nil {
		return nil
	}

	// Default retention: 30 days
	result, err := p.db.ExecContext(ctx,
		"DELETE FROM run_logs WHERE ts < datetime('now', '-30 days')")
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Info().Int64("deleted", rows).Msg("old logs cleaned up")
	}

	return nil
}

// metricsCollectorTask collects system metrics.
func (p *Pool) metricsCollectorTask(ctx context.Context) error {
	if p.db == nil {
		return nil
	}

	// Count active runs
	var activeRuns int
	p.db.GetContext(ctx, &activeRuns,
		"SELECT COUNT(*) FROM pipeline_runs WHERE status IN ('running', 'queued', 'pending')")

	// Count online agents
	var onlineAgents int
	p.db.GetContext(ctx, &onlineAgents,
		"SELECT COUNT(*) FROM agents WHERE status IN ('online', 'busy')")

	return nil
}

// staleRunRecoveryTask recovers pipeline runs that are stuck in DB.
// On server restart, runs that were "running" are marked as failed,
// and runs that are "queued" but not in the in-memory queue are re-enqueued.
func (p *Pool) staleRunRecoveryTask(ctx context.Context) error {
	if p.db == nil {
		return nil
	}

	// Mark runs stuck in "running" as failed (these were interrupted by a crash/restart)
	result, err := p.db.ExecContext(ctx,
		`UPDATE pipeline_runs SET status = 'failure', finished_at = CURRENT_TIMESTAMP
		 WHERE status = 'running'
		 AND datetime(started_at) < datetime('now', '-5 minutes')`)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Warn().Int64("runs", rows).Msg("stale running pipeline runs marked as failed")
	}

	// Re-enqueue runs stuck in "queued" status
	p.mu.Lock()
	engine := p.engine
	p.mu.Unlock()

	if engine == nil {
		return nil
	}

	type queuedRun struct {
		ID string `db:"id"`
	}
	var staleQueued []queuedRun
	err = p.db.SelectContext(ctx, &staleQueued,
		`SELECT id FROM pipeline_runs
		 WHERE status = 'queued'
		 AND datetime(created_at) < datetime('now', '-10 seconds')
		 ORDER BY created_at ASC
		 LIMIT 50`)
	if err != nil {
		return err
	}

	for _, run := range staleQueued {
		if err := engine.ReenqueueRun(ctx, run.ID); err != nil {
			log.Warn().Err(err).Str("run_id", run.ID).Msg("failed to re-enqueue stale run")
		} else {
			log.Info().Str("run_id", run.ID).Msg("re-enqueued stale queued run")
		}
	}

	return nil
}
