package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type RunRepo struct {
	db *sqlx.DB
}

// PipelineRun
func (r *RunRepo) GetByID(ctx context.Context, id string) (*models.PipelineRun, error) {
	run := &models.PipelineRun{}
	err := r.db.GetContext(ctx, run, "SELECT * FROM pipeline_runs WHERE id = ?", id)
	return run, err
}

func (r *RunRepo) ListByPipeline(ctx context.Context, pipelineID string, limit, offset int) ([]models.PipelineRun, error) {
	runs := []models.PipelineRun{}
	err := r.db.SelectContext(ctx, &runs, "SELECT * FROM pipeline_runs WHERE pipeline_id = ? ORDER BY number DESC LIMIT ? OFFSET ?", pipelineID, limit, offset)
	return runs, err
}

// ListAllRuns returns runs across all pipelines/projects with metadata, ordered by most recent first.
func (r *RunRepo) ListAllRuns(ctx context.Context, status string, limit, offset int) ([]models.PipelineRunWithMeta, error) {
	runs := []models.PipelineRunWithMeta{}
	query := `
		SELECT
			pr.*,
			p.name AS pipeline_name,
			p.is_active AS pipeline_is_active,
			proj.id AS project_id,
			proj.name AS project_name
		FROM pipeline_runs pr
		JOIN pipelines p ON p.id = pr.pipeline_id
		JOIN projects proj ON proj.id = p.project_id
		WHERE proj.deleted_at IS NULL`

	args := []interface{}{}
	if status != "" {
		query += " AND pr.status = ?"
		args = append(args, status)
	}
	query += " ORDER BY pr.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := r.db.SelectContext(ctx, &runs, query, args...)
	return runs, err
}

func (r *RunRepo) Create(ctx context.Context, run *models.PipelineRun) error {
	run.ID = uuid.New().String()
	run.CreatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO pipeline_runs (id, pipeline_id, number, status, trigger_type, trigger_data, commit_sha, commit_message, branch, tag, author, created_by, created_at)
		VALUES (:id, :pipeline_id, :number, :status, :trigger_type, :trigger_data, :commit_sha, :commit_message, :branch, :tag, :author, :created_by, :created_at)`,
		run)
	return err
}

func (r *RunRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE pipeline_runs SET status = ? WHERE id = ?", status, id)
	return err
}

func (r *RunRepo) SetStarted(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, "UPDATE pipeline_runs SET status = 'running', started_at = ? WHERE id = ?", now, id)
	return err
}

func (r *RunRepo) SetFinished(ctx context.Context, id, status string, durationMs int, errorSummary string) error {
	now := time.Now()
	if errorSummary != "" {
		_, err := r.db.ExecContext(ctx, "UPDATE pipeline_runs SET status = ?, finished_at = ?, duration_ms = ?, error_summary = ? WHERE id = ?", status, now, durationMs, errorSummary, id)
		return err
	}
	_, err := r.db.ExecContext(ctx, "UPDATE pipeline_runs SET status = ?, finished_at = ?, duration_ms = ? WHERE id = ?", status, now, durationMs, id)
	return err
}

func (r *RunRepo) GetNextNumber(ctx context.Context, pipelineID string) (int, error) {
	var n int
	err := r.db.GetContext(ctx, &n, "SELECT COALESCE(MAX(number), 0) + 1 FROM pipeline_runs WHERE pipeline_id = ?", pipelineID)
	return n, err
}

func (r *RunRepo) SetDeployURL(ctx context.Context, id, url string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE pipeline_runs SET deploy_url = ? WHERE id = ?", url, id)
	return err
}

// StageRun
func (r *RunRepo) CreateStageRun(ctx context.Context, s *models.StageRun) error {
	s.ID = uuid.New().String()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO stage_runs (id, run_id, name, status, position) VALUES (:id, :run_id, :name, :status, :position)`,
		s)
	return err
}

