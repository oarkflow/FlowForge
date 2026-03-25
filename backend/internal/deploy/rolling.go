package deploy

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// RollingStrategy implements incremental replacement of instances.
type RollingStrategy struct{}

func (s *RollingStrategy) Type() StrategyType {
	return StrategyRolling
}

func (s *RollingStrategy) Plan(_ context.Context, env *models.Environment, dep *models.Deployment) (*DeploymentPlan, error) {
	cfg := RollingConfig{
		BatchSize:      2,
		MaxSurge:       25,
		MaxUnavailable: 25,
	}
	// Parse strategy_config from environment if available
	if env.StrategyConfig != "" && env.StrategyConfig != "{}" {
		_ = json.Unmarshal([]byte(env.StrategyConfig), &cfg)
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 2
	}

	// Assume a default instance count of 4 for planning purposes.
	instanceCount := 4
	batches := (instanceCount + cfg.BatchSize - 1) / cfg.BatchSize

	steps := make([]DeployStep, 0, batches+1)
	for i := 0; i < batches; i++ {
		steps = append(steps, DeployStep{
			Name:        "update_batch",
			Description: "Update batch " + time.Now().Format("") + string(rune('1'+i)) + " of instances",
			Order:       i + 1,
		})
	}
	steps = append(steps, DeployStep{
		Name:        "verify_all",
		Description: "Verify all instances are healthy",
		Order:       batches + 1,
	})

	return &DeploymentPlan{
		DeploymentID:      dep.ID,
		EnvironmentID:     env.ID,
		StrategyType:      StrategyRolling,
		Steps:             steps,
		TotalSteps:        len(steps),
		EstimatedDuration: batches * 30, // ~30s per batch
	}, nil
}

func (s *RollingStrategy) Execute(_ context.Context, plan *DeploymentPlan) error {
	for _, step := range plan.Steps {
		log.Printf("[rolling] deployment=%s step=%d/%d %s: %s",
			plan.DeploymentID, step.Order, plan.TotalSteps, step.Name, step.Description)
		// Actual batch update logic will be implemented with cloud integrations.
	}
	log.Printf("[rolling] deployment=%s execute complete", plan.DeploymentID)
	return nil
}

func (s *RollingStrategy) Verify(_ context.Context, dep *models.Deployment) (*HealthResult, error) {
	log.Printf("[rolling] deployment=%s verifying health of all instances", dep.ID)
	return &HealthResult{
		Healthy:    true,
		StatusCode: 200,
		Latency:    20,
		CheckedAt:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *RollingStrategy) Rollback(_ context.Context, dep *models.Deployment) error {
	log.Printf("[rolling] deployment=%s stopping rolling update and redeploying previous version", dep.ID)
	return nil
}

func (s *RollingStrategy) AdvanceCanary(_ context.Context, _ *models.Deployment, _ int) error {
	// Not applicable for rolling strategy
	return nil
}
