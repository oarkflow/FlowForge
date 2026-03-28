package auth

import (
	"context"
	"encoding/json"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/models"
)

// AuditLogger provides a convenient wrapper around the audit log repository
// for recording user actions throughout the application.
type AuditLogger struct {
	repo *queries.AuditLogRepo
}

// NewAuditLogger creates a new AuditLogger wrapping the given AuditLogRepo.
func NewAuditLogger(repo *queries.AuditLogRepo) *AuditLogger {
	return &AuditLogger{repo: repo}
}

// LogAction records an action in the audit log. The changes parameter is
// serialized to JSON; pass nil if there are no changes to record.
func (a *AuditLogger) LogAction(ctx context.Context, actorID, actorIP, action, resource, resourceID string, changes interface{}) error {
	var changesJSON *string
	if changes != nil {
		data, err := json.Marshal(changes)
		if err != nil {
			return err
		}
		s := string(data)
		changesJSON = &s
	}

	entry := &models.AuditLog{
		ActorID:    strPtr(actorID),
		ActorIP:    strPtr(actorIP),
		Action:     action,
		Resource:   resource,
		ResourceID: strPtr(resourceID),
		Changes:    changesJSON,
	}

	return a.repo.Insert(ctx, entry)
}

// strPtr returns a pointer to s, or nil if s is empty.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
