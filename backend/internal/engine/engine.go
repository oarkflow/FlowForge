package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/internal/config"
	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/engine/queue"
	"github.com/oarkflow/deploy/backend/internal/engine/scheduler"
	"github.com/oarkflow/deploy/backend/internal/models"
	"github.com/oarkflow/deploy/backend/internal/websocket"
)

// Engine is the central pipeline execution engine. It manages the job queue,
// scheduler, and runner to orchestrate pipeline execution.
type Engine struct {
	db        *sqlx.DB
	hub       *websocket.Hub
	cfg       *config.Config
	repos     *queries.Repositories
	queue     *queue.PriorityQueue
	scheduler *scheduler.Scheduler
	runner    *Runner

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new Engine with the given database, WebSocket hub, and configuration.
func New(db *sqlx.DB, hub *websocket.Hub, cfg *config.Config) *Engine {
	repos := queries.NewRepositories(db)
	q := queue.NewPriorityQueue()
	sched := scheduler.New(q, db, hub)
	runner := NewRunner(repos, hub)

	// Wire the scheduler to use the runner for executing pipelines
	sched.SetRunFunc(runner.RunPipeline)

	return &Engine{
		db:        db,
		hub:       hub,
		cfg:       cfg,
		repos:     repos,
		queue:     q,
		scheduler: sched,
		runner:    runner,
	}
}

// Start begins the engine's background goroutines (scheduler loop).
// It is non-blocking; call Stop() to shut down.
func (e *Engine) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		log.Info().Msg("engine: starting scheduler")
		e.scheduler.Start(ctx)
	}()

	log.Info().Msg("engine: started")
}

// Stop gracefully shuts down the engine, waiting for in-progress work to complete.
func (e *Engine) Stop() {
	log.Info().Msg("engine: stopping")
	if e.cancel != nil {
		e.cancel()
	}
	e.queue.Close()
	e.wg.Wait()
	log.Info().Msg("engine: stopped")
}

// TriggerPipeline creates a new pipeline run and enqueues it for execution.
func (e *Engine) TriggerPipeline(ctx context.Context, pipelineID, triggerType string, triggerData map[string]string) (*models.PipelineRun, error) {
	// Fetch the pipeline
	pipeline, err := e.repos.Pipelines.GetByID(ctx, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("pipeline not found: %w", err)
	}

	if pipeline.IsActive == 0 {
		return nil, fmt.Errorf("pipeline %q is not active", pipeline.Name)
	}

	// Get next run number
	number, err := e.repos.Runs.GetNextNumber(ctx, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next run number: %w", err)
	}

	// Serialize trigger data
	var triggerDataJSON *string
	if len(triggerData) > 0 {
		data, err := json.Marshal(triggerData)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize trigger data: %w", err)
		}
		s := string(data)
		triggerDataJSON = &s
	}

	// Create the pipeline run record
	run := &models.PipelineRun{
		PipelineID:  pipelineID,
		Number:      number,
		Status:      "queued",
		TriggerType: triggerType,
		TriggerData: triggerDataJSON,
	}

	// Extract common trigger data fields
	if branch, ok := triggerData["branch"]; ok {
		run.Branch = &branch
	}
	if sha, ok := triggerData["commit_sha"]; ok {
		run.CommitSHA = &sha
	}
	if msg, ok := triggerData["commit_message"]; ok {
		run.CommitMessage = &msg
	}
	if author, ok := triggerData["author"]; ok {
		run.Author = &author
	}
	if tag, ok := triggerData["tag"]; ok {
		run.Tag = &tag
	}

	if err := e.repos.Runs.Create(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to create pipeline run: %w", err)
	}

	// Build the pipeline config for the queue job.
	// If the pipeline has config_content, use it; otherwise use a default.
	pipelineConfig := e.buildPipelineConfig(pipeline)

	configJSON, err := json.Marshal(pipelineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize pipeline config: %w", err)
	}

	// Enqueue the job
	job := &queue.Job{
		ID:            uuid.New().String(),
		PipelineRunID: run.ID,
		Priority:      determinePriority(triggerType),
		CreatedAt:     time.Now(),
		Config:        configJSON,
	}

	if err := e.queue.Enqueue(job); err != nil {
		// If enqueue fails, mark the run as failed
		_ = e.repos.Runs.UpdateStatus(ctx, run.ID, "failure")
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	log.Info().
		Str("pipeline_id", pipelineID).
		Str("run_id", run.ID).
		Int("number", run.Number).
		Str("trigger", triggerType).
		Msg("engine: pipeline triggered")

	return run, nil
}

// buildPipelineConfig converts the pipeline's stored config into a scheduler.PipelineConfig.
// If the pipeline has YAML/JSON content, it attempts to parse it. Otherwise it returns
// a minimal default config.
func (e *Engine) buildPipelineConfig(pipeline *models.Pipeline) scheduler.PipelineConfig {
	// Try to parse stored config_content as a PipelineConfig
	if pipeline.ConfigContent != nil && *pipeline.ConfigContent != "" {
		var config scheduler.PipelineConfig
		if err := json.Unmarshal([]byte(*pipeline.ConfigContent), &config); err == nil && len(config.Stages) > 0 {
			return config
		}
	}

	// Return a default minimal config
	return scheduler.PipelineConfig{
		Stages: []scheduler.StageConfig{
			{
				Name: "default",
				Jobs: []scheduler.JobConfig{
					{
						Name:         "build",
						ExecutorType: "local",
						Steps: []scheduler.StepConfig{
							{
								Name:    "echo",
								Command: "echo 'No pipeline configuration found'",
							},
						},
					},
				},
			},
		},
	}
}

// determinePriority assigns a priority based on trigger type.
// Manual triggers get highest priority, followed by API and push.
func determinePriority(triggerType string) int {
	switch triggerType {
	case "manual":
		return 100
	case "api":
		return 80
	case "push":
		return 50
	case "pull_request":
		return 50
	case "schedule":
		return 30
	case "pipeline":
		return 60
	default:
		return 50
	}
}

// Queue returns the engine's priority queue (for external inspection or testing).
func (e *Engine) Queue() *queue.PriorityQueue {
	return e.queue
}
