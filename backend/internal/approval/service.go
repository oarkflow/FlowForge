package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/models"
)

// Service handles approval logic. It is stateless — all state lives in the database.
type Service struct {
	repos *queries.Repositories
}

// NewService creates a new approval service.
func NewService(repos *queries.Repositories) *Service {
	return &Service{repos: repos}
}

// ApprovalRequest holds the parameters for creating a new approval.
type ApprovalRequest struct {
	Type          string
	DeploymentID  *string
	PipelineRunID *string
	EnvironmentID *string
	ProjectID     string
	RequestedBy   string
	MinApprovals  int
	Approvers     []string // user IDs
	ExpiresIn     time.Duration
}

// RequestApproval creates a new approval request.
func (s *Service) RequestApproval(ctx context.Context, req *ApprovalRequest) (*models.Approval, error) {
	approversJSON, err := json.Marshal(req.Approvers)
	if err != nil {
		return nil, fmt.Errorf("marshal approvers: %w", err)
	}

	minApprovals := req.MinApprovals
	if minApprovals < 1 {
		minApprovals = 1
	}

	expiresIn := req.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 72 * time.Hour // default: 72 hours
	}
	expiresAt := time.Now().Add(expiresIn)

	approval := &models.Approval{
		Type:              req.Type,
		DeploymentID:      req.DeploymentID,
		PipelineRunID:     req.PipelineRunID,
		EnvironmentID:     req.EnvironmentID,
		ProjectID:         req.ProjectID,
		RequestedBy:       req.RequestedBy,
		Status:            "pending",
		RequiredApprovers: string(approversJSON),
		MinApprovals:      minApprovals,
		CurrentApprovals:  0,
		ExpiresAt:         &expiresAt,
	}

	if err := s.repos.Approvals.Create(ctx, approval); err != nil {
		return nil, fmt.Errorf("create approval: %w", err)
	}

	return approval, nil
}

// Respond records an approval response and evaluates if the threshold is met.
// Rejection by any single approver rejects the whole approval.
// Responses are idempotent — a user cannot approve/reject twice.
func (s *Service) Respond(ctx context.Context, approvalID, approverID, approverName, decision, comment string) (*models.Approval, error) {
	// Get approval
	approval, err := s.repos.Approvals.GetByID(ctx, approvalID)
	if err != nil {
		return nil, fmt.Errorf("approval not found: %w", err)
	}

	// Check approval is still pending
	if approval.Status != "pending" {
		return nil, fmt.Errorf("approval is no longer pending (status: %s)", approval.Status)
	}

	// Check approver hasn't already responded
	responded, err := s.repos.ApprovalResponses.HasResponded(ctx, approvalID, approverID)
	if err != nil {
		return nil, fmt.Errorf("check responded: %w", err)
	}
	if responded {
		return nil, fmt.Errorf("approver has already responded to this approval")
	}

	// Check approver is in required_approvers list
	var requiredApprovers []string
	if err := json.Unmarshal([]byte(approval.RequiredApprovers), &requiredApprovers); err != nil {
		return nil, fmt.Errorf("parse required approvers: %w", err)
	}

	isAuthorized := false
	for _, id := range requiredApprovers {
		if id == approverID {
			isAuthorized = true
			break
		}
	}
	if !isAuthorized {
		return nil, fmt.Errorf("user is not an authorized approver for this request")
	}

	// Record the response
	response := &models.ApprovalResponse{
		ApprovalID:   approvalID,
		ApproverID:   approverID,
		ApproverName: approverName,
		Decision:     decision,
		Comment:      comment,
	}
	if err := s.repos.ApprovalResponses.Create(ctx, response); err != nil {
		return nil, fmt.Errorf("create response: %w", err)
	}

	// Evaluate decision
	if decision == "reject" {
		// Rejection by any single approver rejects the whole approval
		if err := s.repos.Approvals.UpdateStatus(ctx, approvalID, "rejected"); err != nil {
			return nil, fmt.Errorf("update approval status: %w", err)
		}
	} else if decision == "approve" {
		// Increment approval count
		if err := s.repos.Approvals.IncrementApprovals(ctx, approvalID); err != nil {
			return nil, fmt.Errorf("increment approvals: %w", err)
		}
		// Check if threshold is met
		if approval.CurrentApprovals+1 >= approval.MinApprovals {
			if err := s.repos.Approvals.UpdateStatus(ctx, approvalID, "approved"); err != nil {
				return nil, fmt.Errorf("update approval status: %w", err)
			}
		}
	}

	// Fetch and return updated approval
	updated, err := s.repos.Approvals.GetByID(ctx, approvalID)
	if err != nil {
		return nil, fmt.Errorf("fetch updated approval: %w", err)
	}

	return updated, nil
}

// ProtectionRules represents the JSON structure stored in environment.protection_rules.
type ProtectionRules struct {
	RequireApproval bool `json:"require_approval"`
	MinApprovals    int  `json:"min_approvals"`
}

// CheckRequired checks if an environment requires approval for deployment.
func (s *Service) CheckRequired(env *models.Environment) (bool, *ProtectionRules) {
	if env.ProtectionRules == "" || env.ProtectionRules == "{}" {
		return false, nil
	}

	var rules ProtectionRules
	if err := json.Unmarshal([]byte(env.ProtectionRules), &rules); err != nil {
		return false, nil
	}

	return rules.RequireApproval, &rules
}

// GetApprovers parses the required_approvers JSON from an environment.
func (s *Service) GetApprovers(env *models.Environment) []string {
	if env.RequiredApprovers == "" || env.RequiredApprovers == "[]" {
		return nil
	}

	var approvers []string
	if err := json.Unmarshal([]byte(env.RequiredApprovers), &approvers); err != nil {
		return nil
	}

	return approvers
}

// ExpireStale expires pending approvals past their deadline.
func (s *Service) ExpireStale(ctx context.Context) (int, error) {
	return s.repos.Approvals.ExpirePending(ctx)
}
