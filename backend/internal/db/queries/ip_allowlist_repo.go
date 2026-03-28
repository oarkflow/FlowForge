package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// IPAllowlistRepo handles CRUD for the ip_allowlist table.
type IPAllowlistRepo struct {
	db *sqlx.DB
}

func (r *IPAllowlistRepo) GetByID(ctx context.Context, id string) (*models.IPAllowlistEntry, error) {
	e := &models.IPAllowlistEntry{}
	err := r.db.GetContext(ctx, e, "SELECT * FROM ip_allowlist WHERE id = ?", id)
	return e, err
}

func (r *IPAllowlistRepo) ListAll(ctx context.Context) ([]models.IPAllowlistEntry, error) {
	var entries []models.IPAllowlistEntry
	err := r.db.SelectContext(ctx, &entries, "SELECT * FROM ip_allowlist ORDER BY scope, created_at DESC")
	return entries, err
}

func (r *IPAllowlistRepo) ListByProject(ctx context.Context, projectID string) ([]models.IPAllowlistEntry, error) {
	var entries []models.IPAllowlistEntry
	err := r.db.SelectContext(ctx, &entries,
		"SELECT * FROM ip_allowlist WHERE project_id = ? OR scope = 'global' ORDER BY scope, created_at DESC",
		projectID)
	return entries, err
}

func (r *IPAllowlistRepo) ListGlobal(ctx context.Context) ([]models.IPAllowlistEntry, error) {
	var entries []models.IPAllowlistEntry
	err := r.db.SelectContext(ctx, &entries,
		"SELECT * FROM ip_allowlist WHERE scope = 'global' ORDER BY created_at DESC")
	return entries, err
}

func (r *IPAllowlistRepo) Create(ctx context.Context, e *models.IPAllowlistEntry) error {
	e.ID = uuid.New().String()
	e.CreatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO ip_allowlist (id, project_id, scope, cidr, label, created_by, created_at)
		VALUES (:id, :project_id, :scope, :cidr, :label, :created_by, :created_at)`,
		e)
	return err
}

func (r *IPAllowlistRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ip_allowlist WHERE id = ?", id)
	return err
}

func (r *IPAllowlistRepo) DeleteByProject(ctx context.Context, projectID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ip_allowlist WHERE project_id = ?", projectID)
	return err
}
