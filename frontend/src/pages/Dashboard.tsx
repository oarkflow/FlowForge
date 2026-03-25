import type { Component } from 'solid-js';
import { createSignal, onMount, For, Show } from 'solid-js';
import { A } from '@solidjs/router';
import Card from '../components/ui/Card';
import Badge from '../components/ui/Badge';
import Table, { type TableColumn } from '../components/ui/Table';
import type { PipelineRun, RunStatus, Agent, AgentStatus } from '../types';

// ---------------------------------------------------------------------------
// Status helpers
// ---------------------------------------------------------------------------
const statusVariant = (status: RunStatus) => {
  const map: Record<RunStatus, 'success' | 'error' | 'warning' | 'running' | 'queued' | 'default' | 'info'> = {
    success: 'success',
    failure: 'error',
    cancelled: 'warning',
    running: 'running',
    queued: 'queued',
    pending: 'queued',
    skipped: 'default',
    waiting_approval: 'info',
  };
  return map[status] ?? 'default';
};

const statusLabel = (status: RunStatus) => {
  const map: Record<RunStatus, string> = {
    success: 'Success',
    failure: 'Failed',
    cancelled: 'Cancelled',
    running: 'Running',
    queued: 'Queued',
    pending: 'Pending',
    skipped: 'Skipped',
    waiting_approval: 'Awaiting Approval',
  };
  return map[status] ?? status;
};

const agentStatusVariant = (status: AgentStatus) => {
  const map: Record<AgentStatus, 'success' | 'error' | 'running' | 'warning'> = {
    online: 'success',
    offline: 'error',
    busy: 'running',
    draining: 'warning',
  };
  return map[status];
};

const formatDuration = (ms?: number): string => {
  if (!ms) return '-';
  if (ms < 1000) return `${ms}ms`;
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}m ${remainingSeconds}s`;
};

const formatTimeAgo = (dateStr: string): string => {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSeconds = Math.floor(diffMs / 1000);
  if (diffSeconds < 60) return 'just now';
  const diffMinutes = Math.floor(diffSeconds / 60);
  if (diffMinutes < 60) return `${diffMinutes}m ago`;
  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays}d ago`;
};

// ---------------------------------------------------------------------------
// Mock data (until backend is connected)
// ---------------------------------------------------------------------------
const mockRuns: PipelineRun[] = [
  {
    id: '1', pipeline_id: 'p1', number: 142, status: 'success', trigger_type: 'push',
    commit_sha: 'a1b2c3d', commit_message: 'fix: resolve auth token refresh race condition',
    branch: 'main', author: 'sujit', duration_ms: 127000, created_at: new Date(Date.now() - 300000).toISOString(),
  },
  {
    id: '2', pipeline_id: 'p1', number: 141, status: 'failure', trigger_type: 'pull_request',
    commit_sha: 'e4f5g6h', commit_message: 'feat: add pipeline matrix builds support',
    branch: 'feature/matrix', author: 'dev01', duration_ms: 89000, created_at: new Date(Date.now() - 1200000).toISOString(),
  },
  {
    id: '3', pipeline_id: 'p2', number: 87, status: 'running', trigger_type: 'push',
    commit_sha: 'i7j8k9l', commit_message: 'chore: update dependencies to latest versions',
    branch: 'develop', author: 'sujit', duration_ms: undefined, created_at: new Date(Date.now() - 60000).toISOString(),
  },
  {
    id: '4', pipeline_id: 'p3', number: 56, status: 'queued', trigger_type: 'schedule',
    commit_sha: 'm0n1o2p', commit_message: 'ci: nightly integration test run',
    branch: 'main', author: 'system', duration_ms: undefined, created_at: new Date(Date.now() - 30000).toISOString(),
  },
  {
    id: '5', pipeline_id: 'p1', number: 140, status: 'cancelled', trigger_type: 'manual',
    commit_sha: 'q3r4s5t', commit_message: 'test: add e2e tests for deployment flow',
    branch: 'test/e2e', author: 'dev02', duration_ms: 34000, created_at: new Date(Date.now() - 3600000).toISOString(),
  },
];

const mockAgents: Agent[] = [
  { id: 'a1', name: 'agent-linux-01', labels: ['linux', 'docker'], executor: 'docker', status: 'online', os: 'Linux', arch: 'amd64', cpu_cores: 8, memory_mb: 16384, created_at: '' },
  { id: 'a2', name: 'agent-linux-02', labels: ['linux', 'docker'], executor: 'docker', status: 'busy', os: 'Linux', arch: 'amd64', cpu_cores: 4, memory_mb: 8192, created_at: '' },
  { id: 'a3', name: 'agent-k8s-pool', labels: ['kubernetes', 'scalable'], executor: 'kubernetes', status: 'online', os: 'Linux', arch: 'amd64', cpu_cores: 16, memory_mb: 32768, created_at: '' },
  { id: 'a4', name: 'agent-mac-01', labels: ['macos', 'xcode'], executor: 'local', status: 'offline', os: 'macOS', arch: 'arm64', cpu_cores: 10, memory_mb: 16384, created_at: '' },
];

