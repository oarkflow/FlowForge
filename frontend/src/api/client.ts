import type {
  ApiError, AuthTokens, User, Organization, OrgMember, Project, Repository,
  Pipeline, PipelineVersion, PipelineRun, RunWithMeta, StageRun, JobRun, StepRun,
  Secret, EnvVar, Agent, Artifact, NotificationChannel, AuditLog, LogLine,
  PaginatedResponse, SystemHealth, LoginRequest, RegisterRequest,
  ProviderRepo, ImportDetectRequest, ImportDetectResponse,
  ImportCreateProjectRequest, ImportCreateProjectResponse,
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
};
