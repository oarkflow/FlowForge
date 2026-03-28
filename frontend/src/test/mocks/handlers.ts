import { http, HttpResponse } from 'msw';
import type {
  User, AuthTokens, Project, Pipeline, PipelineRun, Agent, Secret,
  PaginatedResponse, SystemHealth, NotificationPreference, DashboardPreference,
} from '../../types';

const API = '/api/v1';

// ---------------------------------------------------------------------------
// Mock data factories
// ---------------------------------------------------------------------------
export const mockUser: User = {
  id: 'user-1',
  email: 'test@flowforge.dev',
  username: 'testuser',
  display_name: 'Test User',
  avatar_url: undefined,
  role: 'admin',
  totp_enabled: false,
  is_active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

export const mockTokens: AuthTokens = {
  access_token: 'mock-access-token',
  refresh_token: 'mock-refresh-token',
  expires_in: 3600,
};

export const mockProjects: Project[] = [
  {
    id: 'proj-1',
    name: 'Frontend App',
    slug: 'frontend-app',
    description: 'A SolidJS frontend application',
    visibility: 'private',
    created_by: 'user-1',
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
  {
    id: 'proj-2',
    name: 'Backend API',
    slug: 'backend-api',
    description: 'Go backend service',
    visibility: 'internal',
    created_by: 'user-1',
    created_at: '2026-02-01T10:00:00Z',
    updated_at: '2026-02-01T10:00:00Z',
  },
];

export const mockPipelines: Pipeline[] = [
  {
    id: 'pipe-1',
    project_id: 'proj-1',
    name: 'CI Pipeline',
    description: 'Continuous integration pipeline',
    config_source: 'db',
    config_path: '.flowforge.yml',
    config_content: 'version: "1"\nname: CI Pipeline',
    config_version: 1,
    triggers: {},
    is_active: true,
    created_by: 'user-1',
    created_at: '2026-01-15T12:00:00Z',
    updated_at: '2026-01-15T12:00:00Z',
  },
];

export const mockRuns: PipelineRun[] = [
  {
    id: 'run-1',
    pipeline_id: 'pipe-1',
    number: 1,
    status: 'success',
    trigger_type: 'push',
    commit_sha: 'abc1234567890',
    commit_message: 'Initial commit',
    branch: 'main',
    author: 'testuser',
    started_at: '2026-01-15T12:05:00Z',
    finished_at: '2026-01-15T12:10:00Z',
    duration_ms: 300000,
    created_at: '2026-01-15T12:05:00Z',
  },
  {
    id: 'run-2',
    pipeline_id: 'pipe-1',
    number: 2,
    status: 'running',
    trigger_type: 'push',
    commit_sha: 'def4567890123',
    commit_message: 'Add feature',
    branch: 'feature/new',
    author: 'testuser',
    started_at: '2026-01-15T13:00:00Z',
    duration_ms: undefined,
    created_at: '2026-01-15T13:00:00Z',
  },
];

export const mockAgents: Agent[] = [
  {
    id: 'agent-1',
    name: 'docker-runner-1',
    labels: ['docker', 'linux'],
    executor: 'docker',
    status: 'online',
    version: '1.0.0',
    os: 'linux',
    arch: 'amd64',
    cpu_cores: 4,
    memory_mb: 8192,
    ip_address: '10.0.0.1',
    last_seen_at: '2026-01-15T12:00:00Z',
    created_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'agent-2',
    name: 'k8s-runner-1',
    labels: ['kubernetes'],
    executor: 'kubernetes',
    status: 'offline',
    version: '1.0.0',
    os: 'linux',
    arch: 'arm64',
    cpu_cores: 8,
    memory_mb: 16384,
    ip_address: '10.0.0.2',
    last_seen_at: '2026-01-14T08:00:00Z',
    created_at: '2026-01-01T00:00:00Z',
  },
];

export const mockSecrets: Secret[] = [
  {
    id: 'secret-1',
    project_id: 'proj-1',
    scope: 'project',
    key: 'API_KEY',
    masked: true,
    created_by: 'user-1',
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
];

export const mockHealth: SystemHealth = {
  status: 'healthy',
  version: '1.0.0',
  uptime_seconds: 86400,
  database: { status: 'ok' },
  agents: { online: 1, total: 2 },
};

// ---------------------------------------------------------------------------
// MSW handlers
// ---------------------------------------------------------------------------
export const handlers = [
  // --- Auth ---
  http.post(`${API}/auth/login`, async ({ request }) => {
    const body = await request.json() as Record<string, string>;
    if (body.email === 'test@flowforge.dev' && body.password === 'password123') {
      return HttpResponse.json(mockTokens);
    }
    return HttpResponse.json(
      { error: 'auth', message: 'Invalid credentials', status: 401 },
      { status: 401 },
    );
  }),

  http.post(`${API}/auth/register`, () => {
    return HttpResponse.json(mockTokens);
  }),

  http.post(`${API}/auth/refresh`, () => {
    return HttpResponse.json(mockTokens);
  }),

  http.post(`${API}/auth/logout`, () => {
    return new HttpResponse(null, { status: 204 });
  }),

  // --- Users ---
  http.get(`${API}/users/me`, () => {
    return HttpResponse.json(mockUser);
  }),

  http.put(`${API}/users/me`, async ({ request }) => {
    const body = await request.json() as Partial<User>;
    return HttpResponse.json({ ...mockUser, ...body });
  }),

  // --- Projects ---
  http.get(`${API}/projects`, ({ request }) => {
    const url = new URL(request.url);
    const page = parseInt(url.searchParams.get('page') || '1');
    const perPage = parseInt(url.searchParams.get('per_page') || '50');
    const response: PaginatedResponse<Project> = {
      data: mockProjects,
      total: mockProjects.length,
      page,
      per_page: perPage,
      total_pages: 1,
    };
    return HttpResponse.json(response);
  }),

  http.post(`${API}/projects`, async ({ request }) => {
    const body = await request.json() as Partial<Project>;
    const created: Project = {
      id: 'proj-new',
      name: body.name || 'New Project',
      slug: (body.name || 'new-project').toLowerCase().replace(/\s+/g, '-'),
      description: body.description,
      visibility: body.visibility || 'private',
      created_by: 'user-1',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    return HttpResponse.json(created, { status: 201 });
  }),

  http.get(`${API}/projects/:id`, ({ params }) => {
    const project = mockProjects.find(p => p.id === params.id);
    if (!project) {
      return HttpResponse.json({ error: 'not_found', message: 'Project not found', status: 404 }, { status: 404 });
    }
    return HttpResponse.json(project);
  }),

  http.put(`${API}/projects/:id`, async ({ params, request }) => {
    const body = await request.json() as Partial<Project>;
    const project = mockProjects.find(p => p.id === params.id);
    if (!project) {
      return HttpResponse.json({ error: 'not_found', message: 'Project not found', status: 404 }, { status: 404 });
    }
    return HttpResponse.json({ ...project, ...body });
  }),

  http.delete(`${API}/projects/:id`, () => {
    return new HttpResponse(null, { status: 204 });
  }),

  // --- Pipelines ---
  http.get(`${API}/projects/:projectId/pipelines`, () => {
    return HttpResponse.json(mockPipelines);
  }),

  http.get(`${API}/projects/:projectId/pipelines/:pipelineId`, ({ params }) => {
    const pipeline = mockPipelines.find(p => p.id === params.pipelineId);
    if (!pipeline) {
      return HttpResponse.json({ error: 'not_found', message: 'Pipeline not found', status: 404 }, { status: 404 });
    }
    return HttpResponse.json(pipeline);
  }),

  http.post(`${API}/projects/:projectId/pipelines/:pipelineId/trigger`, () => {
    return HttpResponse.json(mockRuns[1], { status: 201 });
  }),

  // --- Runs ---
  http.get(`${API}/runs`, () => {
    return HttpResponse.json(mockRuns);
  }),

  http.get(`${API}/projects/:projectId/pipelines/:pipelineId/runs`, () => {
    return HttpResponse.json(mockRuns);
  }),

  http.get(`${API}/projects/:projectId/pipelines/:pipelineId/runs/:runId`, ({ params }) => {
    const run = mockRuns.find(r => r.id === params.runId);
    if (!run) {
      return HttpResponse.json({ error: 'not_found', message: 'Run not found', status: 404 }, { status: 404 });
    }
    return HttpResponse.json({
      ...run,
      stages: [{ id: 'stage-1', run_id: run.id, name: 'build', status: run.status, position: 0 }],
      jobs: [{ id: 'job-1', stage_run_id: 'stage-1', run_id: run.id, name: 'build', status: run.status, executor_type: 'docker' }],
      steps: [{ id: 'step-1', job_run_id: 'job-1', name: 'checkout', status: run.status }],
    });
  }),

  http.post(`${API}/projects/:projectId/pipelines/:pipelineId/runs/:runId/cancel`, () => {
    return new HttpResponse(null, { status: 204 });
  }),

  http.get(`${API}/projects/:projectId/pipelines/:pipelineId/runs/:runId/logs`, () => {
    return HttpResponse.json([
      { id: 1, run_id: 'run-1', stream: 'stdout', content: 'Step 1: Checkout', ts: '2026-01-15T12:05:00Z' },
      { id: 2, run_id: 'run-1', stream: 'stdout', content: 'Step 2: Build', ts: '2026-01-15T12:06:00Z' },
    ]);
  }),

  http.get(`${API}/projects/:projectId/pipelines/:pipelineId/runs/:runId/artifacts`, () => {
    return HttpResponse.json([]);
  }),

  // --- Agents ---
  http.get(`${API}/agents`, () => {
    return HttpResponse.json(mockAgents);
  }),

  http.post(`${API}/agents`, async ({ request }) => {
    const body = await request.json() as Partial<Agent>;
    return HttpResponse.json({
      id: 'agent-new',
      name: body.name || 'new-agent',
      labels: body.labels || [],
      executor: body.executor || 'docker',
      status: 'offline',
      created_at: new Date().toISOString(),
      token: 'mock-agent-token-xyz',
    }, { status: 201 });
  }),

  http.delete(`${API}/agents/:id`, () => {
    return new HttpResponse(null, { status: 204 });
  }),

  http.post(`${API}/agents/:id/drain`, () => {
    return new HttpResponse(null, { status: 204 });
  }),

  // --- Secrets ---
  http.get(`${API}/projects/:projectId/secrets`, () => {
    return HttpResponse.json(mockSecrets);
  }),

  // --- System ---
  http.get(`${API}/system/health`, () => {
    return HttpResponse.json(mockHealth);
  }),

  // --- Notifications ---
  http.get(`${API}/notifications/preferences`, () => {
    const prefs: NotificationPreference = {
      id: 'pref-1',
      user_id: 'user-1',
      email_enabled: true,
      in_app_enabled: true,
      pipeline_success: true,
      pipeline_failure: true,
      deployment_success: true,
      deployment_failure: true,
      approval_requested: true,
      approval_resolved: true,
      agent_offline: true,
      security_alerts: true,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    };
    return HttpResponse.json(prefs);
  }),

  http.put(`${API}/notifications/preferences`, async ({ request }) => {
    const body = await request.json();
    return HttpResponse.json(body);
  }),

  // --- Dashboard prefs ---
  http.get(`${API}/dashboard/preferences`, () => {
    const prefs: DashboardPreference = {
      id: 'dash-1',
      user_id: 'user-1',
      layout: '[]',
      theme: 'dark',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    };
    return HttpResponse.json(prefs);
  }),

  // --- Approvals ---
  http.get(`${API}/approvals/pending`, () => {
    return HttpResponse.json([]);
  }),

  // --- Deployments ---
  http.get(`${API}/deployments/recent`, () => {
    return HttpResponse.json([]);
  }),

  // --- Scaling ---
  http.get(`${API}/scaling/policies`, () => {
    return HttpResponse.json([]);
  }),

  http.get(`${API}/scaling/metrics`, () => {
    return HttpResponse.json({
      total_agents: 2,
      online_agents: 1,
      busy_agents: 0,
      queue_depth: 0,
      agents_by_executor: { docker: 1, kubernetes: 1 },
      agents_by_label: {},
    });
  }),

  http.get(`${API}/scaling/events`, () => {
    return HttpResponse.json([]);
  }),

  // --- API keys ---
  http.get(`${API}/users/me/api-keys`, () => {
    return HttpResponse.json([]);
  }),

  // --- Audit logs ---
  http.get(`${API}/audit-logs`, () => {
    return HttpResponse.json([]);
  }),

  // --- Deployment Providers ---
  http.get(`${API}/projects/:projectId/deployment-providers`, () => {
    return HttpResponse.json([]);
  }),

  http.post(`${API}/projects/:projectId/deployment-providers`, async ({ request }) => {
    const body = await request.json() as Record<string, unknown>;
    return HttpResponse.json({
      id: 'dp-new',
      project_id: 'proj-1',
      name: body.name || 'New Provider',
      provider_type: body.provider_type || 'aws',
      config: {},
      is_active: body.is_active ?? true,
      is_default: false,
      capabilities: '',
      created_by: 'user-1',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }, { status: 201 });
  }),

  http.put(`${API}/projects/:projectId/deployment-providers/:dpId`, async ({ request, params }) => {
    const body = await request.json() as Record<string, unknown>;
    return HttpResponse.json({
      id: params.dpId,
      project_id: params.projectId,
      name: body.name || 'Updated Provider',
      provider_type: body.provider_type || 'aws',
      config: {},
      is_active: body.is_active ?? true,
      is_default: false,
      capabilities: '',
      created_by: 'user-1',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: new Date().toISOString(),
    });
  }),

  http.delete(`${API}/projects/:projectId/deployment-providers/:dpId`, () => {
    return HttpResponse.json({ message: 'Provider deleted' });
  }),

  http.post(`${API}/projects/:projectId/deployment-providers/:dpId/test`, () => {
    return HttpResponse.json({ success: true, message: 'Connection successful' });
  }),

  // --- Environment Chain ---
  http.get(`${API}/projects/:projectId/environment-chain`, () => {
    return HttpResponse.json([]);
  }),

  http.put(`${API}/projects/:projectId/environment-chain`, async ({ request }) => {
    const body = await request.json() as unknown[];
    return HttpResponse.json(body);
  }),

  // --- Pipeline Stage→Environment Mappings ---
  http.get(`${API}/projects/:projectId/pipelines/:pipelineId/stage-environments`, () => {
    return HttpResponse.json([]);
  }),

  http.put(`${API}/projects/:projectId/pipelines/:pipelineId/stage-environments`, async ({ request }) => {
    const body = await request.json() as unknown[];
    return HttpResponse.json(body);
  }),
];
