package models

import "time"

type PipelineTemplate struct {
	ID          string    `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	Category    string    `db:"category" json:"category"`
	Config      string    `db:"config" json:"config"`
	IsBuiltin   int       `db:"is_builtin" json:"is_builtin"`
	IsPublic    int       `db:"is_public" json:"is_public"`
	Author      string    `db:"author" json:"author"`
	Downloads   int       `db:"downloads" json:"downloads"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}
