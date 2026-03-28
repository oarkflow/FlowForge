package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type FeatureFlagRepo struct {
	db *sqlx.DB
}

func (r *FeatureFlagRepo) GetByID(ctx context.Context, id string) (*models.FeatureFlag, error) {
	f := &models.FeatureFlag{}
	err := r.db.GetContext(ctx, f, "SELECT * FROM feature_flags WHERE id = ?", id)
	return f, err
}

func (r *FeatureFlagRepo) GetByName(ctx context.Context, name string) (*models.FeatureFlag, error) {
	f := &models.FeatureFlag{}
	err := r.db.GetContext(ctx, f, "SELECT * FROM feature_flags WHERE name = ?", name)
	return f, err
}

func (r *FeatureFlagRepo) List(ctx context.Context, limit, offset int) ([]models.FeatureFlag, error) {
	flags := []models.FeatureFlag{}
	err := r.db.SelectContext(ctx, &flags, "SELECT * FROM feature_flags ORDER BY name LIMIT ? OFFSET ?", limit, offset)
	return flags, err
}

func (r *FeatureFlagRepo) Create(ctx context.Context, f *models.FeatureFlag) error {
	f.ID = uuid.New().String()
	f.CreatedAt = time.Now()
	f.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO feature_flags (id, name, description, enabled, rollout_percentage, target_users, target_orgs, created_at, updated_at)
		VALUES (:id, :name, :description, :enabled, :rollout_percentage, :target_users, :target_orgs, :created_at, :updated_at)`,
		f)
	return err
}

func (r *FeatureFlagRepo) Update(ctx context.Context, f *models.FeatureFlag) error {
	f.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE feature_flags SET name=:name, description=:description, enabled=:enabled, rollout_percentage=:rollout_percentage, target_users=:target_users, target_orgs=:target_orgs, updated_at=:updated_at WHERE id=:id`,
		f)
	return err
}

func (r *FeatureFlagRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM feature_flags WHERE id = ?", id)
	return err
}
