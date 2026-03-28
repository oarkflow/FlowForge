import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@solidjs/testing-library';
import { Router } from '@solidjs/router';

// Mock @solidjs/router's useParams
vi.mock('@solidjs/router', async () => {
  const actual = await vi.importActual('@solidjs/router');
  return {
    ...actual,
    useParams: () => ({ id: 'proj-1', pid: 'pipe-1', rid: 'run-1' }),
    useNavigate: () => vi.fn(),
  };
});

// Mock the API client
vi.mock('../../api/client', () => ({
  api: {
    runs: {
      getDetail: vi.fn().mockResolvedValue({
        run: {
          id: 'run-1',
          pipeline_id: 'pipe-1',
          number: 42,
          status: 'success',
          trigger_type: 'push',
          branch: 'main',
          commit_sha: 'abc1234567890',
          commit_message: 'Fix tests',
          author: 'developer',
          duration_ms: 65000,
          started_at: '2026-03-01T10:00:00Z',
          finished_at: '2026-03-01T10:01:05Z',
          created_at: '2026-03-01T09:59:55Z',
        },
        stages: [
          { id: 'stage-1', run_id: 'run-1', name: 'build', status: 'success', position: 0, started_at: '2026-03-01T10:00:00Z', finished_at: '2026-03-01T10:00:30Z' },
          { id: 'stage-2', run_id: 'run-1', name: 'test', status: 'success', position: 1, started_at: '2026-03-01T10:00:30Z', finished_at: '2026-03-01T10:01:00Z' },
        ],
        jobs: [
          { id: 'job-1', stage_run_id: 'stage-1', run_id: 'run-1', name: 'build-go', status: 'success', executor_type: 'docker', started_at: '2026-03-01T10:00:00Z', finished_at: '2026-03-01T10:00:30Z' },
          { id: 'job-2', stage_run_id: 'stage-2', run_id: 'run-1', name: 'test-unit', status: 'success', executor_type: 'docker', started_at: '2026-03-01T10:00:30Z', finished_at: '2026-03-01T10:01:00Z' },
        ],
        steps: [
          { id: 'step-1', job_run_id: 'job-1', name: 'Checkout', status: 'success', exit_code: 0, duration_ms: 5000, started_at: '2026-03-01T10:00:00Z', finished_at: '2026-03-01T10:00:05Z' },
          { id: 'step-2', job_run_id: 'job-1', name: 'Build', status: 'success', exit_code: 0, duration_ms: 25000, started_at: '2026-03-01T10:00:05Z', finished_at: '2026-03-01T10:00:30Z' },
          { id: 'step-3', job_run_id: 'job-2', name: 'Run tests', status: 'success', exit_code: 0, duration_ms: 30000, started_at: '2026-03-01T10:00:30Z', finished_at: '2026-03-01T10:01:00Z' },
        ],
      }),
      getLogs: vi.fn().mockResolvedValue([
        { id: 1, run_id: 'run-1', step_run_id: 'step-1', stream: 'stdout', content: 'Cloning repository...', ts: '2026-03-01T10:00:00Z' },
        { id: 2, run_id: 'run-1', step_run_id: 'step-2', stream: 'stdout', content: 'Building binary...', ts: '2026-03-01T10:00:05Z' },
      ]),
      cancel: vi.fn().mockResolvedValue({}),
      rerun: vi.fn().mockResolvedValue({ id: 'run-2' }),
      getArtifacts: vi.fn().mockResolvedValue([]),
    },
    projects: {
      get: vi.fn().mockResolvedValue({ id: 'proj-1', name: 'Test Project' }),
    },
    pipelines: {
      get: vi.fn().mockResolvedValue({ id: 'pipe-1', name: 'CI Pipeline' }),
    },
  },
  ApiRequestError: class ApiRequestError extends Error {
    constructor(message: string, public status: number) {
      super(message);
    }
  },
}));

// Mock the WebSocket log socket
vi.mock('../../api/websocket', () => ({
  createRunLogSocket: vi.fn().mockReturnValue({
    on: vi.fn().mockReturnValue(vi.fn()),
    connect: vi.fn(),
    disconnect: vi.fn(),
    off: vi.fn(),
    send: vi.fn(),
    isConnected: false,
  }),
}));

describe('RunDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the run detail page', async () => {
    const { default: RunDetailPage } = await import('./RunDetailPage');
    const { container } = render(() => (
      <Router>
        <RunDetailPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(container).toBeDefined();
  });

  it('renders run number', async () => {
    const { default: RunDetailPage } = await import('./RunDetailPage');
    render(() => (
      <Router>
        <RunDetailPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText(/#42/)).toBeInTheDocument();
  });

  it('renders stage names', async () => {
    const { default: RunDetailPage } = await import('./RunDetailPage');
    render(() => (
      <Router>
        <RunDetailPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('build')).toBeInTheDocument();
    expect(screen.getByText('test')).toBeInTheDocument();
  });

  it('renders job names', async () => {
    const { default: RunDetailPage } = await import('./RunDetailPage');
    render(() => (
      <Router>
        <RunDetailPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('build-go')).toBeInTheDocument();
    expect(screen.getByText('test-unit')).toBeInTheDocument();
  });

  it('renders step names', async () => {
    const { default: RunDetailPage } = await import('./RunDetailPage');
    render(() => (
      <Router>
        <RunDetailPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('Checkout')).toBeInTheDocument();
    expect(screen.getByText('Build')).toBeInTheDocument();
    expect(screen.getByText('Run tests')).toBeInTheDocument();
  });

  it('renders commit info', async () => {
    const { default: RunDetailPage } = await import('./RunDetailPage');
    render(() => (
      <Router>
        <RunDetailPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('Fix tests')).toBeInTheDocument();
  });

  it('renders branch name', async () => {
    const { default: RunDetailPage } = await import('./RunDetailPage');
    render(() => (
      <Router>
        <RunDetailPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('main')).toBeInTheDocument();
  });
});
