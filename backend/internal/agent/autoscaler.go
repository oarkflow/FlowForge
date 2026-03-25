package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/models"
	"github.com/rs/zerolog/log"
)

// AutoScaler evaluates scaling policies and determines if agent capacity
// should be adjusted based on job queue depth and workload.
type AutoScaler struct {
	repos      *queries.Repositories
	pool       *Pool
	dispatcher *Dispatcher
}

// NewAutoScaler creates a new AutoScaler instance.
func NewAutoScaler(repos *queries.Repositories, pool *Pool, dispatcher *Dispatcher) *AutoScaler {
	return &AutoScaler{
		repos:      repos,
		pool:       pool,
		dispatcher: dispatcher,
	}
}

// Evaluate checks all enabled scaling policies and determines if scaling actions are needed.
func (a *AutoScaler) Evaluate(ctx context.Context) error {
	policies, err := a.repos.ScalingPolicies.ListEnabledPolicies()
	if err != nil {
		return fmt.Errorf("list enabled policies: %w", err)
	}

	for _, policy := range policies {
		if err := ctx.Err(); err != nil {
			return err
		}
		a.evaluatePolicy(ctx, &policy)
	}
	return nil
}

// evaluatePolicy evaluates a single scaling policy and records the scaling decision.
func (a *AutoScaler) evaluatePolicy(ctx context.Context, policy *models.ScalingPolicy) {
	logger := log.With().
		Str("policy_id", policy.ID).
		Str("policy_name", policy.Name).
		Logger()

	// 1. Get current metrics
	queueDepth := a.dispatcher.QueueDepth()
	matchingAgents := a.pool.CountByLabels(policy.ExecutorType, policy.Labels)

	// 2. Update policy metrics in DB
	if err := a.repos.ScalingPolicies.UpdateMetrics(policy.ID, queueDepth, matchingAgents); err != nil {
		logger.Error().Err(err).Msg("failed to update policy metrics")
	}

	// 3. Check cooldown
	if policy.LastScaleAt != nil {
		cooldownEnd := policy.LastScaleAt.Add(time.Duration(policy.CooldownSeconds) * time.Second)
		if time.Now().Before(cooldownEnd) {
			logger.Debug().
				Time("cooldown_until", cooldownEnd).
				Msg("policy in cooldown, skipping evaluation")
			return
		}
	}

	// 4. Determine scaling action
	var action string
	var newDesired int
	var reason string

	if queueDepth > policy.ScaleUpThreshold && matchingAgents < policy.MaxAgents {
		// Scale UP
		action = "scale_up"
		newDesired = matchingAgents + policy.ScaleUpStep
		if newDesired > policy.MaxAgents {
			newDesired = policy.MaxAgents
		}
		reason = fmt.Sprintf("Queue depth %d exceeds threshold %d", queueDepth, policy.ScaleUpThreshold)
	} else if queueDepth <= policy.ScaleDownThreshold && matchingAgents > policy.MinAgents {
		// Scale DOWN
		action = "scale_down"
		newDesired = matchingAgents - policy.ScaleDownStep
		if newDesired < policy.MinAgents {
			newDesired = policy.MinAgents
		}
		reason = fmt.Sprintf("Queue depth %d at or below threshold %d", queueDepth, policy.ScaleDownThreshold)
	} else {
		action = "no_action"
		newDesired = matchingAgents
	}

	// 5. Record event
	event := &models.ScalingEvent{
		PolicyID:     policy.ID,
		Action:       action,
		FromCount:    matchingAgents,
		ToCount:      newDesired,
		Reason:       reason,
		QueueDepth:   queueDepth,
		ActiveAgents: matchingAgents,
	}
	if err := a.repos.ScalingEvents.Create(event); err != nil {
		logger.Error().Err(err).Msg("failed to record scaling event")
	}

	// 6. Update policy with new desired count and last scale time
	if action != "no_action" {
		if err := a.repos.ScalingPolicies.RecordScaleAction(policy.ID, action, newDesired); err != nil {
			logger.Error().Err(err).Msg("failed to record scale action")
		}

		logger.Info().
			Str("action", action).
			Int("from", matchingAgents).
			Int("to", newDesired).
			Int("queue_depth", queueDepth).
			Str("reason", reason).
			Msg("scaling decision recorded")
	} else {
		logger.Debug().
			Int("queue_depth", queueDepth).
			Int("matching_agents", matchingAgents).
			Msg("no scaling action needed")
	}

	// Note: Actual agent provisioning (launching VMs, K8s pods, etc.) is a placeholder.
	// The auto-scaler records the desired state; external systems or future phases
	// will handle the actual provisioning.
}
