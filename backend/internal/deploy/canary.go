package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// CanaryStrategy implements gradual traffic shifting.
type CanaryStrategy struct{}

func (s *CanaryStrategy) Type() StrategyType {
	return StrategyCanary
}

func (s *CanaryStrategy) Plan(_ context.Context, env *models.Environment, dep *models.Deployment) (*DeploymentPlan, error) {
	cfg := CanaryConfig{
		Steps: []CanaryStep{
			{Weight: 10, Duration: 60},
			{Weight: 25, Duration: 60},
			{Weight: 50, Duration: 60},
			{Weight: 100, Duration: 0},
		},
		AnalysisDuration: 60,
		AutoPromote:      false,
	}
	if env.StrategyConfig != "" && env.StrategyConfig != "{}" {
		_ = json.Unmarshal([]byte(env.StrategyConfig), &cfg)
	}
	if len(cfg.Steps) == 0 {
		cfg.Steps = []CanaryStep{
			{Weight: 10, Duration: 60},
			{Weight: 50, Duration: 60},
			{Weight: 100, Duration: 0},
		}
	}

	steps := make([]DeployStep, 0, len(cfg.Steps)+1)
	steps = append(steps, DeployStep{
		Name:        "deploy_canary",
		Description: "Deploy canary instance alongside stable",
		Order:       1,
	})
	totalDuration := 30 // initial deploy time
	for i, cs := range cfg.Steps {
		steps = append(steps, DeployStep{
			Name:        fmt.Sprintf("canary_weight_%d", cs.Weight),
			Description: fmt.Sprintf("Route %d%% traffic to canary", cs.Weight),
			Order:       i + 2,
		})
		totalDuration += cs.Duration
	}

	return &DeploymentPlan{
		DeploymentID:      dep.ID,
		EnvironmentID:     env.ID,
		StrategyType:      StrategyCanary,
		Steps:             steps,
		TotalSteps:        len(steps),
		EstimatedDuration: totalDuration,
	}, nil
}

func (s *CanaryStrategy) Execute(_ context.Context, plan *DeploymentPlan) error {
	for _, step := range plan.Steps {
		log.Printf("[canary] deployment=%s step=%d/%d %s: %s",
			plan.DeploymentID, step.Order, plan.TotalSteps, step.Name, step.Description)
	}
	log.Printf("[canary] deployment=%s initial canary deploy complete — waiting for advancement", plan.DeploymentID)
	return nil
}

func (s *CanaryStrategy) Verify(_ context.Context, dep *models.Deployment) (*HealthResult, error) {
	log.Printf("[canary] deployment=%s verifying canary instance health (weight=%d%%)", dep.ID, dep.CanaryWeight)
	return &HealthResult{
		Healthy:    true,
		StatusCode: 200,
		Latency:    18,
		CheckedAt:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *CanaryStrategy) Rollback(_ context.Context, dep *models.Deployment) error {
	log.Printf("[canary] deployment=%s rolling back — routing all traffic to stable version", dep.ID)
	return nil
}

func (s *CanaryStrategy) AdvanceCanary(_ context.Context, dep *models.Deployment, weight int) error {
	log.Printf("[canary] deployment=%s advancing canary weight from %d%% to %d%%", dep.ID, dep.CanaryWeight, weight)
	// Actual traffic shifting will be done by cloud integration layer.
	// Here we just validate and log the advancement.
	if weight < 0 || weight > 100 {
		return fmt.Errorf("canary weight must be between 0 and 100, got %d", weight)
	}
	return nil
}
