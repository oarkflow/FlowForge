package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type DeadLetterRepo struct {
	db *sqlx.DB
}

func (r *DeadLetterRepo) GetByID(ctx context.Context, id string) (*models.DeadLetterItem, error) {
	item := &models.DeadLetterItem{}
	err := r.db.GetContext(ctx, item, "SELECT * FROM dead_letter_queue WHERE id = ?", id)
	return item, err
}

func (r *DeadLetterRepo) List(ctx context.Context, status string, limit, offset int) ([]models.DeadLetterItem, error) {
	items := []models.DeadLetterItem{}
	query := "SELECT * FROM dead_letter_queue"
	args := []interface{}{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := r.db.SelectContext(ctx, &items, query, args...)
	return items, err
}

func (r *DeadLetterRepo) Create(ctx context.Context, item *models.DeadLetterItem) error {
	item.ID = uuid.New().String()
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO dead_letter_queue (id, job_id, pipeline_run_id, failure_reason, retry_count, max_retries, job_data, status, created_at, updated_at)
		VALUES (:id, :job_id, :pipeline_run_id, :failure_reason, :retry_count, :max_retries, :job_data, :status, :created_at, :updated_at)`,
		item)
	return err
}

func (r *DeadLetterRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE dead_letter_queue SET status = ?, updated_at = ? WHERE id = ?", status, time.Now(), id)
	return err
}

func (r *DeadLetterRepo) IncrementRetry(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE dead_letter_queue SET retry_count = retry_count + 1, updated_at = ? WHERE id = ?", time.Now(), id)
	return err
}

func (r *DeadLetterRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM dead_letter_queue WHERE id = ?", id)
	return err
}

func (r *DeadLetterRepo) PurgeAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM dead_letter_queue WHERE status = 'pending'")
	return err
}
