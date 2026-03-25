package executor

import (
	"context"
	"fmt"
)

// KubernetesExecutor runs steps as Kubernetes Jobs.
// This is currently a placeholder implementation.
type KubernetesExecutor struct{}

// NewKubernetesExecutor creates a new KubernetesExecutor.
func NewKubernetesExecutor() *KubernetesExecutor {
	return &KubernetesExecutor{}
}

// Execute runs the step as a Kubernetes Job.
// Currently returns an error indicating that the Kubernetes executor is not configured.
func (e *KubernetesExecutor) Execute(ctx context.Context, step ExecutionStep) (*ExecutionResult, error) {
	return nil, fmt.Errorf("kubernetes executor not configured: Kubernetes client-go integration is required to run steps as jobs")
}
