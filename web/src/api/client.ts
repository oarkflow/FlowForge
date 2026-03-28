import type {
	ApiError, AuthTokens, User, Organization, OrgMember, Project, Repository,
	Pipeline, PipelineVersion, PipelineRun, RunWithMeta, StageRun, JobRun, StepRun,
	Secret, EnvVar, Agent, Artifact, NotificationChannel, AuditLog, LogLine,
	PaginatedResponse, SystemHealth, LoginRequest, RegisterRequest,
	ProviderRepo, ImportDetectRequest, ImportDetectResponse,
	ImportCreateProjectRequest, ImportCreateProjectResponse,
	Environment, Deployment, EnvOverride, HealthResult, DeploymentPlan,
	Registry, RegistryType, RegistryImage, RegistryTag,
	Approval, ApprovalResponse, ApprovalDetail,
	PipelineSchedule,
	ScalingPolicy, ScalingEvent, ScalingMetrics,
	PipelineLink, PipelineDAG,
	InAppNotification, NotificationPreference, DashboardPreference, SearchResults,
	ProjectDeploymentProvider, CreateDeploymentProviderRequest, UpdateDeploymentProviderRequest,
	TestDeploymentProviderResponse,
	ProjectEnvironmentChainEdge, UpdateEnvironmentChainRequest,
	PipelineStageEnvironmentMapping, UpdateStageEnvironmentMappingRequest,
} from '../types';

const API_BASE = '/api/v1';

// ---------------------------------------------------------------------------
// Token helpers
// ---------------------------------------------------------------------------
let accessToken: string | null = localStorage.getItem('ff_access_token');
let refreshToken: string | null = localStorage.getItem('ff_refresh_token');

export function setTokens(tokens: AuthTokens): void {
	accessToken = tokens.access_token;
	refreshToken = tokens.refresh_token;
	localStorage.setItem('ff_access_token', tokens.access_token);
	localStorage.setItem('ff_refresh_token', tokens.refresh_token);
}

export function clearTokens(): void {
	accessToken = null;
	refreshToken = null;
	localStorage.removeItem('ff_access_token');
	localStorage.removeItem('ff_refresh_token');
}

export function getAccessToken(): string | null {
	return accessToken;
}

// ---------------------------------------------------------------------------
// Custom error class
// ---------------------------------------------------------------------------
export class ApiRequestError extends Error {
	status: number;
	details?: Record<string, string[]>;

	constructor(apiError: ApiError) {
		super(apiError.message || apiError.error);
		this.name = 'ApiRequestError';
		this.status = apiError.status;
		this.details = apiError.details;
	}
}

// ---------------------------------------------------------------------------
// Refresh logic
// ---------------------------------------------------------------------------
let refreshPromise: Promise<void> | null = null;

async function doRefresh(): Promise<void> {
	if (!refreshToken) {
		clearTokens();
		throw new ApiRequestError({ error: 'auth', message: 'No refresh token', status: 401 });
	}
	const res = await fetch(`${API_BASE}/auth/refresh`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ refresh_token: refreshToken }),
	});
	if (!res.ok) {
		clearTokens();
		throw new ApiRequestError({ error: 'auth', message: 'Token refresh failed', status: 401 });
	}
	const tokens: AuthTokens = await res.json();
	setTokens(tokens);
}

async function refreshAccessToken(): Promise<void> {
	if (!refreshPromise) {
		refreshPromise = doRefresh().finally(() => { refreshPromise = null; });
	}
	return refreshPromise;
}

// ---------------------------------------------------------------------------
// Core request function
// ---------------------------------------------------------------------------
async function request<T>(
	method: string,
	path: string,
	body?: unknown,
	options: { retry401?: boolean } = { retry401: true },
): Promise<T> {
	const headers: Record<string, string> = {};
	if (body !== undefined && body !== null && !(body instanceof FormData)) {
		headers['Content-Type'] = 'application/json';
	}
	if (accessToken) {
		headers['Authorization'] = `Bearer ${accessToken}`;
	}
	const fetchOptions: RequestInit = { method, headers };
	if (body !== undefined && body !== null) {
		fetchOptions.body = body instanceof FormData ? body : JSON.stringify(body);
	}

	const res = await fetch(`${API_BASE}${path}`, fetchOptions);

	if (res.status === 401 && options.retry401 && refreshToken) {
		await refreshAccessToken();
		return request<T>(method, path, body, { retry401: false });
	}
	if (res.status === 204) return undefined as T;

	const data = await res.json();
	if (!res.ok) {
		throw new ApiRequestError({
			error: data.error || 'unknown',
			message: data.message || res.statusText,
			status: res.status,
			details: data.details,
		});
	}
	return data as T;
}

