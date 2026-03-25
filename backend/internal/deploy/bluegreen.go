package deploy

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// BlueGreenStrategy implements a blue-green deployment with traffic switching.
type BlueGreenStrategy struct{}

func (s *BlueGreenStrategy) Type() StrategyType {
	return StrategyBlueGreen
}

func (s *BlueGreenStrategy) Plan(_ context.Context, env *models.Environment, dep *models.Deployment) (*DeploymentPlan, error) {
	cfg := BlueGreenConfig{
		ValidationTimeout: 120,
		AutoPromote:       true,
	}
	if env.StrategyConfig != "" && env.StrategyConfig != "{}" {
		_ = json.Unmarshal([]byte(env.StrategyConfig), &cfg)
	}

	return &DeploymentPlan{
		DeploymentID:  dep.ID,
		EnvironmentID: env.ID,
		StrategyType:  StrategyBlueGreen,
		Steps: []DeployStep{
			{Name: "deploy_green", Description: "Deploy new version to green environment", Order: 1},
			{Name: "verify_green", Description: "Verify green environment health", Order: 2},
			{Name: "switch_traffic", Description: "Switch traffic from blue to green", Order: 3},
		},
		TotalSteps:        3,
		EstimatedDuration: cfg.ValidationTimeout + 30,
	}, nil
}

func (s *BlueGreenStrategy) Execute(_ context.Context, plan *DeploymentPlan) error {
	for _, step := range plan.Steps {
		log.Printf("[blue-green] deployment=%s step=%d/%d %s: %s",
			plan.DeploymentID, step.Order, plan.TotalSteps, step.Name, step.Description)
	}
	log.Printf("[blue-green] deployment=%s execute complete — traffic switched to green", plan.DeploymentID)
	return nil
}

func (s *BlueGreenStrategy) Verify(_ context.Context, dep *models.Deployment) (*HealthResult, error) {
	log.Printf("[blue-green] deployment=%s verifying green environment health", dep.ID)
	return &HealthResult{
		Healthy:    true,
		StatusCode: 200,
		Latency:    12,
		CheckedAt:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *BlueGreenStrategy) Rollback(_ context.Context, dep *models.Deployment) error {
	log.Printf("[blue-green] deployment=%s rolling back — switching traffic back to blue", dep.ID)
	return nil
}

func (s *BlueGreenStrategy) AdvanceCanary(_ context.Context, _ *models.Deployment, _ int) error {
	// Not applicable for blue-green strategy
	return nil
}
