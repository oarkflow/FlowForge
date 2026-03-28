package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// ProjectEnvironmentChainRepo manages allowed promotion edges per project.
type ProjectEnvironmentChainRepo struct {
	db *sqlx.DB
}

func (r *ProjectEnvironmentChainRepo) ListByProject(ctx context.Context, projectID string) ([]models.ProjectEnvironmentChainEdge, error) {
	edges := []models.ProjectEnvironmentChainEdge{}
	err := r.db.SelectContext(ctx, &edges,
		`SELECT * FROM project_environment_chain
		 WHERE project_id = ?
		 ORDER BY position ASC, created_at ASC`,
		projectID)
	return edges, err
}

func (r *ProjectEnvironmentChainRepo) ReplaceForProject(ctx context.Context, projectID string, edges []models.ProjectEnvironmentChainEdge) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM project_environment_chain WHERE project_id = ?", projectID); err != nil {
		return err
	}

	now := time.Now()
	for i := range edges {
		edges[i].ID = uuid.New().String()
		edges[i].ProjectID = projectID
		edges[i].CreatedAt = now
		if _, err := tx.NamedExecContext(ctx,
			`INSERT INTO project_environment_chain
			 (id, project_id, source_environment_id, target_environment_id, position, created_at)
			 VALUES
			 (:id, :project_id, :source_environment_id, :target_environment_id, :position, :created_at)`,
			&edges[i]); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *ProjectEnvironmentChainRepo) IsPromotionAllowed(ctx context.Context, projectID, sourceEnvID, targetEnvID string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(1)
		 FROM project_environment_chain
		 WHERE project_id = ?
		   AND source_environment_id = ?
		   AND target_environment_id = ?`,
		projectID, sourceEnvID, targetEnvID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
