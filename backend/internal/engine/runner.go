package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/engine/executor"
	"github.com/oarkflow/deploy/backend/internal/engine/scheduler"
	"github.com/oarkflow/deploy/backend/internal/models"
	"github.com/oarkflow/deploy/backend/internal/pipeline"
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

// preCreatedIDs holds the pre-created DB record IDs for the full pipeline tree.
type preCreatedIDs struct {
	// stageIDs[stageIndex] = stage_run.ID
	stageIDs []string
	// jobIDs[stageIndex][jobIndex] = job_run.ID
	jobIDs [][]string
	// stepIDs[stageIndex][jobIndex][stepIndex] = step_run.ID
	stepIDs [][][]string
}

// RunPipeline executes the pipeline run described by the given config.
// It pre-creates all stage/job/step records as "pending" so the UI can display
// the full tree immediately, then executes stages — either sequentially or in
// parallel via DAG-based level grouping when stages declare `needs` dependencies.
func (r *Runner) RunPipeline(ctx context.Context, runID string, config scheduler.PipelineConfig) error {
	// Pre-create all stage, job, and step records with "pending" status
	ids, err := r.preCreateRecords(ctx, runID, config)
	if err != nil {
		return fmt.Errorf("failed to pre-create run records: %w", err)
	}

	// Broadcast so the frontend can immediately see the full tree
	r.broadcastStatusChange(runID, "pipeline", runID, "", "running")

	// Create a single workspace volume for the entire run, shared across all
	// stages and jobs. This ensures that files from the install stage (e.g.
	// node_modules, vendor/) persist into test and build stages.
	volumeName := fmt.Sprintf("flowforge-ws-%s", runID)
	tempExec, _ := executor.NewExecutor("docker")
	if dockerExec, ok := tempExec.(*executor.DockerExecutor); ok {
		if err := dockerExec.CreateVolume(ctx, volumeName); err != nil {
			return fmt.Errorf("create workspace volume: %w", err)
		}
		defer dockerExec.RemoveVolume(context.Background(), volumeName)
		r.broadcastLog(runID, "", "system", fmt.Sprintf("Created workspace volume: %s\n", volumeName))
	}

	// Check if any stage has Needs defined — if so, use DAG-based execution
	hasNeeds := false
	depsMap := make(map[string][]string, len(config.Stages))
	stageNames := make([]string, len(config.Stages))
	stageByName := make(map[string]int, len(config.Stages))
	for i, stageCfg := range config.Stages {
		stageNames[i] = stageCfg.Name
		stageByName[stageCfg.Name] = i
		if len(stageCfg.Needs) > 0 {
			hasNeeds = true
			depsMap[stageCfg.Name] = stageCfg.Needs
		}
	}

	if hasNeeds {
		return r.runStagesDAG(ctx, runID, config, ids, depsMap, stageNames, stageByName, volumeName)
	}

	// Fallback: sequential execution (backward compatibility)
	return r.runStagesSequential(ctx, runID, config, ids, volumeName)
}

// runStagesSequential executes stages one by one in order (original behavior).
func (r *Runner) runStagesSequential(ctx context.Context, runID string, config scheduler.PipelineConfig, ids *preCreatedIDs, volumeName string) error {
	for i, stageCfg := range config.Stages {
		if err := ctx.Err(); err != nil {
			for j := i; j < len(config.Stages); j++ {
				r.repos.Runs.UpdateStageRunStatus(ctx, ids.stageIDs[j], "cancelled")
				r.broadcastStatusChange(runID, "stage", ids.stageIDs[j], config.Stages[j].Name, "cancelled")
			}
			return fmt.Errorf("pipeline cancelled: %w", err)
		}

		if err := r.runStage(ctx, runID, ids.stageIDs[i], stageCfg, ids.jobIDs[i], ids.stepIDs[i], volumeName); err != nil {
			for j := i + 1; j < len(config.Stages); j++ {
				r.repos.Runs.UpdateStageRunStatus(ctx, ids.stageIDs[j], "skipped")
				r.broadcastStatusChange(runID, "stage", ids.stageIDs[j], config.Stages[j].Name, "skipped")
			}
			return fmt.Errorf("stage %q failed: %w", stageCfg.Name, err)
		}
	}
	return nil
}

