package queries

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/oarkflow/deploy/backend/internal/models"
)

type ScalingPolicyRepo struct {
	db *sqlx.DB
}

func (r *ScalingPolicyRepo) ListPolicies() ([]models.ScalingPolicy, error) {
	var policies []models.ScalingPolicy
	err := r.db.Select(&policies, "SELECT * FROM scaling_policies ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	return policies, nil
}

func (r *ScalingPolicyRepo) GetPolicy(id string) (*models.ScalingPolicy, error) {
	var policy models.ScalingPolicy
	err := r.db.Get(&policy, "SELECT * FROM scaling_policies WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

func (r *ScalingPolicyRepo) CreatePolicy(policy *models.ScalingPolicy) error {
	_, err := r.db.Exec(`
		INSERT INTO scaling_policies (name, description, enabled, executor_type, labels,
			min_agents, max_agents, desired_agents, scale_up_threshold, scale_down_threshold,
			scale_up_step, scale_down_step, cooldown_seconds)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		policy.Name, policy.Description, policy.Enabled, policy.ExecutorType, policy.Labels,
		policy.MinAgents, policy.MaxAgents, policy.DesiredAgents, policy.ScaleUpThreshold,
		policy.ScaleDownThreshold, policy.ScaleUpStep, policy.ScaleDownStep, policy.CooldownSeconds,
	)
	if err != nil {
		return err
	}
	// Retrieve the created policy to get the auto-generated ID
	return r.db.Get(policy, `
		SELECT * FROM scaling_policies WHERE rowid = last_insert_rowid()`)
}

func (r *ScalingPolicyRepo) UpdatePolicy(policy *models.ScalingPolicy) error {
	_, err := r.db.Exec(`
		UPDATE scaling_policies SET
			name = ?, description = ?, enabled = ?, executor_type = ?, labels = ?,
			min_agents = ?, max_agents = ?, desired_agents = ?, scale_up_threshold = ?,
			scale_down_threshold = ?, scale_up_step = ?, scale_down_step = ?,
			cooldown_seconds = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		policy.Name, policy.Description, policy.Enabled, policy.ExecutorType, policy.Labels,
		policy.MinAgents, policy.MaxAgents, policy.DesiredAgents, policy.ScaleUpThreshold,
		policy.ScaleDownThreshold, policy.ScaleUpStep, policy.ScaleDownStep,
		policy.CooldownSeconds, policy.ID,
	)
	return err
}

func (r *ScalingPolicyRepo) DeletePolicy(id string) error {
	_, err := r.db.Exec("DELETE FROM scaling_policies WHERE id = ?", id)
	return err
}

func (r *ScalingPolicyRepo) SetEnabled(id string, enabled bool) error {
	_, err := r.db.Exec("UPDATE scaling_policies SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		enabled, id)
	return err
}

func (r *ScalingPolicyRepo) UpdateMetrics(id string, queueDepth, activeAgents int) error {
	_, err := r.db.Exec(`
		UPDATE scaling_policies SET queue_depth = ?, active_agents = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		queueDepth, activeAgents, id)
	return err
}

func (r *ScalingPolicyRepo) RecordScaleAction(id, action string, desiredAgents int) error {
	now := time.Now()
	_, err := r.db.Exec(`
		UPDATE scaling_policies SET
			last_scale_action = ?, last_scale_at = ?, desired_agents = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		action, now, desiredAgents, id)
	return err
}

func (r *ScalingPolicyRepo) ListEnabledPolicies() ([]models.ScalingPolicy, error) {
	var policies []models.ScalingPolicy
	err := r.db.Select(&policies, "SELECT * FROM scaling_policies WHERE enabled = 1 ORDER BY created_at ASC")
	if err != nil {
		return nil, err
	}
	return policies, nil
}
