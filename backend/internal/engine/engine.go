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
	"github.com/oarkflow/deploy/backend/internal/pipeline"
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
// It supports both the FlowForge YAML DSL and direct JSON scheduler config.
func (e *Engine) buildPipelineConfig(p *models.Pipeline) scheduler.PipelineConfig {
	if p.ConfigContent == nil || *p.ConfigContent == "" {
		return defaultPipelineConfig()
	}

	content := *p.ConfigContent

	// 1. Try direct JSON unmarshal into scheduler.PipelineConfig (for configs stored as scheduler JSON)
	var direct scheduler.PipelineConfig
	if err := json.Unmarshal([]byte(content), &direct); err == nil && len(direct.Stages) > 0 {
		return direct
	}

	// 2. Parse as FlowForge YAML/JSON DSL using the pipeline parser
	spec, err := pipeline.Parse(content)
	if err != nil {
		log.Warn().Err(err).Str("pipeline_id", p.ID).Msg("engine: failed to parse pipeline config, using default")
		return defaultPipelineConfig()
	}

	return specToSchedulerConfig(spec)
}

// specToSchedulerConfig converts a parsed PipelineSpec into a scheduler.PipelineConfig
// by grouping jobs by stage and mapping step fields.
func specToSchedulerConfig(spec *pipeline.PipelineSpec) scheduler.PipelineConfig {
	if spec == nil || len(spec.Jobs) == 0 {
		return defaultPipelineConfig()
	}

	// Determine stage ordering: use explicit stages list if provided,
	// otherwise collect stages from jobs in insertion order (map iteration order is random,
	// so fall back to alphabetical).
	stageOrder := spec.Stages
	if len(stageOrder) == 0 {
		seen := make(map[string]bool)
		for _, job := range spec.Jobs {
			stageName := job.Stage
			if stageName == "" {
				stageName = "default"
			}
			if !seen[stageName] {
				stageOrder = append(stageOrder, stageName)
				seen[stageName] = true
			}
		}
	}

	// Group jobs by stage name
	stageJobs := make(map[string][]scheduler.JobConfig)
	for jobName, jobSpec := range spec.Jobs {
		stageName := jobSpec.Stage
		if stageName == "" {
			stageName = "default"
		}

		// Convert steps
		var steps []scheduler.StepConfig
		for i, stepSpec := range jobSpec.Steps {
			cmd := stepSpec.Run
			if cmd == "" && stepSpec.Uses != "" {
				// Built-in actions aren't directly executable — emit a placeholder
				cmd = fmt.Sprintf("echo '[flowforge] action: %s (not yet implemented)'", stepSpec.Uses)
			}
			if cmd == "" {
				continue // Skip steps with no command
			}

			name := stepSpec.Name
			if name == "" {
				name = fmt.Sprintf("step-%d", i+1)
			}

			// Merge job-level env into step env
			env := make(map[string]string)
			for k, v := range spec.Env {
				env[k] = v
			}
			for k, v := range jobSpec.Env {
				env[k] = v
			}
			for k, v := range stepSpec.Env {
				env[k] = v
			}

			steps = append(steps, scheduler.StepConfig{
				Name:    name,
				Command: cmd,
				Env:     env,
				Timeout: jobSpec.Timeout,
			})
		}

		if len(steps) == 0 {
			continue // Skip jobs with no executable steps
		}

		// Determine executor type — fall back to "local" since docker/k8s
		// may not be available on the embedded worker
		executorType := jobSpec.Executor
		if executorType == "" && spec.Defaults != nil {
			executorType = spec.Defaults.Executor
		}
		if executorType == "" {
			executorType = "local"
		}

		jc := scheduler.JobConfig{
			Name:         jobName,
			ExecutorType: executorType,
			Steps:        steps,
		}

		stageJobs[stageName] = append(stageJobs[stageName], jc)
	}

	// Build ordered stage configs
	var stages []scheduler.StageConfig
	for _, stageName := range stageOrder {
		jobs, ok := stageJobs[stageName]
		if !ok || len(jobs) == 0 {
			continue
		}
		stages = append(stages, scheduler.StageConfig{
			Name: stageName,
			Jobs: jobs,
		})
	}

	if len(stages) == 0 {
		return defaultPipelineConfig()
	}

	return scheduler.PipelineConfig{Stages: stages}
}

// defaultPipelineConfig returns a minimal pipeline config that just echoes a message.
func defaultPipelineConfig() scheduler.PipelineConfig {
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

// ReenqueueRun re-enqueues a pipeline run that is stuck in "queued" status in the DB.
// This is used by the stale run recovery worker to pick up orphaned runs after a restart.
func (e *Engine) ReenqueueRun(ctx context.Context, runID string) error {
	run, err := e.repos.Runs.GetByID(ctx, runID)
	if err != nil {
		return fmt.Errorf("run not found: %w", err)
	}

	if run.Status != "queued" {
		return nil // Already running or finished
	}

	pipeline, err := e.repos.Pipelines.GetByID(ctx, run.PipelineID)
	if err != nil {
		return fmt.Errorf("pipeline not found: %w", err)
	}

	pipelineConfig := e.buildPipelineConfig(pipeline)
	configJSON, err := json.Marshal(pipelineConfig)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	job := &queue.Job{
		ID:            uuid.New().String(),
		PipelineRunID: run.ID,
		Priority:      determinePriority(run.TriggerType),
		CreatedAt:     time.Now(),
		Config:        configJSON,
	}

	if err := e.queue.Enqueue(job); err != nil {
		return fmt.Errorf("failed to enqueue: %w", err)
	}

	log.Info().
		Str("run_id", runID).
		Msg("engine: re-enqueued stale run")

	return nil
}
