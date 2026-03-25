import type { Component } from 'solid-js';
import { createSignal, createResource, onMount, onCleanup, Show, For, createMemo } from 'solid-js';
import { A, useNavigate } from '@solidjs/router';
import PageContainer from '../../components/layout/PageContainer';
import Card from '../../components/ui/Card';
import Badge from '../../components/ui/Badge';
import Button from '../../components/ui/Button';
import Table, { type TableColumn } from '../../components/ui/Table';
import Tabs from '../../components/ui/Tabs';
import { toast } from '../../components/ui/Toast';
import { api, ApiRequestError } from '../../api/client';
import { getEventSocket } from '../../api/websocket';
import type { RunWithMeta, RunStatus, Project } from '../../types';
import {
  formatDuration,
  formatRelativeTime,
  getStatusBgColor,
  getStatusBadgeVariant,
  truncateCommitSha,
} from '../../utils/helpers';

// ---------------------------------------------------------------------------
// Data fetcher — tries global endpoint, falls back to per-project aggregation
// ---------------------------------------------------------------------------
async function fetchAllRuns(): Promise<RunWithMeta[]> {
  // Try the global endpoint first
  try {
    const res = await api.runs.listAll({ per_page: '100' });
    return res.data;
  } catch (err) {
    if (err instanceof ApiRequestError && err.status === 404) {
      // Fallback: aggregate across projects
    } else {
      throw err;
    }
  }

  // Fallback: fetch projects → pipelines → runs
  const runs: RunWithMeta[] = [];
  const projectsRes = await api.projects.list({ page: '1', per_page: '100' });

  for (const project of projectsRes.data) {
    try {
      const pipelines = await api.pipelines.list(project.id);
      for (const pipeline of pipelines) {
        try {
          const runsRes = await api.runs.list(project.id, pipeline.id, { per_page: '20' });
          for (const run of runsRes.data) {
            runs.push({
              ...run,
              pipeline_name: pipeline.name,
              project_id: project.id,
              project_name: project.name,
            });
          }
        } catch { /* pipeline may have no runs */ }
      }
    } catch { /* project may have no pipelines */ }
  }

  runs.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
  return runs;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const RunsPage: Component = () => {
  const navigate = useNavigate();
  const [allRuns, { refetch }] = createResource(fetchAllRuns);
  const [activeTab, setActiveTab] = createSignal('all');
  const [searchQuery, setSearchQuery] = createSignal('');
  const [cancellingId, setCancellingId] = createSignal<string | null>(null);
  const [rerunningId, setRerunningId] = createSignal<string | null>(null);

  // Subscribe to real-time events
  onMount(() => {
    const ws = getEventSocket();
    if (!ws.isConnected) ws.connect();

    const unsub1 = ws.on('run.completed', () => { refetch(); });
    const unsub2 = ws.on('run.started', () => { refetch(); });

    onCleanup(() => { unsub1(); unsub2(); });
  });

  const runs = () => allRuns() ?? [];

  // Stats
  const stats = createMemo(() => {
    const r = runs();
    return {
      total: r.length,
      running: r.filter(x => x.status === 'running' || x.status === 'queued').length,
      passed: r.filter(x => x.status === 'success').length,
      failed: r.filter(x => x.status === 'failure').length,
    };
  });

  // Filter by tab
  const tabFiltered = createMemo(() => {
    const tab = activeTab();
    const r = runs();
    if (tab === 'all') return r;
    if (tab === 'running') return r.filter(x => x.status === 'running' || x.status === 'queued');
    if (tab === 'success') return r.filter(x => x.status === 'success');
    if (tab === 'failure') return r.filter(x => x.status === 'failure');
    return r;
  });

  // Filter by search query
  const filteredRuns = createMemo(() => {
    const q = searchQuery().toLowerCase().trim();
    if (!q) return tabFiltered();
    return tabFiltered().filter(r =>
      (r.pipeline_name ?? '').toLowerCase().includes(q) ||
      (r.project_name ?? '').toLowerCase().includes(q) ||
      (r.commit_message ?? '').toLowerCase().includes(q) ||
      (r.branch ?? '').toLowerCase().includes(q) ||
      (r.commit_sha ?? '').toLowerCase().includes(q)
    );
  });

  const tabs = createMemo(() => [
    { id: 'all', label: `All (${stats().total})` },
    { id: 'running', label: `Running (${stats().running})` },
    { id: 'success', label: `Passed (${stats().passed})` },
    { id: 'failure', label: `Failed (${stats().failed})` },
  ]);

  const handleCancel = async (e: Event, run: RunWithMeta) => {
    e.stopPropagation();
    setCancellingId(run.id);
    try {
      await api.runs.cancel(run.project_id, run.pipeline_id, run.id);
      toast.success(`Run #${run.number} cancelled`);
      refetch();
    } catch (err) {
      toast.error(err instanceof ApiRequestError ? err.message : 'Failed to cancel run');
    } finally {
      setCancellingId(null);
    }
  };

  const handleRerun = async (e: Event, run: RunWithMeta) => {
    e.stopPropagation();
    setRerunningId(run.id);
    try {
      const newRun = await api.runs.rerun(run.project_id, run.pipeline_id, run.id);
      toast.success(`Re-run started as #${newRun.number}`);
      refetch();
    } catch (err) {
      toast.error(err instanceof ApiRequestError ? err.message : 'Failed to rerun');
    } finally {
      setRerunningId(null);
    }
  };

  const runColumns: TableColumn<RunWithMeta>[] = [
    {
      key: 'status',
      header: '',
      width: '40px',
      align: 'center',
      render: (row) => (
        <span
          class={`w-2.5 h-2.5 rounded-full inline-block ${getStatusBgColor(row.status)} ${row.status === 'running' ? 'animate-pulse' : ''}`}
        />
      ),
    },
    {
      key: 'pipeline',
      header: 'Pipeline',
      render: (row) => (
        <div>
          <div class="flex items-center gap-2">
            <span class="font-medium text-[var(--color-text-primary)]">{row.pipeline_name}</span>
            <span class="text-[var(--color-text-tertiary)]">#{row.number}</span>
          </div>
          <div class="text-xs text-[var(--color-text-tertiary)] mt-0.5 truncate max-w-xs">
            {row.project_name}
            <Show when={row.commit_message}>
              {' '}&mdash; {row.commit_message}
            </Show>
          </div>
        </div>
      ),
    },
    {
      key: 'branch',
      header: 'Branch',
      render: (row) =>
        row.branch ? (
          <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-[var(--color-bg-tertiary)] text-xs text-[var(--color-text-secondary)] font-mono">
            {row.branch}
          </span>
        ) : (
          <span class="text-[var(--color-text-tertiary)]">-</span>
        ),
    },
    {
      key: 'commit',
      header: 'Commit',
      render: (row) => (
        <span class="text-xs font-mono text-[var(--color-text-tertiary)]">
          {truncateCommitSha(row.commit_sha)}
        </span>
      ),
    },
    {
      key: 'trigger',
      header: 'Trigger',
      render: (row) => (
        <span class="text-xs text-[var(--color-text-secondary)] capitalize">
          {row.trigger_type.replace('_', ' ')}
        </span>
      ),
    },
    {
      key: 'duration',
      header: 'Duration',
      align: 'right',
      render: (row) => (
        <span class="text-xs font-mono text-[var(--color-text-secondary)]">
          {row.status === 'running' ? (
            <span class="text-violet-400 flex items-center gap-1 justify-end">
              <svg class="animate-spin h-3 w-3" viewBox="0 0 24 24" fill="none">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              running
            </span>
          ) : (
            formatDuration(row.duration_ms)
          )}
        </span>
      ),
    },
    {
      key: 'time',
      header: 'Started',
      align: 'right',
      render: (row) => (
        <span class="text-xs text-[var(--color-text-tertiary)]">
          {formatRelativeTime(row.started_at ?? row.created_at)}
        </span>
      ),
    },
    {
      key: 'actions',
      header: '',
      width: '90px',
      align: 'right',
      render: (row) => (
        <div class="flex items-center gap-1 justify-end">
          <Show when={row.status === 'running' || row.status === 'queued'}>
            <button
              class="px-2 py-1 text-xs rounded bg-red-500/10 text-red-400 hover:bg-red-500/20 transition-colors cursor-pointer"
              onClick={(e) => handleCancel(e, row)}
              disabled={cancellingId() === row.id}
            >
              {cancellingId() === row.id ? '...' : 'Cancel'}
            </button>
          </Show>
          <Show when={row.status === 'failure' || row.status === 'cancelled'}>
            <button
              class="px-2 py-1 text-xs rounded bg-indigo-500/10 text-indigo-400 hover:bg-indigo-500/20 transition-colors cursor-pointer"
              onClick={(e) => handleRerun(e, row)}
              disabled={rerunningId() === row.id}
            >
              {rerunningId() === row.id ? '...' : 'Rerun'}
            </button>
          </Show>
        </div>
      ),
    },
  ];

  return (
    <PageContainer
      title="Builds"
      description="All pipeline runs across your projects"
      actions={
        <div class="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => { refetch(); toast.info('Refreshing...'); }}
            loading={allRuns.loading}
          >
            Refresh
          </Button>
        </div>
      }
    >
      {/* Error state */}
      <Show when={allRuns.error}>
        <div class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30">
          <div class="flex items-center justify-between">
            <p class="text-sm text-red-400">
              Failed to load runs: {(allRuns.error as Error)?.message || 'Unknown error'}
            </p>
            <Button size="sm" variant="outline" onClick={() => refetch()}>Retry</Button>
          </div>
        </div>
      </Show>

      {/* Stats cards */}
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
        <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-lg px-4 py-3">
          <p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider font-medium">Total</p>
          <p class="text-xl font-bold text-[var(--color-text-primary)] mt-1">{stats().total}</p>
        </div>
        <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-lg px-4 py-3">
          <p class="text-xs text-violet-400 uppercase tracking-wider font-medium">Running</p>
          <p class="text-xl font-bold text-violet-400 mt-1">{stats().running}</p>
        </div>
        <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-lg px-4 py-3">
          <p class="text-xs text-emerald-400 uppercase tracking-wider font-medium">Passed</p>
          <p class="text-xl font-bold text-emerald-400 mt-1">{stats().passed}</p>
        </div>
        <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-lg px-4 py-3">
          <p class="text-xs text-red-400 uppercase tracking-wider font-medium">Failed</p>
          <p class="text-xl font-bold text-red-400 mt-1">{stats().failed}</p>
        </div>
      </div>

      {/* Search */}
      <div class="mb-4">
        <div class="relative">
          <svg
            class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--color-text-tertiary)]"
            viewBox="0 0 20 20"
            fill="currentColor"
          >
            <path
              fill-rule="evenodd"
              d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z"
              clip-rule="evenodd"
            />
          </svg>
          <input
            type="text"
            placeholder="Search by pipeline, project, commit, branch..."
            value={searchQuery()}
            onInput={(e) => setSearchQuery(e.currentTarget.value)}
            class="w-full pl-10 pr-4 py-2 text-sm bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-lg text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)] focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
          />
        </div>
      </div>

      {/* Tabs + Table */}
      <Card padding={false}>
        <Tabs
          tabs={tabs()}
          activeTab={activeTab()}
          onTabChange={setActiveTab}
          class="px-4"
        />
        <Show
          when={!allRuns.loading}
          fallback={
            <div class="p-6 space-y-3">
              {Array.from({ length: 8 }).map(() => (
                <div class="h-12 bg-[var(--color-bg-tertiary)] rounded animate-pulse" />
              ))}
            </div>
          }
        >
          <Table
            columns={runColumns}
            data={filteredRuns()}
            emptyMessage="No pipeline runs match the current filter."
            onRowClick={(row) => {
              navigate(`/projects/${row.project_id}/pipelines/${row.pipeline_id}/runs/${row.id}`);
            }}
          />
        </Show>
      </Card>
    </PageContainer>
  );
};

export default RunsPage;
