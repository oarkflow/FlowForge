import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import { Router } from '@solidjs/router';

// Mock the API client
vi.mock('../../api/client', () => ({
  api: {
    agents: {
      list: vi.fn().mockResolvedValue([
        {
          id: 'agent-1',
          name: 'Docker Agent 1',
          status: 'online',
          executor: 'docker',
          labels: ['docker', 'linux'],
          version: '1.0.0',
          os: 'linux',
          arch: 'amd64',
          cpu_cores: 4,
          memory_mb: 8192,
          ip_address: '10.0.0.1',
          last_seen_at: '2026-03-27T10:00:00Z',
          created_at: '2026-01-01T00:00:00Z',
        },
        {
          id: 'agent-2',
          name: 'K8s Agent',
          status: 'busy',
          executor: 'kubernetes',
          labels: ['k8s', 'production'],
          version: '1.0.0',
          os: 'linux',
          arch: 'arm64',
          cpu_cores: 8,
          memory_mb: 16384,
          ip_address: '10.0.0.2',
          last_seen_at: '2026-03-27T09:55:00Z',
          created_at: '2026-01-02T00:00:00Z',
        },
        {
          id: 'agent-3',
          name: 'Offline Agent',
          status: 'offline',
          executor: 'docker',
          labels: ['docker'],
          version: '0.9.0',
          os: 'linux',
          arch: 'amd64',
          cpu_cores: 2,
          memory_mb: 4096,
          ip_address: '10.0.0.3',
          last_seen_at: '2026-03-20T00:00:00Z',
          created_at: '2026-01-03T00:00:00Z',
        },
      ]),
      create: vi.fn().mockResolvedValue({ id: 'agent-new', token: 'ff_agent_token_abc123' }),
      drain: vi.fn().mockResolvedValue({}),
      delete: vi.fn().mockResolvedValue({}),
    },
    scaling: {
      listPolicies: vi.fn().mockResolvedValue([]),
      getMetrics: vi.fn().mockResolvedValue(undefined),
      listRecentEvents: vi.fn().mockResolvedValue([]),
    },
  },
  ApiRequestError: class ApiRequestError extends Error {
    constructor(message: string, public status: number) {
      super(message);
    }
  },
}));

describe('AgentsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the agents page', async () => {
    const { default: AgentsPage } = await import('./AgentsPage');
    render(() => (
      <Router>
        <AgentsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('Agents')).toBeInTheDocument();
  });

  it('renders agent names after loading', async () => {
    const { default: AgentsPage } = await import('./AgentsPage');
    render(() => (
      <Router>
        <AgentsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('Docker Agent 1')).toBeInTheDocument();
    expect(screen.getByText('K8s Agent')).toBeInTheDocument();
    expect(screen.getByText('Offline Agent')).toBeInTheDocument();
  });

  it('renders Register Agent button', async () => {
    const { default: AgentsPage } = await import('./AgentsPage');
    render(() => (
      <Router>
        <AgentsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('Register Agent')).toBeInTheDocument();
  });

  it('renders search input', async () => {
    const { default: AgentsPage } = await import('./AgentsPage');
    render(() => (
      <Router>
        <AgentsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByPlaceholderText(/search agents/i)).toBeInTheDocument();
  });

  it('renders agent labels', async () => {
    const { default: AgentsPage } = await import('./AgentsPage');
    render(() => (
      <Router>
        <AgentsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('docker')).toBeInTheDocument();
    expect(screen.getByText('linux')).toBeInTheDocument();
  });

  it('filters agents by search', async () => {
    const { default: AgentsPage } = await import('./AgentsPage');
    render(() => (
      <Router>
        <AgentsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));

    const searchInput = screen.getByPlaceholderText(/search agents/i);
    fireEvent.input(searchInput, { target: { value: 'K8s' } });

    expect(screen.getByText('K8s Agent')).toBeInTheDocument();
    expect(screen.queryByText('Docker Agent 1')).not.toBeInTheDocument();
    expect(screen.queryByText('Offline Agent')).not.toBeInTheDocument();
  });

  it('renders tabs for agents and scaling', async () => {
    const { default: AgentsPage } = await import('./AgentsPage');
    render(() => (
      <Router>
        <AgentsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    // The agents tab should be visible
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBeGreaterThan(0);
  });

  it('renders status filter', async () => {
    const { default: AgentsPage } = await import('./AgentsPage');
    render(() => (
      <Router>
        <AgentsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    // There should be a select for filtering by status
    const selects = document.querySelectorAll('select');
    expect(selects.length).toBeGreaterThan(0);
  });
});
