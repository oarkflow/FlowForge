package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// PipelineStageEnvironmentMappingRepo stores stage -> environment links for a pipeline.
type PipelineStageEnvironmentMappingRepo struct {
	db *sqlx.DB
}

func (r *PipelineStageEnvironmentMappingRepo) ListByPipeline(ctx context.Context, projectID, pipelineID string) ([]models.PipelineStageEnvironmentMapping, error) {
	mappings := []models.PipelineStageEnvironmentMapping{}
	err := r.db.SelectContext(ctx, &mappings,
		`SELECT * FROM pipeline_stage_environment_mappings
		 WHERE project_id = ? AND pipeline_id = ?
		 ORDER BY stage_name ASC`,
		projectID, pipelineID)
	return mappings, err
}

func (r *PipelineStageEnvironmentMappingRepo) ReplaceForPipeline(ctx context.Context, projectID, pipelineID string, mappings []models.PipelineStageEnvironmentMapping) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		"DELETE FROM pipeline_stage_environment_mappings WHERE project_id = ? AND pipeline_id = ?",
		projectID, pipelineID); err != nil {
		return err
	}

	now := time.Now()
	for i := range mappings {
		mappings[i].ID = uuid.New().String()
		mappings[i].ProjectID = projectID
		mappings[i].PipelineID = pipelineID
		mappings[i].CreatedAt = now
		mappings[i].UpdatedAt = now
		if _, err := tx.NamedExecContext(ctx,
			`INSERT INTO pipeline_stage_environment_mappings
			 (id, project_id, pipeline_id, stage_name, environment_id, created_at, updated_at)
			 VALUES
			 (:id, :project_id, :pipeline_id, :stage_name, :environment_id, :created_at, :updated_at)`,
			&mappings[i]); err != nil {
			return err
		}
	}

	return tx.Commit()
}
