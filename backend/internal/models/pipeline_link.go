package models

import "time"

// PipelineLink represents a composition link between two pipelines.
// It allows one pipeline to trigger, fan-out to, or fan-in from another.
type PipelineLink struct {
	ID               string    `db:"id" json:"id"`
	SourcePipelineID string    `db:"source_pipeline_id" json:"source_pipeline_id"`
	TargetPipelineID string    `db:"target_pipeline_id" json:"target_pipeline_id"`
	LinkType         string    `db:"link_type" json:"link_type"` // trigger, fan_out, fan_in
	Condition        string    `db:"condition" json:"condition"`
	PassVariables    bool      `db:"pass_variables" json:"pass_variables"`
	Enabled          bool      `db:"enabled" json:"enabled"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}
