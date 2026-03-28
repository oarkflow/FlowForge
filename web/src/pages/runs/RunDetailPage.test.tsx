import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@solidjs/testing-library';
import { Router, Route } from '@solidjs/router';
import { api } from '../../api/client';

const socketState = vi.hoisted(() => {
  const handlers = new Map<string, ((payload: unknown) => void)[]>();
  return {
    reset() {
      handlers.clear();
    },
    emit(event: string, payload: unknown) {
      for (const handler of handlers.get(event) ?? []) handler(payload);
    },
    add(event: string, handler: (payload: unknown) => void) {
      const current = handlers.get(event) ?? [];
      handlers.set(event, [...current, handler]);
      return () => {
        handlers.set(event, (handlers.get(event) ?? []).filter((h) => h !== handler));
      };
    },
  };
});

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
      get: vi.fn().mockResolvedValue({
          id: 'run-1',
          pipeline_id: 'pipe-1',
          number: 42,
          status: 'running',
          trigger_type: 'push',
          branch: 'main',
          commit_sha: 'abc1234567890',
          commit_message: 'Fix tests',
          author: 'developer',
          duration_ms: 65000,
          started_at: '2026-03-01T10:00:00Z',
          finished_at: null,
          created_at: '2026-03-01T09:59:55Z',
        stages: [
          { id: 'stage-1', run_id: 'run-1', name: 'build', status: 'running', position: 0, started_at: '2026-03-01T10:00:00Z', finished_at: null },
          { id: 'stage-2', run_id: 'run-1', name: 'test', status: 'pending', position: 1, started_at: null, finished_at: null },
        ],
        jobs: [
          { id: 'job-1', stage_run_id: 'stage-1', run_id: 'run-1', name: 'build-go', status: 'running', executor_type: 'docker', started_at: '2026-03-01T10:00:00Z', finished_at: null },
          { id: 'job-2', stage_run_id: 'stage-2', run_id: 'run-1', name: 'test-unit', status: 'pending', executor_type: 'docker', started_at: null, finished_at: null },
        ],
        steps: [
          { id: 'step-1', job_run_id: 'job-1', name: 'Checkout', status: 'success', exit_code: 0, duration_ms: 5000, started_at: '2026-03-01T10:00:00Z', finished_at: '2026-03-01T10:00:05Z' },
          { id: 'step-2', job_run_id: 'job-1', name: 'Build', status: 'running', exit_code: null, duration_ms: null, started_at: '2026-03-01T10:00:05Z', finished_at: null },
          { id: 'step-3', job_run_id: 'job-2', name: 'Run tests', status: 'pending', exit_code: null, duration_ms: null, started_at: null, finished_at: null },
        ],
      }),
      getLogs: vi.fn().mockResolvedValue([
        { id: 1, run_id: 'run-1', step_run_id: 'step-1', stream: 'stdout', content: 'Cloning repository...', ts: '2026-03-01T10:00:00Z' },
        { id: 2, run_id: 'run-1', step_run_id: 'step-2', stream: 'stdout', content: 'Building binary...', ts: '2026-03-01T10:00:05Z' },
      ]),
      cancel: vi.fn().mockResolvedValue({}),
      rerun: vi.fn().mockResolvedValue({ id: 'run-2' }),
      getArtifacts: vi.fn().mockResolvedValue([]),
      list: vi.fn().mockResolvedValue({ data: [{ id: 'run-1', number: 42 }] }),
      approve: vi.fn().mockResolvedValue({}),
    },
    projects: {
      get: vi.fn().mockResolvedValue({ id: 'proj-1', name: 'Test Project' }),
    },
    pipelines: {
      get: vi.fn().mockResolvedValue({ id: 'pipe-1', name: 'CI Pipeline' }),
    },
    runMetrics: {
      testResults: vi.fn().mockResolvedValue({ xml: '' }),
      coverage: vi.fn().mockResolvedValue(null),
      resources: vi.fn().mockResolvedValue({ points: [], steps: [] }),
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
  createRunLogSocket: vi.fn().mockImplementation(() => ({
    on: vi.fn((event: string, handler: (payload: unknown) => void) => socketState.add(event, handler)),
    connect: vi.fn(),
    disconnect: vi.fn(),
    off: vi.fn(),
    send: vi.fn(),
    isConnected: false,
  })),
}));

