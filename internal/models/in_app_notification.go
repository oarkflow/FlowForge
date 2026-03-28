package models

import "time"

type InAppNotification struct {
	ID        string    `db:"id" json:"id"`
	UserID    string    `db:"user_id" json:"user_id"`
	Title     string    `db:"title" json:"title"`
	Message   string    `db:"message" json:"message"`
	Type      string    `db:"type" json:"type"`           // info, success, warning, error
	Category  string    `db:"category" json:"category"`   // system, pipeline, deployment, approval, agent, security
	Link      string    `db:"link" json:"link"`           // deep link to relevant page
	IsRead    bool      `db:"is_read" json:"is_read"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type NotificationPreference struct {
	ID                string    `db:"id" json:"id"`
	UserID            string    `db:"user_id" json:"user_id"`
	EmailEnabled      bool      `db:"email_enabled" json:"email_enabled"`
	InAppEnabled      bool      `db:"in_app_enabled" json:"in_app_enabled"`
	PipelineSuccess   bool      `db:"pipeline_success" json:"pipeline_success"`
	PipelineFailure   bool      `db:"pipeline_failure" json:"pipeline_failure"`
	DeploymentSuccess bool      `db:"deployment_success" json:"deployment_success"`
	DeploymentFailure bool      `db:"deployment_failure" json:"deployment_failure"`
	ApprovalRequested bool      `db:"approval_requested" json:"approval_requested"`
	ApprovalResolved  bool      `db:"approval_resolved" json:"approval_resolved"`
	AgentOffline      bool      `db:"agent_offline" json:"agent_offline"`
	SecurityAlerts    bool      `db:"security_alerts" json:"security_alerts"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}
