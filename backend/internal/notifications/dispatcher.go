package notifications

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/pkg/crypto"
)

// Dispatcher listens for pipeline events and dispatches notifications to
// configured channels based on notification rules.
type Dispatcher struct {
	repos         *queries.Repositories
	encryptionKey []byte
	bus           *EventBus
}

// NewDispatcher creates a new notification dispatcher.
func NewDispatcher(repos *queries.Repositories, encryptionKey []byte, bus *EventBus) *Dispatcher {
	return &Dispatcher{
		repos:         repos,
		encryptionKey: encryptionKey,
		bus:           bus,
	}
}

// Start subscribes to the event bus and begins dispatching notifications.
// Call this once during application startup.
func (d *Dispatcher) Start(ctx context.Context) {
	d.bus.Subscribe(func(event *Event) {
		d.handleEvent(ctx, event)
	})
	d.bus.Start(ctx)
}

// handleEvent processes a single event by loading notification channels for
// the project and sending to each matching channel.
func (d *Dispatcher) handleEvent(ctx context.Context, event *Event) {
	channels, err := d.repos.Notifications.ListByProject(ctx, event.ProjectID, 100, 0)
	if err != nil {
		log.Error().Err(err).Str("project_id", event.ProjectID).Msg("dispatcher: failed to load notification channels")
		return
	}

	for _, ch := range channels {
		if ch.IsActive == 0 {
			continue
		}

		notifier, err := d.buildNotifier(ch.Type, ch.ConfigEnc)
		if err != nil {
			log.Error().Err(err).Str("channel_id", ch.ID).Str("type", ch.Type).Msg("dispatcher: failed to build notifier")
			continue
		}

		if shouldNotify(event, ch.Type) {
			if err := notifier.Send(event); err != nil {
				log.Error().Err(err).Str("channel_id", ch.ID).Str("type", ch.Type).Msg("dispatcher: send failed")
			} else {
				log.Info().Str("channel_id", ch.ID).Str("type", ch.Type).Str("run_id", event.RunID).Msg("dispatcher: notification sent")
			}
		}
	}
}

// buildNotifier decrypts the channel config and creates the appropriate notifier.
func (d *Dispatcher) buildNotifier(channelType, configEnc string) (Notifier, error) {
	configJSON, err := crypto.Decrypt(d.encryptionKey, configEnc)
	if err != nil {
		return nil, err
	}

	switch channelType {
	case "slack":
		var cfg SlackConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return nil, err
		}
		return NewSlackNotifier(cfg), nil
	case "email":
		var cfg EmailConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return nil, err
		}
		return NewEmailNotifier(cfg), nil
	case "teams":
		var cfg TeamsConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return nil, err
		}
		return NewTeamsNotifier(cfg), nil
	case "discord":
		var cfg DiscordConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return nil, err
		}
		return NewDiscordNotifier(cfg), nil
	case "pagerduty":
		var cfg PagerDutyConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return nil, err
		}
		return NewPagerDutyNotifier(cfg), nil
	case "webhook":
		var cfg WebhookConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return nil, err
		}
		return NewWebhookNotifier(cfg), nil
	default:
		return nil, nil
	}
}

// shouldNotify determines if a notification should be sent based on event type
// and channel type (e.g. PagerDuty only fires on failures).
func shouldNotify(event *Event, channelType string) bool {
	switch channelType {
	case "pagerduty":
		return event.Type == EventRunFailure
	default:
		return true
	}
}
