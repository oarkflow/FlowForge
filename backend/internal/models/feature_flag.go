package models

import "time"

type FeatureFlag struct {
	ID                string    `db:"id" json:"id"`
	Name              string    `db:"name" json:"name"`
	Description       string    `db:"description" json:"description"`
	Enabled           int       `db:"enabled" json:"enabled"`
	RolloutPercentage int       `db:"rollout_percentage" json:"rollout_percentage"`
	TargetUsers       string    `db:"target_users" json:"target_users"` // JSON array of user IDs
	TargetOrgs        string    `db:"target_orgs" json:"target_orgs"`   // JSON array of org IDs
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}
