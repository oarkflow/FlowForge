package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/proto/agentpb"
)

// GRPCAgentServer implements the agentpb.AgentServiceServer interface and
// bridges gRPC-style agent communication to the existing Pool and Dispatcher.
type GRPCAgentServer struct {
	pool       *Pool
	dispatcher *Dispatcher
	db         *sqlx.DB
	hub        LogBroadcaster

	// agentStreams tracks bidirectional connections to gRPC agents so the
	// dispatcher can push jobs to them.
	mu           sync.RWMutex
	agentStreams map[string]*agentConnection
}

// LogBroadcaster is an interface for broadcasting log lines to WebSocket clients.
// This avoids importing the websocket package directly.
type LogBroadcaster interface {
	BroadcastToRun(runID string, content []byte)
}

// agentConnection holds the state for a single gRPC-connected agent.
type agentConnection struct {
	agentID   string
	cancel    context.CancelFunc
	jobCh     chan *agentpb.ExecuteJobRequest
	eventSink chan *agentpb.JobEvent
}

// NewGRPCAgentServer creates a new gRPC agent server.
func NewGRPCAgentServer(pool *Pool, dispatcher *Dispatcher, db *sqlx.DB, hub LogBroadcaster) *GRPCAgentServer {
	return &GRPCAgentServer{
		pool:         pool,
		dispatcher:   dispatcher,
		db:           db,
		hub:          hub,
		agentStreams: make(map[string]*agentConnection),
	}
}

// Register handles agent registration over gRPC.
func (s *GRPCAgentServer) Register(ctx context.Context, req *agentpb.RegisterRequest) (*agentpb.RegisterResponse, error) {
	// Validate token against database
	agentID, err := s.validateAgentToken(ctx, req.Token)
	if err != nil {
		log.Warn().Err(err).Str("name", req.Name).Msg("gRPC agent registration rejected")
		return &agentpb.RegisterResponse{
			Accepted: false,
			Message:  "invalid agent token",
		}, nil
	}

	// Register in pool
	agent := &AgentInfo{
		ID:       agentID,
		Name:     req.Name,
		Labels:   req.Labels,
		Executor: req.Executor,
		OS:       req.Os,
		Arch:     req.Arch,
		CPUCores: req.CpuCores,
		MemoryMB: req.MemoryMb,
		Version:  req.Version,
	}

	if err := s.pool.Register(agent); err != nil {
		log.Error().Err(err).Str("agent_id", agentID).Msg("failed to register gRPC agent")
		return &agentpb.RegisterResponse{
			Accepted: false,
			Message:  "registration failed",
		}, nil
	}

	// Update agent status in database
	s.updateAgentDB(ctx, agentID, "online", req)

	// Register a dispatcher handler for this agent so the dispatcher can push jobs.
	s.registerDispatchHandler(agentID)

	log.Info().
		Str("agent_id", agentID).
		Str("name", req.Name).
		Str("executor", req.Executor).
		Str("transport", "grpc").
		Msg("agent registered via gRPC")

	return &agentpb.RegisterResponse{
		AgentId:                  agentID,
		Accepted:                 true,
		Message:                  "registration successful (gRPC)",
		HeartbeatIntervalSeconds: 10,
	}, nil
}

// Heartbeat handles bidirectional heartbeat streaming.
func (s *GRPCAgentServer) Heartbeat(stream agentpb.HeartbeatStream) error {
	ctx := stream.Context()
	var agentID string

	for {
		select {
		case <-ctx.Done():
			if agentID != "" {
				s.removeAgentStream(agentID)
				log.Info().Str("agent_id", agentID).Msg("gRPC heartbeat stream ended (context cancelled)")
			}
			return ctx.Err()
		default:
		}

		req, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			if agentID != "" {
				s.removeAgentStream(agentID)
			}
			return err
		}

		agentID = req.AgentId

		// Update pool
		if err := s.pool.UpdateHeartbeat(req.AgentId, req.ActiveJobs, req.CpuUsage, req.MemoryUsage); err != nil {
			log.Warn().Err(err).Str("agent_id", req.AgentId).Msg("gRPC heartbeat: agent not in pool")
			if sendErr := stream.Send(&agentpb.HeartbeatResponse{
				Command: "",
				Message: "agent not registered; please re-register",
			}); sendErr != nil {
				return sendErr
			}
			continue
		}

		// Update last_seen in DB
		s.updateAgentLastSeen(ctx, req.AgentId)

		// Check if agent should drain
		var command string
		agent, ok := s.pool.Get(req.AgentId)
		if ok && agent.Status == "draining" {
			command = "drain"
		}

		if err := stream.Send(&agentpb.HeartbeatResponse{
			Command: command,
			Message: "ok",
		}); err != nil {
			s.removeAgentStream(agentID)
			return err
		}
	}

	if agentID != "" {
		s.removeAgentStream(agentID)
		log.Info().Str("agent_id", agentID).Msg("gRPC heartbeat stream closed by agent")
	}
	return nil
}

