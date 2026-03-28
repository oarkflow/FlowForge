package models

import "time"

// ScalingPolicy defines an auto-scaling policy that controls agent capacity
// based on job queue depth and workload.
type ScalingPolicy struct {
	ID                 string     `db:"id" json:"id"`
	Name               string     `db:"name" json:"name"`
	Description        string     `db:"description" json:"description"`
	Enabled            bool       `db:"enabled" json:"enabled"`
	ExecutorType       string     `db:"executor_type" json:"executor_type"`
	Labels             string     `db:"labels" json:"labels"`
	MinAgents          int        `db:"min_agents" json:"min_agents"`
	MaxAgents          int        `db:"max_agents" json:"max_agents"`
	DesiredAgents      int        `db:"desired_agents" json:"desired_agents"`
	ScaleUpThreshold   int        `db:"scale_up_threshold" json:"scale_up_threshold"`
	ScaleDownThreshold int        `db:"scale_down_threshold" json:"scale_down_threshold"`
	ScaleUpStep        int        `db:"scale_up_step" json:"scale_up_step"`
	ScaleDownStep      int        `db:"scale_down_step" json:"scale_down_step"`
	CooldownSeconds    int        `db:"cooldown_seconds" json:"cooldown_seconds"`
	LastScaleAction    string     `db:"last_scale_action" json:"last_scale_action"`
	LastScaleAt        *time.Time `db:"last_scale_at" json:"last_scale_at"`
	QueueDepth         int        `db:"queue_depth" json:"queue_depth"`
	ActiveAgents       int        `db:"active_agents" json:"active_agents"`
	CreatedAt          time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at" json:"updated_at"`
}

// ScalingEvent records a scaling decision made by the auto-scaler for audit trail.
type ScalingEvent struct {
	ID           string    `db:"id" json:"id"`
	PolicyID     string    `db:"policy_id" json:"policy_id"`
	Action       string    `db:"action" json:"action"`
	FromCount    int       `db:"from_count" json:"from_count"`
	ToCount      int       `db:"to_count" json:"to_count"`
	Reason       string    `db:"reason" json:"reason"`
	QueueDepth   int       `db:"queue_depth" json:"queue_depth"`
	ActiveAgents int       `db:"active_agents" json:"active_agents"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
