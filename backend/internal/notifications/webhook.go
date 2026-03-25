package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// WebhookConfig holds configuration for a generic outbound webhook.
type WebhookConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Secret  string            `json:"secret,omitempty"`
}

// WebhookNotifier sends notifications as JSON POST requests to a configured URL
// with exponential backoff retry.
type WebhookNotifier struct {
	config     WebhookConfig
	httpClient *http.Client
	maxRetries int
}

// NewWebhookNotifier creates a new generic webhook notifier.
func NewWebhookNotifier(config WebhookConfig) *WebhookNotifier {
	return &WebhookNotifier{
		config:     config,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		maxRetries: 3,
	}
}

func (w *WebhookNotifier) Type() string { return "webhook" }

func (w *WebhookNotifier) Send(event *Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("webhook: marshal: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			log.Debug().Int("attempt", attempt).Dur("delay", delay).Msg("webhook: retrying")
			time.Sleep(delay)
		}

		req, err := http.NewRequest(http.MethodPost, w.config.URL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("webhook: create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "FlowForge-Webhook/1.0")

		for k, v := range w.config.Headers {
			req.Header.Set(k, v)
		}

		resp, err := w.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("webhook: send: %w", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("webhook: unexpected status %d", resp.StatusCode)

		// Don't retry client errors (4xx)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return lastErr
		}
	}

	return fmt.Errorf("webhook: all retries exhausted: %w", lastErr)
}
