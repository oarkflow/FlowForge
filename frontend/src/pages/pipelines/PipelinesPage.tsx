import type { Component } from 'solid-js';
import { createSignal, createResource, createMemo, For, Show, onMount } from 'solid-js';
import { useParams, A } from '@solidjs/router';
import PageContainer from '../../components/layout/PageContainer';
import Card from '../../components/ui/Card';
import Badge from '../../components/ui/Badge';
import Button from '../../components/ui/Button';
import Input from '../../components/ui/Input';
import Modal from '../../components/ui/Modal';
import Select from '../../components/ui/Select';
import Table, { type TableColumn } from '../../components/ui/Table';
import Tabs from '../../components/ui/Tabs';
import { toast } from '../../components/ui/Toast';
import { api, ApiRequestError } from '../../api/client';
import type { Pipeline, PipelineRun, RunStatus } from '../../types';
import { formatRelativeTime, formatDuration, getStatusBadgeVariant, truncateCommitSha } from '../../utils/helpers';

// ---------------------------------------------------------------------------
// Status helpers
// ---------------------------------------------------------------------------
const statusLabel: Record<RunStatus, string> = {
  success: 'Success', failure: 'Failed', cancelled: 'Cancelled', running: 'Running',
  queued: 'Queued', pending: 'Pending', skipped: 'Skipped', waiting_approval: 'Awaiting Approval',
};

// ---------------------------------------------------------------------------
// Extended pipeline type with computed run stats
// ---------------------------------------------------------------------------
interface PipelineWithStats extends Pipeline {
  last_run?: PipelineRun;
  run_count: number;
  success_rate: number;
}

// ---------------------------------------------------------------------------
// Fetcher
// ---------------------------------------------------------------------------
async function fetchPipelinesData(projectId: string): Promise<{ pipelines: PipelineWithStats[]; runs: PipelineRun[] }> {
  const pipelines = await api.pipelines.list(projectId);

  // Fetch runs for each pipeline
  const allRuns: PipelineRun[] = [];
  const pipelinesWithStats: PipelineWithStats[] = [];

  await Promise.all(
    pipelines.map(async (pipeline) => {
      try {
        const runsRes = await api.runs.list(projectId, pipeline.id, { page: '1', per_page: '50' });
        const runs = runsRes.data;
        allRuns.push(...runs);

        const successCount = runs.filter(r => r.status === 'success').length;
        const completedCount = runs.filter(r => ['success', 'failure'].includes(r.status)).length;

        pipelinesWithStats.push({
          ...pipeline,
          last_run: runs[0],
          run_count: runsRes.total,
          success_rate: completedCount > 0 ? Math.round((successCount / completedCount) * 100) : 0,
        });
      } catch {
        pipelinesWithStats.push({ ...pipeline, run_count: 0, success_rate: 0 });
      }
    })
  );

  // Sort all runs by created_at descending
  allRuns.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());

  return { pipelines: pipelinesWithStats, runs: allRuns };
}

