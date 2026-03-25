package models

import "time"

type Environment struct {
	ID                  string     `db:"id" json:"id"`
	ProjectID           string     `db:"project_id" json:"project_id"`
	Name                string     `db:"name" json:"name"`
	Slug                string     `db:"slug" json:"slug"`
	Description         string     `db:"description" json:"description"`
	URL                 string     `db:"url" json:"url"`
	IsProduction        bool       `db:"is_production" json:"is_production"`
	AutoDeployBranch    string     `db:"auto_deploy_branch" json:"auto_deploy_branch"`
	RequiredApprovers   string     `db:"required_approvers" json:"required_approvers"`
	ProtectionRules     string     `db:"protection_rules" json:"protection_rules"`
	DeployFreeze        bool       `db:"deploy_freeze" json:"deploy_freeze"`
	LockOwnerID         *string    `db:"lock_owner_id" json:"lock_owner_id"`
	LockReason          string     `db:"lock_reason" json:"lock_reason"`
	LockedAt            *time.Time `db:"locked_at" json:"locked_at"`
	CurrentDeploymentID *string    `db:"current_deployment_id" json:"current_deployment_id"`

	// Deployment strategy fields
	Strategy                  string `db:"strategy" json:"strategy"`
	StrategyConfig            string `db:"strategy_config" json:"strategy_config"`
	HealthCheckURL            string `db:"health_check_url" json:"health_check_url"`
	HealthCheckInterval       int    `db:"health_check_interval" json:"health_check_interval"`
	HealthCheckTimeout        int    `db:"health_check_timeout" json:"health_check_timeout"`
	HealthCheckRetries        int    `db:"health_check_retries" json:"health_check_retries"`
	HealthCheckPath           string `db:"health_check_path" json:"health_check_path"`
	HealthCheckExpectedStatus int    `db:"health_check_expected_status" json:"health_check_expected_status"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
