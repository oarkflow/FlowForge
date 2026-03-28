package executor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// LocalExecutor runs commands on the local machine using os/exec.
type LocalExecutor struct{}

// NewLocalExecutor creates a new LocalExecutor.
func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{}
}

// Execute runs a command locally and captures stdout/stderr.
func (e *LocalExecutor) Execute(ctx context.Context, step ExecutionStep) (*ExecutionResult, error) {
	return e.ExecuteWithLogs(ctx, step, nil)
}

// ExecuteWithLogs runs a command locally, capturing stdout/stderr and streaming
// output to the logWriter callback in real time.
func (e *LocalExecutor) ExecuteWithLogs(ctx context.Context, step ExecutionStep, logWriter LogWriter) (*ExecutionResult, error) {
	// Apply step-level timeout if set
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, step.Timeout)
		defer cancel()
	}

	// Build the command using shell invocation so that pipes, redirects, and
	// multi-line scripts work correctly.
	cmd := exec.CommandContext(ctx, "sh", "-c", step.Command)

	if step.WorkDir != "" {
		cmd.Dir = step.WorkDir
	}

	// Merge environment variables
	if len(step.Env) > 0 {
		cmd.Env = buildEnv(step.Env)
	}

	// Capture stdout and stderr via pipes for real-time streaming
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	start := time.Now()

	if err := cmd.Start(); err != nil {
		return &ExecutionResult{
			ExitCode: -1,
			Stderr:   err.Error(),
			Duration: time.Since(start),
		}, fmt.Errorf("failed to start command: %w", err)
	}

	// Read stdout and stderr concurrently
	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		streamOutput(stdoutPipe, &stdoutBuf, "stdout", logWriter)
	}()
	go func() {
		defer wg.Done()
		streamOutput(stderrPipe, &stderrBuf, "stderr", logWriter)
	}()

	// Wait for all output to be consumed before waiting on the command
	wg.Wait()

	duration := time.Since(start)

	// Wait for the command to finish
	exitCode := 0
	waitErr := cmd.Wait()
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Context cancelled or other error
			exitCode = -1
		}
	}

	result := &ExecutionResult{
		ExitCode: exitCode,
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		Duration: duration,
	}

	// If the context was cancelled (timeout), return a descriptive error
	if ctx.Err() == context.DeadlineExceeded {
		return result, fmt.Errorf("step %q timed out after %s", step.Name, step.Timeout)
	}
	if ctx.Err() == context.Canceled {
		return result, fmt.Errorf("step %q was cancelled", step.Name)
	}

	return result, nil
}

// streamOutput reads from the given reader line-by-line, writes to buf for
// capturing, and invokes logWriter for real-time streaming.
func streamOutput(r io.Reader, buf *bytes.Buffer, stream string, logWriter LogWriter) {
	scanner := bufio.NewScanner(r)
	// Allow up to 1MB lines for ANSI-heavy output
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		// Write to capture buffer
		buf.Write(line)
		buf.WriteByte('\n')
		// Stream to log writer
		if logWriter != nil {
			// Send the line with a newline so the consumer gets complete lines
			lineWithNL := make([]byte, len(line)+1)
			copy(lineWithNL, line)
			lineWithNL[len(line)] = '\n'
			logWriter(stream, lineWithNL)
		}
	}
}

// buildEnv merges the given env map on top of the current process environment.
func buildEnv(envMap map[string]string) []string {
	// Start from current environment variables
	existing := make(map[string]string)
	for _, e := range envFromOS() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			existing[parts[0]] = parts[1]
		}
	}
	// Override with step env
	for k, v := range envMap {
		existing[k] = v
	}
	// Build final slice
	result := make([]string, 0, len(existing))
	for k, v := range existing {
		result = append(result, k+"="+v)
	}
	return result
}

// envFromOS returns the current process environment. Separated for testability.
func envFromOS() []string {
	return exec.Command("env").Environ()
}
