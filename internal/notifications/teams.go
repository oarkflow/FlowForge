package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TeamsConfig holds the configuration for a Microsoft Teams notification channel.
type TeamsConfig struct {
	WebhookURL string `json:"webhook_url"`
}

// TeamsNotifier sends notifications to Microsoft Teams via incoming webhooks
// using Adaptive Cards.
type TeamsNotifier struct {
	config     TeamsConfig
	httpClient *http.Client
}

// NewTeamsNotifier creates a new Teams notifier.
func NewTeamsNotifier(config TeamsConfig) *TeamsNotifier {
	return &TeamsNotifier{
		config:     config,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *TeamsNotifier) Type() string { return "teams" }

func (t *TeamsNotifier) Send(event *Event) error {
	color := "good"
	if event.Type == EventRunFailure {
		color = "attention"
	} else if event.Type == EventRunCancelled {
		color = "warning"
	}

	sha := event.CommitSHA
	if len(sha) > 8 {
		sha = sha[:8]
	}

	// Adaptive Card payload
	card := map[string]interface{}{
		"type":    "message",
		"summary": fmt.Sprintf("Pipeline %s #%d — %s", event.PipelineName, event.RunNumber, event.Status),
		"attachments": []map[string]interface{}{
			{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content": map[string]interface{}{
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
					"type":    "AdaptiveCard",
					"version": "1.4",
					"body": []map[string]interface{}{
						{
							"type":   "TextBlock",
							"size":   "Medium",
							"weight": "Bolder",
							"text":   fmt.Sprintf("Pipeline %s #%d", event.PipelineName, event.RunNumber),
							"color":  color,
						},
						{
							"type": "FactSet",
							"facts": []map[string]string{
								{"title": "Status", "value": event.Status},
								{"title": "Branch", "value": event.Branch},
								{"title": "Author", "value": event.Author},
								{"title": "Commit", "value": sha},
							},
						},
					},
				},
			},
		},
	}

	body, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("teams: marshal: %w", err)
	}

	resp, err := t.httpClient.Post(t.config.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("teams: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("teams: unexpected status %d", resp.StatusCode)
	}
	return nil
}
