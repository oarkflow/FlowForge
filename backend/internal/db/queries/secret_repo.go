package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type SecretRepo struct {
	db *sqlx.DB
}

func (r *SecretRepo) GetByID(ctx context.Context, id string) (*models.Secret, error) {
	s := &models.Secret{}
	err := r.db.GetContext(ctx, s, "SELECT * FROM secrets WHERE id = ?", id)
	return s, err
}

func (r *SecretRepo) ListByProject(ctx context.Context, projectID string, limit, offset int) ([]models.Secret, error) {
	secrets := []models.Secret{}
	err := r.db.SelectContext(ctx, &secrets, "SELECT * FROM secrets WHERE project_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?", projectID, limit, offset)
	return secrets, err
}

func (r *SecretRepo) ListByOrg(ctx context.Context, orgID string, limit, offset int) ([]models.Secret, error) {
	secrets := []models.Secret{}
	err := r.db.SelectContext(ctx, &secrets, "SELECT * FROM secrets WHERE org_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?", orgID, limit, offset)
	return secrets, err
}

func (r *SecretRepo) Create(ctx context.Context, s *models.Secret) error {
	s.ID = uuid.New().String()
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO secrets (id, project_id, org_id, scope, key, value_enc, masked, created_by, created_at, updated_at)
		VALUES (:id, :project_id, :org_id, :scope, :key, :value_enc, :masked, :created_by, :created_at, :updated_at)`,
		s)
	return err
}

func (r *SecretRepo) Update(ctx context.Context, s *models.Secret) error {
	s.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE secrets SET key=:key, value_enc=:value_enc, masked=:masked, updated_at=:updated_at WHERE id=:id`,
		s)
	return err
}

func (r *SecretRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM secrets WHERE id = ?", id)
	return err
}
