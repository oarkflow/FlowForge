package models

import "time"

type Deployment struct {
	ID                string     `db:"id" json:"id"`
	EnvironmentID     string     `db:"environment_id" json:"environment_id"`
	PipelineRunID     *string    `db:"pipeline_run_id" json:"pipeline_run_id"`
	Version           string     `db:"version" json:"version"`
	Status            string     `db:"status" json:"status"`
	CommitSHA         string     `db:"commit_sha" json:"commit_sha"`
	ImageTag          string     `db:"image_tag" json:"image_tag"`
	DeployedBy        string     `db:"deployed_by" json:"deployed_by"`
	StartedAt         *time.Time `db:"started_at" json:"started_at"`
	FinishedAt        *time.Time `db:"finished_at" json:"finished_at"`
	HealthCheckStatus string     `db:"health_check_status" json:"health_check_status"`
	RollbackFromID    *string    `db:"rollback_from_id" json:"rollback_from_id"`
	Metadata          string     `db:"metadata" json:"metadata"`

	// Deployment strategy fields
	Strategy           string `db:"strategy" json:"strategy"`
	CanaryWeight       int    `db:"canary_weight" json:"canary_weight"`
	HealthCheckResults string `db:"health_check_results" json:"health_check_results"`
	StrategyState      string `db:"strategy_state" json:"strategy_state"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
