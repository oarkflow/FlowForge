package models

import "time"

type RunLog struct {
	ID        int       `db:"id" json:"id"`
	RunID     string    `db:"run_id" json:"run_id"`
	StepRunID *string   `db:"step_run_id" json:"step_run_id,omitempty"`
	Stream    string    `db:"stream" json:"stream"`
	Content   string    `db:"content" json:"content"`
	Ts        time.Time `db:"ts" json:"ts"`
}
