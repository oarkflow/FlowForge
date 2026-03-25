package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type ProjectRepo struct {
	db *sqlx.DB
}

func (r *ProjectRepo) GetByID(ctx context.Context, id string) (*models.Project, error) {
	p := &models.Project{}
	err := r.db.GetContext(ctx, p, "SELECT * FROM projects WHERE id = ? AND deleted_at IS NULL", id)
	return p, err
}

func (r *ProjectRepo) List(ctx context.Context, limit, offset int) ([]models.Project, error) {
	projects := []models.Project{}
	err := r.db.SelectContext(ctx, &projects, "SELECT * FROM projects WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT ? OFFSET ?", limit, offset)
	return projects, err
}

func (r *ProjectRepo) ListByOrg(ctx context.Context, orgID string, limit, offset int) ([]models.Project, error) {
	projects := []models.Project{}
	err := r.db.SelectContext(ctx, &projects, "SELECT * FROM projects WHERE org_id = ? AND deleted_at IS NULL ORDER BY created_at DESC LIMIT ? OFFSET ?", orgID, limit, offset)
	return projects, err
}

func (r *ProjectRepo) Create(ctx context.Context, p *models.Project) error {
	p.ID = uuid.New().String()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO projects (id, org_id, name, slug, description, visibility, created_by, created_at, updated_at)
		VALUES (:id, :org_id, :name, :slug, :description, :visibility, :created_by, :created_at, :updated_at)`,
		p)
	return err
}

func (r *ProjectRepo) Update(ctx context.Context, p *models.Project) error {
	p.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE projects SET name=:name, slug=:slug, description=:description, visibility=:visibility, updated_at=:updated_at WHERE id=:id`,
		p)
	return err
}

func (r *ProjectRepo) SoftDelete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE projects SET deleted_at = ? WHERE id = ?", time.Now(), id)
	return err
}
