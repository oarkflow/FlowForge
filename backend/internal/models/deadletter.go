package models

import "time"

type DeadLetterItem struct {
	ID            string    `db:"id" json:"id"`
	JobID         string    `db:"job_id" json:"job_id"`
	PipelineRunID string    `db:"pipeline_run_id" json:"pipeline_run_id"`
	FailureReason string    `db:"failure_reason" json:"failure_reason"`
	RetryCount    int       `db:"retry_count" json:"retry_count"`
	MaxRetries    int       `db:"max_retries" json:"max_retries"`
	JobData       string    `db:"job_data" json:"job_data"`
	Status        string    `db:"status" json:"status"` // pending, retried, purged
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}
