package queries

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type LogRepo struct {
	db *sqlx.DB
}

func (r *LogRepo) Insert(ctx context.Context, log *models.RunLog) error {
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO run_logs (run_id, step_run_id, stream, content) VALUES (:run_id, :step_run_id, :stream, :content)`,
		log)
	return err
}

func (r *LogRepo) GetByRunID(ctx context.Context, runID string, limit, offset int) ([]models.RunLog, error) {
	logs := []models.RunLog{}
	err := r.db.SelectContext(ctx, &logs, "SELECT * FROM run_logs WHERE run_id = ? ORDER BY id LIMIT ? OFFSET ?", runID, limit, offset)
	return logs, err
}

func (r *LogRepo) GetByStepRunID(ctx context.Context, stepRunID string, limit, offset int) ([]models.RunLog, error) {
	logs := []models.RunLog{}
	err := r.db.SelectContext(ctx, &logs, "SELECT * FROM run_logs WHERE step_run_id = ? ORDER BY id LIMIT ? OFFSET ?", stepRunID, limit, offset)
	return logs, err
}

// DeleteBefore deletes all logs older than the given time. Returns the number of rows deleted.
func (r *LogRepo) DeleteBefore(ctx context.Context, before string) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM run_logs WHERE ts < ?", before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DeleteByProjectBefore deletes logs for a given project older than the given time.
func (r *LogRepo) DeleteByProjectBefore(ctx context.Context, projectID string, before string) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM run_logs WHERE run_id IN (
			SELECT pr.id FROM pipeline_runs pr
			JOIN pipelines p ON pr.pipeline_id = p.id
			WHERE p.project_id = ?
		) AND ts < ?`, projectID, before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
