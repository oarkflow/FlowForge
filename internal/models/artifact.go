package models

import "time"

type Artifact struct {
	ID             string     `db:"id" json:"id"`
	RunID          string     `db:"run_id" json:"run_id"`
	StepRunID      *string    `db:"step_run_id" json:"step_run_id,omitempty"`
	Name           string     `db:"name" json:"name"`
	Path           string     `db:"path" json:"path"`
	SizeBytes      *int       `db:"size_bytes" json:"size_bytes,omitempty"`
	ChecksumSHA256 *string    `db:"checksum_sha256" json:"checksum_sha256,omitempty"`
	StorageBackend string     `db:"storage_backend" json:"storage_backend"`
	StorageKey     string     `db:"storage_key" json:"storage_key"`
	ExpireAt       *time.Time `db:"expire_at" json:"expire_at,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}
