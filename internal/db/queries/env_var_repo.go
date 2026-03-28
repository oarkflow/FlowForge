package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// EnvVarRepo handles CRUD operations for project environment variables.
type EnvVarRepo struct {
	db *sqlx.DB
}

// NewEnvVarRepo creates a new EnvVarRepo.
func NewEnvVarRepo(db *sqlx.DB) *EnvVarRepo {
	return &EnvVarRepo{db: db}
}

// ListByProject returns all environment variables for a project.
func (r *EnvVarRepo) ListByProject(ctx context.Context, projectID string) ([]models.EnvVar, error) {
	var vars []models.EnvVar
	err := r.db.SelectContext(ctx, &vars,
		"SELECT id, project_id, key, value, created_at, updated_at FROM project_env_vars WHERE project_id = ? ORDER BY key ASC",
		projectID,
	)
	if err != nil {
		return nil, err
	}
	if vars == nil {
		vars = []models.EnvVar{}
	}
	return vars, nil
}

// Create inserts a new environment variable.
func (r *EnvVarRepo) Create(ctx context.Context, ev *models.EnvVar) error {
	if ev.ID == "" {
		ev.ID = uuid.New().String()
	}
	now := time.Now()
	ev.CreatedAt = now
	ev.UpdatedAt = now
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO project_env_vars (id, project_id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		ev.ID, ev.ProjectID, ev.Key, ev.Value, ev.CreatedAt, ev.UpdatedAt,
	)
	return err
}

// Update modifies an existing environment variable's key and value.
func (r *EnvVarRepo) Update(ctx context.Context, ev *models.EnvVar) error {
	ev.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		"UPDATE project_env_vars SET key = ?, value = ?, updated_at = ? WHERE id = ?",
		ev.Key, ev.Value, ev.UpdatedAt, ev.ID,
	)
	return err
}

// Delete removes an environment variable by ID.
func (r *EnvVarRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM project_env_vars WHERE id = ?", id)
	return err
}

// BulkSave replaces all environment variables for a project. It deletes
// existing vars and inserts the provided set in a single transaction.
func (r *EnvVarRepo) BulkSave(ctx context.Context, projectID string, vars []models.EnvVar) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete all existing vars for the project
	if _, err := tx.ExecContext(ctx, "DELETE FROM project_env_vars WHERE project_id = ?", projectID); err != nil {
		return err
	}

	// Insert new vars
	now := time.Now()
	insertQuery := "INSERT INTO project_env_vars (id, project_id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)"
	for _, ev := range vars {
		id := ev.ID
		if id == "" {
			id = uuid.New().String()
		}
		if _, err := tx.ExecContext(ctx,
			insertQuery,
			id, projectID, ev.Key, ev.Value, now, now,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetForInjection returns all environment variables for a project as a
// key=value map suitable for environment variable injection.
func (r *EnvVarRepo) GetForInjection(ctx context.Context, projectID string) (map[string]string, error) {
	vars, err := r.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(vars))
	for _, v := range vars {
		result[v.Key] = v.Value
	}
	return result, nil
}
