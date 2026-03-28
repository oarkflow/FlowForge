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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/internal/engine/executor"
	"github.com/oarkflow/deploy/backend/proto/agentpb"
)

const agentVersion = "1.0.0"

type agentConfig struct {
	ServerURL string
	GRPCAddr  string // host:port for gRPC server (e.g., "localhost:9090")
	UseGRPC   bool   // --grpc flag enables gRPC mode
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

	transport := "http"
	if cfg.UseGRPC {
		transport = "grpc"
	}

	log.Info().
		Str("server", cfg.ServerURL).
		Str("grpc_addr", cfg.GRPCAddr).
		Str("transport", transport).
		Str("name", cfg.Name).
		Str("executor", cfg.Executor).
		Strs("labels", cfg.Labels).
		Msg("FlowForge Agent starting")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create executor
	exec, err := executor.NewExecutor(cfg.Executor)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create executor")
	}

	if cfg.UseGRPC {
		ga := &GRPCAgent{
			config:   cfg,
			executor: exec,
		}
		ga.run(ctx)
	} else {
		agent := &Agent{
			config:     cfg,
			httpClient: &http.Client{Timeout: 30 * time.Second},
			executor:   exec,
		}
		agent.run(ctx)
	}

	log.Info().Msg("agent stopped")
}

func parseFlags() agentConfig {
	cfg := agentConfig{}

	flag.StringVar(&cfg.ServerURL, "server", envOrDefault("FLOWFORGE_SERVER_URL", "http://localhost:8082"), "FlowForge server URL (HTTP mode)")
	flag.StringVar(&cfg.GRPCAddr, "grpc-addr", envOrDefault("FLOWFORGE_GRPC_ADDR", "localhost:9090"), "gRPC server address (host:port)")
	flag.BoolVar(&cfg.UseGRPC, "grpc", envOrDefault("FLOWFORGE_USE_GRPC", "false") == "true", "Use gRPC transport instead of HTTP")
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

// ============================================================================
// GRPCAgent — gRPC-based agent with bidirectional streaming
// ============================================================================

// GRPCAgent connects to the FlowForge server via gRPC for registration,
// heartbeat, job execution, and log streaming.
type GRPCAgent struct {
	config   agentConfig
	agentID  string
	executor executor.Executor
	client   *agentpb.AgentServiceClient
	conn     *agentpb.ClientConn

	activeJobs atomic.Int32
}

func (ga *GRPCAgent) run(ctx context.Context) {
	// Connect with exponential backoff
	if err := ga.connectWithRetry(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to gRPC server")
	}
	defer ga.conn.Close()

	// Register
	if err := ga.register(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to register via gRPC")
	}

	// Start heartbeat stream
	go ga.heartbeatLoop(ctx)

	// Start job polling/receiving loop
	go ga.jobLoop(ctx)

	log.Info().Str("agent_id", ga.agentID).Str("transport", "grpc").Msg("agent is running")

	// Wait for shutdown
	<-ctx.Done()
	log.Info().Msg("shutting down gRPC agent")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ga.waitForJobs(shutdownCtx)
}

func (ga *GRPCAgent) connectWithRetry(ctx context.Context) error {
	var conn *agentpb.ClientConn
	var err error

	delay := 1 * time.Second
	maxDelay := 30 * time.Second
	maxAttempts := 10

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		log.Info().
			Str("addr", ga.config.GRPCAddr).
			Int("attempt", attempt).
			Msg("connecting to gRPC server")

		conn, err = agentpb.Dial(
			ga.config.GRPCAddr,
			agentpb.WithMaxRetries(3),
			agentpb.WithBaseDelay(1*time.Second),
			agentpb.WithMaxDelay(30*time.Second),
			agentpb.WithConnectTimeout(10*time.Second),
			agentpb.WithOnReconnect(func() {
				log.Info().Msg("gRPC connection re-established, re-registering")
				if ga.agentID != "" {
					if regErr := ga.register(context.Background()); regErr != nil {
						log.Error().Err(regErr).Msg("re-registration failed after reconnect")
					}
				}
			}),
		)
		if err == nil {
			ga.conn = conn
			ga.client = agentpb.NewAgentServiceClient(conn)
			log.Info().Str("addr", ga.config.GRPCAddr).Msg("gRPC connection established")
			return nil
		}

		log.Warn().Err(err).Int("attempt", attempt).Msg("gRPC connection failed, retrying")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}

	return fmt.Errorf("failed to connect after %d attempts: %w", maxAttempts, err)
}

