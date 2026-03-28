package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type ApprovalResponseRepo struct {
	db *sqlx.DB
}

// ListByApproval returns all responses for a given approval.
func (r *ApprovalResponseRepo) ListByApproval(ctx context.Context, approvalID string) ([]models.ApprovalResponse, error) {
	responses := []models.ApprovalResponse{}
	err := r.db.SelectContext(ctx, &responses,
		"SELECT * FROM approval_responses WHERE approval_id = ? ORDER BY created_at ASC",
		approvalID)
	return responses, err
}

// Create inserts a new approval response.
func (r *ApprovalResponseRepo) Create(ctx context.Context, response *models.ApprovalResponse) error {
	response.ID = uuid.New().String()
	response.CreatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO approval_responses (id, approval_id, approver_id, approver_name, decision, comment, created_at)
		VALUES (:id, :approval_id, :approver_id, :approver_name, :decision, :comment, :created_at)`,
		response)
	return err
}

// HasResponded checks whether the given approver has already responded to an approval.
func (r *ApprovalResponseRepo) HasResponded(ctx context.Context, approvalID, approverID string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM approval_responses WHERE approval_id = ? AND approver_id = ?",
		approvalID, approverID)
	return count > 0, err
}
