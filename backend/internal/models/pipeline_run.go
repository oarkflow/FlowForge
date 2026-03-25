package models

import "time"

type PipelineRun struct {
	ID            string     `db:"id" json:"id"`
	PipelineID    string     `db:"pipeline_id" json:"pipeline_id"`
	Number        int        `db:"number" json:"number"`
	Status        string     `db:"status" json:"status"`
	TriggerType   string     `db:"trigger_type" json:"trigger_type"`
	TriggerData   *string    `db:"trigger_data" json:"trigger_data,omitempty"`
	CommitSHA     *string    `db:"commit_sha" json:"commit_sha,omitempty"`
	CommitMessage *string    `db:"commit_message" json:"commit_message,omitempty"`
	Branch        *string    `db:"branch" json:"branch,omitempty"`
	Tag           *string    `db:"tag" json:"tag,omitempty"`
	Author        *string    `db:"author" json:"author,omitempty"`
	StartedAt     *time.Time `db:"started_at" json:"started_at,omitempty"`
	FinishedAt    *time.Time `db:"finished_at" json:"finished_at,omitempty"`
	DurationMs    *int       `db:"duration_ms" json:"duration_ms,omitempty"`
	CreatedBy     *string    `db:"created_by" json:"created_by,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
}

// PipelineRunWithMeta extends PipelineRun with pipeline and project context for global views.
type PipelineRunWithMeta struct {
	PipelineRun
	PipelineName string `db:"pipeline_name" json:"pipeline_name"`
	ProjectID    string `db:"project_id" json:"project_id"`
	ProjectName  string `db:"project_name" json:"project_name"`
}

type StageRun struct {
	ID         string     `db:"id" json:"id"`
	RunID      string     `db:"run_id" json:"run_id"`
	Name       string     `db:"name" json:"name"`
	Status     string     `db:"status" json:"status"`
	Position   int        `db:"position" json:"position"`
	StartedAt  *time.Time `db:"started_at" json:"started_at,omitempty"`
	FinishedAt *time.Time `db:"finished_at" json:"finished_at,omitempty"`
}

type JobRun struct {
	ID           string     `db:"id" json:"id"`
	StageRunID   string     `db:"stage_run_id" json:"stage_run_id"`
	RunID        string     `db:"run_id" json:"run_id"`
	Name         string     `db:"name" json:"name"`
	Status       string     `db:"status" json:"status"`
	AgentID      *string    `db:"agent_id" json:"agent_id,omitempty"`
	ExecutorType string     `db:"executor_type" json:"executor_type"`
	StartedAt    *time.Time `db:"started_at" json:"started_at,omitempty"`
	FinishedAt   *time.Time `db:"finished_at" json:"finished_at,omitempty"`
}

type StepRun struct {
	ID           string     `db:"id" json:"id"`
	JobRunID     string     `db:"job_run_id" json:"job_run_id"`
	Name         string     `db:"name" json:"name"`
	Status       string     `db:"status" json:"status"`
	ExitCode     *int       `db:"exit_code" json:"exit_code,omitempty"`
	ErrorMessage *string    `db:"error_message" json:"error_message,omitempty"`
	StartedAt    *time.Time `db:"started_at" json:"started_at,omitempty"`
	FinishedAt   *time.Time `db:"finished_at" json:"finished_at,omitempty"`
	DurationMs   *int       `db:"duration_ms" json:"duration_ms,omitempty"`
}
