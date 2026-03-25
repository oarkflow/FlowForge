package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type EnvOverrideRepo struct {
	db *sqlx.DB
}

func (r *EnvOverrideRepo) ListByEnvironment(ctx context.Context, envID string) ([]models.EnvOverride, error) {
	overrides := []models.EnvOverride{}
	err := r.db.SelectContext(ctx, &overrides,
		"SELECT * FROM env_overrides WHERE environment_id = ? ORDER BY key ASC", envID)
	return overrides, err
}

func (r *EnvOverrideRepo) Upsert(ctx context.Context, override *models.EnvOverride) error {
	if override.ID == "" {
		override.ID = uuid.New().String()
	}
	override.CreatedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO env_overrides (id, environment_id, key, value_enc, is_secret, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(environment_id, key) DO UPDATE SET value_enc=excluded.value_enc, is_secret=excluded.is_secret`,
		override.ID, override.EnvironmentID, override.Key, override.ValueEnc, override.IsSecret, override.CreatedAt)
	return err
}

func (r *EnvOverrideRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM env_overrides WHERE id = ?", id)
	return err
}

func (r *EnvOverrideRepo) BulkSave(ctx context.Context, envID string, overrides []models.EnvOverride) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete all existing overrides for the environment
	if _, err := tx.ExecContext(ctx, "DELETE FROM env_overrides WHERE environment_id = ?", envID); err != nil {
		return err
	}

	// Insert new overrides
	now := time.Now()
	insertQuery := "INSERT INTO env_overrides (id, environment_id, key, value_enc, is_secret, created_at) VALUES (?, ?, ?, ?, ?, ?)"
	for _, o := range overrides {
		id := o.ID
		if id == "" {
			id = uuid.New().String()
		}
		if _, err := tx.ExecContext(ctx,
			insertQuery,
			id, envID, o.Key, o.ValueEnc, o.IsSecret, now,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
