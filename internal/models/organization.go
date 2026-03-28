package models

import "time"

type Organization struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Slug      string    `db:"slug" json:"slug"`
	LogoURL   *string   `db:"logo_url" json:"logo_url,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type OrgMember struct {
	OrgID    string    `db:"org_id" json:"org_id"`
	UserID   string    `db:"user_id" json:"user_id"`
	Role     string    `db:"role" json:"role"`
	JoinedAt time.Time `db:"joined_at" json:"joined_at"`
}
