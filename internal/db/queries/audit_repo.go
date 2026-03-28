package queries

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type AuditLogRepo struct {
	db *sqlx.DB
}

func (r *AuditLogRepo) Insert(ctx context.Context, log *models.AuditLog) error {
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO audit_logs (actor_id, actor_ip, action, resource, resource_id, changes)
		VALUES (:actor_id, :actor_ip, :action, :resource, :resource_id, :changes)`,
		log)
	return err
}

func (r *AuditLogRepo) List(ctx context.Context, limit, offset int) ([]models.AuditLog, error) {
	logs := []models.AuditLog{}
	err := r.db.SelectContext(ctx, &logs, "SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT ? OFFSET ?", limit, offset)
	return logs, err
}

func (r *AuditLogRepo) ListByActor(ctx context.Context, actorID string, limit, offset int) ([]models.AuditLog, error) {
	logs := []models.AuditLog{}
	err := r.db.SelectContext(ctx, &logs, "SELECT * FROM audit_logs WHERE actor_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?", actorID, limit, offset)
	return logs, err
}

func (r *AuditLogRepo) ListByResource(ctx context.Context, resource, resourceID string, limit, offset int) ([]models.AuditLog, error) {
	logs := []models.AuditLog{}
	err := r.db.SelectContext(ctx, &logs, "SELECT * FROM audit_logs WHERE resource = ? AND resource_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?", resource, resourceID, limit, offset)
	return logs, err
}
