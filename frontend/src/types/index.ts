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
	triggers: Record<string, unknown> | string;
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
	error_summary?: string;
	deploy_url?: string;
	created_by?: string;
	created_at: string;
}

export interface RunWithMeta extends PipelineRun {
	pipeline_name: string;
	project_id: string;
	project_name: string;
	pipeline_is_active?: boolean;
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
	is_empty?: boolean;
	created_by?: string;
	created_at: string;
	updated_at: string;
}

// --- Environment Variable ---
export interface EnvVar {
	id: string;
	project_id: string;
	key: string;
	value: string;
	created_at: string;
	updated_at: string;
}

// --- Environment ---
export type DeploymentStatus = 'pending' | 'deploying' | 'live' | 'failed' | 'rolled_back';

export type DeployStrategy = 'recreate' | 'rolling' | 'blue_green' | 'canary';

export interface Environment {
	id: string;
	project_id: string;
	name: string;
	slug: string;
	description: string;
	url: string;
	is_production: boolean;
	auto_deploy_branch: string;
	required_approvers: string;
	protection_rules: string;
	deploy_freeze: boolean;
	lock_owner_id: string | null;
	lock_reason: string;
	locked_at: string | null;
	current_deployment_id: string | null;
	strategy: DeployStrategy;
	strategy_config: string;
	health_check_url: string;
	health_check_interval: number;
	health_check_timeout: number;
	health_check_retries: number;
	health_check_path: string;
	health_check_expected_status: number;
	created_at: string;
	updated_at: string;
}

export interface Deployment {
	id: string;
	environment_id: string;
	pipeline_run_id: string | null;
	version: string;
	status: DeploymentStatus;
	commit_sha: string;
	image_tag: string;
	deployed_by: string;
	started_at: string | null;
	finished_at: string | null;
	health_check_status: string;
	rollback_from_id: string | null;
	metadata: string;
	strategy: string;
	canary_weight: number;
	health_check_results: string;
	strategy_state: string;
	created_at: string;
}

export interface EnvOverride {
	id: string;
	environment_id: string;
	key: string;
	value_enc?: string;
	is_secret: boolean;
	created_at: string;
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
	extracted_env_vars: ExtractedVariable[];
	extracted_secrets: ExtractedVariable[];
}

export interface ExtractedVariable {
	name: string;
	type: 'env_var' | 'secret';
	source: string;
	has_value: boolean;
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
	extracted_env_vars?: ExtractedVariable[];
	extracted_secrets?: ExtractedVariable[];
}

export interface ImportCreateProjectResponse {
	project: Project;
	repository: Repository | null;
	pipeline: Pipeline | null;
}

// --- Deploy Strategy Config Types ---
export interface RollingConfig {
	max_surge: number;
	max_unavailable: number;
	batch_size: number;
}

export interface BlueGreenConfig {
	validation_timeout: number;
	auto_promote: boolean;
}

export interface CanaryStep {
	weight: number;
	duration: number;
}

export interface CanaryConfig {
	steps: CanaryStep[];
	analysis_duration: number;
	auto_promote: boolean;
}

export interface HealthResult {
	healthy: boolean;
	status_code: number;
	latency_ms: number;
	error?: string;
	checked_at: string;
}

export interface DeploymentPlan {
	steps: { name: string; description: string; order: number }[];
	total_steps: number;
	estimated_duration: number;
}

// --- Approval ---
export type ApprovalStatus = 'pending' | 'approved' | 'rejected' | 'expired' | 'cancelled';
export type ApprovalType = 'deployment' | 'pipeline_run';

export interface Approval {
	id: string;
	type: ApprovalType;
	deployment_id: string | null;
	pipeline_run_id: string | null;
	environment_id: string | null;
	project_id: string;
	requested_by: string;
	status: ApprovalStatus;
	required_approvers: string;
	min_approvals: number;
	current_approvals: number;
	expires_at: string | null;
	resolved_at: string | null;
	created_at: string;
}

export interface ApprovalResponse {
	id: string;
	approval_id: string;
	approver_id: string;
	approver_name: string;
	decision: 'approve' | 'reject';
	comment: string;
	created_at: string;
}

export interface ApprovalDetail extends Approval {
	responses: ApprovalResponse[];
}

// --- Container Registry ---
export type RegistryType = 'dockerhub' | 'ecr' | 'gcr' | 'acr' | 'harbor' | 'ghcr' | 'generic';

export interface Registry {
	id: string;
	project_id: string;
	name: string;
	type: RegistryType;
	url: string;
	username: string;
	is_default: boolean;
	created_at: string;
	updated_at: string;
}

export interface RegistryImage {
	name: string;
	tags: string[];
	size: number;
	digest: string;
	pushed_at: string;
	pull_count: number;
}

export interface RegistryTag {
	name: string;
	digest: string;
	size: number;
	created_at: string;
}

