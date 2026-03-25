package models

import "time"

type Secret struct {
	ID        string    `db:"id" json:"id"`
	ProjectID *string   `db:"project_id" json:"project_id,omitempty"`
	OrgID     *string   `db:"org_id" json:"org_id,omitempty"`
	Scope     string    `db:"scope" json:"scope"`
	Key       string    `db:"key" json:"key"`
	ValueEnc  string    `db:"value_enc" json:"-"`
	Masked    int       `db:"masked" json:"masked"`
	CreatedBy *string   `db:"created_by" json:"created_by,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
