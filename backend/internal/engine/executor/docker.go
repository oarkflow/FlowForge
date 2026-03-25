package executor

import (
	"context"
	"fmt"
)

// DockerExecutor runs steps inside Docker containers.
// This is currently a placeholder implementation.
type DockerExecutor struct{}

// NewDockerExecutor creates a new DockerExecutor.
func NewDockerExecutor() *DockerExecutor {
	return &DockerExecutor{}
}

// Execute runs the step inside a Docker container.
// Currently returns an error indicating that the Docker executor is not configured.
func (e *DockerExecutor) Execute(ctx context.Context, step ExecutionStep) (*ExecutionResult, error) {
	return nil, fmt.Errorf("docker executor not configured: Docker SDK integration is required to run steps in containers")
}
