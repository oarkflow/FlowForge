package queries

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// ScheduleRepo provides database operations for pipeline schedules.
type ScheduleRepo struct {
	db *sqlx.DB
}

// ListByPipeline returns all schedules for a pipeline.
func (r *ScheduleRepo) ListByPipeline(ctx context.Context, pipelineID string) ([]models.PipelineSchedule, error) {
	var schedules []models.PipelineSchedule
	err := r.db.SelectContext(ctx, &schedules,
		`SELECT * FROM pipeline_schedules WHERE pipeline_id = ? ORDER BY created_at DESC`, pipelineID)
	if err != nil {
		return nil, err
	}
	if schedules == nil {
		schedules = []models.PipelineSchedule{}
	}
	return schedules, nil
}

// ListByProject returns all schedules for a project.
func (r *ScheduleRepo) ListByProject(ctx context.Context, projectID string) ([]models.PipelineSchedule, error) {
	var schedules []models.PipelineSchedule
	err := r.db.SelectContext(ctx, &schedules,
		`SELECT * FROM pipeline_schedules WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	if schedules == nil {
		schedules = []models.PipelineSchedule{}
	}
	return schedules, nil
}

// ListDue returns all enabled schedules whose next_run_at is at or before now.
func (r *ScheduleRepo) ListDue(ctx context.Context, now time.Time) ([]models.PipelineSchedule, error) {
	var schedules []models.PipelineSchedule
	err := r.db.SelectContext(ctx, &schedules,
		`SELECT * FROM pipeline_schedules WHERE enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ? ORDER BY next_run_at ASC`,
		true, now)
	if err != nil {
		return nil, err
	}
	if schedules == nil {
		schedules = []models.PipelineSchedule{}
	}
	return schedules, nil
}

// GetByID returns a schedule by its ID.
func (r *ScheduleRepo) GetByID(ctx context.Context, id string) (*models.PipelineSchedule, error) {
	var schedule models.PipelineSchedule
	err := r.db.GetContext(ctx, &schedule,
		`SELECT * FROM pipeline_schedules WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

// Create inserts a new schedule into the database.
func (r *ScheduleRepo) Create(ctx context.Context, schedule *models.PipelineSchedule) error {
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO pipeline_schedules (id, pipeline_id, project_id, cron_expression, timezone, description, enabled, branch, environment_id, variables, next_run_at, last_run_at, last_run_status, last_run_id, run_count, created_by, created_at, updated_at)
		VALUES (:id, :pipeline_id, :project_id, :cron_expression, :timezone, :description, :enabled, :branch, :environment_id, :variables, :next_run_at, :last_run_at, :last_run_status, :last_run_id, :run_count, :created_by, :created_at, :updated_at)`,
		schedule)
	return err
}

// Update updates an existing schedule.
func (r *ScheduleRepo) Update(ctx context.Context, schedule *models.PipelineSchedule) error {
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE pipeline_schedules SET
			cron_expression = :cron_expression,
			timezone = :timezone,
			description = :description,
			enabled = :enabled,
			branch = :branch,
			environment_id = :environment_id,
			variables = :variables,
			next_run_at = :next_run_at,
			updated_at = :updated_at
		WHERE id = :id`,
		schedule)
	return err
}

// Delete removes a schedule by ID.
func (r *ScheduleRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM pipeline_schedules WHERE id = ?`, id)
	return err
}

// SetEnabled toggles the enabled state of a schedule.
func (r *ScheduleRepo) SetEnabled(ctx context.Context, id string, enabled bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_schedules SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		enabled, id)
	return err
}

// UpdateAfterRun updates schedule fields after a run completes.
func (r *ScheduleRepo) UpdateAfterRun(ctx context.Context, id string, lastRunAt time.Time, nextRunAt time.Time, runID string, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_schedules SET
			last_run_at = ?,
			next_run_at = ?,
			last_run_id = ?,
			last_run_status = ?,
			run_count = run_count + 1,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		lastRunAt, nextRunAt, runID, status, id)
	return err
}
