package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// PagerDutyConfig holds PagerDuty integration configuration.
type PagerDutyConfig struct {
	RoutingKey string `json:"routing_key"` // Events API v2 routing key
}

// PagerDutyNotifier sends incidents to PagerDuty on pipeline failures.
type PagerDutyNotifier struct {
	config     PagerDutyConfig
	httpClient *http.Client
}

// NewPagerDutyNotifier creates a new PagerDuty notifier.
func NewPagerDutyNotifier(config PagerDutyConfig) *PagerDutyNotifier {
	return &PagerDutyNotifier{
		config:     config,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *PagerDutyNotifier) Type() string { return "pagerduty" }

func (p *PagerDutyNotifier) Send(event *Event) error {
	severity := "info"
	action := "resolve"
	if event.Type == EventRunFailure {
		severity = "critical"
		action = "trigger"
	} else if event.Type == EventRunCancelled {
		severity = "warning"
		action = "trigger"
	}

	payload := map[string]interface{}{
		"routing_key":  p.config.RoutingKey,
		"event_action": action,
		"dedup_key":    fmt.Sprintf("flowforge-%s-%s", event.PipelineID, event.RunID),
		"payload": map[string]interface{}{
			"summary":   fmt.Sprintf("Pipeline %s #%d — %s", event.PipelineName, event.RunNumber, event.Status),
			"source":    "flowforge",
			"severity":  severity,
			"timestamp": event.Timestamp.Format(time.RFC3339),
			"custom_details": map[string]string{
				"pipeline": event.PipelineName,
				"branch":   event.Branch,
				"author":   event.Author,
				"commit":   event.CommitSHA,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("pagerduty: marshal: %w", err)
	}

	resp, err := p.httpClient.Post(
		"https://events.pagerduty.com/v2/enqueue",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("pagerduty: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pagerduty: unexpected status %d", resp.StatusCode)
	}
	return nil
}
