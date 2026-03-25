package deploy

import (
	"context"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// StrategyType enum
type StrategyType string

const (
	StrategyRecreate  StrategyType = "recreate"
	StrategyRolling   StrategyType = "rolling"
	StrategyBlueGreen StrategyType = "blue_green"
	StrategyCanary    StrategyType = "canary"
)

// RollingConfig holds rolling-update specific configuration.
type RollingConfig struct {
	MaxSurge       int `json:"max_surge"`       // max extra instances during update (default: 25%)
	MaxUnavailable int `json:"max_unavailable"` // max unavailable instances (default: 25%)
	BatchSize      int `json:"batch_size"`      // number of instances to update at once
}

// BlueGreenConfig holds blue-green specific configuration.
type BlueGreenConfig struct {
	ValidationTimeout int  `json:"validation_timeout"` // seconds to wait for green to be healthy
	AutoPromote       bool `json:"auto_promote"`       // auto switch traffic after health check
}

// CanaryConfig holds canary-specific configuration.
type CanaryConfig struct {
	Steps            []CanaryStep `json:"steps"`             // weight progression steps
	AnalysisDuration int          `json:"analysis_duration"` // seconds per step
	AutoPromote      bool         `json:"auto_promote"`
}

// CanaryStep represents a single step in canary progression.
type CanaryStep struct {
	Weight   int `json:"weight"`   // percentage of traffic (0-100)
	Duration int `json:"duration"` // seconds to hold this weight
}

// Strategy interface defines the contract for all deployment strategies.
type Strategy interface {
	Type() StrategyType
	Plan(ctx context.Context, env *models.Environment, deployment *models.Deployment) (*DeploymentPlan, error)
	Execute(ctx context.Context, plan *DeploymentPlan) error
	Verify(ctx context.Context, deployment *models.Deployment) (*HealthResult, error)
	Rollback(ctx context.Context, deployment *models.Deployment) error
	AdvanceCanary(ctx context.Context, deployment *models.Deployment, weight int) error // only for canary
}

// DeploymentPlan describes what the strategy will do.
type DeploymentPlan struct {
	DeploymentID      string       `json:"deployment_id"`
	EnvironmentID     string       `json:"environment_id"`
	StrategyType      StrategyType `json:"strategy_type"`
	Steps             []DeployStep `json:"steps"`
	TotalSteps        int          `json:"total_steps"`
	EstimatedDuration int          `json:"estimated_duration"` // seconds
}

// DeployStep is a single step in a deployment plan.
type DeployStep struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Order       int    `json:"order"`
}

// HealthResult from verification.
type HealthResult struct {
	Healthy    bool   `json:"healthy"`
	StatusCode int    `json:"status_code"`
	Latency    int    `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
	CheckedAt  string `json:"checked_at"`
}

// NewStrategy factory returns the correct strategy implementation for the given type.
func NewStrategy(strategyType StrategyType) Strategy {
	switch strategyType {
	case StrategyRolling:
		return &RollingStrategy{}
	case StrategyBlueGreen:
		return &BlueGreenStrategy{}
	case StrategyCanary:
		return &CanaryStrategy{}
	default:
		return &RecreateStrategy{}
	}
}
