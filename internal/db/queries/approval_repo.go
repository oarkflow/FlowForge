package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type ApprovalRepo struct {
	db *sqlx.DB
}

// ListPending returns pending approvals where the given user is in required_approvers.
func (r *ApprovalRepo) ListPending(ctx context.Context, userID string) ([]models.Approval, error) {
	approvals := []models.Approval{}
	// Use LIKE pattern matching on the JSON string to check if user is in required_approvers
	err := r.db.SelectContext(ctx, &approvals,
		`SELECT * FROM approvals
		 WHERE status = 'pending'
		 AND required_approvers LIKE '%' || ? || '%'
		 ORDER BY created_at DESC`,
		userID)
	return approvals, err
}

// ListByProject returns all approvals for a project.
func (r *ApprovalRepo) ListByProject(ctx context.Context, projectID string) ([]models.Approval, error) {
	approvals := []models.Approval{}
	err := r.db.SelectContext(ctx, &approvals,
		"SELECT * FROM approvals WHERE project_id = ? ORDER BY created_at DESC",
		projectID)
	return approvals, err
}

// GetByID returns an approval by its ID.
func (r *ApprovalRepo) GetByID(ctx context.Context, id string) (*models.Approval, error) {
	approval := &models.Approval{}
	err := r.db.GetContext(ctx, approval, "SELECT * FROM approvals WHERE id = ?", id)
	return approval, err
}

// GetByDeployment returns the approval for a given deployment.
func (r *ApprovalRepo) GetByDeployment(ctx context.Context, deploymentID string) (*models.Approval, error) {
	approval := &models.Approval{}
	err := r.db.GetContext(ctx, approval,
		"SELECT * FROM approvals WHERE deployment_id = ? ORDER BY created_at DESC LIMIT 1",
		deploymentID)
	return approval, err
}

// Create inserts a new approval record.
func (r *ApprovalRepo) Create(ctx context.Context, approval *models.Approval) error {
	approval.ID = uuid.New().String()
	approval.CreatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO approvals (id, type, deployment_id, pipeline_run_id, environment_id,
			project_id, requested_by, status, required_approvers, min_approvals,
			current_approvals, expires_at, resolved_at, created_at)
		VALUES (:id, :type, :deployment_id, :pipeline_run_id, :environment_id,
			:project_id, :requested_by, :status, :required_approvers, :min_approvals,
			:current_approvals, :expires_at, :resolved_at, :created_at)`,
		approval)
	return err
}

// UpdateStatus sets the status and resolved_at timestamp.
func (r *ApprovalRepo) UpdateStatus(ctx context.Context, id, status string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		"UPDATE approvals SET status = ?, resolved_at = ? WHERE id = ?",
		status, now, id)
	return err
}

// IncrementApprovals increments the current_approvals counter.
func (r *ApprovalRepo) IncrementApprovals(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE approvals SET current_approvals = current_approvals + 1 WHERE id = ?",
		id)
	return err
}

// ExpirePending expires all pending approvals past their expires_at deadline.
func (r *ApprovalRepo) ExpirePending(ctx context.Context) (int, error) {
	now := time.Now()
	result, err := r.db.ExecContext(ctx,
		`UPDATE approvals SET status = 'expired', resolved_at = CURRENT_TIMESTAMP
		 WHERE status = 'pending'
		 AND expires_at IS NOT NULL
		 AND expires_at < ?`,
		now)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// Cancel sets an approval status to cancelled.
func (r *ApprovalRepo) Cancel(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		"UPDATE approvals SET status = 'cancelled', resolved_at = ? WHERE id = ? AND status = 'pending'",
		now, id)
	return err
}

// ListAll returns all approvals across all projects, ordered by created_at desc.
func (r *ApprovalRepo) ListAll(ctx context.Context, status string, limit, offset int) ([]models.Approval, error) {
	approvals := []models.Approval{}
	if status != "" {
		err := r.db.SelectContext(ctx, &approvals,
			"SELECT * FROM approvals WHERE status = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
			status, limit, offset)
		return approvals, err
	}
	err := r.db.SelectContext(ctx, &approvals,
		"SELECT * FROM approvals ORDER BY created_at DESC LIMIT ? OFFSET ?",
		limit, offset)
	return approvals, err
}
