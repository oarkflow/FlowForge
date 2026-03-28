// Code generated to match agent.proto. DO NOT EDIT.
// source: agent.proto

package agentpb

// RegisterRequest is sent by agents to register with the server.
type RegisterRequest struct {
	Token    string   `json:"token,omitempty"`
	Name     string   `json:"name,omitempty"`
	Labels   []string `json:"labels,omitempty"`
	Executor string   `json:"executor,omitempty"`
	Os       string   `json:"os,omitempty"`
	Arch     string   `json:"arch,omitempty"`
	CpuCores int32    `json:"cpu_cores,omitempty"`
	MemoryMb int64    `json:"memory_mb,omitempty"`
	Version  string   `json:"version,omitempty"`
}

// RegisterResponse confirms agent registration.
type RegisterResponse struct {
	AgentId                  string `json:"agent_id,omitempty"`
	Accepted                 bool   `json:"accepted,omitempty"`
	Message                  string `json:"message,omitempty"`
	HeartbeatIntervalSeconds int32  `json:"heartbeat_interval_seconds,omitempty"`
}

// HeartbeatRequest is sent periodically by the agent.
type HeartbeatRequest struct {
	AgentId     string  `json:"agent_id,omitempty"`
	ActiveJobs  int32   `json:"active_jobs,omitempty"`
	CpuUsage    float64 `json:"cpu_usage,omitempty"`
	MemoryUsage float64 `json:"memory_usage,omitempty"`
	TimestampMs int64   `json:"timestamp_ms,omitempty"`
}

// HeartbeatResponse is sent by the server in response to heartbeats.
type HeartbeatResponse struct {
	Command string `json:"command,omitempty"`
	Message string `json:"message,omitempty"`
}

// ExecuteJobRequest contains all information needed to execute a job.
type ExecuteJobRequest struct {
	JobRunId      string            `json:"job_run_id,omitempty"`
	PipelineRunId string            `json:"pipeline_run_id,omitempty"`
	ConfigJson    string            `json:"config_json,omitempty"`
	EnvVars       map[string]string `json:"env_vars,omitempty"`
	Secrets       []string          `json:"secrets,omitempty"`
	Artifacts     []*ArtifactRef    `json:"artifacts,omitempty"`
	CloneUrl      string            `json:"clone_url,omitempty"`
	CommitSha     string            `json:"commit_sha,omitempty"`
	Branch        string            `json:"branch,omitempty"`
	ExecutorType  string            `json:"executor_type,omitempty"`
	Image         string            `json:"image,omitempty"`
	Steps         []*StepConfig     `json:"steps,omitempty"`
}

// StepConfig defines a single step within a job.
type StepConfig struct {
	StepRunId       string            `json:"step_run_id,omitempty"`
	Name            string            `json:"name,omitempty"`
	Command         string            `json:"command,omitempty"`
	WorkingDir      string            `json:"working_dir,omitempty"`
	Env             map[string]string `json:"env,omitempty"`
	TimeoutSeconds  int64             `json:"timeout_seconds,omitempty"`
	ContinueOnError bool              `json:"continue_on_error,omitempty"`
	RetryCount      int32             `json:"retry_count,omitempty"`
}

// ArtifactRef references an artifact from a previous step/job.
type ArtifactRef struct {
	ArtifactId  string `json:"artifact_id,omitempty"`
	Name        string `json:"name,omitempty"`
	DownloadUrl string `json:"download_url,omitempty"`
	Path        string `json:"path,omitempty"`
}

// JobEvent is streamed from agent to server during job execution.
// Exactly one of the event fields will be set.
type JobEvent struct {
	// Exactly one of these will be non-nil.
	Log        *LogLine          `json:"log,omitempty"`
	StepStatus *StepStatus       `json:"step_status,omitempty"`
	Complete   *JobComplete      `json:"complete,omitempty"`
	Artifact   *ArtifactUploaded `json:"artifact,omitempty"`
}

// EventType returns a string identifier for the type of event contained.
func (e *JobEvent) EventType() string {
	switch {
	case e.Log != nil:
		return "log"
	case e.StepStatus != nil:
		return "step_status"
	case e.Complete != nil:
		return "complete"
	case e.Artifact != nil:
		return "artifact"
	default:
		return "unknown"
	}
}

// LogLine represents a single line of output from a step.
type LogLine struct {
	StepRunId   string `json:"step_run_id,omitempty"`
	Stream      string `json:"stream,omitempty"`
	Content     string `json:"content,omitempty"`
	TimestampMs int64  `json:"timestamp_ms,omitempty"`
}

// StepStatus reports the status change of a step.
type StepStatus struct {
	StepRunId    string `json:"step_run_id,omitempty"`
	Status       string `json:"status,omitempty"`
	ExitCode     int32  `json:"exit_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	StartedAtMs  int64  `json:"started_at_ms,omitempty"`
	FinishedAtMs int64  `json:"finished_at_ms,omitempty"`
	DurationMs   int64  `json:"duration_ms,omitempty"`
}

// JobComplete reports the final status of a job.
type JobComplete struct {
	JobRunId     string `json:"job_run_id,omitempty"`
	Status       string `json:"status,omitempty"`
	DurationMs   int64  `json:"duration_ms,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// ArtifactUploaded reports that an artifact has been uploaded.
type ArtifactUploaded struct {
	StepRunId      string `json:"step_run_id,omitempty"`
	Name           string `json:"name,omitempty"`
	Path           string `json:"path,omitempty"`
	SizeBytes      int64  `json:"size_bytes,omitempty"`
	ChecksumSha256 string `json:"checksum_sha256,omitempty"`
}

// ReportStatusRequest is the final status report for a job.
type ReportStatusRequest struct {
	AgentId      string        `json:"agent_id,omitempty"`
	JobRunId     string        `json:"job_run_id,omitempty"`
	Status       string        `json:"status,omitempty"`
	DurationMs   int64         `json:"duration_ms,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
	StepResults  []*StepResult `json:"step_results,omitempty"`
}

// StepResult contains the result of a single step execution.
type StepResult struct {
	StepRunId    string `json:"step_run_id,omitempty"`
	Status       string `json:"status,omitempty"`
	ExitCode     int32  `json:"exit_code,omitempty"`
	DurationMs   int64  `json:"duration_ms,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// ReportStatusResponse acknowledges the status report.
type ReportStatusResponse struct {
	Acknowledged bool   `json:"acknowledged,omitempty"`
	Message      string `json:"message,omitempty"`
}