// ---------------------------------------------------------------------------
// Stat card
// ---------------------------------------------------------------------------
interface StatCardProps {
  label: string;
  value: string | number;
  change?: string;
  changeType?: 'up' | 'down' | 'neutral';
  icon: any;
}

const StatCard: Component<StatCardProps> = (props) => {
  return (
    <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-5">
      <div class="flex items-start justify-between">
        <div>
          <p class="text-xs font-medium uppercase tracking-wider text-[var(--color-text-tertiary)]">
            {props.label}
          </p>
          <p class="text-2xl font-bold text-[var(--color-text-primary)] mt-1.5 tracking-tight">
            {props.value}
          </p>
          <Show when={props.change}>
            <p class={`text-xs mt-1.5 font-medium ${
              props.changeType === 'up' ? 'text-emerald-400' :
              props.changeType === 'down' ? 'text-red-400' :
              'text-[var(--color-text-tertiary)]'
            }`}>
              {props.changeType === 'up' && (
                <span class="inline-block mr-0.5">&#8593;</span>
              )}
              {props.changeType === 'down' && (
                <span class="inline-block mr-0.5">&#8595;</span>
              )}
              {props.change}
            </p>
          </Show>
        </div>
        <div class="p-2.5 rounded-lg bg-[var(--color-accent-bg)]">
          {props.icon}
        </div>
      </div>
    </div>
  );
};

// ---------------------------------------------------------------------------
// Table columns
// ---------------------------------------------------------------------------
const runColumns: TableColumn<PipelineRun>[] = [
  {
    key: 'status',
    header: 'Status',
    width: '120px',
    render: (row) => (
      <Badge variant={statusVariant(row.status)} dot size="sm">
        {statusLabel(row.status)}
      </Badge>
    ),
  },
  {
    key: 'pipeline',
    header: 'Pipeline',
    render: (row) => (
      <div>
        <p class="font-medium text-[var(--color-text-primary)]">
          Run #{row.number}
        </p>
        <p class="text-xs text-[var(--color-text-tertiary)] mt-0.5 truncate max-w-xs">
          {row.commit_message}
        </p>
      </div>
    ),
  },
  {
    key: 'branch',
    header: 'Branch',
    render: (row) => (
      <div class="flex items-center gap-1.5">
        <svg class="w-3.5 h-3.5 text-[var(--color-text-tertiary)] shrink-0" viewBox="0 0 16 16" fill="currentColor">
          <path fill-rule="evenodd" d="M11.75 2.5a.75.75 0 100 1.5.75.75 0 000-1.5zm-2.25.75a2.25 2.25 0 113 2.122V6A2.5 2.5 0 0110 8.5H6a1 1 0 00-1 1v1.128a2.251 2.251 0 11-1.5 0V5.372a2.25 2.25 0 111.5 0v1.836A2.492 2.492 0 016 7h4a1 1 0 001-1v-.628A2.25 2.25 0 019.5 3.25zM4.25 12a.75.75 0 100 1.5.75.75 0 000-1.5zM3.5 3.25a.75.75 0 111.5 0 .75.75 0 01-1.5 0z" />
        </svg>
        <span class="text-sm text-[var(--color-text-secondary)] font-mono text-xs">{row.branch}</span>
      </div>
    ),
  },
  {
    key: 'trigger',
    header: 'Trigger',
    render: (row) => (
      <span class="text-xs text-[var(--color-text-tertiary)] capitalize">{row.trigger_type.replace('_', ' ')}</span>
    ),
  },
  {
    key: 'duration',
    header: 'Duration',
    align: 'right' as const,
    render: (row) => (
      <span class="text-sm text-[var(--color-text-secondary)] font-mono text-xs">
        {row.status === 'running' ? (
          <span class="text-violet-400 flex items-center gap-1 justify-end">
            <svg class="animate-spin h-3 w-3" viewBox="0 0 24 24" fill="none">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            running
          </span>
        ) : formatDuration(row.duration_ms)}
      </span>
    ),
  },
  {
    key: 'time',
    header: 'Started',
    align: 'right' as const,
    render: (row) => (
      <span class="text-xs text-[var(--color-text-tertiary)]">{formatTimeAgo(row.created_at)}</span>
    ),
  },
];