// ---------------------------------------------------------------------------
// Run table columns
// ---------------------------------------------------------------------------
function makeRunColumns(projectId: string): TableColumn<PipelineRun>[] {
  return [
    { key: 'number', header: '#', width: '70px', render: (row) => (
      <A href={`/projects/${projectId}/pipelines/${row.pipeline_id}/runs/${row.id}`} class="text-sm font-mono font-medium text-indigo-400 hover:text-indigo-300">#{row.number}</A>
    )},
    { key: 'status', header: 'Status', width: '140px', render: (row) => (
      <Badge variant={getStatusBadgeVariant(row.status)} dot size="sm">{statusLabel[row.status]}</Badge>
    )},
    { key: 'commit', header: 'Commit', render: (row) => (
      <div class="min-w-0">
        <p class="text-sm text-[var(--color-text-primary)] truncate max-w-md">{row.commit_message}</p>
        <div class="flex items-center gap-2 mt-0.5">
          <Show when={row.commit_sha}>
            <span class="text-xs font-mono text-[var(--color-text-tertiary)]">{truncateCommitSha(row.commit_sha)}</span>
          </Show>
          <Show when={row.author}>
            <span class="text-xs text-[var(--color-text-tertiary)]">by {row.author}</span>
          </Show>
        </div>
      </div>
    )},
    { key: 'branch', header: 'Branch', render: (row) => (
      <Show when={row.branch}>
        <span class="inline-flex items-center gap-1.5 text-xs font-mono px-2 py-0.5 rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] border border-[var(--color-border-primary)]">
          <svg class="w-3 h-3" viewBox="0 0 16 16" fill="currentColor"><path fill-rule="evenodd" d="M11.75 2.5a.75.75 0 100 1.5.75.75 0 000-1.5zm-2.25.75a2.25 2.25 0 113 2.122V6A2.5 2.5 0 0110 8.5H6a1 1 0 00-1 1v1.128a2.251 2.251 0 11-1.5 0V5.372a2.25 2.25 0 111.5 0v1.836A2.492 2.492 0 016 7h4a1 1 0 001-1v-.628A2.25 2.25 0 019.5 3.25zM4.25 12a.75.75 0 100 1.5.75.75 0 000-1.5zM3.5 3.25a.75.75 0 111.5 0 .75.75 0 01-1.5 0z" /></svg>
          {row.branch}
        </span>
      </Show>
    )},
    { key: 'trigger', header: 'Trigger', width: '110px', render: (row) => (
      <span class="text-xs text-[var(--color-text-tertiary)] capitalize">{row.trigger_type.replace('_', ' ')}</span>
    )},
    { key: 'duration', header: 'Duration', width: '100px', align: 'right' as const, render: (row) => (
      <span class="text-xs font-mono text-[var(--color-text-secondary)]">
        {row.status === 'running' ? (
          <span class="text-violet-400 flex items-center gap-1 justify-end">
            <svg class="animate-spin h-3 w-3" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
            running
          </span>
        ) : row.status === 'queued' ? (
          <span class="text-gray-400">queued</span>
        ) : formatDuration(row.duration_ms)}
      </span>
    )},
    { key: 'time', header: 'Started', width: '90px', align: 'right' as const, render: (row) => (
      <span class="text-xs text-[var(--color-text-tertiary)]">{formatRelativeTime(row.created_at)}</span>
    )},
  ];
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const PipelinesPage: Component = () => {
  const params = useParams<{ id: string }>();
  const [activeTab, setActiveTab] = createSignal('pipelines');
  const [runFilter, setRunFilter] = createSignal('all');
  const [showCreate, setShowCreate] = createSignal(false);
  const [creating, setCreating] = createSignal(false);
  const [showTrigger, setShowTrigger] = createSignal(false);
  const [triggerPipelineId, setTriggerPipelineId] = createSignal('');
  const [triggerBranch, setTriggerBranch] = createSignal('main');
  const [triggering, setTriggering] = createSignal(false);

  // Create form
  const [newPipelineName, setNewPipelineName] = createSignal('');
  const [newPipelineDesc, setNewPipelineDesc] = createSignal('');
  const [newPipelineSource, setNewPipelineSource] = createSignal('db');
  const [newPipelineConfigPath, setNewPipelineConfigPath] = createSignal('.flowforge.yml');
  const [createError, setCreateError] = createSignal('');

  // Fetch data
  const [data, { refetch }] = createResource(
    () => params.id,
    (projectId) => fetchPipelinesData(projectId)
  );

  const pipelines = () => data()?.pipelines ?? [];
  const allRuns = () => data()?.runs ?? [];

  // Project name for breadcrumb
  const [projectName, setProjectName] = createSignal('Project');
  onMount(async () => {
    try {
      const project = await api.projects.get(params.id);
      setProjectName(project.name);
    } catch { /* breadcrumb fallback */ }
  });

  const runColumns = createMemo(() => makeRunColumns(params.id));

  const filteredRuns = () => {
    const tab = runFilter();
    if (tab === 'all') return allRuns();
    return allRuns().filter((r) => r.status === tab);
  };

  const statusCounts = () => {
    const counts: Record<string, number> = { all: 0, running: 0, success: 0, failure: 0 };
    for (const r of allRuns()) {
      counts.all++;
      if (r.status in counts) counts[r.status]++;
    }
    return counts;
  };

  const runFilterTabs = () => [
    { id: 'all', label: `All (${statusCounts().all})` },
    { id: 'running', label: `Running (${statusCounts().running})` },
    { id: 'success', label: `Passed (${statusCounts().success})` },
    { id: 'failure', label: `Failed (${statusCounts().failure})` },
  ];

  const handleCreate = async () => {
    if (!newPipelineName().trim()) return;
    setCreating(true);
    setCreateError('');
    try {
      await api.pipelines.create(params.id, {
        name: newPipelineName().trim(),
        description: newPipelineDesc().trim() || undefined,
        config_source: newPipelineSource() as 'db' | 'repo',
        config_path: newPipelineSource() === 'repo' ? newPipelineConfigPath().trim() : undefined,
      });
      setShowCreate(false);
      setNewPipelineName('');
      setNewPipelineDesc('');
      setNewPipelineSource('db');
      toast.success('Pipeline created');
      refetch();
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to create pipeline';
      setCreateError(msg);
      toast.error(msg);
    } finally {
      setCreating(false);
    }
  };

  const handleTrigger = async () => {
    if (!triggerPipelineId()) return;
    setTriggering(true);
    try {
      await api.pipelines.trigger(params.id, triggerPipelineId(), {
        branch: triggerBranch().trim() || 'main',
      });
      setShowTrigger(false);
      toast.success('Pipeline triggered');
      refetch();
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to trigger pipeline';
      toast.error(msg);
    } finally {
      setTriggering(false);
    }
  };

  const openTriggerModal = (pipelineId?: string) => {
    setTriggerPipelineId(pipelineId || (pipelines()[0]?.id ?? ''));
    setTriggerBranch('main');
    setShowTrigger(true);
  };

  return (
    <PageContainer
      title="Pipelines"
      description="Manage pipelines and view execution history"
      breadcrumbs={[
        { label: 'Projects', href: '/projects' },
        { label: projectName(), href: `/projects/${params.id}` },
        { label: 'Pipelines' },
      ]}
      actions={
        <div class="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => setShowCreate(true)}
            icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>}
          >
            New Pipeline
          </Button>
          <Button size="sm" onClick={() => openTriggerModal()}
            icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M6.3 2.841A1.5 1.5 0 004 4.11V15.89a1.5 1.5 0 002.3 1.269l9.344-5.89a1.5 1.5 0 000-2.538L6.3 2.84z" /></svg>}
          >
            Trigger Run
          </Button>
        </div>
      }
    >
      {/* Error state */}
      <Show when={data.error}>
        <div class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30 flex items-center justify-between">
          <p class="text-sm text-red-400">Failed to load pipelines: {(data.error as Error)?.message}</p>
          <Button size="sm" variant="outline" onClick={refetch}>Retry</Button>
        </div>
      </Show>

      <Tabs
        tabs={[
          { id: 'pipelines', label: `Pipelines (${pipelines().length})` },
          { id: 'runs', label: `All Runs (${allRuns().length})` },
        ]}
        activeTab={activeTab()}
        onTabChange={setActiveTab}
        class="mb-6"
      />

      <Show when={!data.loading} fallback={
        <div class="space-y-4">
          <For each={[1, 2, 3]}>{() => <div class="h-24 bg-[var(--color-bg-secondary)] rounded-xl animate-pulse" />}</For>
        </div>
      }>
        <Show when={activeTab() === 'pipelines'}>
          <Show when={pipelines().length > 0} fallback={
            <div class="text-center py-16">
              <svg class="w-12 h-12 mx-auto text-[var(--color-text-tertiary)] mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25H12" /></svg>
              <p class="text-[var(--color-text-secondary)] mb-2">No pipelines yet</p>
              <p class="text-sm text-[var(--color-text-tertiary)] mb-4">Create your first pipeline to automate your builds.</p>
              <Button onClick={() => setShowCreate(true)}>Create Pipeline</Button>
            </div>
          }>
            <div class="space-y-4">
              <For each={pipelines()}>
                {(pipeline) => (
                  <A href={`/projects/${params.id}/pipelines/${pipeline.id}`} class="block">
                    <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-5 hover:border-[var(--color-border-secondary)] transition-all">
                      <div class="flex items-start justify-between">
                        <div class="flex items-start gap-4">
                          <div class={`w-3 h-3 rounded-full mt-1.5 ${pipeline.is_active ? 'bg-emerald-400' : 'bg-gray-500'}`} />
                          <div>
                            <div class="flex items-center gap-2">
                              <h3 class="text-sm font-semibold text-[var(--color-text-primary)]">{pipeline.name}</h3>
                              <Badge variant={pipeline.is_active ? 'success' : 'default'} size="sm">
                                {pipeline.is_active ? 'Active' : 'Disabled'}
                              </Badge>
                            </div>
                            <Show when={pipeline.description}>
                              <p class="text-xs text-[var(--color-text-tertiary)] mt-0.5">{pipeline.description}</p>
                            </Show>

                            {/* Triggers */}
                            <div class="flex items-center gap-2 mt-2">
                              <Show when={pipeline.triggers.push}>
                                <span class="text-xs px-1.5 py-0.5 rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">push</span>
                              </Show>
                              <Show when={pipeline.triggers.pull_request}>
                                <span class="text-xs px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-400 border border-blue-500/20">PR</span>
                              </Show>
                              <Show when={pipeline.triggers.manual}>
                                <span class="text-xs px-1.5 py-0.5 rounded bg-violet-500/10 text-violet-400 border border-violet-500/20">manual</span>
                              </Show>
                              <Show when={pipeline.triggers.schedule}>
                                <span class="text-xs px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-400 border border-amber-500/20">schedule</span>
                              </Show>
                            </div>
                          </div>
                        </div>

                        <div class="flex items-center gap-6 text-right">
                          <div>
                            <p class="text-xs text-[var(--color-text-tertiary)]">{pipeline.run_count} runs</p>
                            <Show when={pipeline.success_rate > 0}>
                              <p class={`text-xs font-medium mt-0.5 ${pipeline.success_rate >= 90 ? 'text-emerald-400' : pipeline.success_rate >= 70 ? 'text-amber-400' : 'text-red-400'}`}>
                                {pipeline.success_rate}% pass rate
                              </p>
                            </Show>
                          </div>
                          <Show when={pipeline.last_run}>
                            <Badge variant={getStatusBadgeVariant(pipeline.last_run!.status)} dot size="sm">
                              {statusLabel[pipeline.last_run!.status]}
                            </Badge>
                          </Show>
                        </div>
                      </div>

                      {/* Last run info */}
                      <Show when={pipeline.last_run}>
                        <div class="flex items-center gap-3 mt-3 pt-3 border-t border-[var(--color-border-primary)] text-xs text-[var(--color-text-tertiary)]">
                          <span>#{pipeline.last_run!.number}</span>
                          <Show when={pipeline.last_run!.commit_sha}>
                            <span class="font-mono">{truncateCommitSha(pipeline.last_run!.commit_sha)}</span>
                          </Show>
                          <Show when={pipeline.last_run!.branch}>
                            <span>on {pipeline.last_run!.branch}</span>
                          </Show>
                          <Show when={pipeline.last_run!.author}>
                            <span>by {pipeline.last_run!.author}</span>
                          </Show>
                          <Show when={pipeline.last_run!.duration_ms}>
                            <span>{formatDuration(pipeline.last_run!.duration_ms)}</span>
                          </Show>
                          <span>{formatRelativeTime(pipeline.last_run!.created_at)}</span>
                        </div>
                      </Show>
                    </div>
                  </A>
                )}
              </For>
            </div>
          </Show>
        </Show>

        <Show when={activeTab() === 'runs'}>
          {/* Run stats */}
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
            <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-lg px-4 py-3">
              <p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider font-medium">Total</p>
              <p class="text-xl font-bold text-[var(--color-text-primary)] mt-1">{allRuns().length}</p>
            </div>
            <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-lg px-4 py-3">
              <p class="text-xs text-emerald-400 uppercase tracking-wider font-medium">Passed</p>
              <p class="text-xl font-bold text-emerald-400 mt-1">{allRuns().filter(r => r.status === 'success').length}</p>
            </div>
            <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-lg px-4 py-3">
              <p class="text-xs text-red-400 uppercase tracking-wider font-medium">Failed</p>
              <p class="text-xl font-bold text-red-400 mt-1">{allRuns().filter(r => r.status === 'failure').length}</p>
            </div>
            <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-lg px-4 py-3">
              <p class="text-xs text-violet-400 uppercase tracking-wider font-medium">Running</p>
              <p class="text-xl font-bold text-violet-400 mt-1">{allRuns().filter(r => r.status === 'running').length}</p>
            </div>
          </div>

          {/* Filter + table */}
          <Card padding={false}>
            <Tabs tabs={runFilterTabs()} activeTab={runFilter()} onTabChange={setRunFilter} class="px-4" />
            <Table columns={runColumns()} data={filteredRuns()} emptyMessage="No pipeline runs match the current filter." />
          </Card>
        </Show>
      </Show>

      {/* Create Pipeline Modal */}
      <Show when={showCreate()}>
        <Modal open={showCreate()} onClose={() => { setShowCreate(false); setCreateError(''); }} title="Create Pipeline" description="Set up a new CI/CD pipeline" footer={
          <>
            <Button variant="ghost" onClick={() => { setShowCreate(false); setCreateError(''); }}>Cancel</Button>
            <Button disabled={!newPipelineName().trim()} loading={creating()} onClick={handleCreate}>Create Pipeline</Button>
          </>
        }>
          <div class="space-y-4">
            <Show when={createError()}>
              <div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-sm text-red-400">{createError()}</div>
            </Show>
            <Input label="Pipeline Name" placeholder="e.g. CI Pipeline" value={newPipelineName()} onInput={(e) => setNewPipelineName(e.currentTarget.value)} />
            <Input label="Description" placeholder="Brief description..." value={newPipelineDesc()} onInput={(e) => setNewPipelineDesc(e.currentTarget.value)} />
            <Select label="Configuration Source" value={newPipelineSource()} onChange={(e) => setNewPipelineSource(e.currentTarget.value)} options={[
              { value: 'db', label: 'Database — Edit in UI' },
              { value: 'repo', label: 'Repository — Read from .flowforge.yml' },
            ]} />
            <Show when={newPipelineSource() === 'repo'}>
              <Input label="Config Path" value={newPipelineConfigPath()} onInput={(e) => setNewPipelineConfigPath(e.currentTarget.value)} hint="Path to the pipeline YAML file in the repository" />
            </Show>
          </div>
        </Modal>
      </Show>

      {/* Trigger Pipeline Modal */}
      <Show when={showTrigger()}>
        <Modal open={showTrigger()} onClose={() => setShowTrigger(false)} title="Trigger Pipeline" description="Manually start a new pipeline run" footer={
          <>
            <Button variant="ghost" onClick={() => setShowTrigger(false)}>Cancel</Button>
            <Button onClick={handleTrigger} loading={triggering()}>Trigger Run</Button>
          </>
        }>
          <div class="space-y-4">
            <Show when={pipelines().length > 1}>
              <Select label="Pipeline" value={triggerPipelineId()} onChange={(e) => setTriggerPipelineId(e.currentTarget.value)} options={
                pipelines().map(p => ({ value: p.id, label: p.name }))
              } />
            </Show>
            <Input label="Branch" value={triggerBranch()} onInput={(e) => setTriggerBranch(e.currentTarget.value)} placeholder="main" />
            <p class="text-xs text-[var(--color-text-tertiary)]">The pipeline will execute against the latest commit on this branch.</p>
          </div>
        </Modal>
      </Show>
    </PageContainer>
  );
};

export default PipelinesPage;
