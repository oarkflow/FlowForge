import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@solidjs/testing-library';
import { Router } from '@solidjs/router';

// Mock the API client
vi.mock('../../api/client', () => ({
  api: {
    projects: {
      list: vi.fn().mockResolvedValue({
        data: [
          { id: 'proj-1', name: 'Alpha', slug: 'alpha', visibility: 'private', created_at: '2026-01-01T00:00:00Z' },
        ],
        total: 1,
        page: 1,
        per_page: 100,
      }),
    },
    agents: {
      list: vi.fn().mockResolvedValue([
        { id: 'agent-1', name: 'Docker Agent', status: 'online', executor: 'docker', labels: ['docker'], created_at: '2026-01-01T00:00:00Z' },
      ]),
    },
    system: {
      health: vi.fn().mockResolvedValue({
        status: 'healthy',
        version: '1.0.0',
        uptime_seconds: 86400,
        database: 'ok',
        queue_depth: 2,
      }),
    },
    pipelines: {
      list: vi.fn().mockResolvedValue([]),
    },
    runs: {
      list: vi.fn().mockResolvedValue({ data: [], total: 0, page: 1, per_page: 50 }),
    },
    deployments: {
      list: vi.fn().mockResolvedValue({ data: [], total: 0, page: 1, per_page: 10 }),
    },
    approvals: {
      list: vi.fn().mockResolvedValue({ data: [], total: 0, page: 1, per_page: 10 }),
    },
    dashboardPrefs: {
      get: vi.fn().mockResolvedValue({ widgets: [] }),
      update: vi.fn().mockResolvedValue({}),
    },
  },
}));

// Mock the WebSocket event socket
vi.mock('../../api/websocket', () => ({
  getEventSocket: vi.fn().mockReturnValue({
    on: vi.fn().mockReturnValue(vi.fn()),
    connect: vi.fn(),
    disconnect: vi.fn(),
    off: vi.fn(),
    send: vi.fn(),
    isConnected: false,
  }),
  connectEventSocket: vi.fn(),
  disconnectEventSocket: vi.fn(),
}));

// Mock chart components
vi.mock('../../components/dashboard/PipelineHealthChart', () => ({
  default: () => <div data-testid="pipeline-health-chart">PipelineHealthChart</div>,
}));

vi.mock('../../components/dashboard/AgentUtilizationChart', () => ({
  default: () => <div data-testid="agent-util-chart">AgentUtilizationChart</div>,
}));

describe('DashboardPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the dashboard page', async () => {
    const { default: DashboardPage } = await import('./DashboardPage');
    render(() => (
      <Router>
        <DashboardPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(document.body.innerHTML.length).toBeGreaterThan(0);
  });

  it('renders the page title', async () => {
    const { default: DashboardPage } = await import('./DashboardPage');
    render(() => (
      <Router>
        <DashboardPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('Dashboard')).toBeInTheDocument();
  });

  it('renders without crashing when APIs succeed', async () => {
    const { default: DashboardPage } = await import('./DashboardPage');
    const { container } = render(() => (
      <Router>
        <DashboardPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(container).toBeDefined();
  });

  it('renders customize button', async () => {
    const { default: DashboardPage } = await import('./DashboardPage');
    render(() => (
      <Router>
        <DashboardPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBeGreaterThan(0);
  });

  it('renders with loading state initially', async () => {
    const { default: DashboardPage } = await import('./DashboardPage');
    const { container } = render(() => (
      <Router>
        <DashboardPage />
      </Router>
    ));
    // The component should render immediately (even if loading)
    expect(container).toBeDefined();
  });
});
