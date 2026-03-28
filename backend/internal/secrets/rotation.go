package secrets

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// RotationStatus represents the rotation state of a single secret.
type RotationStatus struct {
	SecretID         string     `db:"id" json:"secret_id"`
	Key              string     `db:"key" json:"key"`
	ProjectID        *string    `db:"project_id" json:"project_id,omitempty"`
	RotationInterval *string    `db:"rotation_interval" json:"rotation_interval,omitempty"`
	LastRotatedAt    *time.Time `db:"last_rotated_at" json:"last_rotated_at,omitempty"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
	IsOverdue        bool       `json:"is_overdue"`
}

// RotationTracker monitors secret rotation schedules and emits events when
// secrets become overdue for rotation.
type RotationTracker struct {
	db       *sqlx.DB
	interval time.Duration // how often to check for stale secrets
	notify   func(ctx context.Context, status RotationStatus) // callback for overdue secrets

	cancel context.CancelFunc
	done   chan struct{}
}

// NewRotationTracker creates a tracker that checks every `checkInterval` for
// secrets whose last rotation exceeds their configured rotation_interval.
// The notify function is called for every overdue secret discovered.
func NewRotationTracker(db *sqlx.DB, checkInterval time.Duration, notify func(ctx context.Context, status RotationStatus)) *RotationTracker {
	if checkInterval <= 0 {
		checkInterval = 1 * time.Hour
	}
	return &RotationTracker{
		db:       db,
		interval: checkInterval,
		notify:   notify,
		done:     make(chan struct{}),
	}
}

// Start begins the background check loop. It is safe to call from a goroutine.
func (t *RotationTracker) Start(ctx context.Context) {
	rCtx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	go t.run(rCtx)
}

// Stop gracefully terminates the background loop.
func (t *RotationTracker) Stop() {
	if t.cancel != nil {
		t.cancel()
		<-t.done
	}
}

func (t *RotationTracker) run(ctx context.Context) {
	defer close(t.done)

	// Run immediately on start, then on each tick.
	t.checkOnce(ctx)

	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.checkOnce(ctx)
		}
	}
}

func (t *RotationTracker) checkOnce(ctx context.Context) {
	overdue, err := t.ListOverdue(ctx)
	if err != nil {
		log.Error().Err(err).Msg("rotation tracker: failed to query overdue secrets")
		return
	}
	for _, s := range overdue {
		log.Warn().
			Str("secret_id", s.SecretID).
			Str("key", s.Key).
			Msg("secret is overdue for rotation")
		if t.notify != nil {
			t.notify(ctx, s)
		}
	}
}

// ListOverdue returns secrets that are past their rotation deadline.
func (t *RotationTracker) ListOverdue(ctx context.Context) ([]RotationStatus, error) {
	query := `
		SELECT id, key, project_id, rotation_interval, last_rotated_at, updated_at
		FROM secrets
		WHERE rotation_interval IS NOT NULL
		  AND rotation_interval != ''
		  AND (
			  last_rotated_at IS NULL
			  OR (
				  CASE rotation_interval
					  WHEN '24h'  THEN datetime(last_rotated_at, '+1 day')
					  WHEN '48h'  THEN datetime(last_rotated_at, '+2 days')
					  WHEN '7d'   THEN datetime(last_rotated_at, '+7 days')
					  WHEN '30d'  THEN datetime(last_rotated_at, '+30 days')
					  WHEN '60d'  THEN datetime(last_rotated_at, '+60 days')
					  WHEN '90d'  THEN datetime(last_rotated_at, '+90 days')
					  WHEN '180d' THEN datetime(last_rotated_at, '+180 days')
					  WHEN '365d' THEN datetime(last_rotated_at, '+365 days')
					  ELSE datetime(last_rotated_at, '+30 days')
				  END
			  ) < datetime('now')
		  )
	`

	var rows []RotationStatus
	if err := t.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("query overdue secrets: %w", err)
	}
	for i := range rows {
		rows[i].IsOverdue = true
	}
	return rows, nil
}

// ListAll returns rotation status for all secrets that have a rotation
// interval configured.
func (t *RotationTracker) ListAll(ctx context.Context, projectID string) ([]RotationStatus, error) {
	query := `
		SELECT id, key, project_id, rotation_interval, last_rotated_at, updated_at
		FROM secrets
		WHERE rotation_interval IS NOT NULL
		  AND rotation_interval != ''
		  AND (project_id = ? OR ? = '')
		ORDER BY last_rotated_at ASC NULLS FIRST
	`

	var rows []RotationStatus
	if err := t.db.SelectContext(ctx, &rows, query, projectID, projectID); err != nil {
		return nil, fmt.Errorf("query rotation status: %w", err)
	}

	now := time.Now()
	for i := range rows {
		rows[i].IsOverdue = isOverdue(rows[i], now)
	}
	return rows, nil
}

// SetRotationPolicy updates the rotation interval and/or marks a secret as
// just rotated.
func (t *RotationTracker) SetRotationPolicy(ctx context.Context, secretID, interval string, markRotated bool) error {
	var query string
	var args []interface{}

	if markRotated {
		query = `UPDATE secrets SET rotation_interval = ?, last_rotated_at = datetime('now'), updated_at = datetime('now') WHERE id = ?`
		args = []interface{}{interval, secretID}
	} else {
		query = `UPDATE secrets SET rotation_interval = ?, updated_at = datetime('now') WHERE id = ?`
		args = []interface{}{interval, secretID}
	}

	res, err := t.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update rotation policy: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("secret %q not found", secretID)
	}
	return nil
}

// MarkRotated updates the last_rotated_at timestamp for a secret.
func (t *RotationTracker) MarkRotated(ctx context.Context, secretID string) error {
	_, err := t.db.ExecContext(ctx,
		`UPDATE secrets SET last_rotated_at = datetime('now'), updated_at = datetime('now') WHERE id = ?`,
		secretID,
	)
	if err != nil {
		return fmt.Errorf("mark rotated: %w", err)
	}
	return nil
}

func isOverdue(s RotationStatus, now time.Time) bool {
	if s.LastRotatedAt == nil {
		return true // never rotated
	}
	if s.RotationInterval == nil {
		return false
	}
	d := parseDuration(*s.RotationInterval)
	return now.After(s.LastRotatedAt.Add(d))
}

// parseDuration interprets shorthand duration strings like "30d", "7d", "24h".
func parseDuration(s string) time.Duration {
	switch s {
	case "24h":
		return 24 * time.Hour
	case "48h":
		return 48 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	case "60d":
		return 60 * 24 * time.Hour
	case "90d":
		return 90 * 24 * time.Hour
	case "180d":
		return 180 * 24 * time.Hour
	case "365d":
		return 365 * 24 * time.Hour
	default:
		// Fallback: try Go's time.ParseDuration.
		d, err := time.ParseDuration(s)
		if err != nil {
			return 30 * 24 * time.Hour // safe default
		}
		return d
	}
}
