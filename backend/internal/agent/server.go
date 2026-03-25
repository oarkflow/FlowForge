package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// Server provides HTTP endpoints for agent communication.
// This serves as an alternative to gRPC, using JSON over HTTP for agent registration,
// heartbeat, and job execution.
type Server struct {
	pool       *Pool
	dispatcher *Dispatcher
	db         *sqlx.DB
	mu         sync.RWMutex
}

// NewServer creates a new agent server.
func NewServer(pool *Pool, dispatcher *Dispatcher, db *sqlx.DB) *Server {
	return &Server{
		pool:       pool,
		dispatcher: dispatcher,
		db:         db,
	}
}

// RegisterRequest is the agent registration payload.
type RegisterRequest struct {
	Token    string   `json:"token"`
	Name     string   `json:"name"`
	Labels   []string `json:"labels"`
	Executor string   `json:"executor"`
	OS       string   `json:"os"`
	Arch     string   `json:"arch"`
	CPUCores int32    `json:"cpu_cores"`
	MemoryMB int64    `json:"memory_mb"`
	Version  string   `json:"version"`
}

// RegisterResponse is returned after agent registration.
type RegisterResponse struct {
	AgentID                  string `json:"agent_id"`
	Accepted                 bool   `json:"accepted"`
	Message                  string `json:"message"`
	HeartbeatIntervalSeconds int    `json:"heartbeat_interval_seconds"`
}

// HeartbeatRequest is the agent heartbeat payload.
type HeartbeatPayload struct {
	AgentID     string  `json:"agent_id"`
	ActiveJobs  int32   `json:"active_jobs"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	TimestampMS int64   `json:"timestamp_ms"`
}

// HeartbeatResponse is returned after a heartbeat.
type HeartbeatResponse struct {
	Command string `json:"command"`
	Message string `json:"message"`
}

// HandleRegister processes agent registration requests.
func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate token against database
	agentID, err := s.validateAgentToken(r.Context(), req.Token)
	if err != nil {
		log.Warn().Err(err).Str("name", req.Name).Msg("agent registration rejected")
		writeJSON(w, http.StatusUnauthorized, RegisterResponse{
			Accepted: false,
			Message:  "invalid agent token",
		})
		return
	}

	// Register in pool
	agent := &AgentInfo{
		ID:       agentID,
		Name:     req.Name,
		Labels:   req.Labels,
		Executor: req.Executor,
		OS:       req.OS,
		Arch:     req.Arch,
		CPUCores: req.CPUCores,
		MemoryMB: req.MemoryMB,
		Version:  req.Version,
	}

	if err := s.pool.Register(agent); err != nil {
		log.Error().Err(err).Str("agent_id", agentID).Msg("failed to register agent")
		writeJSON(w, http.StatusInternalServerError, RegisterResponse{
			Accepted: false,
			Message:  "registration failed",
		})
		return
	}

	// Update agent status in database
	s.updateAgentDB(r.Context(), agentID, "online", req)

	log.Info().
		Str("agent_id", agentID).
		Str("name", req.Name).
		Str("executor", req.Executor).
		Msg("agent registered")

	writeJSON(w, http.StatusOK, RegisterResponse{
		AgentID:                  agentID,
		Accepted:                 true,
		Message:                  "registration successful",
		HeartbeatIntervalSeconds: 10,
	})
}

// HandleHeartbeat processes agent heartbeat requests.
func (s *Server) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req HeartbeatPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.pool.UpdateHeartbeat(req.AgentID, req.ActiveJobs, req.CPUUsage, req.MemoryUsage); err != nil {
		writeJSON(w, http.StatusNotFound, HeartbeatResponse{
			Command: "",
			Message: err.Error(),
		})
		return
	}

	// Update last_seen in database
	s.updateAgentLastSeen(r.Context(), req.AgentID)

	// Check if agent should drain
	agent, ok := s.pool.Get(req.AgentID)
	var command string
	if ok && agent.Status == "draining" {
		command = "drain"
	}

	writeJSON(w, http.StatusOK, HeartbeatResponse{
		Command: command,
		Message: "ok",
	})
}

// HandleJobPoll allows agents to poll for available jobs.
func (s *Server) HandleJobPoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	agent, ok := s.pool.Get(payload.AgentID)
	if !ok {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	if agent.Status == "draining" || agent.Status == "offline" {
		writeJSON(w, http.StatusOK, map[string]any{"job": nil, "message": "agent not accepting jobs"})
		return
	}

	// For now, respond with no job (jobs are pushed via dispatcher)
	writeJSON(w, http.StatusOK, map[string]any{"job": nil, "message": "no pending jobs"})
}

// HandleJobComplete processes job completion reports from agents.
type JobCompletePayload struct {
	AgentID    string       `json:"agent_id"`
	JobRunID   string       `json:"job_run_id"`
	Status     string       `json:"status"`
	DurationMS int64        `json:"duration_ms"`
	Error      string       `json:"error,omitempty"`
	Steps      []StepResult `json:"steps,omitempty"`
}

type StepResult struct {
	StepRunID    string `json:"step_run_id"`
	Status       string `json:"status"`
	ExitCode     int    `json:"exit_code"`
	DurationMS   int64  `json:"duration_ms"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func (s *Server) HandleJobComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload JobCompletePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	log.Info().
		Str("agent_id", payload.AgentID).
		Str("job_run_id", payload.JobRunID).
		Str("status", payload.Status).
		Int64("duration_ms", payload.DurationMS).
		Msg("job completed")

	// Decrement active jobs
	s.pool.DecrementActiveJobs(payload.AgentID)

	// Update job status in database
	s.updateJobRunStatus(r.Context(), payload)

	writeJSON(w, http.StatusOK, map[string]any{
		"acknowledged": true,
		"message":      "status recorded",
	})
}

