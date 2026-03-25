package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type ArtifactRepo struct {
	db *sqlx.DB
}

func (r *ArtifactRepo) GetByID(ctx context.Context, id string) (*models.Artifact, error) {
	a := &models.Artifact{}
	err := r.db.GetContext(ctx, a, "SELECT * FROM artifacts WHERE id = ?", id)
	return a, err
}

func (r *ArtifactRepo) ListByRunID(ctx context.Context, runID string) ([]models.Artifact, error) {
	artifacts := []models.Artifact{}
	err := r.db.SelectContext(ctx, &artifacts, "SELECT * FROM artifacts WHERE run_id = ? ORDER BY created_at DESC", runID)
	return artifacts, err
}

func (r *ArtifactRepo) Create(ctx context.Context, a *models.Artifact) error {
	a.ID = uuid.New().String()
	a.CreatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO artifacts (id, run_id, step_run_id, name, path, size_bytes, checksum_sha256, storage_backend, storage_key, expire_at, created_at)
		VALUES (:id, :run_id, :step_run_id, :name, :path, :size_bytes, :checksum_sha256, :storage_backend, :storage_key, :expire_at, :created_at)`,
		a)
	return err
}

func (r *ArtifactRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM artifacts WHERE id = ?", id)
	return err
}

func (r *ArtifactRepo) ListExpired(ctx context.Context) ([]models.Artifact, error) {
	artifacts := []models.Artifact{}
	err := r.db.SelectContext(ctx, &artifacts, "SELECT * FROM artifacts WHERE expire_at IS NOT NULL AND expire_at < ?", time.Now())
	return artifacts, err
}

func (r *ArtifactRepo) DeleteExpired(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx, "DELETE FROM artifacts WHERE expire_at IS NOT NULL AND expire_at < ?", time.Now())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
