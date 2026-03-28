package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// ProjectDeploymentProviderRepo handles CRUD for project_deployment_providers.
type ProjectDeploymentProviderRepo struct {
	db *sqlx.DB
}

func (r *ProjectDeploymentProviderRepo) GetByID(ctx context.Context, id string) (*models.ProjectDeploymentProvider, error) {
	provider := &models.ProjectDeploymentProvider{}
	err := r.db.GetContext(ctx, provider, "SELECT * FROM project_deployment_providers WHERE id = ?", id)
	return provider, err
}

func (r *ProjectDeploymentProviderRepo) GetByName(ctx context.Context, projectID, name string) (*models.ProjectDeploymentProvider, error) {
	provider := &models.ProjectDeploymentProvider{}
	err := r.db.GetContext(ctx, provider,
		"SELECT * FROM project_deployment_providers WHERE project_id = ? AND name = ?",
		projectID, name)
	return provider, err
}

func (r *ProjectDeploymentProviderRepo) ListByProject(ctx context.Context, projectID string, limit, offset int) ([]models.ProjectDeploymentProvider, error) {
	providers := []models.ProjectDeploymentProvider{}
	err := r.db.SelectContext(ctx, &providers,
		`SELECT * FROM project_deployment_providers
		 WHERE project_id = ?
		 ORDER BY created_at DESC
		 LIMIT ? OFFSET ?`,
		projectID, limit, offset)
	return providers, err
}

func (r *ProjectDeploymentProviderRepo) Create(ctx context.Context, provider *models.ProjectDeploymentProvider) error {
	provider.ID = uuid.New().String()
	now := time.Now()
	provider.CreatedAt = now
	provider.UpdatedAt = now
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO project_deployment_providers
		 (id, project_id, name, provider_type, config_enc, is_active, created_by, created_at, updated_at)
		 VALUES
		 (:id, :project_id, :name, :provider_type, :config_enc, :is_active, :created_by, :created_at, :updated_at)`,
		provider)
	return err
}

func (r *ProjectDeploymentProviderRepo) Update(ctx context.Context, provider *models.ProjectDeploymentProvider) error {
	provider.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE project_deployment_providers
		 SET name = :name,
		     provider_type = :provider_type,
		     config_enc = :config_enc,
		     is_active = :is_active,
		     updated_at = :updated_at
		 WHERE id = :id`,
		provider)
	return err
}

func (r *ProjectDeploymentProviderRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM project_deployment_providers WHERE id = ?", id)
	return err
}
