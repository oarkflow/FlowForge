package models

import "time"

type DashboardPreference struct {
	ID        string    `db:"id" json:"id"`
	UserID    string    `db:"user_id" json:"user_id"`
	Layout    string    `db:"layout" json:"layout"` // JSON string of widget configuration
	Theme     string    `db:"theme" json:"theme"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
