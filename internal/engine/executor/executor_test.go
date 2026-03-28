package executor

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewExecutor_Supported(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"local"},
		{"docker"},
		{"kubernetes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec, err := NewExecutor(tt.name)
			if err != nil {
				t.Fatal(err)
			}
			if exec == nil {
				t.Error("executor should not be nil")
			}
		})
	}
}

func TestNewExecutor_Unsupported(t *testing.T) {
	_, err := NewExecutor("unknown")
	if err == nil {
		t.Error("should return error for unsupported type")
	}
}

func TestLocalExecutor_Execute_SimpleCommand(t *testing.T) {
	exec := NewLocalExecutor()
	ctx := context.Background()

	result, err := exec.Execute(ctx, ExecutionStep{
		Name:    "echo",
		Command: "echo hello world",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "hello world") {
		t.Errorf("Stdout = %q, want to contain 'hello world'", result.Stdout)
	}
}

func TestLocalExecutor_Execute_FailingCommand(t *testing.T) {
	exec := NewLocalExecutor()
	ctx := context.Background()

	result, err := exec.Execute(ctx, ExecutionStep{
		Name:    "fail",
		Command: "exit 42",
	})
	// err may or may not be set depending on implementation
	_ = err
	if result == nil {
		t.Fatal("result should not be nil for failing command")
	}
	if result.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", result.ExitCode)
	}
}

func TestLocalExecutor_Execute_Stderr(t *testing.T) {
	exec := NewLocalExecutor()
	ctx := context.Background()

	result, _ := exec.Execute(ctx, ExecutionStep{
		Name:    "stderr",
		Command: "echo error_msg >&2",
	})
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if !strings.Contains(result.Stderr, "error_msg") {
		t.Errorf("Stderr = %q, want to contain 'error_msg'", result.Stderr)
	}
}

func TestLocalExecutor_Execute_WithEnv(t *testing.T) {
	exec := NewLocalExecutor()
	ctx := context.Background()

	result, err := exec.Execute(ctx, ExecutionStep{
		Name:    "env-test",
		Command: "echo $MY_TEST_VAR",
		Env:     map[string]string{"MY_TEST_VAR": "hello_from_env"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Stdout, "hello_from_env") {
		t.Errorf("env var not propagated: %q", result.Stdout)
	}
}

func TestLocalExecutor_Execute_WithWorkDir(t *testing.T) {
	exec := NewLocalExecutor()
	ctx := context.Background()

	result, err := exec.Execute(ctx, ExecutionStep{
		Name:    "workdir",
		Command: "pwd",
		WorkDir: "/tmp",
	})
	if err != nil {
		t.Fatal(err)
	}
	// On macOS /tmp symlinks to /private/tmp
	if !strings.Contains(result.Stdout, "tmp") {
		t.Errorf("WorkDir not applied: %q", result.Stdout)
	}
}

func TestLocalExecutor_Execute_Timeout(t *testing.T) {
	exec := NewLocalExecutor()
	ctx := context.Background()

	result, err := exec.Execute(ctx, ExecutionStep{
		Name:    "timeout",
		Command: "sleep 10",
		Timeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Error("should return error for timeout")
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error should mention timeout: %v", err)
	}
}

func TestLocalExecutor_Execute_ContextCancel(t *testing.T) {
	exec := NewLocalExecutor()
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	result, err := exec.Execute(ctx, ExecutionStep{
		Name:    "cancel",
		Command: "sleep 10",
	})
	if err == nil {
		t.Error("should return error for cancelled context")
	}
	_ = result
}

func TestLocalExecutor_ExecuteWithLogs(t *testing.T) {
	exec := NewLocalExecutor()
	ctx := context.Background()

	var logs []string
	var mu sync.Mutex

	logWriter := func(stream string, content []byte) {
		mu.Lock()
		logs = append(logs, stream+": "+strings.TrimSpace(string(content)))
		mu.Unlock()
	}

	result, err := exec.ExecuteWithLogs(ctx, ExecutionStep{
		Name:    "logs",
		Command: "echo line1 && echo line2",
	}, logWriter)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d", result.ExitCode)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(logs) < 2 {
		t.Errorf("expected at least 2 log lines, got %d: %v", len(logs), logs)
	}
}

func TestLocalExecutor_Execute_MultiLineScript(t *testing.T) {
	exec := NewLocalExecutor()
	ctx := context.Background()

	script := `
A=hello
B=world
echo "$A $B"
`
	result, err := exec.Execute(ctx, ExecutionStep{
		Name:    "multiline",
		Command: script,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Stdout, "hello world") {
		t.Errorf("multi-line script output = %q", result.Stdout)
	}
}

func TestLocalExecutor_Execute_Duration(t *testing.T) {
	exec := NewLocalExecutor()
	ctx := context.Background()

	result, _ := exec.Execute(ctx, ExecutionStep{
		Name:    "duration",
		Command: "sleep 0.1",
	})
	if result.Duration < 50*time.Millisecond {
		t.Errorf("Duration = %v, expected at least 50ms", result.Duration)
	}
}

func TestStreamOutput(t *testing.T) {
	input := "line1\nline2\nline3\n"
	reader := strings.NewReader(input)
	var buf bytes.Buffer
	var lines []string
	var mu sync.Mutex

	streamOutput(reader, &buf, "stdout", func(stream string, content []byte) {
		mu.Lock()
		lines = append(lines, strings.TrimSpace(string(content)))
		mu.Unlock()
	})

	if buf.String() != input {
		t.Errorf("buffer = %q, want %q", buf.String(), input)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(lines) != 3 {
		t.Errorf("lines = %d, want 3", len(lines))
	}
}

func TestBuildEnv(t *testing.T) {
	env := buildEnv(map[string]string{
		"CUSTOM_VAR": "custom_value",
	})
	found := false
	for _, e := range env {
		if e == "CUSTOM_VAR=custom_value" {
			found = true
		}
	}
	if !found {
		t.Error("CUSTOM_VAR should be in the resulting environment")
	}
}
