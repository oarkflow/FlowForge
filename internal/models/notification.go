package models

import "time"

type NotificationChannel struct {
	ID        string    `db:"id" json:"id"`
	ProjectID *string   `db:"project_id" json:"project_id,omitempty"`
	Type      string    `db:"type" json:"type"`
	Name      string    `db:"name" json:"name"`
	ConfigEnc string    `db:"config_enc" json:"-"`
	IsActive  int       `db:"is_active" json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
