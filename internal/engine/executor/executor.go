package executor

import (
	"context"
	"fmt"
	"time"
)

// ExecutionStep defines a single step to be executed.
type ExecutionStep struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	WorkDir string            `json:"work_dir"`
	Env     map[string]string `json:"env"`
	Timeout time.Duration     `json:"timeout"`
}

// ExecutionResult contains the result of executing a step.
type ExecutionResult struct {
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Duration time.Duration `json:"duration"`
}

// LogWriter is a callback invoked for each line of output during execution.
// stream is "stdout" or "stderr". content is the raw output bytes.
type LogWriter func(stream string, content []byte)

// Executor is the interface all executors must implement.
type Executor interface {
	Execute(ctx context.Context, step ExecutionStep) (*ExecutionResult, error)
}

// StreamingExecutor extends Executor with real-time log output support.
type StreamingExecutor interface {
	Executor
	// ExecuteWithLogs runs the step and sends output in real time via the log writer.
	ExecuteWithLogs(ctx context.Context, step ExecutionStep, logWriter LogWriter) (*ExecutionResult, error)
}

// NewExecutor creates an executor of the given type.
// Supported types: "local", "docker", "kubernetes".
func NewExecutor(executorType string) (Executor, error) {
	switch executorType {
	case "local":
		return NewLocalExecutor(), nil
	case "docker":
		return NewDockerExecutor(), nil
	case "kubernetes":
		return NewKubernetesExecutor(), nil
	default:
		return nil, fmt.Errorf("unsupported executor type: %s", executorType)
	}
}
