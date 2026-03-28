package models

import "time"

// SecretProviderConfig represents a configured external secret provider.
type SecretProviderConfig struct {
	ID           string    `db:"id" json:"id"`
	ProjectID    *string   `db:"project_id" json:"project_id,omitempty"`
	Name         string    `db:"name" json:"name"`
	ProviderType string    `db:"provider_type" json:"provider_type"`
	ConfigEnc    string    `db:"config_enc" json:"-"`
	IsActive     int       `db:"is_active" json:"is_active"`
	Priority     int       `db:"priority" json:"priority"`
	CreatedBy    *string   `db:"created_by" json:"created_by,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// IPAllowlistEntry represents an IP allowlist rule.
type IPAllowlistEntry struct {
	ID        string    `db:"id" json:"id"`
	ProjectID *string   `db:"project_id" json:"project_id,omitempty"`
	Scope     string    `db:"scope" json:"scope"`
	CIDR      string    `db:"cidr" json:"cidr"`
	Label     string    `db:"label" json:"label"`
	CreatedBy *string   `db:"created_by" json:"created_by,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
