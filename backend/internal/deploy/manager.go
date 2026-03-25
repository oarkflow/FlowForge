package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/models"
)

// Manager orchestrates deployment execution using the environment's configured strategy.
type Manager struct {
	repos   *queries.Repositories
	checker *HealthChecker
}

// NewManager creates a new deployment manager.
func NewManager(repos *queries.Repositories) *Manager {
	return &Manager{
		repos:   repos,
		checker: NewHealthChecker(),
	}
}

// Deploy executes a deployment using the environment's configured strategy.
func (m *Manager) Deploy(ctx context.Context, env *models.Environment, deployment *models.Deployment) error {
	strategyType := StrategyType(env.Strategy)
	if strategyType == "" {
		strategyType = StrategyRecreate
	}
	strategy := NewStrategy(strategyType)

	// 1. Create deployment plan
	plan, err := strategy.Plan(ctx, env, deployment)
	if err != nil {
		log.Printf("[manager] deployment=%s plan failed: %v", deployment.ID, err)
		_ = m.repos.Deployments.UpdateStatus(ctx, deployment.ID, "failed")
		return fmt.Errorf("failed to create deployment plan: %w", err)
	}

	// Save the plan as strategy state
	planJSON, _ := json.Marshal(plan)
	_ = m.repos.Deployments.UpdateStrategyState(ctx, deployment.ID, string(planJSON))

	// 2. Update deployment status to "deploying"
	if err := m.repos.Deployments.UpdateStatus(ctx, deployment.ID, "deploying"); err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	// 3. Execute the strategy
	if err := strategy.Execute(ctx, plan); err != nil {
		log.Printf("[manager] deployment=%s execute failed: %v", deployment.ID, err)
		_ = strategy.Rollback(ctx, deployment)
		_ = m.repos.Deployments.UpdateStatus(ctx, deployment.ID, "failed")
		return fmt.Errorf("deployment execution failed: %w", err)
	}

	// 4. Verify health
	result, err := strategy.Verify(ctx, deployment)
	if err != nil {
		log.Printf("[manager] deployment=%s verify error: %v", deployment.ID, err)
		_ = m.repos.Deployments.UpdateStatus(ctx, deployment.ID, "failed")
		return fmt.Errorf("health verification error: %w", err)
	}

	// Record health check result
	m.recordHealthResult(ctx, deployment.ID, result)

	// 5. Update deployment based on result
	if result.Healthy {
		now := time.Now()
		_ = m.repos.Deployments.UpdateStatus(ctx, deployment.ID, "live")
		_ = m.repos.Deployments.SetFinished(ctx, deployment.ID, "live")
		_ = m.repos.Environments.UpdateCurrentDeployment(ctx, env.ID, deployment.ID)
		log.Printf("[manager] deployment=%s is live (finished at %s)", deployment.ID, now.Format(time.RFC3339))
	} else {
		log.Printf("[manager] deployment=%s health check failed, rolling back", deployment.ID)
		_ = strategy.Rollback(ctx, deployment)
		_ = m.repos.Deployments.UpdateStatus(ctx, deployment.ID, "failed")
		_ = m.repos.Deployments.SetFinished(ctx, deployment.ID, "failed")
	}

	return nil
}

// AdvanceCanary advances a canary deployment to the specified weight.
func (m *Manager) AdvanceCanary(ctx context.Context, deploymentID string, weight int) error {
	dep, err := m.repos.Deployments.GetByID(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("deployment not found: %w", err)
	}

	if dep.Strategy != string(StrategyCanary) {
		return fmt.Errorf("deployment is not a canary deployment (strategy=%s)", dep.Strategy)
	}

	if dep.Status != "deploying" && dep.Status != "live" {
		return fmt.Errorf("cannot advance canary — deployment status is %s", dep.Status)
	}

	if weight < 0 || weight > 100 {
		return fmt.Errorf("weight must be between 0 and 100")
	}

	strategy := NewStrategy(StrategyCanary)
	if err := strategy.AdvanceCanary(ctx, dep, weight); err != nil {
		return fmt.Errorf("failed to advance canary: %w", err)
	}

	// Update canary weight in DB
	if err := m.repos.Deployments.UpdateCanaryWeight(ctx, deploymentID, weight); err != nil {
		return fmt.Errorf("failed to update canary weight: %w", err)
	}

	// If weight is 100%, mark as fully promoted
	if weight == 100 {
		_ = m.repos.Deployments.UpdateStatus(ctx, deploymentID, "live")
		_ = m.repos.Deployments.SetFinished(ctx, deploymentID, "live")
		log.Printf("[manager] canary deployment=%s fully promoted (100%%)", deploymentID)
	}

	return nil
}

// RollbackDeployment rolls back to a specific previous deployment.
func (m *Manager) RollbackDeployment(ctx context.Context, env *models.Environment, targetDeploymentID string) error {
	targetDep, err := m.repos.Deployments.GetByID(ctx, targetDeploymentID)
	if err != nil {
		return fmt.Errorf("target deployment not found: %w", err)
	}

	strategyType := StrategyType(env.Strategy)
	if strategyType == "" {
		strategyType = StrategyRecreate
	}
	strategy := NewStrategy(strategyType)

	if err := strategy.Rollback(ctx, targetDep); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	// Mark the current deployment as rolled_back
	if env.CurrentDeploymentID != nil {
		_ = m.repos.Deployments.SetFinished(ctx, *env.CurrentDeploymentID, "rolled_back")
	}

	// Update environment to point to the target deployment
	_ = m.repos.Environments.UpdateCurrentDeployment(ctx, env.ID, targetDeploymentID)

	log.Printf("[manager] environment=%s rolled back to deployment=%s", env.ID, targetDeploymentID)
	return nil
}

// CheckHealth performs a health check against the environment.
func (m *Manager) CheckHealth(ctx context.Context, env *models.Environment, deploymentID string) (*HealthResult, error) {
	result, err := m.checker.CheckWithRetry(ctx, env)
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	// Record the result
	m.recordHealthResult(ctx, deploymentID, result)

	return result, nil
}

// recordHealthResult appends a health result to the deployment's health_check_results.
func (m *Manager) recordHealthResult(ctx context.Context, deploymentID string, result *HealthResult) {
	dep, err := m.repos.Deployments.GetByID(ctx, deploymentID)
	if err != nil {
		return
	}

	var results []HealthResult
	if dep.HealthCheckResults != "" && dep.HealthCheckResults != "[]" {
		_ = json.Unmarshal([]byte(dep.HealthCheckResults), &results)
	}

	results = append(results, *result)

	// Keep only the last 20 results
	if len(results) > 20 {
		results = results[len(results)-20:]
	}

	resultsJSON, _ := json.Marshal(results)
	_ = m.repos.Deployments.UpdateHealthCheckResults(ctx, deploymentID, string(resultsJSON))
}