// ---------------------------------------------------------------------------
// Dashboard
// ---------------------------------------------------------------------------
const Dashboard: Component = () => {
  const [runs] = createSignal(mockRuns);
  const [agents] = createSignal(mockAgents);

  const onlineAgents = () => agents().filter(a => a.status === 'online' || a.status === 'busy').length;
  const successRate = () => {
    const completed = runs().filter(r => r.status === 'success' || r.status === 'failure');
    if (completed.length === 0) return '—';
    const successes = completed.filter(r => r.status === 'success').length;
    return `${Math.round((successes / completed.length) * 100)}%`;
  };
  const avgDuration = () => {
    const durations = runs().filter(r => r.duration_ms).map(r => r.duration_ms!);
    if (durations.length === 0) return '—';
    const avg = durations.reduce((a, b) => a + b, 0) / durations.length;
    return formatDuration(avg);
  };

  return (
    <div class="animate-fade-in space-y-6">
      {/* Header */}
      <div>
        <h1 class="text-2xl font-bold text-[var(--color-text-primary)] tracking-tight">Dashboard</h1>
        <p class="text-sm text-[var(--color-text-secondary)] mt-1">
          Overview of your CI/CD pipelines and infrastructure.
        </p>
      </div>

      {/* Stat cards */}
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          label="Total Runs"
          value={runs().length}
          change="+12 this week"
          changeType="up"
          icon={
            <svg class="w-5 h-5 text-indigo-400" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M5.5 17a4.5 4.5 0 01-1.44-8.765 4.5 4.5 0 018.302-3.046 3.5 3.5 0 014.504 4.272A4 4 0 0115 17H5.5zm3.75-2.75a.75.75 0 001.5 0V9.66l1.95 2.1a.75.75 0 101.1-1.02l-3.25-3.5a.75.75 0 00-1.1 0l-3.25 3.5a.75.75 0 101.1 1.02l1.95-2.1v4.59z" clip-rule="evenodd" />
            </svg>
          }
        />
        <StatCard
          label="Success Rate"
          value={successRate()}
          change="+3% from last week"
          changeType="up"
          icon={
            <svg class="w-5 h-5 text-emerald-400" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" />
            </svg>
          }
        />
        <StatCard
          label="Avg Duration"
          value={avgDuration()}
          change="2s faster"
          changeType="up"
          icon={
            <svg class="w-5 h-5 text-amber-400" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm.75-13a.75.75 0 00-1.5 0v5c0 .414.336.75.75.75h4a.75.75 0 000-1.5h-3.25V5z" clip-rule="evenodd" />
            </svg>
          }
        />
        <StatCard
          label="Active Agents"
          value={`${onlineAgents()} / ${agents().length}`}
          change="All healthy"
          changeType="neutral"
          icon={
            <svg class="w-5 h-5 text-blue-400" viewBox="0 0 20 20" fill="currentColor">
              <path d="M4.632 3.533A2 2 0 016.577 2h6.846a2 2 0 011.945 1.533l1.976 8.234A3.489 3.489 0 0016 11.5H4c-.476 0-.93.095-1.344.267l1.976-8.234z" />
              <path fill-rule="evenodd" d="M4 13a2 2 0 100 4h12a2 2 0 100-4H4zm11.24 2a.75.75 0 01.75-.75H16a.75.75 0 01.75.75v.01a.75.75 0 01-.75.75h-.01a.75.75 0 01-.75-.75V15zm-2.25-.75a.75.75 0 00-.75.75v.01c0 .414.336.75.75.75H13a.75.75 0 00.75-.75V15a.75.75 0 00-.75-.75h-.01z" clip-rule="evenodd" />
            </svg>
          }
        />
      </div>

      {/* Recent runs */}
      <Card
        title="Recent Pipeline Runs"
        description="Latest pipeline executions across all projects"
        padding={false}
        actions={
          <A
            href="/projects"
            class="text-xs text-indigo-400 hover:text-indigo-300 transition-colors font-medium"
          >
            View all
          </A>
        }
      >
        <Table columns={runColumns} data={runs()} emptyMessage="No pipeline runs yet." />
      </Card>

      {/* Agent overview */}
      <Card
        title="Agent Status"
        description="Connected build agents and their current state"
        actions={
          <A
            href="/agents"
            class="text-xs text-indigo-400 hover:text-indigo-300 transition-colors font-medium"
          >
            Manage agents
          </A>
        }
      >
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
          <For each={agents()}>
            {(agent) => (
              <div class="flex items-center gap-3 p-3 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)]">
                <div class={`w-2.5 h-2.5 rounded-full shrink-0 ${
                  agent.status === 'online' ? 'bg-emerald-400' :
                  agent.status === 'busy' ? 'bg-violet-400 animate-pulse-dot' :
                  agent.status === 'draining' ? 'bg-amber-400' :
                  'bg-gray-500'
                }`} />
                <div class="min-w-0 flex-1">
                  <p class="text-sm font-medium text-[var(--color-text-primary)] truncate">{agent.name}</p>
                  <div class="flex items-center gap-2 mt-0.5">
                    <span class="text-xs text-[var(--color-text-tertiary)]">{agent.executor}</span>
                    <span class="text-xs text-[var(--color-text-tertiary)]">
                      {agent.os}/{agent.arch}
                    </span>
                  </div>
                </div>
                <Badge variant={agentStatusVariant(agent.status)} size="sm">
                  {agent.status}
                </Badge>
              </div>
            )}
          </For>
        </div>
      </Card>
    </div>
  );
};

export default Dashboard;
