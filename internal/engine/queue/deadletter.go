package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/models"
)

// DeadLetterQueue manages failed job dispatches that need manual intervention or retry.
type DeadLetterQueue struct {
	repo       *queries.DeadLetterRepo
	maxRetries int
}

// NewDeadLetterQueue creates a new DLQ.
func NewDeadLetterQueue(repo *queries.DeadLetterRepo) *DeadLetterQueue {
	return &DeadLetterQueue{
		repo:       repo,
		maxRetries: 3,
	}
}

// Add places a failed job into the dead-letter queue.
func (dlq *DeadLetterQueue) Add(ctx context.Context, job *Job, reason string) error {
	jobData, _ := json.Marshal(job)
	item := &models.DeadLetterItem{
		JobID:         job.ID,
		PipelineRunID: job.PipelineRunID,
		FailureReason: reason,
		RetryCount:    0,
		MaxRetries:    dlq.maxRetries,
		JobData:       string(jobData),
		Status:        "pending",
	}
	if err := dlq.repo.Create(ctx, item); err != nil {
		log.Error().Err(err).Str("job_id", job.ID).Msg("dlq: failed to add item")
		return err
	}
	log.Warn().Str("job_id", job.ID).Str("reason", reason).Msg("dlq: job added to dead-letter queue")
	return nil
}

// List returns items from the DLQ with optional status filtering.
func (dlq *DeadLetterQueue) List(ctx context.Context, status string, limit, offset int) ([]models.DeadLetterItem, error) {
	return dlq.repo.List(ctx, status, limit, offset)
}

// Retry attempts to retry a DLQ item. Returns the original Job for re-enqueue.
func (dlq *DeadLetterQueue) Retry(ctx context.Context, id string) (*Job, error) {
	item, err := dlq.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.Status != "pending" {
		return nil, nil
	}
	if item.RetryCount >= item.MaxRetries {
		return nil, nil
	}

	var job Job
	if err := json.Unmarshal([]byte(item.JobData), &job); err != nil {
		return nil, err
	}

	// Update status
	_ = dlq.repo.IncrementRetry(ctx, id)
	_ = dlq.repo.UpdateStatus(ctx, id, "retried")

	// Create a new job with a fresh ID for retry
	job.CreatedAt = time.Now()
	job.Cancelled = false

	return &job, nil
}

// Purge removes a DLQ item.
func (dlq *DeadLetterQueue) Purge(ctx context.Context, id string) error {
	return dlq.repo.UpdateStatus(ctx, id, "purged")
}