describe('RunDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    socketState.reset();
  });

  const renderPage = async () => {
    const { default: RunDetailPage } = await import('./RunDetailPage');
    return render(() => (
      <Router>
        <Route path="/" component={RunDetailPage} />
      </Router>
    ));
  };

  it('renders the run detail page', async () => {
    const { container } = await renderPage();
    await new Promise(r => setTimeout(r, 200));
    expect(container).toBeDefined();
  });

  it('renders run number', async () => {
    await renderPage();
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText(/#42/)).toBeInTheDocument();
  });

  it('renders stage names', async () => {
    await renderPage();
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('build')).toBeInTheDocument();
    expect(screen.getByText('test')).toBeInTheDocument();
  });

  it('renders run actions for an active run', async () => {
    await renderPage();
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('Cancel')).toBeInTheDocument();
  });

  it('renders visible log content', async () => {
    await renderPage();
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('Cloning repository...')).toBeInTheDocument();
    expect(screen.getByText('Building binary...')).toBeInTheDocument();
  });

  it('renders commit sha', async () => {
    await renderPage();
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('abc1234')).toBeInTheDocument();
  });

  it('renders branch name', async () => {
    await renderPage();
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('main')).toBeInTheDocument();
  });

  it('renders live websocket logs alongside historical logs', async () => {
    await renderPage();

    await waitFor(() => {
      expect(screen.getByText('Building binary...')).toBeInTheDocument();
    });

    socketState.emit('log', {
      step_run_id: 'step-2',
      stream: 'stdout',
      content: 'Bundling assets...',
    });

    await waitFor(() => {
      expect(screen.getByText('Bundling assets...')).toBeInTheDocument();
    });
  });

  it('polls active runs so terminal failure appears without a page refresh', async () => {
    vi.useFakeTimers();
    const getRun = vi.mocked(api.runs.get);

    getRun.mockReset();
    getRun
      .mockResolvedValueOnce({
        id: 'run-1',
        pipeline_id: 'pipe-1',
        number: 42,
        status: 'running',
        trigger_type: 'push',
        branch: 'main',
        commit_sha: 'abc1234567890',
        commit_message: 'Fix tests',
        author: 'developer',
        duration_ms: 65000,
        started_at: '2026-03-01T10:00:00Z',
        finished_at: null,
        created_at: '2026-03-01T09:59:55Z',
        stages: [
          { id: 'stage-1', run_id: 'run-1', name: 'build', status: 'running', position: 0, started_at: '2026-03-01T10:00:00Z', finished_at: null },
        ],
        jobs: [
          { id: 'job-1', stage_run_id: 'stage-1', run_id: 'run-1', name: 'build-go', status: 'running', executor_type: 'docker', started_at: '2026-03-01T10:00:00Z', finished_at: null },
        ],
        steps: [
          { id: 'step-2', job_run_id: 'job-1', name: 'Run linter', status: 'running', exit_code: null, duration_ms: null, started_at: '2026-03-01T10:00:05Z', finished_at: null },
        ],
      })
      .mockResolvedValue({
        id: 'run-1',
        pipeline_id: 'pipe-1',
        number: 42,
        status: 'failure',
        trigger_type: 'push',
        branch: 'main',
        commit_sha: 'abc1234567890',
        commit_message: 'Fix tests',
        author: 'developer',
        duration_ms: 65000,
        started_at: '2026-03-01T10:00:00Z',
        finished_at: '2026-03-01T10:01:00Z',
        created_at: '2026-03-01T09:59:55Z',
        stages: [
          { id: 'stage-1', run_id: 'run-1', name: 'build', status: 'failure', position: 0, started_at: '2026-03-01T10:00:00Z', finished_at: '2026-03-01T10:01:00Z' },
        ],
        jobs: [
          { id: 'job-1', stage_run_id: 'stage-1', run_id: 'run-1', name: 'build-go', status: 'failure', executor_type: 'docker', started_at: '2026-03-01T10:00:00Z', finished_at: '2026-03-01T10:01:00Z' },
        ],
        steps: [
          { id: 'step-2', job_run_id: 'job-1', name: 'Run linter', status: 'failure', exit_code: 1, duration_ms: 2000, started_at: '2026-03-01T10:00:05Z', finished_at: '2026-03-01T10:00:07Z' },
        ],
      });

    try {
      await renderPage();

      await waitFor(() => {
        expect(screen.getByText('Running')).toBeInTheDocument();
      });

      await vi.advanceTimersByTimeAsync(2500);

      await waitFor(() => {
        expect(screen.getByText('Failed')).toBeInTheDocument();
      });
    } finally {
      vi.useRealTimers();
    }
  });
});
