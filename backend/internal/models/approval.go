package models

import "time"

type Approval struct {
	ID                string     `db:"id" json:"id"`
	Type              string     `db:"type" json:"type"`
	DeploymentID      *string    `db:"deployment_id" json:"deployment_id"`
	PipelineRunID     *string    `db:"pipeline_run_id" json:"pipeline_run_id"`
	EnvironmentID     *string    `db:"environment_id" json:"environment_id"`
	ProjectID         string     `db:"project_id" json:"project_id"`
	RequestedBy       string     `db:"requested_by" json:"requested_by"`
	Status            string     `db:"status" json:"status"`
	RequiredApprovers string     `db:"required_approvers" json:"required_approvers"`
	MinApprovals      int        `db:"min_approvals" json:"min_approvals"`
	CurrentApprovals  int        `db:"current_approvals" json:"current_approvals"`
	ExpiresAt         *time.Time `db:"expires_at" json:"expires_at"`
	ResolvedAt        *time.Time `db:"resolved_at" json:"resolved_at"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
}

type ApprovalResponse struct {
	ID           string    `db:"id" json:"id"`
	ApprovalID   string    `db:"approval_id" json:"approval_id"`
	ApproverID   string    `db:"approver_id" json:"approver_id"`
	ApproverName string    `db:"approver_name" json:"approver_name"`
	Decision     string    `db:"decision" json:"decision"`
	Comment      string    `db:"comment" json:"comment"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
