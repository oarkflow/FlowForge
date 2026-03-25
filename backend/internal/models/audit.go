package models

import "time"

type AuditLog struct {
	ID         int       `db:"id" json:"id"`
	ActorID    *string   `db:"actor_id" json:"actor_id,omitempty"`
	ActorIP    *string   `db:"actor_ip" json:"actor_ip,omitempty"`
	Action     string    `db:"action" json:"action"`
	Resource   string    `db:"resource" json:"resource"`
	ResourceID *string   `db:"resource_id" json:"resource_id,omitempty"`
	Changes    *string   `db:"changes" json:"changes,omitempty"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