// ExecuteJob handles the server-side of job execution. The server pushes a job
// to the agent and the agent streams events (logs, status, completion) back.
func (s *GRPCAgentServer) ExecuteJob(ctx context.Context, req *agentpb.ExecuteJobRequest, stream agentpb.JobEventSender) error {
	log.Info().
		Str("job_run_id", req.JobRunId).
		Str("pipeline_run_id", req.PipelineRunId).
		Msg("executing job via gRPC")

	// The server-side implementation receives events from the agent (pushed
	// through the agentConnection.eventSink by the agent binary). For a
	// server-push model, the server sends ExecuteJobRequest to the agent,
	// and the agent calls back with events. This method processes those events.
	//
	// The actual event stream is handled by processJobEvents, which is called
	// when the dispatcher pushes a job to a connected agent.
	return nil
}

// ReportStatus handles the final status report for a job.
func (s *GRPCAgentServer) ReportStatus(ctx context.Context, req *agentpb.ReportStatusRequest) (*agentpb.ReportStatusResponse, error) {
	log.Info().
		Str("agent_id", req.AgentId).
		Str("job_run_id", req.JobRunId).
		Str("status", req.Status).
		Int64("duration_ms", req.DurationMs).
		Msg("gRPC job status report")

	// Decrement active jobs
	s.pool.DecrementActiveJobs(req.AgentId)

	// Update job run in database
	s.updateJobRunStatusFromGRPC(ctx, req)

	return &agentpb.ReportStatusResponse{
		Acknowledged: true,
		Message:      "status recorded",
	}, nil
}

// registerDispatchHandler sets up a JobHandler in the dispatcher for a gRPC agent.
// When the dispatcher assigns a job to this agent, the handler pushes the job
// through the agent's gRPC connection.
func (s *GRPCAgentServer) registerDispatchHandler(agentID string) {
	s.mu.Lock()
	conn := &agentConnection{
		agentID:   agentID,
		jobCh:     make(chan *agentpb.ExecuteJobRequest, 8),
		eventSink: make(chan *agentpb.JobEvent, 256),
	}
	s.agentStreams[agentID] = conn
	s.mu.Unlock()

	s.dispatcher.RegisterHandler(agentID, func(ctx context.Context, aid string, req *JobRequest) error {
		return s.pushJobToAgent(ctx, aid, req)
	})
}

// pushJobToAgent converts a dispatcher JobRequest to a gRPC ExecuteJobRequest
// and queues it for the connected agent.
func (s *GRPCAgentServer) pushJobToAgent(ctx context.Context, agentID string, req *JobRequest) error {
	s.mu.RLock()
	conn, ok := s.agentStreams[agentID]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("agent %s has no active gRPC connection", agentID)
	}

	// Convert dispatcher types to proto types
	protoSteps := make([]*agentpb.StepConfig, len(req.Steps))
	for i, step := range req.Steps {
		protoSteps[i] = &agentpb.StepConfig{
			StepRunId:       step.StepRunID,
			Name:            step.Name,
			Command:         step.Command,
			WorkingDir:      step.WorkingDir,
			Env:             step.Env,
			TimeoutSeconds:  step.TimeoutSeconds,
			ContinueOnError: step.ContinueOnError,
			RetryCount:      step.RetryCount,
		}
	}

	configJSON, _ := json.Marshal(req)

	grpcReq := &agentpb.ExecuteJobRequest{
		JobRunId:      req.JobRunID,
		PipelineRunId: req.PipelineRunID,
		ConfigJson:    string(configJSON),
		EnvVars:       req.EnvVars,
		CloneUrl:      req.CloneURL,
		CommitSha:     req.CommitSHA,
		Branch:        req.Branch,
		ExecutorType:  req.ExecutorType,
		Image:         req.Image,
		Steps:         protoSteps,
	}

	select {
	case conn.jobCh <- grpcReq:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout pushing job to agent %s", agentID)
	}
}

