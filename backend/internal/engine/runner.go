package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/engine/executor"
	"github.com/oarkflow/deploy/backend/internal/engine/scheduler"
	"github.com/oarkflow/deploy/backend/internal/models"
	"github.com/oarkflow/deploy/backend/internal/websocket"
)

// Runner executes a full pipeline run: stages sequentially, jobs in parallel
// within a stage, and steps sequentially within a job.
type Runner struct {
	repos *queries.Repositories
	hub   *websocket.Hub
}

// NewRunner creates a new Runner.
func NewRunner(repos *queries.Repositories, hub *websocket.Hub) *Runner {
	return &Runner{
		repos: repos,
		hub:   hub,
	}
}

// RunPipeline executes the pipeline run described by the given config.
// Stages are executed sequentially. Within each stage, jobs run in parallel.
// Within each job, steps run sequentially. If any stage fails, the pipeline stops.
func (r *Runner) RunPipeline(ctx context.Context, runID string, config scheduler.PipelineConfig) error {
	for i, stageCfg := range config.Stages {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("pipeline cancelled: %w", err)
		}

		if err := r.runStage(ctx, runID, stageCfg, i); err != nil {
			return fmt.Errorf("stage %q failed: %w", stageCfg.Name, err)
		}
	}
	return nil
}

// runStage creates a stage_run record, executes all jobs in parallel, and
// updates the stage status when complete.
func (r *Runner) runStage(ctx context.Context, runID string, stageCfg scheduler.StageConfig, position int) error {
	// Create stage run record
	stageRun := &models.StageRun{
		RunID:    runID,
		Name:     stageCfg.Name,
		Status:   "running",
		Position: position,
	}
	now := time.Now()
	stageRun.StartedAt = &now
	if err := r.repos.Runs.CreateStageRun(ctx, stageRun); err != nil {
		return fmt.Errorf("failed to create stage run: %w", err)
	}

	r.broadcastLog(runID, "", "system", fmt.Sprintf("=== Stage: %s ===\n", stageCfg.Name))

	// Run all jobs within this stage in parallel
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		jobErrs []error
	)

	for _, jobCfg := range stageCfg.Jobs {
		wg.Add(1)
		go func(jc scheduler.JobConfig) {
			defer wg.Done()
			if err := r.runJob(ctx, runID, stageRun.ID, jc); err != nil {
				mu.Lock()
				jobErrs = append(jobErrs, err)
				mu.Unlock()
			}
		}(jobCfg)
	}

	wg.Wait()

	// Determine stage status
	status := "success"
	if len(jobErrs) > 0 {
		status = "failure"
	}

	if err := r.repos.Runs.SetStageRunFinished(ctx, stageRun.ID, status); err != nil {
		log.Error().Err(err).Str("stage_run_id", stageRun.ID).Msg("runner: failed to update stage status")
	}

	r.broadcastLog(runID, "", "system", fmt.Sprintf("=== Stage: %s — %s ===\n", stageCfg.Name, status))

	if status == "failure" {
		return fmt.Errorf("stage %q: %d job(s) failed", stageCfg.Name, len(jobErrs))
	}
	return nil
}

// runJob creates a job_run record, executes all steps sequentially, and updates status.
func (r *Runner) runJob(ctx context.Context, runID, stageRunID string, jobCfg scheduler.JobConfig) error {
	executorType := jobCfg.ExecutorType
	if executorType == "" {
		executorType = "local"
	}

	jobRun := &models.JobRun{
		StageRunID:   stageRunID,
		RunID:        runID,
		Name:         jobCfg.Name,
		Status:       "running",
		ExecutorType: executorType,
	}
	now := time.Now()
	jobRun.StartedAt = &now
	if err := r.repos.Runs.CreateJobRun(ctx, jobRun); err != nil {
		return fmt.Errorf("failed to create job run: %w", err)
	}

	r.broadcastLog(runID, "", "system", fmt.Sprintf("--- Job: %s (executor: %s) ---\n", jobCfg.Name, executorType))

	// Create the executor
	exec, err := executor.NewExecutor(executorType)
	if err != nil {
		r.failJob(ctx, jobRun.ID, err.Error())
		return err
	}

	// Execute steps sequentially
	for _, stepCfg := range jobCfg.Steps {
		if err := ctx.Err(); err != nil {
			r.failJob(ctx, jobRun.ID, "cancelled")
			return fmt.Errorf("job cancelled: %w", err)
		}

		if err := r.runStep(ctx, runID, jobRun.ID, stepCfg, exec); err != nil {
			r.failJob(ctx, jobRun.ID, err.Error())
			return fmt.Errorf("step %q failed: %w", stepCfg.Name, err)
		}
	}

	// Mark job as successful
	if err := r.repos.Runs.UpdateJobRunStatus(ctx, jobRun.ID, "success"); err != nil {
		log.Error().Err(err).Str("job_run_id", jobRun.ID).Msg("runner: failed to update job status")
	}

	return nil
}