func (r *RunRepo) ListStageRuns(ctx context.Context, runID string) ([]models.StageRun, error) {
	stages := []models.StageRun{}
	err := r.db.SelectContext(ctx, &stages, "SELECT * FROM stage_runs WHERE run_id = ? ORDER BY position", runID)
	return stages, err
}

func (r *RunRepo) UpdateStageRunStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE stage_runs SET status = ? WHERE id = ?", status, id)
	return err
}

// JobRun
func (r *RunRepo) CreateJobRun(ctx context.Context, j *models.JobRun) error {
	j.ID = uuid.New().String()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO job_runs (id, stage_run_id, run_id, name, status, agent_id, executor_type)
		VALUES (:id, :stage_run_id, :run_id, :name, :status, :agent_id, :executor_type)`,
		j)
	return err
}

func (r *RunRepo) ListJobRuns(ctx context.Context, stageRunID string) ([]models.JobRun, error) {
	jobs := []models.JobRun{}
	err := r.db.SelectContext(ctx, &jobs, "SELECT * FROM job_runs WHERE stage_run_id = ?", stageRunID)
	return jobs, err
}

func (r *RunRepo) UpdateJobRunStatus(ctx context.Context, id, status string) error {
	now := time.Now()
	if status == "running" {
		_, err := r.db.ExecContext(ctx, "UPDATE job_runs SET status = ?, started_at = ? WHERE id = ?", status, now, id)
		return err
	}
	_, err := r.db.ExecContext(ctx, "UPDATE job_runs SET status = ?, finished_at = ? WHERE id = ?", status, now, id)
	return err
}

func (r *RunRepo) SetStageRunStarted(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, "UPDATE stage_runs SET status = 'running', started_at = ? WHERE id = ?", now, id)
	return err
}

func (r *RunRepo) SetStageRunFinished(ctx context.Context, id, status string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, "UPDATE stage_runs SET status = ?, finished_at = ? WHERE id = ?", status, now, id)
	return err
}

// StepRun
func (r *RunRepo) CreateStepRun(ctx context.Context, s *models.StepRun) error {
	s.ID = uuid.New().String()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO step_runs (id, job_run_id, name, status) VALUES (:id, :job_run_id, :name, :status)`,
		s)
	return err
}

func (r *RunRepo) ListStepRuns(ctx context.Context, jobRunID string) ([]models.StepRun, error) {
	steps := []models.StepRun{}
	err := r.db.SelectContext(ctx, &steps, "SELECT * FROM step_runs WHERE job_run_id = ?", jobRunID)
	return steps, err
}

func (r *RunRepo) UpdateStepRun(ctx context.Context, s *models.StepRun) error {
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE step_runs SET status=:status, exit_code=:exit_code, error_message=:error_message, started_at=:started_at, finished_at=:finished_at, duration_ms=:duration_ms WHERE id=:id`,
		s)
	return err
}

// UpdateStepRunStatus updates just the status field of a step run.
func (r *RunRepo) UpdateStepRunStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE step_runs SET status = ? WHERE id = ?", status, id)
	return err
}

// ListJobRunsByRunID returns all job runs for a given pipeline run.
func (r *RunRepo) ListJobRunsByRunID(ctx context.Context, runID string) ([]models.JobRun, error) {
	jobs := []models.JobRun{}
	err := r.db.SelectContext(ctx, &jobs, "SELECT * FROM job_runs WHERE run_id = ? ORDER BY id ASC", runID)
	return jobs, err
}

// ListStepRunsByRunID returns all step runs for a given pipeline run via job_runs.
func (r *RunRepo) ListStepRunsByRunID(ctx context.Context, runID string) ([]models.StepRun, error) {
	steps := []models.StepRun{}
	err := r.db.SelectContext(ctx, &steps,
		`SELECT s.* FROM step_runs s
		 JOIN job_runs j ON j.id = s.job_run_id
		 WHERE j.run_id = ?
		 ORDER BY s.id ASC`, runID)
	return steps, err
}
