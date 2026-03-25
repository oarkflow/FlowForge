package models

import "time"

// PipelineSchedule represents a cron-based schedule for triggering a pipeline.
type PipelineSchedule struct {
	ID             string     `db:"id" json:"id"`
	PipelineID     string     `db:"pipeline_id" json:"pipeline_id"`
	ProjectID      string     `db:"project_id" json:"project_id"`
	CronExpression string     `db:"cron_expression" json:"cron_expression"`
	Timezone       string     `db:"timezone" json:"timezone"`
	Description    string     `db:"description" json:"description"`
	Enabled        bool       `db:"enabled" json:"enabled"`
	Branch         string     `db:"branch" json:"branch"`
	EnvironmentID  *string    `db:"environment_id" json:"environment_id"`
	Variables      string     `db:"variables" json:"variables"`
	NextRunAt      *time.Time `db:"next_run_at" json:"next_run_at"`
	LastRunAt      *time.Time `db:"last_run_at" json:"last_run_at"`
	LastRunStatus  string     `db:"last_run_status" json:"last_run_status"`
	LastRunID      *string    `db:"last_run_id" json:"last_run_id"`
	RunCount       int        `db:"run_count" json:"run_count"`
	CreatedBy      string     `db:"created_by" json:"created_by"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
}
