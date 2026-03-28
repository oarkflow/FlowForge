package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type AgentRepo struct {
	db *sqlx.DB
}

func (r *AgentRepo) GetByID(ctx context.Context, id string) (*models.Agent, error) {
	agent := &models.Agent{}
	err := r.db.GetContext(ctx, agent, "SELECT * FROM agents WHERE id = ?", id)
	return agent, err
}

func (r *AgentRepo) List(ctx context.Context, limit, offset int) ([]models.Agent, error) {
	agents := []models.Agent{}
	err := r.db.SelectContext(ctx, &agents, "SELECT * FROM agents ORDER BY created_at DESC LIMIT ? OFFSET ?", limit, offset)
	return agents, err
}

func (r *AgentRepo) ListByStatus(ctx context.Context, status string) ([]models.Agent, error) {
	agents := []models.Agent{}
	err := r.db.SelectContext(ctx, &agents, "SELECT * FROM agents WHERE status = ?", status)
	return agents, err
}

func (r *AgentRepo) Create(ctx context.Context, agent *models.Agent) error {
	agent.ID = uuid.New().String()
	agent.CreatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO agents (id, name, token_hash, labels, executor, status, version, os, arch, cpu_cores, memory_mb, ip_address, created_at)
		VALUES (:id, :name, :token_hash, :labels, :executor, :status, :version, :os, :arch, :cpu_cores, :memory_mb, :ip_address, :created_at)`,
		agent)
	return err
}

func (r *AgentRepo) UpdateStatus(ctx context.Context, id, status string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, "UPDATE agents SET status = ?, last_seen_at = ? WHERE id = ?", status, now, id)
	return err
}

func (r *AgentRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM agents WHERE id = ?", id)
	return err
}
