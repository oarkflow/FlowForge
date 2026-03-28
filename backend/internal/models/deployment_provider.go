package models

import "time"

// ProjectDeploymentProvider stores project-scoped deployment vendor configuration.
type ProjectDeploymentProvider struct {
	ID           string    `db:"id" json:"id"`
	ProjectID    string    `db:"project_id" json:"project_id"`
	Name         string    `db:"name" json:"name"`
	ProviderType string    `db:"provider_type" json:"provider_type"`
	ConfigEnc    string    `db:"config_enc" json:"-"`
	IsActive     int       `db:"is_active" json:"is_active"`
	CreatedBy    *string   `db:"created_by" json:"created_by,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// ProjectEnvironmentChainEdge defines an allowed source -> target promotion edge.
type ProjectEnvironmentChainEdge struct {
	ID                  string    `db:"id" json:"id"`
	ProjectID           string    `db:"project_id" json:"project_id"`
	SourceEnvironmentID string    `db:"source_environment_id" json:"source_environment_id"`
	TargetEnvironmentID string    `db:"target_environment_id" json:"target_environment_id"`
	Position            int       `db:"position" json:"position"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
}

// PipelineStageEnvironmentMapping maps a pipeline stage to a project environment.
type PipelineStageEnvironmentMapping struct {
	ID            string    `db:"id" json:"id"`
	ProjectID     string    `db:"project_id" json:"project_id"`
	PipelineID    string    `db:"pipeline_id" json:"pipeline_id"`
	StageName     string    `db:"stage_name" json:"stage_name"`
	EnvironmentID string    `db:"environment_id" json:"environment_id"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}
