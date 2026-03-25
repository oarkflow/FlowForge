package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type PipelineRepo struct {
	db *sqlx.DB
}

func (r *PipelineRepo) GetByID(ctx context.Context, id string) (*models.Pipeline, error) {
	p := &models.Pipeline{}
	err := r.db.GetContext(ctx, p, "SELECT * FROM pipelines WHERE id = ? AND deleted_at IS NULL", id)
	return p, err
}

func (r *PipelineRepo) ListByProject(ctx context.Context, projectID string, limit, offset int) ([]models.Pipeline, error) {
	pipelines := []models.Pipeline{}
	err := r.db.SelectContext(ctx, &pipelines, "SELECT * FROM pipelines WHERE project_id = ? AND deleted_at IS NULL ORDER BY created_at DESC LIMIT ? OFFSET ?", projectID, limit, offset)
	return pipelines, err
}

func (r *PipelineRepo) Create(ctx context.Context, p *models.Pipeline) error {
	p.ID = uuid.New().String()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO pipelines (id, project_id, repository_id, name, description, config_source, config_path, config_content, config_version, triggers, is_active, created_by, created_at, updated_at)
		VALUES (:id, :project_id, :repository_id, :name, :description, :config_source, :config_path, :config_content, :config_version, :triggers, :is_active, :created_by, :created_at, :updated_at)`,
		p)
	return err
}

func (r *PipelineRepo) Update(ctx context.Context, p *models.Pipeline) error {
	p.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE pipelines SET name=:name, description=:description, config_source=:config_source, config_path=:config_path, config_content=:config_content, config_version=:config_version, triggers=:triggers, is_active=:is_active, updated_at=:updated_at WHERE id=:id`,
		p)
	return err
}

func (r *PipelineRepo) SoftDelete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE pipelines SET deleted_at = ? WHERE id = ?", time.Now(), id)
	return err
}

func (r *PipelineRepo) CreateVersion(ctx context.Context, v *models.PipelineVersion) error {
	v.ID = uuid.New().String()
	v.CreatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO pipeline_versions (id, pipeline_id, version, config, message, created_by, created_at)
		VALUES (:id, :pipeline_id, :version, :config, :message, :created_by, :created_at)`,
		v)
	return err
}

func (r *PipelineRepo) ListVersions(ctx context.Context, pipelineID string) ([]models.PipelineVersion, error) {
	versions := []models.PipelineVersion{}
	err := r.db.SelectContext(ctx, &versions, "SELECT * FROM pipeline_versions WHERE pipeline_id = ? ORDER BY version DESC", pipelineID)
	return versions, err
}