// runStagesDAG executes stages grouped by DAG levels. Stages within the same
// level run concurrently; stages at higher levels wait for their dependencies.
func (r *Runner) runStagesDAG(ctx context.Context, runID string, config scheduler.PipelineConfig, ids *preCreatedIDs, depsMap map[string][]string, stageNames []string, stageByName map[string]int, volumeName string) error {
	dag, err := pipeline.BuildStageDAG(stageNames, depsMap)
	if err != nil {
		return fmt.Errorf("invalid stage DAG: %w", err)
	}

	r.broadcastLog(runID, "", "system", fmt.Sprintf("DAG execution: %d levels detected\n", len(dag.Levels)))

	// Track which stages failed so we can skip dependents
	failedStages := make(map[string]bool)
	var failedMu sync.Mutex

	for levelIdx, level := range dag.Levels {
		if err := ctx.Err(); err != nil {
			// Cancel all remaining stages
			for li := levelIdx; li < len(dag.Levels); li++ {
				for _, stageName := range dag.Levels[li] {
					idx := stageByName[stageName]
					r.repos.Runs.UpdateStageRunStatus(ctx, ids.stageIDs[idx], "cancelled")
					r.broadcastStatusChange(runID, "stage", ids.stageIDs[idx], stageName, "cancelled")
				}
			}
			return fmt.Errorf("pipeline cancelled: %w", err)
		}

		r.broadcastLog(runID, "", "system", fmt.Sprintf("=== DAG Level %d: [%s] ===\n", levelIdx, strings.Join(level, ", ")))

		var wg sync.WaitGroup
		var levelErrs []error
		var errMu sync.Mutex

		for _, stageName := range level {
			stageIdx := stageByName[stageName]

			// Check if any dependency failed → skip this stage
			shouldSkip := false
			failedMu.Lock()
			for _, dep := range depsMap[stageName] {
				if failedStages[dep] {
					shouldSkip = true
					break
				}
			}
			failedMu.Unlock()

			if shouldSkip {
				r.repos.Runs.UpdateStageRunStatus(ctx, ids.stageIDs[stageIdx], "skipped")
				r.broadcastStatusChange(runID, "stage", ids.stageIDs[stageIdx], stageName, "skipped")
				r.broadcastLog(runID, "", "system", fmt.Sprintf("Stage %q skipped (dependency failed)\n", stageName))
				failedMu.Lock()
				failedStages[stageName] = true
				failedMu.Unlock()
				continue
			}

			wg.Add(1)
			go func(name string, idx int) {
				defer wg.Done()
				stageCfg := config.Stages[idx]
				if stageErr := r.runStage(ctx, runID, ids.stageIDs[idx], stageCfg, ids.jobIDs[idx], ids.stepIDs[idx], volumeName); stageErr != nil {
					errMu.Lock()
					levelErrs = append(levelErrs, fmt.Errorf("stage %q failed: %w", name, stageErr))
					errMu.Unlock()
					failedMu.Lock()
					failedStages[name] = true
					failedMu.Unlock()
				}
			}(stageName, stageIdx)
		}

		wg.Wait()

		// If all stages in this level failed, we can still continue —
		// dependents will be skipped. Only return error at the very end.
	}

	// Check if any stage failed
	if len(failedStages) > 0 {
		var names []string
		for name := range failedStages {
			names = append(names, name)
		}
		return fmt.Errorf("stages failed: %s", strings.Join(names, ", "))
	}

	return nil
}