// GetPendingJob retrieves the next job waiting for a gRPC-connected agent.
// Called by the agent's gRPC connection handler.
func (s *GRPCAgentServer) GetPendingJob(agentID string) (*agentpb.ExecuteJobRequest, error) {
	s.mu.RLock()
	conn, ok := s.agentStreams[agentID]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("agent %s not connected via gRPC", agentID)
	}

	select {
	case job := <-conn.jobCh:
		return job, nil
	case <-time.After(30 * time.Second):
		return nil, nil // no job available
	}
}

// ProcessJobEvent handles a single job event from a gRPC agent (log, status, completion).
func (s *GRPCAgentServer) ProcessJobEvent(ctx context.Context, agentID string, event *agentpb.JobEvent) {
	switch {
	case event.Log != nil:
		s.processLogEvent(ctx, event.Log)

	case event.StepStatus != nil:
		s.processStepStatusEvent(ctx, event.StepStatus)

	case event.Complete != nil:
		s.processJobCompleteEvent(ctx, agentID, event.Complete)

	case event.Artifact != nil:
		log.Info().
			Str("step_run_id", event.Artifact.StepRunId).
			Str("name", event.Artifact.Name).
			Int64("size", event.Artifact.SizeBytes).
			Msg("artifact uploaded via gRPC")
	}
}

func (s *GRPCAgentServer) processLogEvent(ctx context.Context, logLine *agentpb.LogLine) {
	// Broadcast to WebSocket clients
	if s.hub != nil {
		logJSON, _ := json.Marshal(map[string]any{
			"type":        "log",
			"step_run_id": logLine.StepRunId,
			"stream":      logLine.Stream,
			"content":     logLine.Content,
			"timestamp":   logLine.TimestampMs,
		})
		// Look up the pipeline run ID from the step_run_id to broadcast to the right room.
		// For now, store directly using the step_run_id lookup.
		if s.db != nil {
			var runID string
			err := s.db.GetContext(ctx, &runID,
				`SELECT jr.run_id FROM step_runs sr
				 JOIN job_runs jr ON sr.job_run_id = jr.id
				 WHERE sr.id = ?`, logLine.StepRunId)
			if err == nil && runID != "" {
				s.hub.BroadcastToRun(runID, logJSON)
			}
		}
	}

	// Persist to database
	if s.db != nil {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO run_logs (run_id, step_run_id, stream, content)
			 SELECT jr.run_id, ?, ?, ?
			 FROM step_runs sr JOIN job_runs jr ON sr.job_run_id = jr.id
			 WHERE sr.id = ?`,
			logLine.StepRunId, logLine.Stream, logLine.Content, logLine.StepRunId)
		if err != nil {
			log.Error().Err(err).Str("step_run_id", logLine.StepRunId).Msg("failed to store gRPC log")
		}
	}
}

func (s *GRPCAgentServer) processStepStatusEvent(ctx context.Context, status *agentpb.StepStatus) {
	log.Debug().
		Str("step_run_id", status.StepRunId).
		Str("status", status.Status).
		Int32("exit_code", status.ExitCode).
		Msg("step status update via gRPC")

	if s.db == nil {
		return
	}

	if status.Status == "running" {
		_, err := s.db.ExecContext(ctx,
			"UPDATE step_runs SET status = ?, started_at = CURRENT_TIMESTAMP WHERE id = ?",
			status.Status, status.StepRunId)
		if err != nil {
			log.Error().Err(err).Str("step_run_id", status.StepRunId).Msg("failed to update step start")
		}
	} else {
		_, err := s.db.ExecContext(ctx,
			`UPDATE step_runs SET status = ?, exit_code = ?, error_message = ?,
			 duration_ms = ?, finished_at = CURRENT_TIMESTAMP WHERE id = ?`,
			status.Status, status.ExitCode, status.ErrorMessage,
			status.DurationMs, status.StepRunId)
		if err != nil {
			log.Error().Err(err).Str("step_run_id", status.StepRunId).Msg("failed to update step status")
		}
	}
}

func (s *GRPCAgentServer) processJobCompleteEvent(ctx context.Context, agentID string, complete *agentpb.JobComplete) {
	log.Info().
		Str("agent_id", agentID).
		Str("job_run_id", complete.JobRunId).
		Str("status", complete.Status).
		Int64("duration_ms", complete.DurationMs).
		Msg("job completed via gRPC")

	// Decrement active jobs
	s.pool.DecrementActiveJobs(agentID)

	if s.db == nil {
		return
	}

	_, err := s.db.ExecContext(ctx,
		"UPDATE job_runs SET status = ?, finished_at = CURRENT_TIMESTAMP WHERE id = ?",
		complete.Status, complete.JobRunId)
	if err != nil {
		log.Error().Err(err).Str("job_run_id", complete.JobRunId).Msg("failed to update job run status")
	}
}

// removeAgentStream cleans up agent connection state.
func (s *GRPCAgentServer) removeAgentStream(agentID string) {
	s.mu.Lock()
	if conn, ok := s.agentStreams[agentID]; ok {
		if conn.cancel != nil {
			conn.cancel()
		}
		close(conn.jobCh)
		delete(s.agentStreams, agentID)
	}
	s.mu.Unlock()

	s.dispatcher.UnregisterHandler(agentID)
	log.Info().Str("agent_id", agentID).Msg("gRPC agent stream removed")
}

// ConnectedAgentCount returns the number of agents connected via gRPC.
func (s *GRPCAgentServer) ConnectedAgentCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.agentStreams)
}

// IsAgentConnected checks if an agent has an active gRPC connection.
func (s *GRPCAgentServer) IsAgentConnected(agentID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.agentStreams[agentID]
	return ok
}

// StartServer starts the gRPC server on the given address.
// Returns the server instance and the listener (for retrieving the actual port).
func (s *GRPCAgentServer) StartServer(addr string) (*agentpb.GRPCServer, net.Listener, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	grpcSrv := agentpb.NewGRPCServer(s)

	go func() {
		if err := grpcSrv.Serve(lis); err != nil {
			log.Error().Err(err).Msg("gRPC server error")
		}
	}()

	log.Info().Str("addr", lis.Addr().String()).Msg("gRPC agent server started")
	return grpcSrv, lis, nil
}

// validateAgentToken checks the agent token against the database.
func (s *GRPCAgentServer) validateAgentToken(ctx context.Context, token string) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database not configured")
	}

	var agent struct {
		ID string `db:"id"`
	}

	err := s.db.GetContext(ctx, &agent,
		"SELECT id FROM agents WHERE token_hash = ? AND status != 'offline'", token)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	return agent.ID, nil
}

// updateAgentDB updates agent information in the database.
func (s *GRPCAgentServer) updateAgentDB(ctx context.Context, agentID, status string, req *agentpb.RegisterRequest) {
	if s.db == nil {
		return
	}

	labelsJSON, _ := json.Marshal(req.Labels)

	_, err := s.db.ExecContext(ctx,
		`UPDATE agents SET status = ?, version = ?, os = ?, arch = ?,
		 cpu_cores = ?, memory_mb = ?, labels = ?, last_seen_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		status, req.Version, req.Os, req.Arch,
		req.CpuCores, req.MemoryMb, string(labelsJSON), agentID)
	if err != nil {
		log.Error().Err(err).Str("agent_id", agentID).Msg("failed to update agent in database")
	}
}

