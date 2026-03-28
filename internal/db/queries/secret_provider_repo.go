package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// SecretProviderRepo handles CRUD for the secret_providers table.
type SecretProviderRepo struct {
	db *sqlx.DB
}

func (r *SecretProviderRepo) GetByID(ctx context.Context, id string) (*models.SecretProviderConfig, error) {
	p := &models.SecretProviderConfig{}
	err := r.db.GetContext(ctx, p, "SELECT * FROM secret_providers WHERE id = ?", id)
	return p, err
}

func (r *SecretProviderRepo) ListByProject(ctx context.Context, projectID string, limit, offset int) ([]models.SecretProviderConfig, error) {
	var providers []models.SecretProviderConfig
	err := r.db.SelectContext(ctx, &providers,
		`SELECT * FROM secret_providers WHERE project_id = ? ORDER BY priority ASC, created_at DESC LIMIT ? OFFSET ?`,
		projectID, limit, offset)
	return providers, err
}

func (r *SecretProviderRepo) ListActive(ctx context.Context, projectID string) ([]models.SecretProviderConfig, error) {
	var providers []models.SecretProviderConfig
	err := r.db.SelectContext(ctx, &providers,
		`SELECT * FROM secret_providers WHERE project_id = ? AND is_active = 1 ORDER BY priority ASC`,
		projectID)
	return providers, err
}

func (r *SecretProviderRepo) Create(ctx context.Context, p *models.SecretProviderConfig) error {
	p.ID = uuid.New().String()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO secret_providers (id, project_id, name, provider_type, config_enc, is_active, priority, created_by, created_at, updated_at)
		VALUES (:id, :project_id, :name, :provider_type, :config_enc, :is_active, :priority, :created_by, :created_at, :updated_at)`,
		p)
	return err
}

func (r *SecretProviderRepo) Update(ctx context.Context, p *models.SecretProviderConfig) error {
	p.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE secret_providers SET name=:name, provider_type=:provider_type, config_enc=:config_enc, is_active=:is_active, priority=:priority, updated_at=:updated_at WHERE id=:id`,
		p)
	return err
}

func (r *SecretProviderRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM secret_providers WHERE id = ?", id)
	return err
}