func (ga *GRPCAgent) register(ctx context.Context) error {
	resp, err := ga.client.Register(ctx, &agentpb.RegisterRequest{
		Token:    ga.config.Token,
		Name:     ga.config.Name,
		Labels:   ga.config.Labels,
		Executor: ga.config.Executor,
		Os:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CpuCores: int32(runtime.NumCPU()),
		MemoryMb: 0,
		Version:  agentVersion,
	})
	if err != nil {
		return fmt.Errorf("gRPC register: %w", err)
	}

	if !resp.Accepted {
		return fmt.Errorf("registration rejected: %s", resp.Message)
	}

	ga.agentID = resp.AgentId
	log.Info().
		Str("agent_id", ga.agentID).
		Str("message", resp.Message).
		Int32("heartbeat_interval", resp.HeartbeatIntervalSeconds).
		Msg("registered with server via gRPC")

	return nil
}

func (ga *GRPCAgent) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	recvErrCh := make(chan error, 1)

	startReceiver := func(s *agentpb.HeartbeatClientStream) {
		go func(stream *agentpb.HeartbeatClientStream) {
			for {
				resp, err := stream.Recv()
				if err != nil {
					select {
					case recvErrCh <- err:
					default:
					}
					return
				}
				if resp.Command == "drain" {
					log.Warn().Msg("received drain command from server via gRPC heartbeat")
				}
			}
		}(s)
	}

	reopenStream := func(current *agentpb.HeartbeatClientStream) (*agentpb.HeartbeatClientStream, error) {
		if current != nil {
			current.Close()
		}
		newStream, err := ga.client.Heartbeat(ctx)
		if err != nil {
			return nil, err
		}
		startReceiver(newStream)
		return newStream, nil
	}

	// Open initial heartbeat stream.
	stream, err := reopenStream(nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to open heartbeat stream")
		ga.heartbeatFallback(ctx)
		return
	}
	defer stream.Close()

	// Send heartbeats periodically.
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-recvErrCh:
			log.Warn().Err(err).Msg("heartbeat recv failed, reconnecting stream")
			newStream, newErr := reopenStream(stream)
			if newErr != nil {
				log.Error().Err(newErr).Msg("failed to reopen heartbeat stream")
				ga.heartbeatFallback(ctx)
				return
			}
			stream = newStream
		case <-ticker.C:
			err := stream.Send(&agentpb.HeartbeatRequest{
				AgentId:     ga.agentID,
				ActiveJobs:  ga.activeJobs.Load(),
				CpuUsage:    0.0,
				MemoryUsage: 0.0,
				TimestampMs: time.Now().UnixMilli(),
			})
			if err != nil {
				log.Warn().Err(err).Msg("heartbeat send failed, reconnecting stream")
				newStream, newErr := reopenStream(stream)
				if newErr != nil {
					log.Error().Err(newErr).Msg("failed to reopen heartbeat stream")
					ga.heartbeatFallback(ctx)
					return
				}
				stream = newStream
			}
		}
	}
}

// heartbeatFallback uses the HTTP endpoint if the gRPC stream fails.
func (ga *GRPCAgent) heartbeatFallback(ctx context.Context) {
	log.Warn().Msg("falling back to HTTP heartbeat")
	httpClient := &http.Client{Timeout: 10 * time.Second}
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			payload := map[string]any{
				"agent_id":     ga.agentID,
				"active_jobs":  ga.activeJobs.Load(),
				"cpu_usage":    0.0,
				"memory_usage": 0.0,
				"timestamp_ms": time.Now().UnixMilli(),
			}
			body, _ := json.Marshal(payload)
			url := ga.config.ServerURL + "/api/v1/agents/heartbeat"
			req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
			if err != nil {
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := httpClient.Do(req)
			if err != nil {
				log.Warn().Err(err).Msg("HTTP heartbeat fallback failed")
				continue
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}
}

func (ga *GRPCAgent) jobLoop(ctx context.Context) {
	// In gRPC mode, we still poll via HTTP for job assignment since the
	// server pushes jobs through the dispatcher. The gRPC transport is used
	// for log streaming and event reporting.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	httpClient := &http.Client{Timeout: 30 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ga.pollForJob(ctx, httpClient)
		}
	}
}

