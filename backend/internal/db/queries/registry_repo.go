package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type RegistryRepo struct {
	db *sqlx.DB
}

func (r *RegistryRepo) List(ctx context.Context, projectID string) ([]models.Registry, error) {
	regs := []models.Registry{}
	err := r.db.SelectContext(ctx, &regs,
		"SELECT * FROM registries WHERE project_id = ? ORDER BY is_default DESC, name ASC", projectID)
	return regs, err
}

func (r *RegistryRepo) GetByID(ctx context.Context, id string) (*models.Registry, error) {
	reg := &models.Registry{}
	err := r.db.GetContext(ctx, reg, "SELECT * FROM registries WHERE id = ?", id)
	return reg, err
}

func (r *RegistryRepo) GetDefault(ctx context.Context, projectID string) (*models.Registry, error) {
	reg := &models.Registry{}
	err := r.db.GetContext(ctx, reg,
		"SELECT * FROM registries WHERE project_id = ? AND is_default = ? LIMIT 1", projectID, true)
	return reg, err
}

func (r *RegistryRepo) Create(ctx context.Context, reg *models.Registry) error {
	reg.ID = uuid.New().String()
	now := time.Now()
	reg.CreatedAt = now
	reg.UpdatedAt = now
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO registries (id, project_id, name, type, url, username, credentials_enc, is_default, created_at, updated_at)
		VALUES (:id, :project_id, :name, :type, :url, :username, :credentials_enc, :is_default, :created_at, :updated_at)`,
		reg)
	return err
}

func (r *RegistryRepo) Update(ctx context.Context, reg *models.Registry) error {
	reg.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE registries SET name=:name, type=:type, url=:url, username=:username,
			credentials_enc=:credentials_enc, is_default=:is_default, updated_at=:updated_at
		WHERE id=:id`,
		reg)
	return err
}

func (r *RegistryRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM registries WHERE id = ?", id)
	return err
}

func (r *RegistryRepo) SetDefault(ctx context.Context, projectID, registryID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	// Unset all defaults for this project
	if _, err := tx.ExecContext(ctx,
		"UPDATE registries SET is_default = ?, updated_at = ? WHERE project_id = ?",
		false, now, projectID); err != nil {
		return err
	}
	// Set the specified registry as default
	if _, err := tx.ExecContext(ctx,
		"UPDATE registries SET is_default = ?, updated_at = ? WHERE id = ? AND project_id = ?",
		true, now, registryID, projectID); err != nil {
		return err
	}

	return tx.Commit()
}
