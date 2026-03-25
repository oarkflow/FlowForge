package models

import "time"

type Pipeline struct {
	ID            string     `db:"id" json:"id"`
	ProjectID     string     `db:"project_id" json:"project_id"`
	RepositoryID  *string    `db:"repository_id" json:"repository_id,omitempty"`
	Name          string     `db:"name" json:"name"`
	Description   *string    `db:"description" json:"description,omitempty"`
	ConfigSource  string     `db:"config_source" json:"config_source"`
	ConfigPath    *string    `db:"config_path" json:"config_path,omitempty"`
	ConfigContent *string    `db:"config_content" json:"config_content,omitempty"`
	ConfigVersion int        `db:"config_version" json:"config_version"`
	Triggers      string     `db:"triggers" json:"triggers"`
	IsActive      int        `db:"is_active" json:"is_active"`
	CreatedBy     *string    `db:"created_by" json:"created_by,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt     *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

type PipelineVersion struct {
	ID         string    `db:"id" json:"id"`
	PipelineID string    `db:"pipeline_id" json:"pipeline_id"`
	Version    int       `db:"version" json:"version"`
	Config     string    `db:"config" json:"config"`
	Message    *string   `db:"message" json:"message,omitempty"`
	CreatedBy  *string   `db:"created_by" json:"created_by,omitempty"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
