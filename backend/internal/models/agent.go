package models

import "time"

type Agent struct {
	ID         string     `db:"id" json:"id"`
	Name       string     `db:"name" json:"name"`
	TokenHash  string     `db:"token_hash" json:"-"`
	Labels     string     `db:"labels" json:"labels"`
	Executor   string     `db:"executor" json:"executor"`
	Status     string     `db:"status" json:"status"`
	Version    *string    `db:"version" json:"version,omitempty"`
	OS         *string    `db:"os" json:"os,omitempty"`
	Arch       *string    `db:"arch" json:"arch,omitempty"`
	CPUCores   *int       `db:"cpu_cores" json:"cpu_cores,omitempty"`
	MemoryMB   *int       `db:"memory_mb" json:"memory_mb,omitempty"`
	IPAddress  *string    `db:"ip_address" json:"ip_address,omitempty"`
	LastSeenAt *time.Time `db:"last_seen_at" json:"last_seen_at,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}
