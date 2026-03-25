package models

import "time"

type Repository struct {
	ID             string     `db:"id" json:"id"`
	ProjectID      string     `db:"project_id" json:"project_id"`
	Provider       string     `db:"provider" json:"provider"`
	ProviderID     string     `db:"provider_id" json:"provider_id"`
	FullName       string     `db:"full_name" json:"full_name"`
	CloneURL       string     `db:"clone_url" json:"clone_url"`
	SSHURL         *string    `db:"ssh_url" json:"ssh_url,omitempty"`
	DefaultBranch  string     `db:"default_branch" json:"default_branch"`
	WebhookID      *string    `db:"webhook_id" json:"-"`
	WebhookSecret  *string    `db:"webhook_secret" json:"-"`
	AccessTokenEnc *string    `db:"access_token_enc" json:"-"`
	SSHKeyEnc      *string    `db:"ssh_key_enc" json:"-"`
	IsActive       int        `db:"is_active" json:"is_active"`
	LastSyncAt     *time.Time `db:"last_sync_at" json:"last_sync_at,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}
