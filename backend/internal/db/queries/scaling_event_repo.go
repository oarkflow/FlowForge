package queries

import (
	"github.com/jmoiron/sqlx"
	"github.com/oarkflow/deploy/backend/internal/models"
)

type ScalingEventRepo struct {
	db *sqlx.DB
}

func (r *ScalingEventRepo) ListByPolicy(policyID string, limit int) ([]models.ScalingEvent, error) {
	if limit <= 0 || limit > 1000 {
		limit = 50
	}
	var events []models.ScalingEvent
	err := r.db.Select(&events,
		"SELECT * FROM scaling_events WHERE policy_id = ? ORDER BY created_at DESC LIMIT ?",
		policyID, limit)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (r *ScalingEventRepo) Create(event *models.ScalingEvent) error {
	_, err := r.db.Exec(`
		INSERT INTO scaling_events (policy_id, action, from_count, to_count, reason, queue_depth, active_agents)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.PolicyID, event.Action, event.FromCount, event.ToCount,
		event.Reason, event.QueueDepth, event.ActiveAgents,
	)
	if err != nil {
		return err
	}
	return r.db.Get(event, "SELECT * FROM scaling_events WHERE rowid = last_insert_rowid()")
}

func (r *ScalingEventRepo) ListRecent(limit int) ([]models.ScalingEvent, error) {
	if limit <= 0 || limit > 1000 {
		limit = 50
	}
	var events []models.ScalingEvent
	err := r.db.Select(&events,
		"SELECT * FROM scaling_events ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	return events, nil
}
