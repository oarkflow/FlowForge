package notifications

import "time"

// EventType identifies the kind of pipeline event.
type EventType string

const (
	EventRunSuccess  EventType = "run_success"
	EventRunFailure  EventType = "run_failure"
	EventRunCancelled EventType = "run_cancelled"
	EventDeployment  EventType = "deployment"
	EventApproval    EventType = "approval_required"
)

// Event represents a pipeline event that may trigger notifications.
type Event struct {
	Type         EventType         `json:"type"`
	ProjectID    string            `json:"project_id"`
	PipelineID   string            `json:"pipeline_id"`
	PipelineName string            `json:"pipeline_name"`
	RunID        string            `json:"run_id"`
	RunNumber    int               `json:"run_number"`
	Status       string            `json:"status"`
	Branch       string            `json:"branch"`
	CommitSHA    string            `json:"commit_sha"`
	Author       string            `json:"author"`
	Duration     time.Duration     `json:"duration"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
}

// Notifier is the interface that all notification channel types must implement.
type Notifier interface {
	// Send delivers a notification for the given event.
	Send(event *Event) error
	// Type returns the channel type identifier.
	Type() string
}
