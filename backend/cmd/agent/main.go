package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/internal/engine/executor"
)

const agentVersion = "1.0.0"

type agentConfig struct {
	ServerURL string
	Token     string
	Name      string
	Labels    []string
	Executor  string
	LogLevel  string
}

func main() {
	cfg := parseFlags()

	// Setup logging
	lvl, _ := zerolog.ParseLevel(cfg.LogLevel)
	if lvl == zerolog.NoLevel {
		lvl = zerolog.InfoLevel
	}
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		Level(lvl).With().Timestamp().Str("component", "agent").Logger()

	log.Info().
		Str("server", cfg.ServerURL).
		Str("name", cfg.Name).
		Str("executor", cfg.Executor).
		Strs("labels", cfg.Labels).
		Msg("FlowForge Agent starting")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	agent := &Agent{
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	// Create executor
	exec, err := executor.NewExecutor(cfg.Executor)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create executor")
	}
	agent.executor = exec

	// Register with server
	if err := agent.register(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to register with server")
	}

	// Start heartbeat
	go agent.heartbeatLoop(ctx)

	// Start job polling loop
	go agent.jobLoop(ctx)

	log.Info().Str("agent_id", agent.agentID).Msg("agent is running")

	// Wait for shutdown
	<-ctx.Done()
	log.Info().Msg("shutting down agent")

	// Give in-flight jobs time to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	agent.shutdown(shutdownCtx)
	log.Info().Msg("agent stopped")
}

func parseFlags() agentConfig {
	cfg := agentConfig{}

	flag.StringVar(&cfg.ServerURL, "server", envOrDefault("FLOWFORGE_SERVER_URL", "http://localhost:8081"), "FlowForge server URL")
	flag.StringVar(&cfg.Token, "token", envOrDefault("FLOWFORGE_AGENT_TOKEN", ""), "Agent authentication token")
	flag.StringVar(&cfg.Name, "name", envOrDefault("FLOWFORGE_AGENT_NAME", hostname()), "Agent name")
	flag.StringVar(&cfg.Executor, "executor", envOrDefault("FLOWFORGE_AGENT_EXECUTOR", "local"), "Executor type (local, docker, kubernetes)")
	flag.StringVar(&cfg.LogLevel, "log-level", envOrDefault("FLOWFORGE_LOG_LEVEL", "info"), "Log level")

	labelsStr := flag.String("labels", envOrDefault("FLOWFORGE_AGENT_LABELS", ""), "Comma-separated labels")
	flag.Parse()

	if *labelsStr != "" {
		for _, l := range strings.Split(*labelsStr, ",") {
			l = strings.TrimSpace(l)
			if l != "" {
				cfg.Labels = append(cfg.Labels, l)
			}
		}
	}

	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "Error: agent token is required (--token or FLOWFORGE_AGENT_TOKEN)")
		os.Exit(1)
	}

	return cfg
}

// Agent represents a FlowForge worker agent.
type Agent struct {
	config     agentConfig
	agentID    string
	httpClient *http.Client
	executor   executor.Executor
	activeJobs int32
}

// register sends a registration request to the server.
func (a *Agent) register(ctx context.Context) error {
	payload := map[string]any{
		"token":     a.config.Token,
		"name":      a.config.Name,
		"labels":    a.config.Labels,
		"executor":  a.config.Executor,
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
		"cpu_cores": runtime.NumCPU(),
		"memory_mb": 0, // Not easily available without cgo
		"version":   agentVersion,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal register request: %w", err)
	}

	url := a.config.ServerURL + "/api/v1/agents/register"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create register request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("register request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AgentID                  string `json:"agent_id"`
		Accepted                 bool   `json:"accepted"`
		Message                  string `json:"message"`
		HeartbeatIntervalSeconds int    `json:"heartbeat_interval_seconds"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode register response: %w", err)
	}

	if !result.Accepted {
		return fmt.Errorf("registration rejected: %s", result.Message)
	}

	a.agentID = result.AgentID
	log.Info().
		Str("agent_id", a.agentID).
		Str("message", result.Message).
		Msg("registered with server")

	return nil
}

// heartbeatLoop sends periodic heartbeats to the server.
func (a *Agent) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.sendHeartbeat(ctx)
		}
	}
}

func (a *Agent) sendHeartbeat(ctx context.Context) {
	payload := map[string]any{
		"agent_id":     a.agentID,
		"active_jobs":  a.activeJobs,
		"cpu_usage":    0.0,
		"memory_usage": 0.0,
		"timestamp_ms": time.Now().UnixMilli(),
	}

	body, _ := json.Marshal(payload)
	url := a.config.ServerURL + "/api/v1/agents/heartbeat"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		log.Error().Err(err).Msg("failed to create heartbeat request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		log.Warn().Err(err).Msg("heartbeat failed")
		return
	}
	defer resp.Body.Close()

	var result struct {
		Command string `json:"command"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Command == "drain" {
		log.Warn().Msg("received drain command from server")
	}
}

// jobLoop polls for jobs (or waits for push notifications).
func (a *Agent) jobLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.pollForJob(ctx)
		}
	}
}