// runStep creates a step_run record, executes the command, streams logs,
// and updates the step status.
func (r *Runner) runStep(ctx context.Context, runID, jobRunID string, stepCfg scheduler.StepConfig, exec executor.Executor) error {
	stepRun := &models.StepRun{
		JobRunID: jobRunID,
		Name:     stepCfg.Name,
		Status:   "running",
	}
	now := time.Now()
	stepRun.StartedAt = &now
	if err := r.repos.Runs.CreateStepRun(ctx, stepRun); err != nil {
		return fmt.Errorf("failed to create step run: %w", err)
	}

	r.broadcastLog(runID, stepRun.ID, "system", fmt.Sprintf(">>> Step: %s\n", stepCfg.Name))

	// Parse timeout
	var timeout time.Duration
	if stepCfg.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(stepCfg.Timeout)
		if err != nil {
			log.Warn().Err(err).Str("timeout", stepCfg.Timeout).Msg("runner: invalid timeout, using no timeout")
		}
	}

	step := executor.ExecutionStep{
		Name:    stepCfg.Name,
		Command: stepCfg.Command,
		WorkDir: stepCfg.WorkDir,
		Env:     stepCfg.Env,
		Timeout: timeout,
	}

	// Create log writer for real-time streaming
	logWriter := func(stream string, content []byte) {
		// Persist the log line
		logEntry := &models.RunLog{
			RunID:     runID,
			StepRunID: &stepRun.ID,
			Stream:    stream,
			Content:   string(content),
		}
		if err := r.repos.Logs.Insert(ctx, logEntry); err != nil {
			log.Error().Err(err).Msg("runner: failed to persist log line")
		}

		// Broadcast to WebSocket
		r.broadcastLogEntry(runID, stepRun.ID, stream, content)
	}

	// Execute the step, preferring streaming if supported
	var result *executor.ExecutionResult
	var execErr error

	if streamExec, ok := exec.(executor.StreamingExecutor); ok {
		result, execErr = streamExec.ExecuteWithLogs(ctx, step, logWriter)
	} else {
		result, execErr = exec.Execute(ctx, step)
		// For non-streaming executors, write captured output as logs after execution
		if result != nil {
			if result.Stdout != "" {
				logWriter("stdout", []byte(result.Stdout))
			}
			if result.Stderr != "" {
				logWriter("stderr", []byte(result.Stderr))
			}
		}
	}

	// Update step run with results
	finishedAt := time.Now()
	stepRun.FinishedAt = &finishedAt

	if result != nil {
		stepRun.ExitCode = &result.ExitCode
		durationMs := int(result.Duration.Milliseconds())
		stepRun.DurationMs = &durationMs
	}

	if execErr != nil || (result != nil && result.ExitCode != 0) {
		stepRun.Status = "failure"
		errMsg := ""
		if execErr != nil {
			errMsg = execErr.Error()
		} else if result != nil && result.ExitCode != 0 {
			errMsg = fmt.Sprintf("process exited with code %d", result.ExitCode)
		}
		stepRun.ErrorMessage = &errMsg

		if updateErr := r.repos.Runs.UpdateStepRun(ctx, stepRun); updateErr != nil {
			log.Error().Err(updateErr).Str("step_run_id", stepRun.ID).Msg("runner: failed to update step run")
		}

		r.broadcastLog(runID, stepRun.ID, "system", fmt.Sprintf("<<< Step: %s — FAILED (%s)\n", stepCfg.Name, errMsg))
		return fmt.Errorf("step %q: %s", stepCfg.Name, errMsg)
	}

	stepRun.Status = "success"
	if updateErr := r.repos.Runs.UpdateStepRun(ctx, stepRun); updateErr != nil {
		log.Error().Err(updateErr).Str("step_run_id", stepRun.ID).Msg("runner: failed to update step run")
	}

	r.broadcastLog(runID, stepRun.ID, "system", fmt.Sprintf("<<< Step: %s — SUCCESS\n", stepCfg.Name))
	return nil
}

// failJob marks a job run as failed.
func (r *Runner) failJob(ctx context.Context, jobRunID string, errMsg string) {
	if err := r.repos.Runs.UpdateJobRunStatus(ctx, jobRunID, "failure"); err != nil {
		log.Error().Err(err).Str("job_run_id", jobRunID).Msg("runner: failed to update job status to failure")
	}
}

// broadcastLog sends a system-level log message to WebSocket clients.
func (r *Runner) broadcastLog(runID, stepRunID, stream, content string) {
	r.broadcastLogEntry(runID, stepRunID, stream, []byte(content))
}

// broadcastLogEntry sends a structured log entry over WebSocket.
func (r *Runner) broadcastLogEntry(runID, stepRunID, stream string, content []byte) {
	msg, _ := json.Marshal(map[string]string{
		"type":        "log",
		"run_id":      runID,
		"step_run_id": stepRunID,
		"stream":      stream,
		"content":     string(content),
	})
	r.hub.BroadcastToRun(runID, msg)
}