// preCreateRecords inserts all stage_run, job_run, and step_run records as "pending"
// before any execution begins. This lets the frontend sidebar show the full tree.
func (r *Runner) preCreateRecords(ctx context.Context, runID string, config scheduler.PipelineConfig) (*preCreatedIDs, error) {
	ids := &preCreatedIDs{
		stageIDs: make([]string, len(config.Stages)),
		jobIDs:   make([][]string, len(config.Stages)),
		stepIDs:  make([][][]string, len(config.Stages)),
	}

	for i, stageCfg := range config.Stages {
		stageRun := &models.StageRun{
			RunID:    runID,
			Name:     stageCfg.Name,
			Status:   "pending",
			Position: i,
		}
		if err := r.repos.Runs.CreateStageRun(ctx, stageRun); err != nil {
			return nil, fmt.Errorf("failed to pre-create stage %q: %w", stageCfg.Name, err)
		}
		ids.stageIDs[i] = stageRun.ID

		ids.jobIDs[i] = make([]string, len(stageCfg.Jobs))
		ids.stepIDs[i] = make([][]string, len(stageCfg.Jobs))

		for j, jobCfg := range stageCfg.Jobs {
			executorType := jobCfg.ExecutorType
			if executorType == "" {
				executorType = "docker"
			}
			jobRun := &models.JobRun{
				StageRunID:   stageRun.ID,
				RunID:        runID,
				Name:         jobCfg.Name,
				Status:       "pending",
				ExecutorType: executorType,
			}
			if err := r.repos.Runs.CreateJobRun(ctx, jobRun); err != nil {
				return nil, fmt.Errorf("failed to pre-create job %q: %w", jobCfg.Name, err)
			}
			ids.jobIDs[i][j] = jobRun.ID

			ids.stepIDs[i][j] = make([]string, len(jobCfg.Steps))
			for k, stepCfg := range jobCfg.Steps {
				stepRun := &models.StepRun{
					JobRunID: jobRun.ID,
					Name:     stepCfg.Name,
					Status:   "pending",
				}
				if err := r.repos.Runs.CreateStepRun(ctx, stepRun); err != nil {
					return nil, fmt.Errorf("failed to pre-create step %q: %w", stepCfg.Name, err)
				}
				ids.stepIDs[i][j][k] = stepRun.ID
			}
		}
	}

	return ids, nil
}

// runStage executes all jobs in a stage in parallel and updates status.
func (r *Runner) runStage(ctx context.Context, runID, stageRunID string, stageCfg scheduler.StageConfig, jobIDs []string, stepIDs [][]string, volumeName string) error {
	// Mark stage as running
	now := time.Now()
	r.repos.Runs.SetStageRunStarted(ctx, stageRunID)
	_ = now // started_at set by SetStageRunStarted

	r.broadcastStatusChange(runID, "stage", stageRunID, stageCfg.Name, "running")
	r.broadcastLog(runID, "", "system", fmt.Sprintf("=== Stage: %s ===\n", stageCfg.Name))

	// Run all jobs within this stage in parallel
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		jobErrs []error
	)

	for j, jobCfg := range stageCfg.Jobs {
		wg.Add(1)
		go func(jc scheduler.JobConfig, jobRunID string, stpIDs []string) {
			defer wg.Done()
			if err := r.runJob(ctx, runID, jobRunID, jc, stpIDs, volumeName); err != nil {
				mu.Lock()
				jobErrs = append(jobErrs, err)
				mu.Unlock()
			}
		}(jobCfg, jobIDs[j], stepIDs[j])
	}

	wg.Wait()

	// Determine stage status
	status := "success"
	if len(jobErrs) > 0 {
		status = "failure"
	}

	if err := r.repos.Runs.SetStageRunFinished(ctx, stageRunID, status); err != nil {
		log.Error().Err(err).Str("stage_run_id", stageRunID).Msg("runner: failed to update stage status")
	}

	r.broadcastStatusChange(runID, "stage", stageRunID, stageCfg.Name, status)
	r.broadcastLog(runID, "", "system", fmt.Sprintf("=== Stage: %s — %s ===\n", stageCfg.Name, status))

	if status == "failure" {
		return fmt.Errorf("stage %q: %d job(s) failed", stageCfg.Name, len(jobErrs))
	}
	return nil
}

