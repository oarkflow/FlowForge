package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type RepositoryRepo struct {
	db *sqlx.DB
}

func (r *RepositoryRepo) GetByID(ctx context.Context, id string) (*models.Repository, error) {
	repo := &models.Repository{}
	err := r.db.GetContext(ctx, repo, "SELECT * FROM repositories WHERE id = ?", id)
	return repo, err
}

func (r *RepositoryRepo) ListByProjectID(ctx context.Context, projectID string) ([]models.Repository, error) {
	repos := []models.Repository{}
	err := r.db.SelectContext(ctx, &repos, "SELECT * FROM repositories WHERE project_id = ? ORDER BY created_at DESC", projectID)
	return repos, err
}

func (r *RepositoryRepo) Create(ctx context.Context, repo *models.Repository) error {
	repo.ID = uuid.New().String()
	repo.CreatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO repositories (id, project_id, provider, provider_id, full_name, clone_url, ssh_url, default_branch, webhook_id, webhook_secret, access_token_enc, ssh_key_enc, is_active, created_at)
		VALUES (:id, :project_id, :provider, :provider_id, :full_name, :clone_url, :ssh_url, :default_branch, :webhook_id, :webhook_secret, :access_token_enc, :ssh_key_enc, :is_active, :created_at)`,
		repo)
	return err
}

func (r *RepositoryRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM repositories WHERE id = ?", id)
	return err
}