func (ga *GRPCAgent) pollForJob(ctx context.Context, httpClient *http.Client) {
	payload := map[string]any{
		"agent_id": ga.agentID,
	}
	body, _ := json.Marshal(payload)

	url := ga.config.ServerURL + "/api/v1/agents/poll"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
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

	go ga.executeJobGRPC(ctx, jobReq.JobRunID, jobReq.Steps)
}

// executeJobGRPC executes a job and streams logs/events back via gRPC.
func (ga *GRPCAgent) executeJobGRPC(ctx context.Context, jobRunID string, steps []struct {
	StepRunID string            `json:"step_run_id"`
	Name      string            `json:"name"`
	Command   string            `json:"command"`
	WorkDir   string            `json:"working_dir"`
	Env       map[string]string `json:"env"`
	Timeout   int64             `json:"timeout_seconds"`
}) {
	ga.activeJobs.Add(1)
	defer ga.activeJobs.Add(-1)

	logger := log.With().Str("job_run_id", jobRunID).Logger()
	logger.Info().Int("steps", len(steps)).Msg("executing job (gRPC mode)")

	jobStatus := "success"
	var jobError string
	start := time.Now()

	var stepResults []*agentpb.StepResult

	for _, step := range steps {
		stepStart := time.Now()
		logger.Info().Str("step", step.Name).Msg("executing step")

		// Report step started via gRPC
		ga.reportStepStatus(ctx, &agentpb.StepStatus{
			StepRunId:   step.StepRunID,
			Status:      "running",
			StartedAtMs: stepStart.UnixMilli(),
		})

		execStep := executor.ExecutionStep{
			Name:    step.Name,
			Command: step.Command,
			WorkDir: step.WorkDir,
			Env:     step.Env,
		}
		if step.Timeout > 0 {
			execStep.Timeout = time.Duration(step.Timeout) * time.Second
		}

		var result *executor.ExecutionResult
		var err error

		if se, ok := ga.executor.(executor.StreamingExecutor); ok {
			logWriter := func(stream string, content []byte) {
				ga.sendLogGRPC(ctx, jobRunID, step.StepRunID, stream, string(content))
			}
			result, err = se.ExecuteWithLogs(ctx, execStep, logWriter)
		} else {
			result, err = ga.executor.Execute(ctx, execStep)
			// Send stdout/stderr as log events after execution
			if result != nil {
				if result.Stdout != "" {
					ga.sendLogGRPC(ctx, jobRunID, step.StepRunID, "stdout", result.Stdout)
				}
				if result.Stderr != "" {
					ga.sendLogGRPC(ctx, jobRunID, step.StepRunID, "stderr", result.Stderr)
				}
			}
		}

		stepDuration := time.Since(stepStart)
		stepStatus := "success"
		exitCode := int32(0)
		var errorMsg string

		if err != nil {
			stepStatus = "failure"
			jobStatus = "failure"
			errorMsg = err.Error()
			jobError = errorMsg
			if result != nil {
				exitCode = int32(result.ExitCode)
			}
		} else if result != nil && result.ExitCode != 0 {
			stepStatus = "failure"
			jobStatus = "failure"
			exitCode = int32(result.ExitCode)
			errorMsg = fmt.Sprintf("step %s exited with code %d", step.Name, exitCode)
			jobError = errorMsg
		}

		// Report step completion via gRPC
		ga.reportStepStatus(ctx, &agentpb.StepStatus{
			StepRunId:    step.StepRunID,
			Status:       stepStatus,
			ExitCode:     exitCode,
			ErrorMessage: errorMsg,
			StartedAtMs:  stepStart.UnixMilli(),
			FinishedAtMs: time.Now().UnixMilli(),
			DurationMs:   stepDuration.Milliseconds(),
		})

		stepResults = append(stepResults, &agentpb.StepResult{
			StepRunId:    step.StepRunID,
			Status:       stepStatus,
			ExitCode:     exitCode,
			DurationMs:   stepDuration.Milliseconds(),
			ErrorMessage: errorMsg,
		})

		if stepStatus == "failure" {
			break
		}
	}

	// Report job completion via gRPC
	duration := time.Since(start)
	_, err := ga.client.ReportStatus(ctx, &agentpb.ReportStatusRequest{
		AgentId:      ga.agentID,
		JobRunId:     jobRunID,
		Status:       jobStatus,
		DurationMs:   duration.Milliseconds(),
		ErrorMessage: jobError,
		StepResults:  stepResults,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to report job status via gRPC, falling back to HTTP")
		ga.reportJobCompleteHTTP(ctx, jobRunID, jobStatus, duration.Milliseconds(), jobError, stepResults)
	}

	logger.Info().
		Str("status", jobStatus).
		Dur("duration", duration).
		Str("transport", "grpc").
		Msg("job execution completed")
}

// sendLogGRPC sends a log line to the server via gRPC ReportStatus (piggybacked)
// or via the HTTP fallback if gRPC is unavailable.
func (ga *GRPCAgent) sendLogGRPC(ctx context.Context, jobRunID, stepRunID, stream, content string) {
	// Use HTTP for log delivery since log streaming in the current gRPC protocol
	// happens on the ExecuteJob server-initiated stream, which the agent doesn't
	// directly control. We use the HTTP log endpoint as a reliable fallback.
	payload := map[string]any{
		"agent_id":    ga.agentID,
		"job_run_id":  jobRunID,
		"step_run_id": stepRunID,
		"stream":      stream,
		"content":     content,
	}
	body, _ := json.Marshal(payload)

	url := ga.config.ServerURL + "/api/v1/agents/log"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

// reportStepStatus sends a step status update to the server.
func (ga *GRPCAgent) reportStepStatus(ctx context.Context, status *agentpb.StepStatus) {
	// Use HTTP for step status updates (compatible with the existing server).
	payload := map[string]any{
		"agent_id":      ga.agentID,
		"step_run_id":   status.StepRunId,
		"status":        status.Status,
		"exit_code":     status.ExitCode,
		"error_message": status.ErrorMessage,
		"started_at_ms": status.StartedAtMs,
		"finished_at_ms": status.FinishedAtMs,
		"duration_ms":   status.DurationMs,
	}
	body, _ := json.Marshal(payload)

	url := ga.config.ServerURL + "/api/v1/agents/log"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

// reportJobCompleteHTTP is a fallback that reports job completion via HTTP.
func (ga *GRPCAgent) reportJobCompleteHTTP(ctx context.Context, jobRunID, status string, durationMS int64, errMsg string, steps []*agentpb.StepResult) {
	httpSteps := make([]map[string]any, len(steps))
	for i, s := range steps {
		httpSteps[i] = map[string]any{
			"step_run_id":   s.StepRunId,
			"status":        s.Status,
			"exit_code":     s.ExitCode,
			"duration_ms":   s.DurationMs,
			"error_message": s.ErrorMessage,
		}
	}

	payload := map[string]any{
		"agent_id":    ga.agentID,
		"job_run_id":  jobRunID,
		"status":      status,
		"duration_ms": durationMS,
		"error":       errMsg,
		"steps":       httpSteps,
	}
	body, _ := json.Marshal(payload)

	url := ga.config.ServerURL + "/api/v1/agents/complete"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		log.Error().Err(err).Msg("failed to create HTTP completion report")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		log.Error().Err(err).Msg("failed to send HTTP completion report")
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func (ga *GRPCAgent) waitForJobs(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if ga.activeJobs.Load() <= 0 {
				return
			}
			log.Info().Int32("active_jobs", ga.activeJobs.Load()).Msg("waiting for active jobs to complete")
		}
	}
}

// ============================================================================
// HTTP Agent — original HTTP-based agent (backward compatible)
// ============================================================================

// Agent represents a FlowForge worker agent using HTTP transport.
type Agent struct {
	config     agentConfig
	agentID    string
	httpClient *http.Client
	executor   executor.Executor
	activeJobs atomic.Int32
}

func (a *Agent) run(ctx context.Context) {
	// Register with server
	if err := a.register(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to register with server")
	}

	// Start heartbeat
	go a.heartbeatLoop(ctx)

	// Start job polling loop
	go a.jobLoop(ctx)

	log.Info().Str("agent_id", a.agentID).Str("transport", "http").Msg("agent is running")

	// Wait for shutdown
	<-ctx.Done()
	log.Info().Msg("shutting down agent")

	// Give in-flight jobs time to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a.shutdown(shutdownCtx)
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
		"active_jobs":  a.activeJobs.Load(),
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
	a.activeJobs.Add(1)
	defer func() { a.activeJobs.Add(-1) }()

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
			if a.activeJobs.Load() <= 0 {
				return
			}
			log.Info().Int32("active_jobs", a.activeJobs.Load()).Msg("waiting for active jobs to complete")
		}
	}
}

// ============================================================================
// Helpers
// ============================================================================

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
