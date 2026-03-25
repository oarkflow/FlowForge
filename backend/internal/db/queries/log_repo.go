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
