import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@solidjs/testing-library';
import { Router } from '@solidjs/router';

// Mock @solidjs/router's useParams
vi.mock('@solidjs/router', async () => {
  const actual = await vi.importActual('@solidjs/router');
  return {
    ...actual,
    useParams: () => ({ id: 'proj-1' }),
    useNavigate: () => vi.fn(),
  };
});

// Mock the API client
vi.mock('../../api/client', () => ({
  api: {
    pipelines: {
      list: vi.fn().mockResolvedValue([
        {
          id: 'pipe-1',
          name: 'CI Pipeline',
          description: 'Continuous integration',
          is_active: true,
          triggers: '{"push":{"branches":["main"]}}',
          config_source: 'db',
          config_version: 1,
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
        },
        {
          id: 'pipe-2',
          name: 'Deploy Pipeline',
          description: 'Deploy to staging',
          is_active: false,
          triggers: '{}',
          config_source: 'repo',
          config_version: 1,
          created_at: '2026-01-02T00:00:00Z',
          updated_at: '2026-01-02T00:00:00Z',
        },
      ]),
      create: vi.fn().mockResolvedValue({}),
      trigger: vi.fn().mockResolvedValue({}),
      update: vi.fn().mockResolvedValue({}),
      delete: vi.fn().mockResolvedValue({}),
    },
    runs: {
      list: vi.fn().mockResolvedValue({
        data: [
          {
            id: 'run-1',
            pipeline_id: 'pipe-1',
            number: 42,
            status: 'success',
            trigger_type: 'push',
            branch: 'main',
            commit_sha: 'abc1234567890',
            commit_message: 'Fix tests',
            author: 'dev',
            duration_ms: 65000,
            created_at: '2026-03-01T10:00:00Z',
          },
        ],
        total: 1,
        page: 1,
        per_page: 50,
      }),
    },
    projects: {
      get: vi.fn().mockResolvedValue({ id: 'proj-1', name: 'Test Project' }),
    },
  },
  ApiRequestError: class ApiRequestError extends Error {
    constructor(message: string, public status: number) {
      super(message);
    }
  },
}));

describe('PipelinesPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the pipelines page', async () => {
    const { default: PipelinesPage } = await import('./PipelinesPage');
    render(() => (
      <Router>
        <PipelinesPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('Pipelines')).toBeInTheDocument();
  });

  it('renders the New Pipeline button', async () => {
    const { default: PipelinesPage } = await import('./PipelinesPage');
    render(() => (
      <Router>
        <PipelinesPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('New Pipeline')).toBeInTheDocument();
  });

  it('renders the Trigger Run button', async () => {
    const { default: PipelinesPage } = await import('./PipelinesPage');
    render(() => (
      <Router>
        <PipelinesPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('Trigger Run')).toBeInTheDocument();
  });

  it('renders pipeline names after loading', async () => {
    const { default: PipelinesPage } = await import('./PipelinesPage');
    render(() => (
      <Router>
        <PipelinesPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('CI Pipeline')).toBeInTheDocument();
    expect(screen.getByText('Deploy Pipeline')).toBeInTheDocument();
  });

  it('renders the tabs', async () => {
    const { default: PipelinesPage } = await import('./PipelinesPage');
    render(() => (
      <Router>
        <PipelinesPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    // Tabs should contain pipeline count
    const pipelinesTab = screen.getByText(/Pipelines \(2\)/);
    expect(pipelinesTab).toBeInTheDocument();
  });

  it('renders pipeline descriptions', async () => {
    const { default: PipelinesPage } = await import('./PipelinesPage');
    render(() => (
      <Router>
        <PipelinesPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('Continuous integration')).toBeInTheDocument();
    expect(screen.getByText('Deploy to staging')).toBeInTheDocument();
  });

  it('renders active/disabled badges', async () => {
    const { default: PipelinesPage } = await import('./PipelinesPage');
    render(() => (
      <Router>
        <PipelinesPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('Active')).toBeInTheDocument();
    expect(screen.getByText('Disabled')).toBeInTheDocument();
  });
});