// runJob executes all steps in a job sequentially and updates status.
func (r *Runner) runJob(ctx context.Context, runID, jobRunID string, jobCfg scheduler.JobConfig, stepIDs []string, volumeName string) error {
	executorType := jobCfg.ExecutorType
	if executorType == "" {
		executorType = "docker"
	}

	// Mark job as running
	if err := r.repos.Runs.UpdateJobRunStatus(ctx, jobRunID, "running"); err != nil {
		log.Error().Err(err).Str("job_run_id", jobRunID).Msg("runner: failed to mark job as running")
	}

	r.broadcastStatusChange(runID, "job", jobRunID, jobCfg.Name, "running")
	r.broadcastLog(runID, "", "system", fmt.Sprintf("--- Job: %s (executor: %s) ---\n", jobCfg.Name, executorType))

	// Create the executor
	exec, err := executor.NewExecutor(executorType)
	if err != nil {
		r.failJob(ctx, runID, jobRunID, stepIDs, err.Error())
		return err
	}

	// For Docker executor: use the shared workspace volume created at the run
	// level, so all stages and jobs share the same /workspace directory.
	if dockerExec, ok := exec.(*executor.DockerExecutor); ok {
		dockerExec.WorkspaceVolume = volumeName
	}

	// Execute steps sequentially
	for k, stepCfg := range jobCfg.Steps {
		if err := ctx.Err(); err != nil {
			r.failJob(ctx, runID, jobRunID, stepIDs[k:], "cancelled")
			return fmt.Errorf("job cancelled: %w", err)
		}

		if err := r.runStep(ctx, runID, stepIDs[k], stepCfg, exec); err != nil {
			// Mark remaining steps as skipped
			for _, remainingStepID := range stepIDs[k+1:] {
				r.repos.Runs.UpdateStepRunStatus(ctx, remainingStepID, "skipped")
				r.broadcastStatusChange(runID, "step", remainingStepID, "", "skipped")
			}
			r.failJobStatus(ctx, runID, jobRunID)
			return fmt.Errorf("step %q failed: %w", stepCfg.Name, err)
		}
	}

	// Mark job as successful
	if err := r.repos.Runs.UpdateJobRunStatus(ctx, jobRunID, "success"); err != nil {
		log.Error().Err(err).Str("job_run_id", jobRunID).Msg("runner: failed to update job status")
	}
	r.broadcastStatusChange(runID, "job", jobRunID, jobCfg.Name, "success")

	return nil
}

