// ============================================================
// FlowForge Type Definitions
// ============================================================

// --- Auth ---
export interface User {
  id: string;
  email: string;
  username: string;
  display_name?: string;
  avatar_url?: string;
  role: 'owner' | 'admin' | 'developer' | 'viewer';
  totp_enabled: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface AuthTokens {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface LoginRequest {
  email: string;
  password: string;
  totp_code?: string;
}

export interface RegisterRequest {
  email: string;
  username: string;
  password: string;
  display_name?: string;
}

// --- Organization ---
export interface Organization {
  id: string;
  name: string;
  slug: string;
  logo_url?: string;
  created_at: string;
}

export interface OrgMember {
  org_id: string;
  user_id: string;
  role: string;
  joined_at: string;
  user?: User;
}

// --- Project ---
export interface Project {
  id: string;
  org_id?: string;
  name: string;
  slug: string;
  description?: string;
  visibility: 'private' | 'internal' | 'public';
  created_by?: string;
  created_at: string;
  updated_at: string;
}

// --- Repository ---
export type RepositoryProvider = 'github' | 'gitlab' | 'bitbucket' | 'git' | 'local' | 'upload';

export interface Repository {
  id: string;
  project_id: string;
  provider: RepositoryProvider;
  provider_id: string;
  full_name: string;
  clone_url: string;
  ssh_url?: string;
  default_branch: string;
  is_active: boolean;
  last_sync_at?: string;
  created_at: string;
}

// --- Pipeline ---
export interface Pipeline {
  id: string;
  project_id: string;
  repository_id?: string;
  name: string;
  description?: string;
  config_source: 'db' | 'repo';
  config_path?: string;
  config_content?: string;
  config_version: number;
  triggers: Record<string, unknown>;
  is_active: boolean;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface PipelineVersion {
  id: string;
  pipeline_id: string;
  version: number;
  config: string;
  message?: string;
  created_by?: string;
  created_at: string;
}

// --- Pipeline Run ---
export type RunStatus =
  | 'queued'
  | 'pending'
  | 'running'
  | 'success'
  | 'failure'
  | 'cancelled'
  | 'skipped'
  | 'waiting_approval';

export type TriggerType = 'push' | 'pull_request' | 'schedule' | 'manual' | 'api' | 'pipeline';

export interface PipelineRun {
  id: string;
  pipeline_id: string;
  number: number;
  status: RunStatus;
  trigger_type: TriggerType;
  trigger_data?: Record<string, unknown>;
  commit_sha?: string;
  commit_message?: string;
  branch?: string;
  tag?: string;
  author?: string;
  started_at?: string;
  finished_at?: string;
  duration_ms?: number;
  created_by?: string;
  created_at: string;
}

export interface RunWithMeta extends PipelineRun {
  pipeline_name: string;
  project_id: string;
  project_name: string;
}

export interface StageRun {
  id: string;
  run_id: string;
  name: string;
  status: RunStatus;
  position: number;
  started_at?: string;
  finished_at?: string;
}

export interface JobRun {
  id: string;
  stage_run_id: string;
  run_id: string;
  name: string;
  status: RunStatus;
  agent_id?: string;
  executor_type: string;
  started_at?: string;
  finished_at?: string;
}

export interface StepRun {
  id: string;
  job_run_id: string;
  name: string;
  status: RunStatus;
  exit_code?: number;
  error_message?: string;
  started_at?: string;
  finished_at?: string;
  duration_ms?: number;
}

// --- Agent ---
export type AgentStatus = 'online' | 'offline' | 'busy' | 'draining';

export interface Agent {
  id: string;
  name: string;
  labels: string[];
  executor: 'local' | 'docker' | 'kubernetes';
  status: AgentStatus;
  version?: string;
  os?: string;
  arch?: string;
  cpu_cores?: number;
  memory_mb?: number;
  ip_address?: string;
  last_seen_at?: string;
  created_at: string;
}

// --- Secret ---
export interface Secret {
  id: string;
  project_id?: string;
  org_id?: string;
  scope: 'project' | 'org' | 'global';
  key: string;
  masked: boolean;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

// --- Artifact ---
export interface Artifact {
  id: string;
  run_id: string;
  step_run_id?: string;
  name: string;
  path: string;
  size_bytes?: number;
  checksum_sha256?: string;
  storage_backend: string;
  expire_at?: string;
  created_at: string;
}

// --- Notification Channel ---
export interface NotificationChannel {
  id: string;
  project_id?: string;
  type: 'slack' | 'email' | 'teams' | 'discord' | 'pagerduty' | 'webhook';
  name: string;
  is_active: boolean;
  created_at: string;
}

// --- Audit Log ---
export interface AuditLog {
  id: number;
  actor_id?: string;
  actor_ip?: string;
  action: string;
  resource: string;
  resource_id?: string;
  changes?: Record<string, unknown>;
  created_at: string;
}

// --- Log ---
export interface LogLine {
  id?: number;
  run_id: string;
  step_run_id?: string;
  stream: 'stdout' | 'stderr' | 'system';
  content: string;
  ts: string;
}

// --- API ---
export interface ApiError {
  error: string;
  message: string;
  status: number;
  details?: Record<string, string[]>;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface SystemHealth {
  status: 'healthy' | 'degraded' | 'unhealthy';
  version: string;
  uptime_seconds: number;
  database: { status: string };
  agents: { online: number; total: number };
}

// --- Import Wizard ---
export interface ProviderRepo {
  id: string;
  full_name: string;
  description: string;
  clone_url: string;
  ssh_url: string;
  default_branch: string;
  private: boolean;
  updated_at: string;
}

export interface DetectionResult {
  language: string;
  framework: string;
  confidence: number;
  dependency_file: string;
  build_tool: string;
  runtime_version: string;
}

export interface ImportDetectRequest {
  source_type: string;
  git_url?: string;
  ssh_key?: string;
  branch?: string;
  provider?: string;
  repo_owner?: string;
  repo_name?: string;
  local_path?: string;
  upload_id?: string;
}

export interface ImportDetectResponse {
  session_id: string;
  detections: DetectionResult[];
  generated_pipeline: string;
  default_branch: string;
  clone_url: string;
}

export interface ImportCreateProjectRequest {
  session_id: string;
  project: {
    name: string;
    slug: string;
    description: string;
    visibility: string;
    org_id?: string;
  };
  repository: {
    provider: string;
    provider_id?: string;
    full_name: string;
    clone_url: string;
    ssh_url?: string;
    default_branch: string;
  };
  pipeline_yaml: string;
  setup_webhook: boolean;
}

export interface ImportCreateProjectResponse {
  project: Project;
  repository: Repository | null;
  pipeline: Pipeline | null;
}