// updateAgentLastSeen updates the last_seen_at timestamp in the database.
func (s *GRPCAgentServer) updateAgentLastSeen(ctx context.Context, agentID string) {
	if s.db == nil {
		return
	}

	_, err := s.db.ExecContext(ctx,
		"UPDATE agents SET last_seen_at = CURRENT_TIMESTAMP WHERE id = ?", agentID)
	if err != nil {
		log.Error().Err(err).Str("agent_id", agentID).Msg("failed to update agent last_seen")
	}
}

// updateJobRunStatusFromGRPC updates job and step status from a gRPC status report.
func (s *GRPCAgentServer) updateJobRunStatusFromGRPC(ctx context.Context, req *agentpb.ReportStatusRequest) {
	if s.db == nil {
		return
	}

	_, err := s.db.ExecContext(ctx,
		"UPDATE job_runs SET status = ?, finished_at = CURRENT_TIMESTAMP WHERE id = ?",
		req.Status, req.JobRunId)
	if err != nil {
		log.Error().Err(err).Str("job_run_id", req.JobRunId).Msg("failed to update job run status")
	}

	for _, step := range req.StepResults {
		_, err := s.db.ExecContext(ctx,
			`UPDATE step_runs SET status = ?, exit_code = ?, duration_ms = ?,
			 error_message = ?, finished_at = CURRENT_TIMESTAMP WHERE id = ?`,
			step.Status, step.ExitCode, step.DurationMs, step.ErrorMessage, step.StepRunId)
		if err != nil {
			log.Error().Err(err).Str("step_run_id", step.StepRunId).Msg("failed to update step run status")
		}
	}
}
