package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DiscordConfig holds the configuration for a Discord notification channel.
type DiscordConfig struct {
	WebhookURL string `json:"webhook_url"`
}

// DiscordNotifier sends notifications to Discord via webhooks using embeds.
type DiscordNotifier struct {
	config     DiscordConfig
	httpClient *http.Client
}

// NewDiscordNotifier creates a new Discord notifier.
func NewDiscordNotifier(config DiscordConfig) *DiscordNotifier {
	return &DiscordNotifier{
		config:     config,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (d *DiscordNotifier) Type() string { return "discord" }

func (d *DiscordNotifier) Send(event *Event) error {
	color := 3066993 // green
	if event.Type == EventRunFailure {
		color = 15158332 // red
	} else if event.Type == EventRunCancelled {
		color = 16776960 // yellow
	}

	sha := event.CommitSHA
	if len(sha) > 8 {
		sha = sha[:8]
	}

	fields := []map[string]interface{}{
		{"name": "Branch", "value": event.Branch, "inline": true},
		{"name": "Author", "value": event.Author, "inline": true},
		{"name": "Commit", "value": fmt.Sprintf("`%s`", sha), "inline": true},
	}
	if event.Duration > 0 {
		fields = append(fields, map[string]interface{}{
			"name":   "Duration",
			"value":  event.Duration.Round(time.Second).String(),
			"inline": true,
		})
	}

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       fmt.Sprintf("Pipeline %s #%d — %s", event.PipelineName, event.RunNumber, event.Status),
				"color":       color,
				"fields":      fields,
				"footer":      map[string]string{"text": "FlowForge CI/CD"},
				"timestamp":   event.Timestamp.Format(time.RFC3339),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("discord: marshal: %w", err)
	}

	resp, err := d.httpClient.Post(d.config.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord: unexpected status %d", resp.StatusCode)
	}
	return nil
}
