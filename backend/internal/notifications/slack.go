package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackConfig holds the configuration for a Slack notification channel.
type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel,omitempty"`
	Username   string `json:"username,omitempty"`
}

// SlackNotifier sends notifications to Slack via incoming webhooks.
type SlackNotifier struct {
	config     SlackConfig
	httpClient *http.Client
}

// NewSlackNotifier creates a new Slack notifier.
func NewSlackNotifier(config SlackConfig) *SlackNotifier {
	return &SlackNotifier{
		config:     config,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *SlackNotifier) Type() string { return "slack" }

func (s *SlackNotifier) Send(event *Event) error {
	color := "#36a64f" // green
	icon := ":white_check_mark:"
	if event.Type == EventRunFailure {
		color = "#e01e5a"
		icon = ":x:"
	} else if event.Type == EventRunCancelled {
		color = "#f2c744"
		icon = ":warning:"
	} else if event.Type == EventApproval {
		color = "#2eb886"
		icon = ":hand:"
	}

	text := fmt.Sprintf("%s Pipeline *%s* #%d — *%s*", icon, event.PipelineName, event.RunNumber, event.Status)

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":  color,
				"text":   text,
				"fields": s.buildFields(event),
				"footer": "FlowForge CI/CD",
				"ts":     event.Timestamp.Unix(),
			},
		},
	}

	if s.config.Channel != "" {
		payload["channel"] = s.config.Channel
	}
	if s.config.Username != "" {
		payload["username"] = s.config.Username
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("slack: marshal payload: %w", err)
	}

	resp, err := s.httpClient.Post(s.config.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (s *SlackNotifier) buildFields(event *Event) []map[string]interface{} {
	fields := []map[string]interface{}{
		{"title": "Branch", "value": event.Branch, "short": true},
		{"title": "Author", "value": event.Author, "short": true},
	}
	if event.Duration > 0 {
		fields = append(fields, map[string]interface{}{
			"title": "Duration",
			"value": event.Duration.Round(time.Second).String(),
			"short": true,
		})
	}
	if event.CommitSHA != "" {
		sha := event.CommitSHA
		if len(sha) > 8 {
			sha = sha[:8]
		}
		fields = append(fields, map[string]interface{}{
			"title": "Commit",
			"value": sha,
			"short": true,
		})
	}
	return fields
}