// --- Pipeline Schedule ---
export interface PipelineSchedule {
	id: string;
	pipeline_id: string;
	project_id: string;
	cron_expression: string;
	timezone: string;
	description: string;
	enabled: boolean;
	branch: string;
	environment_id: string | null;
	variables: string;
	next_run_at: string | null;
	last_run_at: string | null;
	last_run_status: string;
	last_run_id: string | null;
	run_count: number;
	created_by: string;
	created_at: string;
	updated_at: string;
}

// --- Auto-Scaling ---
export interface ScalingPolicy {
	id: string;
	name: string;
	description: string;
	enabled: boolean;
	executor_type: string;
	labels: string;
	min_agents: number;
	max_agents: number;
	desired_agents: number;
	scale_up_threshold: number;
	scale_down_threshold: number;
	scale_up_step: number;
	scale_down_step: number;
	cooldown_seconds: number;
	last_scale_action: string;
	last_scale_at: string | null;
	queue_depth: number;
	active_agents: number;
	created_at: string;
	updated_at: string;
}

export interface ScalingEvent {
	id: string;
	policy_id: string;
	action: 'scale_up' | 'scale_down' | 'no_action';
	from_count: number;
	to_count: number;
	reason: string;
	queue_depth: number;
	active_agents: number;
	created_at: string;
}

export interface ScalingMetrics {
	total_agents: number;
	online_agents: number;
	busy_agents: number;
	queue_depth: number;
	agents_by_executor: Record<string, number>;
	agents_by_label: Record<string, number>;
}

// --- Pipeline Link (Composition) ---
export interface PipelineLink {
	id: string;
	source_pipeline_id: string;
	target_pipeline_id: string;
	link_type: 'trigger' | 'fan_out' | 'fan_in';
	condition: string;
	pass_variables: boolean;
	enabled: boolean;
	created_at: string;
}

// --- Pipeline DAG ---
export interface DAGNode {
	name: string;
	dependencies: string[];
	dependents: string[];
	level: number;
}

export interface PipelineDAG {
	nodes: Record<string, DAGNode>;
	levels: string[][];
	has_cycle: boolean;
}

// --- In-App Notification ---
export type NotificationType = 'info' | 'success' | 'warning' | 'error';
export type NotificationCategory = 'system' | 'pipeline' | 'deployment' | 'approval' | 'agent' | 'security';

export interface InAppNotification {
	id: string;
	user_id: string;
	title: string;
	message: string;
	type: NotificationType;
	category: NotificationCategory;
	link: string;
	is_read: boolean;
	created_at: string;
}

export interface NotificationPreference {
	id: string;
	user_id: string;
	email_enabled: boolean;
	in_app_enabled: boolean;
	pipeline_success: boolean;
	pipeline_failure: boolean;
	deployment_success: boolean;
	deployment_failure: boolean;
	approval_requested: boolean;
	approval_resolved: boolean;
	agent_offline: boolean;
	security_alerts: boolean;
	created_at: string;
	updated_at: string;
}

// --- Dashboard Preference ---
export interface DashboardWidgetConfig {
	id: string;
	visible: boolean;
	size: 'full' | 'half';
	order: number;
}

export interface DashboardPreference {
	id: string;
	user_id: string;
	layout: string; // JSON string of DashboardWidgetConfig[]
	theme: string;
	created_at: string;
	updated_at: string;
}

// --- Global Search ---
export interface SearchResults {
	projects: Project[];
	pipelines: Pipeline[];
	runs: PipelineRun[];
}

// --- Deployment Provider ---
export type DeploymentProviderType = 'aws' | 'gcp' | 'azure' | 'digitalocean' | 'custom';

export interface ProjectDeploymentProvider {
	id: string;
	project_id: string;
	name: string;
	provider_type: string;
	config: Record<string, any>;
	is_active: boolean;
	is_default: boolean;
	capabilities: string;
	created_by: string;
	created_at: string;
	updated_at: string;
}

export interface AWSProviderConfig {
	region: string;
	auth_mode: 'access_key' | 'assume_role' | 'default';
	access_key_id?: string;
	secret_access_key?: string;
	role_arn?: string;
	external_id?: string;
	session_name?: string;
}

export interface CreateDeploymentProviderRequest {
	name: string;
	provider_type: string;
	config: Record<string, any>;
	is_active?: boolean;
}

export interface UpdateDeploymentProviderRequest {
	name?: string;
	provider_type?: string;
	config?: Record<string, any>;
	is_active?: boolean;
}

export interface TestDeploymentProviderResponse {
	success: boolean;
	message: string;
}

// --- Environment Chain ---
export interface ProjectEnvironmentChainEdge {
	id: string;
	project_id: string;
	source_environment_id: string;
	target_environment_id: string;
	position: number;
	is_enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface UpdateEnvironmentChainRequest {
	source_environment_id: string;
	target_environment_id: string;
	position?: number;
}

// --- Pipeline Stage→Environment Mapping ---
export interface PipelineStageEnvironmentMapping {
	id: string;
	project_id: string;
	pipeline_id: string;
	stage_name: string;
	environment_id: string;
	promotion_source_stage: string;
	created_at: string;
	updated_at: string;
}

export interface UpdateStageEnvironmentMappingRequest {
	stage_name: string;
	environment_id: string;
}
