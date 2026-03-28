package logging

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
)

// RetentionWorker periodically deletes old logs based on retention policies.
type RetentionWorker struct {
	repos               *queries.Repositories
	globalRetentionDays int
	interval            time.Duration
}

// NewRetentionWorker creates a new log retention worker.
func NewRetentionWorker(repos *queries.Repositories, globalRetentionDays int) *RetentionWorker {
	if globalRetentionDays <= 0 {
		globalRetentionDays = 90
	}
	return &RetentionWorker{
		repos:               repos,
		globalRetentionDays: globalRetentionDays,
		interval:            24 * time.Hour,
	}
}

// Start runs the retention worker in a loop, checking once per interval.
func (w *RetentionWorker) Start(ctx context.Context) {
	log.Info().Int("global_retention_days", w.globalRetentionDays).Msg("retention: worker started")

	// Run once immediately on startup
	w.cleanup(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("retention: worker stopped")
			return
		case <-ticker.C:
			w.cleanup(ctx)
		}
	}
}

func (w *RetentionWorker) cleanup(ctx context.Context) {
	// 1. Apply per-project retention overrides
	projects, err := w.repos.Projects.List(ctx, 10000, 0)
	if err != nil {
		log.Error().Err(err).Msg("retention: failed to list projects")
	} else {
		for _, p := range projects {
			if p.LogRetentionDays > 0 {
				cutoff := time.Now().AddDate(0, 0, -p.LogRetentionDays).Format("2006-01-02 15:04:05")
				deleted, err := w.repos.Logs.DeleteByProjectBefore(ctx, p.ID, cutoff)
				if err != nil {
					log.Error().Err(err).Str("project_id", p.ID).Msg("retention: project cleanup failed")
				} else if deleted > 0 {
					log.Info().
						Str("project_id", p.ID).
						Int64("deleted", deleted).
						Int("retention_days", p.LogRetentionDays).
						Msg("retention: cleaned project logs")
				}
			}
		}
	}

	// 2. Apply global retention for all remaining logs
	cutoff := time.Now().AddDate(0, 0, -w.globalRetentionDays).Format("2006-01-02 15:04:05")
	deleted, err := w.repos.Logs.DeleteBefore(ctx, cutoff)
	if err != nil {
		log.Error().Err(err).Msg("retention: global cleanup failed")
	} else if deleted > 0 {
		log.Info().Int64("deleted", deleted).Str("cutoff", cutoff).Msg("retention: cleaned global logs")
	}

	_ = fmt.Sprintf("retention cleanup complete") // avoid unused import
}
