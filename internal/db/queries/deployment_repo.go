package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type DeploymentRepo struct {
	db *sqlx.DB
}

func (r *DeploymentRepo) ListByEnvironment(ctx context.Context, envID string, limit, offset int) ([]models.Deployment, error) {
	deployments := []models.Deployment{}
	err := r.db.SelectContext(ctx, &deployments,
		"SELECT * FROM deployments WHERE environment_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		envID, limit, offset)
	return deployments, err
}

func (r *DeploymentRepo) GetByID(ctx context.Context, id string) (*models.Deployment, error) {
	dep := &models.Deployment{}
	err := r.db.GetContext(ctx, dep, "SELECT * FROM deployments WHERE id = ?", id)
	return dep, err
}

func (r *DeploymentRepo) Create(ctx context.Context, dep *models.Deployment) error {
	dep.ID = uuid.New().String()
	dep.CreatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO deployments (id, environment_id, pipeline_run_id, version, status,
			commit_sha, image_tag, deployed_by, started_at, finished_at,
			health_check_status, rollback_from_id, metadata,
			strategy, canary_weight, health_check_results, strategy_state,
			created_at)
		VALUES (:id, :environment_id, :pipeline_run_id, :version, :status,
			:commit_sha, :image_tag, :deployed_by, :started_at, :finished_at,
			:health_check_status, :rollback_from_id, :metadata,
			:strategy, :canary_weight, :health_check_results, :strategy_state,
			:created_at)`,
		dep)
	return err
}

func (r *DeploymentRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE deployments SET status = ? WHERE id = ?", status, id)
	return err
}

func (r *DeploymentRepo) SetFinished(ctx context.Context, id, status string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		"UPDATE deployments SET status = ?, finished_at = ? WHERE id = ?",
		status, now, id)
	return err
}

func (r *DeploymentRepo) GetLatestByEnvironment(ctx context.Context, envID string) (*models.Deployment, error) {
	dep := &models.Deployment{}
	err := r.db.GetContext(ctx, dep,
		"SELECT * FROM deployments WHERE environment_id = ? ORDER BY created_at DESC LIMIT 1", envID)
	return dep, err
}

func (r *DeploymentRepo) ListAll(ctx context.Context, limit, offset int) ([]models.Deployment, error) {
	deployments := []models.Deployment{}
	err := r.db.SelectContext(ctx, &deployments,
		"SELECT * FROM deployments ORDER BY created_at DESC LIMIT ? OFFSET ?",
		limit, offset)
	return deployments, err
}

// UpdateCanaryWeight updates the canary weight for a deployment.
func (r *DeploymentRepo) UpdateCanaryWeight(ctx context.Context, id string, weight int) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE deployments SET canary_weight = ? WHERE id = ?",
		weight, id)
	return err
}

// UpdateHealthCheckResults updates the health check results JSON for a deployment.
func (r *DeploymentRepo) UpdateHealthCheckResults(ctx context.Context, id, results string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE deployments SET health_check_results = ? WHERE id = ?",
		results, id)
	return err
}

// UpdateStrategyState updates the strategy state JSON for a deployment.
func (r *DeploymentRepo) UpdateStrategyState(ctx context.Context, id, state string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE deployments SET strategy_state = ? WHERE id = ?",
		state, id)
	return err
}