function normalizePaginatedResponse<T>(
	response: PaginatedResponse<T> | T[],
	params?: Record<string, string>,
): PaginatedResponse<T> {
	if (!Array.isArray(response)) {
		return response;
	}

	const page = Number.parseInt(params?.page ?? '1', 10);
	const requestedPerPage = Number.parseInt(params?.per_page ?? String(response.length || 1), 10);
	const safePage = Number.isFinite(page) && page > 0 ? page : 1;
	const safePerPage = Number.isFinite(requestedPerPage) && requestedPerPage > 0
		? requestedPerPage
		: Math.max(response.length, 1);
	const total = response.length;

	return {
		data: response,
		total,
		page: safePage,
		per_page: safePerPage,
		total_pages: total === 0 ? 0 : Math.ceil(total / safePerPage),
	};
}

// ---------------------------------------------------------------------------
// Request with extra headers (for provider tokens)
// ---------------------------------------------------------------------------
async function requestWithHeaders<T>(
	method: string,
	path: string,
	body?: unknown,
	extraHeaders?: Record<string, string>,
): Promise<T> {
	const headers: Record<string, string> = { ...extraHeaders };
	if (body !== undefined && body !== null && !(body instanceof FormData)) {
		headers['Content-Type'] = 'application/json';
	}
	if (accessToken) {
		headers['Authorization'] = `Bearer ${accessToken}`;
	}
	const fetchOptions: RequestInit = { method, headers };
	if (body !== undefined && body !== null) {
		fetchOptions.body = body instanceof FormData ? body : JSON.stringify(body);
	}
	const res = await fetch(`${API_BASE}${path}`, fetchOptions);
	if (res.status === 204) return undefined as T;
	const data = await res.json();
	if (!res.ok) {
		throw new ApiRequestError({
			error: data.error || 'unknown',
			message: data.message || res.statusText,
			status: res.status,
			details: data.details,
		});
	}
	return data as T;
}

// ---------------------------------------------------------------------------
// Low-level client
// ---------------------------------------------------------------------------
export const apiClient = {
	get<T>(path: string): Promise<T> { return request<T>('GET', path); },
	post<T>(path: string, body?: unknown): Promise<T> { return request<T>('POST', path, body); },
	put<T>(path: string, body?: unknown): Promise<T> { return request<T>('PUT', path, body); },
	delete<T>(path: string): Promise<T> { return request<T>('DELETE', path); },
	upload<T>(path: string, formData: FormData): Promise<T> { return request<T>('POST', path, formData); },
};

// ---------------------------------------------------------------------------
// Run detail (extended)
// ---------------------------------------------------------------------------
export interface RunDetail extends PipelineRun {
	stages: StageRun[];
	jobs: JobRun[];
	steps: StepRun[];
}