// validateAgentToken checks the agent token against the database.
func (s *Server) validateAgentToken(ctx context.Context, token string) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database not configured")
	}

	var agent struct {
		ID string `db:"id"`
	}

	// Token is stored as a hash, but for simplicity we compare directly
	// In production, use bcrypt or similar
	err := s.db.GetContext(ctx, &agent,
		"SELECT id FROM agents WHERE token_hash = ? AND status != 'offline'", token)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	return agent.ID, nil
}

// updateAgentDB updates agent information in the database.
func (s *Server) updateAgentDB(ctx context.Context, agentID, status string, req RegisterRequest) {
	if s.db == nil {
		return
	}

	labelsJSON, _ := json.Marshal(req.Labels)

	_, err := s.db.ExecContext(ctx,
		`UPDATE agents SET status = ?, version = ?, os = ?, arch = ?,
		 cpu_cores = ?, memory_mb = ?, labels = ?, last_seen_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		status, req.Version, req.OS, req.Arch,
		req.CPUCores, req.MemoryMB, string(labelsJSON), agentID)
	if err != nil {
		log.Error().Err(err).Str("agent_id", agentID).Msg("failed to update agent in database")
	}
}

// updateAgentLastSeen updates the last_seen_at timestamp in the database.
func (s *Server) updateAgentLastSeen(ctx context.Context, agentID string) {
	if s.db == nil {
		return
	}

	_, err := s.db.ExecContext(ctx,
		"UPDATE agents SET last_seen_at = CURRENT_TIMESTAMP WHERE id = ?", agentID)
	if err != nil {
		log.Error().Err(err).Str("agent_id", agentID).Msg("failed to update agent last_seen")
	}
}

// updateJobRunStatus updates the job run status in the database.
func (s *Server) updateJobRunStatus(ctx context.Context, payload JobCompletePayload) {
	if s.db == nil {
		return
	}

	_, err := s.db.ExecContext(ctx,
		`UPDATE job_runs SET status = ?, finished_at = CURRENT_TIMESTAMP WHERE id = ?`,
		payload.Status, payload.JobRunID)
	if err != nil {
		log.Error().Err(err).Str("job_run_id", payload.JobRunID).Msg("failed to update job run status")
	}

	// Update step statuses
	for _, step := range payload.Steps {
		_, err := s.db.ExecContext(ctx,
			`UPDATE step_runs SET status = ?, exit_code = ?, duration_ms = ?,
			 error_message = ?, finished_at = CURRENT_TIMESTAMP WHERE id = ?`,
			step.Status, step.ExitCode, step.DurationMS, step.ErrorMessage, step.StepRunID)
		if err != nil {
			log.Error().Err(err).Str("step_run_id", step.StepRunID).Msg("failed to update step run status")
		}
	}
}

// HandleLog processes log lines sent from agents during job execution.
func (s *Server) HandleLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		AgentID   string `json:"agent_id"`
		JobRunID  string `json:"job_run_id"`
		StepRunID string `json:"step_run_id"`
		Stream    string `json:"stream"`
		Content   string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Store log in database
	if s.db != nil {
		_, err := s.db.ExecContext(r.Context(),
			`INSERT INTO run_logs (run_id, step_run_id, stream, content)
			 SELECT jr.run_id, ?, ?, ?
			 FROM job_runs jr WHERE jr.id = ?`,
			payload.StepRunID, payload.Stream, payload.Content, payload.JobRunID)
		if err != nil {
			log.Error().Err(err).Str("job_run_id", payload.JobRunID).Msg("failed to store agent log")
		}
	}

	w.WriteHeader(http.StatusOK)
}

// writeJSON is a helper to write JSON responses.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
