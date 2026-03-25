package queries

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/oarkflow/deploy/backend/internal/models"
)

type NotificationPrefRepo struct {
	db *sqlx.DB
}

func (r *NotificationPrefRepo) GetByUser(userID string) (*models.NotificationPreference, error) {
	var pref models.NotificationPreference
	err := r.db.Get(&pref, "SELECT * FROM notification_preferences WHERE user_id = ?", userID)
	if err == sql.ErrNoRows {
		// Return default preferences if none exist
		return &models.NotificationPreference{
			UserID:            userID,
			EmailEnabled:      true,
			InAppEnabled:      true,
			PipelineSuccess:   true,
			PipelineFailure:   true,
			DeploymentSuccess: true,
			DeploymentFailure: true,
			ApprovalRequested: true,
			ApprovalResolved:  true,
			AgentOffline:      true,
			SecurityAlerts:    true,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &pref, nil
}

func (r *NotificationPrefRepo) Upsert(pref *models.NotificationPreference) error {
	_, err := r.db.Exec(`
		INSERT INTO notification_preferences (user_id, email_enabled, in_app_enabled,
			pipeline_success, pipeline_failure, deployment_success, deployment_failure,
			approval_requested, approval_resolved, agent_offline, security_alerts)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			email_enabled = excluded.email_enabled,
			in_app_enabled = excluded.in_app_enabled,
			pipeline_success = excluded.pipeline_success,
			pipeline_failure = excluded.pipeline_failure,
			deployment_success = excluded.deployment_success,
			deployment_failure = excluded.deployment_failure,
			approval_requested = excluded.approval_requested,
			approval_resolved = excluded.approval_resolved,
			agent_offline = excluded.agent_offline,
			security_alerts = excluded.security_alerts,
			updated_at = CURRENT_TIMESTAMP`,
		pref.UserID, pref.EmailEnabled, pref.InAppEnabled,
		pref.PipelineSuccess, pref.PipelineFailure,
		pref.DeploymentSuccess, pref.DeploymentFailure,
		pref.ApprovalRequested, pref.ApprovalResolved,
		pref.AgentOffline, pref.SecurityAlerts)
	return err
}