func (a *Agent) pollForJob(ctx context.Context) {
	payload := map[string]any{
		"agent_id": a.agentID,
	}
	body, _ := json.Marshal(payload)

	url := a.config.ServerURL + "/api/v1/agents/poll"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var result struct {
		Job json.RawMessage `json:"job"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	if result.Job == nil || string(result.Job) == "null" {
		return
	}

	// Parse and execute job
	var jobReq struct {
		JobRunID string `json:"job_run_id"`
		Steps    []struct {
			StepRunID string            `json:"step_run_id"`
			Name      string            `json:"name"`
			Command   string            `json:"command"`
			WorkDir   string            `json:"working_dir"`
			Env       map[string]string `json:"env"`
			Timeout   int64             `json:"timeout_seconds"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(result.Job, &jobReq); err != nil {
		log.Error().Err(err).Msg("failed to parse job")
		return
	}

	go a.executeJob(ctx, jobReq.JobRunID, jobReq.Steps)
}

func (a *Agent) executeJob(ctx context.Context, jobRunID string, steps []struct {
	StepRunID string            `json:"step_run_id"`
	Name      string            `json:"name"`
	Command   string            `json:"command"`
	WorkDir   string            `json:"working_dir"`
	Env       map[string]string `json:"env"`
	Timeout   int64             `json:"timeout_seconds"`
}) {
	a.activeJobs++
	defer func() { a.activeJobs-- }()

	logger := log.With().Str("job_run_id", jobRunID).Logger()
	logger.Info().Int("steps", len(steps)).Msg("executing job")

	jobStatus := "success"
	var jobError string
	start := time.Now()

	var stepResults []map[string]any

	for _, step := range steps {
		stepStart := time.Now()
		logger.Info().Str("step", step.Name).Msg("executing step")

		execStep := executor.ExecutionStep{
			Name:    step.Name,
			Command: step.Command,
			WorkDir: step.WorkDir,
			Env:     step.Env,
		}
		if step.Timeout > 0 {
			execStep.Timeout = time.Duration(step.Timeout) * time.Second
		}

		// Use streaming executor if available
		var result *executor.ExecutionResult
		var err error

		if se, ok := a.executor.(executor.StreamingExecutor); ok {
			logWriter := func(stream string, content []byte) {
				a.sendLog(ctx, jobRunID, step.StepRunID, stream, string(content))
			}
			result, err = se.ExecuteWithLogs(ctx, execStep, logWriter)
		} else {
			result, err = a.executor.Execute(ctx, execStep)
		}

		stepDuration := time.Since(stepStart)
		stepStatus := "success"
		exitCode := 0

		if err != nil {
			stepStatus = "failure"
			jobStatus = "failure"
			jobError = err.Error()
			if result != nil {
				exitCode = result.ExitCode
			}
		} else if result != nil && result.ExitCode != 0 {
			stepStatus = "failure"
			jobStatus = "failure"
			exitCode = result.ExitCode
			jobError = fmt.Sprintf("step %s exited with code %d", step.Name, exitCode)
		}

		stepResults = append(stepResults, map[string]any{
			"step_run_id":   step.StepRunID,
			"status":        stepStatus,
			"exit_code":     exitCode,
			"duration_ms":   stepDuration.Milliseconds(),
			"error_message": jobError,
		})

		if stepStatus == "failure" {
			break
		}
	}

	// Report completion
	duration := time.Since(start)
	a.reportJobComplete(ctx, jobRunID, jobStatus, duration.Milliseconds(), jobError, stepResults)

	logger.Info().
		Str("status", jobStatus).
		Dur("duration", duration).
		Msg("job execution completed")
}

func (a *Agent) sendLog(ctx context.Context, jobRunID, stepRunID, stream, content string) {
	payload := map[string]any{
		"agent_id":    a.agentID,
		"job_run_id":  jobRunID,
		"step_run_id": stepRunID,
		"stream":      stream,
		"content":     content,
	}
	body, _ := json.Marshal(payload)

	url := a.config.ServerURL + "/api/v1/agents/log"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func (a *Agent) reportJobComplete(ctx context.Context, jobRunID, status string, durationMS int64, errMsg string, steps []map[string]any) {
	payload := map[string]any{
		"agent_id":    a.agentID,
		"job_run_id":  jobRunID,
		"status":      status,
		"duration_ms": durationMS,
		"error":       errMsg,
		"steps":       steps,
	}
	body, _ := json.Marshal(payload)

	url := a.config.ServerURL + "/api/v1/agents/complete"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		log.Error().Err(err).Msg("failed to create completion report")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("failed to send completion report")
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func (a *Agent) shutdown(ctx context.Context) {
	// Wait for active jobs to complete
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if a.activeJobs <= 0 {
				return
			}
			log.Info().Int32("active_jobs", a.activeJobs).Msg("waiting for active jobs to complete")
		}
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "agent"
	}
	return h
}
