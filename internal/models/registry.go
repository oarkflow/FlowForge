package models

import "time"

// Registry represents a container registry configured for a project.
type Registry struct {
	ID             string    `db:"id" json:"id"`
	ProjectID      string    `db:"project_id" json:"project_id"`
	Name           string    `db:"name" json:"name"`
	Type           string    `db:"type" json:"type"`
	URL            string    `db:"url" json:"url"`
	Username       string    `db:"username" json:"username"`
	CredentialsEnc string    `db:"credentials_enc" json:"-"`
	IsDefault      bool      `db:"is_default" json:"is_default"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// RegistryImage represents a container image in a registry.
type RegistryImage struct {
	Name      string   `json:"name"`
	Tags      []string `json:"tags"`
	Size      int64    `json:"size"`
	Digest    string   `json:"digest"`
	PushedAt  string   `json:"pushed_at"`
	PullCount int64    `json:"pull_count"`
}

// RegistryTag represents a tag for a container image.
type RegistryTag struct {
	Name      string `json:"name"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}
