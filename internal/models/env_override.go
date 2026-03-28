package models

import "time"

type EnvOverride struct {
	ID            string    `db:"id" json:"id"`
	EnvironmentID string    `db:"environment_id" json:"environment_id"`
	Key           string    `db:"key" json:"key"`
	ValueEnc      string    `db:"value_enc" json:"value_enc,omitempty"`
	IsSecret      bool      `db:"is_secret" json:"is_secret"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}
