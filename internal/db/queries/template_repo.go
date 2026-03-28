package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type TemplateRepo struct {
	db *sqlx.DB
}

func (r *TemplateRepo) GetByID(ctx context.Context, id string) (*models.PipelineTemplate, error) {
	t := &models.PipelineTemplate{}
	err := r.db.GetContext(ctx, t, "SELECT * FROM pipeline_templates WHERE id = ?", id)
	return t, err
}

func (r *TemplateRepo) List(ctx context.Context, category string, builtinOnly bool, limit, offset int) ([]models.PipelineTemplate, error) {
	templates := []models.PipelineTemplate{}
	query := "SELECT * FROM pipeline_templates WHERE 1=1"
	args := []interface{}{}

	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}
	if builtinOnly {
		query += " AND is_builtin = 1"
	}

	query += " ORDER BY downloads DESC, created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := r.db.SelectContext(ctx, &templates, query, args...)
	return templates, err
}

func (r *TemplateRepo) Create(ctx context.Context, t *models.PipelineTemplate) error {
	t.ID = uuid.New().String()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO pipeline_templates (id, name, description, category, config, is_builtin, is_public, author, downloads, created_at, updated_at)
		VALUES (:id, :name, :description, :category, :config, :is_builtin, :is_public, :author, :downloads, :created_at, :updated_at)`,
		t)
	return err
}

func (r *TemplateRepo) Update(ctx context.Context, t *models.PipelineTemplate) error {
	t.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE pipeline_templates SET name=:name, description=:description, category=:category, config=:config, is_public=:is_public, author=:author, updated_at=:updated_at WHERE id=:id`,
		t)
	return err
}

func (r *TemplateRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM pipeline_templates WHERE id = ?", id)
	return err
}

func (r *TemplateRepo) IncrementDownloads(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE pipeline_templates SET downloads = downloads + 1 WHERE id = ?", id)
	return err
}