// ---------------------------------------------------------------------------
// Typed API — grouped by resource
// ---------------------------------------------------------------------------
export const api = {
	auth: {
		login: (data: LoginRequest) => apiClient.post<AuthTokens>('/auth/login', data),
		register: (data: RegisterRequest) => apiClient.post<AuthTokens>('/auth/register', data),
		refresh: () => apiClient.post<AuthTokens>('/auth/refresh', { refresh_token: refreshToken }),
		logout: () => apiClient.post<void>('/auth/logout'),
		oauthRedirect: (provider: string) => apiClient.get<{ url: string }>(`/auth/oauth/${provider}`),
		totpSetup: () => apiClient.post<{ secret: string; qr_url: string }>('/auth/totp/setup'),
		totpVerify: (code: string) => apiClient.post<void>('/auth/totp/verify', { code }),
	},

	users: {
		me: () => apiClient.get<User>('/users/me'),
		updateMe: (data: Partial<User>) => apiClient.put<User>('/users/me', data),
		get: (id: string) => apiClient.get<User>(`/users/${id}`),
	},

	orgs: {
		list: () => apiClient.get<Organization[]>('/orgs'),
		create: (data: Partial<Organization>) => apiClient.post<Organization>('/orgs', data),
		get: (id: string) => apiClient.get<Organization>(`/orgs/${id}`),
		update: (id: string, data: Partial<Organization>) => apiClient.put<Organization>(`/orgs/${id}`, data),
		delete: (id: string) => apiClient.delete<void>(`/orgs/${id}`),
		listMembers: (id: string) => apiClient.get<OrgMember[]>(`/orgs/${id}/members`),
		addMember: (id: string, userId: string, role: string) =>
			apiClient.post<OrgMember>(`/orgs/${id}/members`, { user_id: userId, role }),
		removeMember: (id: string, userId: string) => apiClient.delete<void>(`/orgs/${id}/members/${userId}`),
	},

	projects: {
		list: (params?: Record<string, string>) => {
			const qs = params ? '?' + new URLSearchParams(params).toString() : '';
			return apiClient
				.get<PaginatedResponse<Project> | Project[]>(`/projects${qs}`)
				.then((response) => normalizePaginatedResponse(response, params));
		},
		create: (data: Partial<Project>) => apiClient.post<Project>('/projects', data),
		get: (id: string) => apiClient.get<Project>(`/projects/${id}`),
		update: (id: string, data: Partial<Project>) => apiClient.put<Project>(`/projects/${id}`, data),
		delete: (id: string) => apiClient.delete<void>(`/projects/${id}`),
	},

	repositories: {
		list: (projectId: string) => apiClient.get<Repository[]>(`/projects/${projectId}/repositories`),
		connect: (projectId: string, data: Partial<Repository>) =>
			apiClient.post<Repository>(`/projects/${projectId}/repositories`, data),
		update: (projectId: string, repoId: string, data: Partial<Repository>) =>
			apiClient.put<Repository>(`/projects/${projectId}/repositories/${repoId}`, data),
		disconnect: (projectId: string, repoId: string) =>
			apiClient.delete<void>(`/projects/${projectId}/repositories/${repoId}`),
		sync: (projectId: string, repoId: string) =>
			apiClient.post<void>(`/projects/${projectId}/repositories/${repoId}/sync`),
	},

	pipelines: {
		list: (projectId: string) => apiClient.get<Pipeline[]>(`/projects/${projectId}/pipelines`),
		create: (projectId: string, data: Partial<Pipeline>) =>
			apiClient.post<Pipeline>(`/projects/${projectId}/pipelines`, data),
		get: (projectId: string, pipelineId: string) =>
			apiClient.get<Pipeline>(`/projects/${projectId}/pipelines/${pipelineId}`),
		update: (projectId: string, pipelineId: string, data: Partial<Pipeline>) =>
			apiClient.put<Pipeline>(`/projects/${projectId}/pipelines/${pipelineId}`, data),
		delete: (projectId: string, pipelineId: string) =>
			apiClient.delete<void>(`/projects/${projectId}/pipelines/${pipelineId}`),
		listVersions: (projectId: string, pipelineId: string) =>
			apiClient.get<PipelineVersion[]>(`/projects/${projectId}/pipelines/${pipelineId}/versions`),
		trigger: (projectId: string, pipelineId: string, data: Record<string, unknown>) =>
			apiClient.post<PipelineRun>(`/projects/${projectId}/pipelines/${pipelineId}/trigger`, data),
		validate: (projectId: string, pipelineId: string, config: string) =>
			apiClient.post<{ valid: boolean; errors?: string[] }>(
				`/projects/${projectId}/pipelines/${pipelineId}/validate`, { config }),
	},

	runs: {
		listAll: (params?: Record<string, string>) => {
			const qs = params ? '?' + new URLSearchParams(params).toString() : '';
			return apiClient
				.get<PaginatedResponse<RunWithMeta> | RunWithMeta[]>(`/runs${qs}`)
				.then((response) => normalizePaginatedResponse(response, params));
		},
		list: (projectId: string, pipelineId: string, params?: Record<string, string>) => {
			const qs = params ? '?' + new URLSearchParams(params).toString() : '';
			return apiClient
				.get<PaginatedResponse<PipelineRun> | PipelineRun[]>(
					`/projects/${projectId}/pipelines/${pipelineId}/runs${qs}`)
				.then((response) => normalizePaginatedResponse(response, params));
		},
		get: (projectId: string, pipelineId: string, runId: string) =>
			apiClient.get<RunDetail>(`/projects/${projectId}/pipelines/${pipelineId}/runs/${runId}`),
		cancel: (projectId: string, pipelineId: string, runId: string) =>
			apiClient.post<void>(`/projects/${projectId}/pipelines/${pipelineId}/runs/${runId}/cancel`),
		rerun: (projectId: string, pipelineId: string, runId: string) =>
			apiClient.post<PipelineRun>(`/projects/${projectId}/pipelines/${pipelineId}/runs/${runId}/rerun`),
		approve: (projectId: string, pipelineId: string, runId: string) =>
			apiClient.post<void>(`/projects/${projectId}/pipelines/${pipelineId}/runs/${runId}/approve`),
		getLogs: (projectId: string, pipelineId: string, runId: string) =>
			apiClient.get<LogLine[]>(`/projects/${projectId}/pipelines/${pipelineId}/runs/${runId}/logs`),
		getArtifacts: (projectId: string, pipelineId: string, runId: string) =>
			apiClient.get<Artifact[]>(`/projects/${projectId}/pipelines/${pipelineId}/runs/${runId}/artifacts`),
	},

	secrets: {
		list: (projectId: string) => apiClient.get<Secret[]>(`/projects/${projectId}/secrets`),
		create: (projectId: string, key: string, value: string) =>
			apiClient.post<Secret>(`/projects/${projectId}/secrets`, { key, value }),
		update: (projectId: string, secretId: string, value: string) =>
			apiClient.put<Secret>(`/projects/${projectId}/secrets/${secretId}`, { value }),
		delete: (projectId: string, secretId: string) =>
			apiClient.delete<void>(`/projects/${projectId}/secrets/${secretId}`),
	},

	envVars: {
		list: (projectId: string) => apiClient.get<EnvVar[]>(`/projects/${projectId}/env-vars`),
		create: (projectId: string, key: string, value: string) =>
			apiClient.post<EnvVar>(`/projects/${projectId}/env-vars`, { key, value }),
		update: (projectId: string, varId: string, key: string, value: string) =>
			apiClient.put<EnvVar>(`/projects/${projectId}/env-vars/${varId}`, { key, value }),
		delete: (projectId: string, varId: string) =>
			apiClient.delete<void>(`/projects/${projectId}/env-vars/${varId}`),
		bulkSave: (projectId: string, vars: { key: string; value: string }[]) =>
			apiClient.put<EnvVar[]>(`/projects/${projectId}/env-vars`, vars),
	},

	agents: {
		list: () => apiClient.get<Agent[]>('/agents'),
		create: (data: Partial<Agent>) => apiClient.post<Agent & { token: string }>('/agents', data),
		get: (id: string) => apiClient.get<Agent>(`/agents/${id}`),
		delete: (id: string) => apiClient.delete<void>(`/agents/${id}`),
		drain: (id: string) => apiClient.post<void>(`/agents/${id}/drain`),
	},

	artifacts: {
		get: (id: string) => apiClient.get<Artifact>(`/artifacts/${id}`),
		downloadUrl: (id: string) => `${API_BASE}/artifacts/${id}/download`,
	},

	notifications: {
		list: (projectId: string) =>
			apiClient.get<NotificationChannel[]>(`/projects/${projectId}/notifications`),
		create: (projectId: string, data: Partial<NotificationChannel> & { config?: Record<string, unknown> }) =>
			apiClient.post<NotificationChannel>(`/projects/${projectId}/notifications`, data),
		update: (projectId: string, channelId: string, data: Partial<NotificationChannel>) =>
			apiClient.put<NotificationChannel>(`/projects/${projectId}/notifications/${channelId}`, data),
		delete: (projectId: string, channelId: string) =>
			apiClient.delete<void>(`/projects/${projectId}/notifications/${channelId}`),
	},

	auditLogs: {
		list: (params?: Record<string, string>) => {
			const qs = params ? '?' + new URLSearchParams(params).toString() : '';
			return apiClient
				.get<PaginatedResponse<AuditLog> | AuditLog[]>(`/audit-logs${qs}`)
				.then((response) => normalizePaginatedResponse(response, params));
		},
	},

	system: {
		health: () => apiClient.get<SystemHealth>('/system/health'),
		metrics: () => apiClient.get<Record<string, unknown>>('/system/metrics'),
		info: () => apiClient.get<Record<string, unknown>>('/system/info'),
	},

	import: {
		detect: (data: ImportDetectRequest, providerToken?: string) => {
			if (providerToken) {
				return requestWithHeaders<ImportDetectResponse>('POST', '/import/detect', data, {
					'X-Provider-Token': providerToken,
				});
			}
			return apiClient.post<ImportDetectResponse>('/import/detect', data);
		},
		upload: (formData: FormData) =>
			apiClient.upload<{ upload_id: string; filename: string }>('/import/upload', formData),
		listRepos: (provider: string, params: { search?: string; page?: number; per_page?: number }, providerToken: string) => {
			const qs = new URLSearchParams();
			if (params.search) qs.set('search', params.search);
			if (params.page) qs.set('page', String(params.page));
			if (params.per_page) qs.set('per_page', String(params.per_page));
			const query = qs.toString() ? '?' + qs.toString() : '';
			return requestWithHeaders<{ repos: ProviderRepo[]; total: number; page: number }>(
				'GET', `/import/providers/${provider}/repos${query}`, undefined, {
				'X-Provider-Token': providerToken,
			});
		},
		createProject: (data: ImportCreateProjectRequest, providerToken?: string) => {
			if (providerToken) {
				return requestWithHeaders<ImportCreateProjectResponse>('POST', '/import/project', data, {
					'X-Provider-Token': providerToken,
				});
			}
			return apiClient.post<ImportCreateProjectResponse>('/import/project', data);
		},
	},

	environments: {
		list: (projectId: string) =>
			apiClient.get<Environment[]>(`/projects/${projectId}/environments`),
		create: (projectId: string, data: Partial<Environment>) =>
			apiClient.post<Environment>(`/projects/${projectId}/environments`, data),
		update: (projectId: string, envId: string, data: Partial<Environment>) =>
			apiClient.put<Environment>(`/projects/${projectId}/environments/${envId}`, data),
		delete: (projectId: string, envId: string) =>
			apiClient.delete<void>(`/projects/${projectId}/environments/${envId}`),
		lock: (projectId: string, envId: string, reason: string) =>
			apiClient.post<Environment>(`/projects/${projectId}/environments/${envId}/lock`, { reason }),
		unlock: (projectId: string, envId: string) =>
			apiClient.post<Environment>(`/projects/${projectId}/environments/${envId}/unlock`),
	},

	deployments: {
		list: (projectId: string, envId: string) =>
			apiClient.get<Deployment[]>(`/projects/${projectId}/environments/${envId}/deployments`),
		trigger: (projectId: string, envId: string, data: { pipeline_id?: string; version?: string; commit_sha?: string; image_tag?: string }) =>
			apiClient.post<Deployment>(`/projects/${projectId}/environments/${envId}/deploy`, data),
		rollback: (projectId: string, envId: string, deploymentId: string) =>
			apiClient.post<Deployment>(`/projects/${projectId}/environments/${envId}/rollback`, { deployment_id: deploymentId }),
		promote: (projectId: string, envId: string, sourceEnvId: string) =>
			apiClient.post<Deployment>(`/projects/${projectId}/environments/${envId}/promote`, { source_env_id: sourceEnvId }),
		recent: () => apiClient.get<Deployment[]>('/deployments/recent'),
		advanceCanary: (projectId: string, envId: string, deploymentId: string, weight: number) =>
			apiClient.post<void>(`/projects/${projectId}/environments/${envId}/deployments/${deploymentId}/advance-canary`, { weight }),
		checkHealth: (projectId: string, envId: string, deploymentId: string) =>
			apiClient.get<HealthResult>(`/projects/${projectId}/environments/${envId}/deployments/${deploymentId}/health`),
		getPlan: (projectId: string, envId: string, deploymentId: string) =>
			apiClient.get<DeploymentPlan>(`/projects/${projectId}/environments/${envId}/deployments/${deploymentId}/plan`),
	},

	strategy: {
		get: (projectId: string, envId: string) =>
			apiClient.get<Environment>(`/projects/${projectId}/environments/${envId}/strategy`),
		update: (projectId: string, envId: string, config: {
			strategy: string;
			strategy_config: string;
			health_check_url: string;
			health_check_interval: number;
			health_check_timeout: number;
			health_check_retries: number;
			health_check_path: string;
			health_check_expected_status: number;
		}) =>
			apiClient.put<Environment>(`/projects/${projectId}/environments/${envId}/strategy`, config),
	},

	envOverrides: {
		list: (projectId: string, envId: string) =>
			apiClient.get<EnvOverride[]>(`/projects/${projectId}/environments/${envId}/overrides`),
		save: (projectId: string, envId: string, overrides: { key: string; value: string; is_secret: boolean }[]) =>
			apiClient.put<EnvOverride[]>(`/projects/${projectId}/environments/${envId}/overrides`, overrides),
	},

	registries: {
		list: (projectId: string) =>
			apiClient.get<Registry[]>(`/projects/${projectId}/registries`),
		create: (projectId: string, data: { name: string; type: RegistryType; url: string; username: string; password: string; is_default?: boolean }) =>
			apiClient.post<Registry>(`/projects/${projectId}/registries`, data),
		update: (projectId: string, registryId: string, data: Partial<Registry & { password?: string }>) =>
			apiClient.put<Registry>(`/projects/${projectId}/registries/${registryId}`, data),
		delete: (projectId: string, registryId: string) =>
			apiClient.delete<void>(`/projects/${projectId}/registries/${registryId}`),
		test: (projectId: string, registryId: string) =>
			apiClient.post<{ success: boolean; message: string }>(`/projects/${projectId}/registries/${registryId}/test`),
		listImages: (projectId: string, registryId: string) =>
			apiClient.get<RegistryImage[]>(`/projects/${projectId}/registries/${registryId}/images`),
		listTags: (projectId: string, registryId: string, imageName: string) =>
			apiClient.get<RegistryTag[]>(`/projects/${projectId}/registries/${registryId}/images/${encodeURIComponent(imageName)}/tags`),
		deleteTag: (projectId: string, registryId: string, imageName: string, tag: string) =>
			apiClient.delete<void>(`/projects/${projectId}/registries/${registryId}/images/${encodeURIComponent(imageName)}/tags/${tag}`),
		setDefault: (projectId: string, registryId: string) =>
			apiClient.post<void>(`/projects/${projectId}/registries/${registryId}/default`),
	},

	approvals: {
		listPending: () => apiClient.get<Approval[]>('/approvals/pending'),
		listByProject: (projectId: string) => apiClient.get<Approval[]>(`/projects/${projectId}/approvals`),
		get: (approvalId: string) => apiClient.get<ApprovalDetail>(`/approvals/${approvalId}`),
		approve: (approvalId: string, comment?: string) =>
			apiClient.post<Approval>(`/approvals/${approvalId}/approve`, { comment: comment || '' }),
		reject: (approvalId: string, comment: string) =>
			apiClient.post<Approval>(`/approvals/${approvalId}/reject`, { comment }),
		cancel: (approvalId: string) =>
			apiClient.post<void>(`/approvals/${approvalId}/cancel`),
		getResponses: (approvalId: string) =>
			apiClient.get<ApprovalResponse[]>(`/approvals/${approvalId}/responses`),
	},

	protectionRules: {
		update: (projectId: string, envId: string, data: { require_approval: boolean; min_approvals: number; required_approvers: string[] }) =>
			apiClient.put<Environment>(`/projects/${projectId}/environments/${envId}/protection`, data),
	},

	schedules: {
		listByPipeline: (pipelineId: string) =>
			apiClient.get<PipelineSchedule[]>(`/pipelines/${pipelineId}/schedules`),
		listByProject: (projectId: string) =>
			apiClient.get<PipelineSchedule[]>(`/projects/${projectId}/schedules`),
		create: (pipelineId: string, data: {
			cron_expression: string;
			timezone: string;
			description?: string;
			branch?: string;
			environment_id?: string;
			variables?: Record<string, string>;
		}) =>
			apiClient.post<PipelineSchedule>(`/pipelines/${pipelineId}/schedules`, data),
		update: (scheduleId: string, data: Partial<PipelineSchedule>) =>
			apiClient.put<PipelineSchedule>(`/schedules/${scheduleId}`, data),
		delete: (scheduleId: string) =>
			apiClient.delete<void>(`/schedules/${scheduleId}`),
		enable: (scheduleId: string) =>
			apiClient.post<void>(`/schedules/${scheduleId}/enable`),
		disable: (scheduleId: string) =>
			apiClient.post<void>(`/schedules/${scheduleId}/disable`),
		getNextRuns: (scheduleId: string, count?: number) =>
			apiClient.get<string[]>(`/schedules/${scheduleId}/next-runs?count=${count || 5}`),
	},

	scaling: {
		listPolicies: () =>
			apiClient.get<ScalingPolicy[]>('/scaling/policies'),
		getPolicy: (policyId: string) =>
			apiClient.get<ScalingPolicy>(`/scaling/policies/${policyId}`),
		createPolicy: (data: Partial<ScalingPolicy>) =>
			apiClient.post<ScalingPolicy>('/scaling/policies', data),
		updatePolicy: (policyId: string, data: Partial<ScalingPolicy>) =>
			apiClient.put<ScalingPolicy>(`/scaling/policies/${policyId}`, data),
		deletePolicy: (policyId: string) =>
			apiClient.delete<void>(`/scaling/policies/${policyId}`),
		enablePolicy: (policyId: string) =>
			apiClient.post<void>(`/scaling/policies/${policyId}/enable`),
		disablePolicy: (policyId: string) =>
			apiClient.post<void>(`/scaling/policies/${policyId}/disable`),
		listEvents: (policyId: string) =>
			apiClient.get<ScalingEvent[]>(`/scaling/policies/${policyId}/events`),
		listRecentEvents: () =>
			apiClient.get<ScalingEvent[]>('/scaling/events'),
		getMetrics: () =>
			apiClient.get<ScalingMetrics>('/scaling/metrics'),
	},

	pipelineLinks: {
		list: (pipelineId: string) =>
			apiClient.get<PipelineLink[]>(`/pipelines/${pipelineId}/links`),
		create: (pipelineId: string, data: Partial<PipelineLink>) =>
			apiClient.post<PipelineLink>(`/pipelines/${pipelineId}/links`, data),
		update: (linkId: string, data: Partial<PipelineLink>) =>
			apiClient.put<PipelineLink>(`/pipeline-links/${linkId}`, data),
		delete: (linkId: string) =>
			apiClient.delete<void>(`/pipeline-links/${linkId}`),
		getDAG: (pipelineId: string) =>
			apiClient.get<PipelineDAG>(`/pipelines/${pipelineId}/dag`),
	},

	inbox: {
		list: (params?: Record<string, string>) => {
			const qs = params ? '?' + new URLSearchParams(params).toString() : '';
			return apiClient.get<InAppNotification[]>(`/notifications/inbox${qs}`);
		},
		unreadCount: () =>
			apiClient.get<{ count: number }>('/notifications/inbox/unread-count'),
		markRead: (notificationId: string) =>
			apiClient.post<void>(`/notifications/inbox/${notificationId}/read`),
		markAllRead: () =>
			apiClient.post<void>('/notifications/inbox/read-all'),
		delete: (notificationId: string) =>
			apiClient.delete<void>(`/notifications/inbox/${notificationId}`),
	},

	notificationPrefs: {
		get: () =>
			apiClient.get<NotificationPreference>('/notifications/preferences'),
		update: (data: Partial<NotificationPreference>) =>
			apiClient.put<NotificationPreference>('/notifications/preferences', data),
	},

	search: {
		query: (q: string) =>
			apiClient.get<SearchResults>(`/search?q=${encodeURIComponent(q)}`),
	},

	dashboardPrefs: {
		get: () =>
			apiClient.get<DashboardPreference>('/dashboard/preferences'),
		update: (data: { layout: string; theme?: string }) =>
			apiClient.put<DashboardPreference>('/dashboard/preferences', data),
	},

	badges: {
		pipelineUrl: (pipelineId: string) => `${API_BASE}/badges/pipeline/${pipelineId}`,
	},

	templates: {
		list: (params?: Record<string, string>) => {
			const qs = params ? '?' + new URLSearchParams(params).toString() : '';
			return apiClient.get<{ id: string; name: string; description: string; category: string; yaml: string; author: string; downloads: number; is_official: boolean }[]>(`/templates${qs}`);
		},
		get: (id: string) =>
			apiClient.get<{ id: string; name: string; description: string; category: string; yaml: string; author: string; downloads: number; is_official: boolean }>(`/templates/${id}`),
		create: (data: { name: string; description: string; category: string; yaml: string; tags?: string[] }) =>
			apiClient.post<{ id: string }>('/templates', data),
		delete: (id: string) =>
			apiClient.delete<void>(`/templates/${id}`),
	},

	runMetrics: {
		testResults: (projectId: string, pipelineId: string, runId: string) =>
			apiClient.get<{ xml_content: string }>(`/projects/${projectId}/pipelines/${pipelineId}/runs/${runId}/test-results`),
		coverage: (projectId: string, pipelineId: string, runId: string) =>
			apiClient.get<{ total_percentage: number; total_lines: number; covered_lines: number; threshold: number; files: { file: string; lines: number; covered: number; percentage: number }[] }>(`/projects/${projectId}/pipelines/${pipelineId}/runs/${runId}/coverage`),
		resources: (projectId: string, pipelineId: string, runId: string) =>
			apiClient.get<{
				points: { timestamp: number; cpu_percent: number; memory_mb: number; step_name: string }[];
				steps: { step_name: string; step_id: string; avg_cpu: number; max_cpu: number; avg_memory: number; max_memory: number; duration_ms: number }[];
			}>(`/projects/${projectId}/pipelines/${pipelineId}/runs/${runId}/resources`),
	},

	pipelineMetrics: {
		healthTrend: (projectId: string, pipelineId: string, days?: number) =>
			apiClient.get<{ date: string; success: number; failure: number; cancelled: number }[]>(
				`/projects/${projectId}/pipelines/${pipelineId}/metrics/health?days=${days ?? 30}`
			),
	},

	agentMetrics: {
		utilization: () =>
			apiClient.get<{ timestamp: string; cpu_percent: number; memory_percent: number; queue_depth: number }[]>('/agents/metrics/utilization'),
	},

	deploymentProviders: {
		list: (projectId: string) =>
			apiClient.get<ProjectDeploymentProvider[]>(`/projects/${projectId}/deployment-providers`),
		create: (projectId: string, data: CreateDeploymentProviderRequest) =>
			apiClient.post<ProjectDeploymentProvider>(`/projects/${projectId}/deployment-providers`, data),
		update: (projectId: string, dpId: string, data: UpdateDeploymentProviderRequest) =>
			apiClient.put<ProjectDeploymentProvider>(`/projects/${projectId}/deployment-providers/${dpId}`, data),
		delete: (projectId: string, dpId: string) =>
			apiClient.delete<{ message: string }>(`/projects/${projectId}/deployment-providers/${dpId}`),
		test: (projectId: string, dpId: string) =>
			apiClient.post<TestDeploymentProviderResponse>(`/projects/${projectId}/deployment-providers/${dpId}/test`),
	},

	environmentChain: {
		get: (projectId: string) =>
			apiClient.get<ProjectEnvironmentChainEdge[]>(`/projects/${projectId}/environment-chain`),
		update: (projectId: string, edges: UpdateEnvironmentChainRequest[]) =>
			apiClient.put<ProjectEnvironmentChainEdge[]>(`/projects/${projectId}/environment-chain`, edges),
	},

	stageEnvironments: {
		get: (projectId: string, pipelineId: string) =>
			apiClient.get<PipelineStageEnvironmentMapping[]>(`/projects/${projectId}/pipelines/${pipelineId}/stage-environments`),
		update: (projectId: string, pipelineId: string, mappings: UpdateStageEnvironmentMappingRequest[]) =>
			apiClient.put<PipelineStageEnvironmentMapping[]>(`/projects/${projectId}/pipelines/${pipelineId}/stage-environments`, mappings),
	},
};
