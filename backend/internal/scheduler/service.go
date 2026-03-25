package scheduler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/models"
)

// PipelineEngine is the interface for triggering pipeline runs.
type PipelineEngine interface {
	TriggerPipeline(ctx context.Context, pipelineID, triggerType string, triggerData map[string]string) (*models.PipelineRun, error)
}

// Service manages pipeline schedule operations.
type Service struct {
	repos  *queries.Repositories
	engine PipelineEngine
}

// NewService creates a new schedule service.
func NewService(repos *queries.Repositories) *Service {
	return &Service{repos: repos}
}

// SetEngine sets the pipeline engine used to trigger runs.
func (s *Service) SetEngine(engine PipelineEngine) {
	s.engine = engine
}

// Create creates a new schedule and computes next_run_at.
func (s *Service) Create(ctx context.Context, schedule *models.PipelineSchedule) error {
	// Validate cron expression
	if err := ValidateCron(schedule.CronExpression); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Validate timezone
	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		return fmt.Errorf("invalid timezone %q: %w", schedule.Timezone, err)
	}

	// Generate ID
	if schedule.ID == "" {
		schedule.ID = generateID()
	}

	// Compute next_run_at
	if schedule.Enabled {
		cron, _ := ParseCron(schedule.CronExpression)
		now := time.Now().In(loc)
		next := cron.Next(now)
		nextUTC := next.UTC()
		schedule.NextRunAt = &nextUTC
	}

	now := time.Now().UTC()
	schedule.CreatedAt = now
	schedule.UpdatedAt = now

	return s.repos.Schedules.Create(ctx, schedule)
}

// Update updates a schedule and recomputes next_run_at.
func (s *Service) Update(ctx context.Context, schedule *models.PipelineSchedule) error {
	// Validate cron expression
	if err := ValidateCron(schedule.CronExpression); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Validate timezone
	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		return fmt.Errorf("invalid timezone %q: %w", schedule.Timezone, err)
	}

	// Recompute next_run_at
	if schedule.Enabled {
		cron, _ := ParseCron(schedule.CronExpression)
		now := time.Now().In(loc)
		next := cron.Next(now)
		nextUTC := next.UTC()
		schedule.NextRunAt = &nextUTC
	} else {
		schedule.NextRunAt = nil
	}

	schedule.UpdatedAt = time.Now().UTC()

	return s.repos.Schedules.Update(ctx, schedule)
}

// Enable enables a schedule and computes next_run_at.
func (s *Service) Enable(ctx context.Context, id string) error {
	schedule, err := s.repos.Schedules.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("schedule not found: %w", err)
	}

	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		loc = time.UTC
	}

	cron, err := ParseCron(schedule.CronExpression)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	now := time.Now().In(loc)
	next := cron.Next(now)
	nextUTC := next.UTC()

	if err := s.repos.Schedules.SetEnabled(ctx, id, true); err != nil {
		return err
	}

	// Also update next_run_at
	schedule.Enabled = true
	schedule.NextRunAt = &nextUTC
	schedule.UpdatedAt = time.Now().UTC()
	return s.repos.Schedules.Update(ctx, schedule)
}

// Disable disables a schedule and clears next_run_at.
func (s *Service) Disable(ctx context.Context, id string) error {
	schedule, err := s.repos.Schedules.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("schedule not found: %w", err)
	}

	if err := s.repos.Schedules.SetEnabled(ctx, id, false); err != nil {
		return err
	}

	schedule.Enabled = false
	schedule.NextRunAt = nil
	schedule.UpdatedAt = time.Now().UTC()
	return s.repos.Schedules.Update(ctx, schedule)
}

// GetDueSchedules returns schedules where next_run_at <= now and enabled.
func (s *Service) GetDueSchedules(ctx context.Context) ([]models.PipelineSchedule, error) {
	return s.repos.Schedules.ListDue(ctx, time.Now().UTC())
}

// AdvanceSchedule updates last_run_at, run_count, and next_run_at after execution.
func (s *Service) AdvanceSchedule(ctx context.Context, id string, runID string, status string) error {
	schedule, err := s.repos.Schedules.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("schedule not found: %w", err)
	}

	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		loc = time.UTC
	}

	cron, err := ParseCron(schedule.CronExpression)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	now := time.Now().UTC()
	next := cron.Next(now.In(loc))
	nextUTC := next.UTC()

	return s.repos.Schedules.UpdateAfterRun(ctx, id, now, nextUTC, runID, status)
}

// GetNextRuns returns the next N scheduled run times for a schedule.
func (s *Service) GetNextRuns(ctx context.Context, id string, count int) ([]time.Time, error) {
	schedule, err := s.repos.Schedules.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("schedule not found: %w", err)
	}

	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		loc = time.UTC
	}

	cron, err := ParseCron(schedule.CronExpression)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	now := time.Now().In(loc)
	return cron.NextN(now, count), nil
}

// ProcessDueSchedules checks for due schedules and triggers pipeline runs.
// This is designed to be called by the background worker.
func (s *Service) ProcessDueSchedules(ctx context.Context) error {
	if s.engine == nil {
		return nil
	}

	schedules, err := s.GetDueSchedules(ctx)
	if err != nil {
		return fmt.Errorf("failed to get due schedules: %w", err)
	}

	for _, schedule := range schedules {
		if err := s.triggerScheduledRun(ctx, &schedule); err != nil {
			log.Error().Err(err).
				Str("schedule_id", schedule.ID).
				Str("pipeline_id", schedule.PipelineID).
				Msg("failed to trigger scheduled pipeline run")
			// Continue processing other schedules; don't fail the whole batch
			continue
		}
	}

	return nil
}

// triggerScheduledRun triggers a single scheduled pipeline run.
func (s *Service) triggerScheduledRun(ctx context.Context, schedule *models.PipelineSchedule) error {
	triggerData := map[string]string{
		"branch":      schedule.Branch,
		"schedule_id": schedule.ID,
		"trigger":     "schedule",
	}

	if schedule.EnvironmentID != nil && *schedule.EnvironmentID != "" {
		triggerData["environment_id"] = *schedule.EnvironmentID
	}

	run, err := s.engine.TriggerPipeline(ctx, schedule.PipelineID, "schedule", triggerData)
	if err != nil {
		// Advance the schedule even on failure to prevent re-triggering
		_ = s.AdvanceSchedule(ctx, schedule.ID, "", "trigger_failed")
		return fmt.Errorf("failed to trigger pipeline %s: %w", schedule.PipelineID, err)
	}

	log.Info().
		Str("schedule_id", schedule.ID).
		Str("pipeline_id", schedule.PipelineID).
		Str("run_id", run.ID).
		Int("run_number", run.Number).
		Msg("scheduled pipeline run triggered")

	return s.AdvanceSchedule(ctx, schedule.ID, run.ID, "triggered")
}

// generateID generates a random hex ID matching the SQLite default pattern.
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
