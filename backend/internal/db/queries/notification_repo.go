package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type NotificationRepo struct {
	db *sqlx.DB
}

func (r *NotificationRepo) GetByID(ctx context.Context, id string) (*models.NotificationChannel, error) {
	n := &models.NotificationChannel{}
	err := r.db.GetContext(ctx, n, "SELECT * FROM notification_channels WHERE id = ?", id)
	return n, err
}

func (r *NotificationRepo) ListByProject(ctx context.Context, projectID string, limit, offset int) ([]models.NotificationChannel, error) {
	channels := []models.NotificationChannel{}
	err := r.db.SelectContext(ctx, &channels, "SELECT * FROM notification_channels WHERE project_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?", projectID, limit, offset)
	return channels, err
}

func (r *NotificationRepo) Create(ctx context.Context, n *models.NotificationChannel) error {
	n.ID = uuid.New().String()
	n.CreatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO notification_channels (id, project_id, type, name, config_enc, is_active, created_at)
		VALUES (:id, :project_id, :type, :name, :config_enc, :is_active, :created_at)`,
		n)
	return err
}

func (r *NotificationRepo) Update(ctx context.Context, n *models.NotificationChannel) error {
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE notification_channels SET name=:name, config_enc=:config_enc, is_active=:is_active WHERE id=:id`,
		n)
	return err
}

func (r *NotificationRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM notification_channels WHERE id = ?", id)
	return err
}
