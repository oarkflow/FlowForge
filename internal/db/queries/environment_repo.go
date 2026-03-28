package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type EnvironmentRepo struct {
	db *sqlx.DB
}

func (r *EnvironmentRepo) List(ctx context.Context, projectID string) ([]models.Environment, error) {
	envs := []models.Environment{}
	err := r.db.SelectContext(ctx, &envs,
		"SELECT * FROM environments WHERE project_id = ? ORDER BY is_production DESC, name ASC", projectID)
	return envs, err
}

func (r *EnvironmentRepo) GetByID(ctx context.Context, id string) (*models.Environment, error) {
	env := &models.Environment{}
	err := r.db.GetContext(ctx, env, "SELECT * FROM environments WHERE id = ?", id)
	return env, err
}

func (r *EnvironmentRepo) GetBySlug(ctx context.Context, projectID, slug string) (*models.Environment, error) {
	env := &models.Environment{}
	err := r.db.GetContext(ctx, env, "SELECT * FROM environments WHERE project_id = ? AND slug = ?", projectID, slug)
	return env, err
}

func (r *EnvironmentRepo) Create(ctx context.Context, env *models.Environment) error {
	env.ID = uuid.New().String()
	now := time.Now()
	env.CreatedAt = now
	env.UpdatedAt = now
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO environments (id, project_id, name, slug, description, url, is_production,
			auto_deploy_branch, required_approvers, protection_rules, deploy_freeze,
			lock_owner_id, lock_reason, locked_at, current_deployment_id,
			strategy, strategy_config, health_check_url, health_check_interval,
			health_check_timeout, health_check_retries, health_check_path, health_check_expected_status,
			created_at, updated_at)
		VALUES (:id, :project_id, :name, :slug, :description, :url, :is_production,
			:auto_deploy_branch, :required_approvers, :protection_rules, :deploy_freeze,
			:lock_owner_id, :lock_reason, :locked_at, :current_deployment_id,
			:strategy, :strategy_config, :health_check_url, :health_check_interval,
			:health_check_timeout, :health_check_retries, :health_check_path, :health_check_expected_status,
			:created_at, :updated_at)`,
		env)
	return err
}

func (r *EnvironmentRepo) Update(ctx context.Context, env *models.Environment) error {
	env.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE environments SET name=:name, slug=:slug, description=:description, url=:url,
			is_production=:is_production, auto_deploy_branch=:auto_deploy_branch,
			required_approvers=:required_approvers, protection_rules=:protection_rules,
			deploy_freeze=:deploy_freeze, updated_at=:updated_at
		WHERE id=:id`,
		env)
	return err
}

func (r *EnvironmentRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM environments WHERE id = ?", id)
	return err
}

func (r *EnvironmentRepo) Lock(ctx context.Context, id, ownerID, reason string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		"UPDATE environments SET lock_owner_id = ?, lock_reason = ?, locked_at = ?, updated_at = ? WHERE id = ?",
		ownerID, reason, now, now, id)
	return err
}

func (r *EnvironmentRepo) Unlock(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		"UPDATE environments SET lock_owner_id = NULL, lock_reason = '', locked_at = NULL, updated_at = ? WHERE id = ?",
		now, id)
	return err
}

func (r *EnvironmentRepo) UpdateCurrentDeployment(ctx context.Context, envID, deploymentID string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		"UPDATE environments SET current_deployment_id = ?, updated_at = ? WHERE id = ?",
		deploymentID, now, envID)
	return err
}

// UpdateStrategy updates the deployment strategy configuration for an environment.
func (r *EnvironmentRepo) UpdateStrategy(ctx context.Context, id string, strategy, strategyConfig,
	healthCheckURL string, healthCheckInterval, healthCheckTimeout, healthCheckRetries int,
	healthCheckPath string, healthCheckExpectedStatus int) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE environments SET strategy = ?, strategy_config = ?,
			health_check_url = ?, health_check_interval = ?, health_check_timeout = ?,
			health_check_retries = ?, health_check_path = ?, health_check_expected_status = ?,
			updated_at = ?
		WHERE id = ?`,
		strategy, strategyConfig,
		healthCheckURL, healthCheckInterval, healthCheckTimeout,
		healthCheckRetries, healthCheckPath, healthCheckExpectedStatus,
		now, id)
	return err
}
