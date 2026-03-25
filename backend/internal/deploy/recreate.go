package deploy

import (
	"context"
	"log"
	"time"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// RecreateStrategy implements a simple stop-and-start deployment.
// All existing instances are torn down before the new version is brought up.
type RecreateStrategy struct{}

func (s *RecreateStrategy) Type() StrategyType {
	return StrategyRecreate
}

func (s *RecreateStrategy) Plan(_ context.Context, env *models.Environment, dep *models.Deployment) (*DeploymentPlan, error) {
	return &DeploymentPlan{
		DeploymentID:  dep.ID,
		EnvironmentID: env.ID,
		StrategyType:  StrategyRecreate,
		Steps: []DeployStep{
			{Name: "stop_current", Description: "Stop current deployment instances", Order: 1},
			{Name: "deploy_new", Description: "Deploy new version", Order: 2},
			{Name: "verify_health", Description: "Verify health of new deployment", Order: 3},
		},
		TotalSteps:        3,
		EstimatedDuration: 60, // ~60 seconds
	}, nil
}

func (s *RecreateStrategy) Execute(_ context.Context, plan *DeploymentPlan) error {
	for _, step := range plan.Steps {
		log.Printf("[recreate] deployment=%s step=%s: %s", plan.DeploymentID, step.Name, step.Description)
		// Actual infrastructure operations (stopping containers, starting new ones)
		// will be implemented in Phase 11 (Cloud Integrations).
		// For now we log the step progression.
	}
	log.Printf("[recreate] deployment=%s execute complete", plan.DeploymentID)
	return nil
}

func (s *RecreateStrategy) Verify(_ context.Context, dep *models.Deployment) (*HealthResult, error) {
	// In production this would use the HealthChecker against the environment's URL.
	// For now return a healthy result as a placeholder.
	log.Printf("[recreate] deployment=%s verifying health", dep.ID)
	return &HealthResult{
		Healthy:    true,
		StatusCode: 200,
		Latency:    15,
		CheckedAt:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *RecreateStrategy) Rollback(_ context.Context, dep *models.Deployment) error {
	log.Printf("[recreate] deployment=%s rolling back — redeploy previous version", dep.ID)
	return nil
}

func (s *RecreateStrategy) AdvanceCanary(_ context.Context, _ *models.Deployment, _ int) error {
	// Not applicable for recreate strategy
	return nil
}
