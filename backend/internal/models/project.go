package models

import "time"

type Project struct {
	ID               string     `db:"id" json:"id"`
	OrgID            *string    `db:"org_id" json:"org_id,omitempty"`
	Name             string     `db:"name" json:"name"`
	Slug             string     `db:"slug" json:"slug"`
	Description      *string    `db:"description" json:"description,omitempty"`
	Visibility       string     `db:"visibility" json:"visibility"`
	LogRetentionDays int        `db:"log_retention_days" json:"log_retention_days"`
	CreatedBy        *string    `db:"created_by" json:"created_by,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt        *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}