// runStep executes a single step command, streams logs, and updates status.
func (r *Runner) runStep(ctx context.Context, runID, stepRunID string, stepCfg scheduler.StepConfig, exec executor.Executor) error {
	// Mark step as running
	now := time.Now()
	stepRun := &models.StepRun{
		ID:        stepRunID,
		Status:    "running",
		StartedAt: &now,
	}
	if err := r.repos.Runs.UpdateStepRun(ctx, stepRun); err != nil {
		log.Error().Err(err).Str("step_run_id", stepRunID).Msg("runner: failed to mark step as running")
	}

	r.broadcastStatusChange(runID, "step", stepRunID, stepCfg.Name, "running")
	r.broadcastLog(runID, stepRunID, "system", fmt.Sprintf(">>> Step: %s\n", stepCfg.Name))

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
	var lastStderrLine string
	logWriter := func(stream string, content []byte) {
		logEntry := &models.RunLog{
			RunID:     runID,
			StepRunID: &stepRunID,
			Stream:    stream,
			Content:   string(content),
		}
		if err := r.repos.Logs.Insert(ctx, logEntry); err != nil {
			log.Error().Err(err).Msg("runner: failed to persist log line")
		}
		if stream == "stderr" {
			if line := lastNonEmptyLine(string(content)); line != "" {
				lastStderrLine = line
			}
		}
		r.broadcastLogEntry(runID, stepRunID, stream, content)
	}

	// Execute the step, preferring streaming if supported
	var result *executor.ExecutionResult
	var execErr error

	if streamExec, ok := exec.(executor.StreamingExecutor); ok {
		// Streaming executor: logWriter is called in real-time during execution.
		// Do NOT re-broadcast the buffered stdout/stderr afterwards.
		result, execErr = streamExec.ExecuteWithLogs(ctx, step, logWriter)
	} else {
		// Non-streaming executor: logs come back as buffered result only.
		result, execErr = exec.Execute(ctx, step)
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
			// Try to extract the actual error message from stderr output
			stderrLine := lastNonEmptyLine(result.Stderr)
			if stderrLine == "" {
				stderrLine = lastStderrLine // fallback to streamed stderr
			}
			if stderrLine != "" {
				errMsg = fmt.Sprintf("%s (exit code %d)", stderrLine, result.ExitCode)
			} else {
				errMsg = fmt.Sprintf("process exited with code %d", result.ExitCode)
			}
		}
		stepRun.ErrorMessage = &errMsg

		if updateErr := r.repos.Runs.UpdateStepRun(ctx, stepRun); updateErr != nil {
			log.Error().Err(updateErr).Str("step_run_id", stepRunID).Msg("runner: failed to update step run")
		}

		r.broadcastStatusChange(runID, "step", stepRunID, stepCfg.Name, "failure")
		r.broadcastLog(runID, stepRunID, "system", fmt.Sprintf("<<< Step: %s — FAILED (%s)\n", stepCfg.Name, errMsg))
		return fmt.Errorf("step %q: %s", stepCfg.Name, errMsg)
	}

	stepRun.Status = "success"
	if updateErr := r.repos.Runs.UpdateStepRun(ctx, stepRun); updateErr != nil {
		log.Error().Err(updateErr).Str("step_run_id", stepRunID).Msg("runner: failed to update step run")
	}

	r.broadcastStatusChange(runID, "step", stepRunID, stepCfg.Name, "success")
	r.broadcastLog(runID, stepRunID, "system", fmt.Sprintf("<<< Step: %s — SUCCESS\n", stepCfg.Name))
	return nil
}

// failJob marks a job and its remaining steps as failed/cancelled.
func (r *Runner) failJob(ctx context.Context, runID, jobRunID string, remainingStepIDs []string, _ string) {
	for _, stepID := range remainingStepIDs {
		r.repos.Runs.UpdateStepRunStatus(ctx, stepID, "cancelled")
		r.broadcastStatusChange(runID, "step", stepID, "", "cancelled")
	}
	r.failJobStatus(ctx, runID, jobRunID)
}

// failJobStatus marks a job as failed.
func (r *Runner) failJobStatus(ctx context.Context, runID, jobRunID string) {
	if err := r.repos.Runs.UpdateJobRunStatus(ctx, jobRunID, "failure"); err != nil {
		log.Error().Err(err).Str("job_run_id", jobRunID).Msg("runner: failed to update job status to failure")
	}
	r.broadcastStatusChange(runID, "job", jobRunID, "", "failure")
}

// broadcastLog sends a system-level log message to WebSocket clients and persists it.
func (r *Runner) broadcastLog(runID, stepRunID, stream, content string) {
	logEntry := &models.RunLog{
		RunID:   runID,
		Stream:  stream,
		Content: content,
	}
	if stepRunID != "" {
		logEntry.StepRunID = &stepRunID
	}
	if err := r.repos.Logs.Insert(context.Background(), logEntry); err != nil {
		log.Error().Err(err).Msg("runner: failed to persist system log")
	}

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

// broadcastStatusChange sends a structured status update for a stage/job/step
// so the frontend can refetch and update the sidebar tree in real-time.
func (r *Runner) broadcastStatusChange(runID, entity, entityID, name, status string) {
	msg, _ := json.Marshal(map[string]string{
		"type":      "status_change",
		"run_id":    runID,
		"entity":    entity,
		"entity_id": entityID,
		"name":      name,
		"status":    status,
	})
	r.hub.BroadcastToRun(runID, msg)
}

// lastNonEmptyLine returns the last non-empty line from the given text.
// Useful for extracting the actual error message from stderr output.
func lastNonEmptyLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}
	return ""
}
