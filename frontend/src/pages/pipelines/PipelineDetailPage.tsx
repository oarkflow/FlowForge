import type { Component } from 'solid-js';
import { createSignal, createResource, For, Show, Switch, Match, onMount } from 'solid-js';
import { useParams, useNavigate, A } from '@solidjs/router';
import PageContainer from '../../components/layout/PageContainer';
import Card from '../../components/ui/Card';
import Badge from '../../components/ui/Badge';
import Button from '../../components/ui/Button';
import Tabs from '../../components/ui/Tabs';
import Table, { type TableColumn } from '../../components/ui/Table';
import Modal from '../../components/ui/Modal';
import Input from '../../components/ui/Input';
import { toast } from '../../components/ui/Toast';
import { api, ApiRequestError, type RunDetail } from '../../api/client';
import type { Pipeline, PipelineRun, PipelineVersion, RunStatus } from '../../types';
import { formatRelativeTime, formatDuration, getStatusBadgeVariant, truncateCommitSha } from '../../utils/helpers';

const statusLabel: Record<RunStatus, string> = {
  success: 'Success', failure: 'Failed', cancelled: 'Cancelled', running: 'Running',
  queued: 'Queued', pending: 'Pending', skipped: 'Skipped', waiting_approval: 'Awaiting Approval',
};

// ---------------------------------------------------------------------------
// Fetcher
// ---------------------------------------------------------------------------
interface PipelineData {
  pipeline: Pipeline;
  runs: PipelineRun[];
  versions: PipelineVersion[];
  totalRuns: number;
}

async function fetchPipelineData(ids: { projectId: string; pipelineId: string }): Promise<PipelineData> {
  const [pipeline, runsRes, versions] = await Promise.all([
    api.pipelines.get(ids.projectId, ids.pipelineId),
    api.runs.list(ids.projectId, ids.pipelineId, { page: '1', per_page: '50' }),
    api.pipelines.listVersions(ids.projectId, ids.pipelineId).catch(() => [] as PipelineVersion[]),
  ]);
  return { pipeline, runs: runsRes.data, versions, totalRuns: runsRes.total };
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const PipelineDetailPage: Component = () => {
  const params = useParams<{ id: string; pid: string }>();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = createSignal('configuration');
  const [showTrigger, setShowTrigger] = createSignal(false);
  const [triggerBranch, setTriggerBranch] = createSignal('main');
  const [triggering, setTriggering] = createSignal(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = createSignal(false);
  const [deleting, setDeleting] = createSignal(false);
  const [saving, setSaving] = createSignal(false);

  // Settings form
  const [editName, setEditName] = createSignal('');
  const [editDesc, setEditDesc] = createSignal('');
  const [editActive, setEditActive] = createSignal(true);

  // Project name for breadcrumb
  const [projectName, setProjectName] = createSignal('Project');
  onMount(async () => {
    try {
      const project = await api.projects.get(params.id);
      setProjectName(project.name);
    } catch { /* fallback */ }
  });

  // Fetch pipeline data
  const [data, { refetch }] = createResource(
    () => ({ projectId: params.id, pipelineId: params.pid }),
    fetchPipelineData
  );

  const pipeline = () => data()?.pipeline;
  const runs = () => data()?.runs ?? [];
  const versions = () => data()?.versions ?? [];
  const totalRuns = () => data()?.totalRuns ?? 0;

  // Sync settings form when data loads
  const syncSettings = () => {
    const p = pipeline();
    if (p) {
      setEditName(p.name);
      setEditDesc(p.description || '');
      setEditActive(p.is_active);
    }
  };

  // Run columns
  const runColumns: TableColumn<PipelineRun>[] = [
    { key: 'number', header: '#', width: '70px', render: (row) => (
      <A href={`/projects/${params.id}/pipelines/${params.pid}/runs/${row.id}`} class="text-sm font-mono font-medium text-indigo-400 hover:text-indigo-300">#{row.number}</A>
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
        <span class="inline-flex items-center gap-1 text-xs font-mono px-2 py-0.5 rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] border border-[var(--color-border-primary)]">
          {row.branch}
        </span>
      </Show>
    )},
    { key: 'duration', header: 'Duration', width: '100px', align: 'right' as const, render: (row) => (
      <span class="text-xs font-mono text-[var(--color-text-secondary)]">
        {row.status === 'running' ? (
          <span class="text-violet-400 flex items-center gap-1 justify-end">
            <svg class="animate-spin h-3 w-3" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
            running
          </span>
        ) : row.status === 'queued' ? 'queued' : formatDuration(row.duration_ms)}
      </span>
    )},
    { key: 'time', header: 'Started', width: '90px', align: 'right' as const, render: (row) => (
      <span class="text-xs text-[var(--color-text-tertiary)]">{formatRelativeTime(row.created_at)}</span>
    )},
  ];

  const tabs = () => [
    { id: 'configuration', label: 'Configuration' },
    { id: 'runs', label: `Runs (${totalRuns()})` },
    { id: 'triggers', label: 'Triggers' },
    { id: 'versions', label: `Versions (${versions().length})` },
    { id: 'settings', label: 'Settings' },
  ];

  const handleTrigger = async () => {
    setTriggering(true);
    try {
      await api.pipelines.trigger(params.id, params.pid, { branch: triggerBranch().trim() || 'main' });
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

  const handleSaveSettings = async () => {
    setSaving(true);
    try {
      await api.pipelines.update(params.id, params.pid, {
        name: editName().trim(),
        description: editDesc().trim() || undefined,
        is_active: editActive(),
      });
      toast.success('Pipeline settings saved');
      refetch();
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to save settings';
      toast.error(msg);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await api.pipelines.delete(params.id, params.pid);
      toast.success('Pipeline deleted');
      navigate(`/projects/${params.id}/pipelines`);
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to delete pipeline';
      toast.error(msg);
    } finally {
      setDeleting(false);
      setShowDeleteConfirm(false);
    }
  };

  // Extract triggers from pipeline config
  const triggers = () => pipeline()?.triggers ?? {};
  const pushTrigger = () => triggers().push as Record<string, unknown> | undefined;
  const prTrigger = () => triggers().pull_request as Record<string, unknown> | undefined;
  const scheduleTrigger = () => triggers().schedule as unknown[] | undefined;
  const manualTrigger = () => triggers().manual;

  // Stats computed from runs
  const successRate = () => {
    const r = runs();
    const completed = r.filter(x => x.status === 'success' || x.status === 'failure');
    if (completed.length === 0) return 0;
    return Math.round(completed.filter(x => x.status === 'success').length / completed.length * 100);
  };

  const avgDuration = () => {
    const durations = runs().filter(r => r.duration_ms).map(r => r.duration_ms!);
    if (durations.length === 0) return 0;
    return Math.round(durations.reduce((a, b) => a + b, 0) / durations.length);
  };

  return (
    <PageContainer
      title={pipeline()?.name ?? 'Pipeline'}
      description={pipeline()?.description}
      breadcrumbs={[
        { label: 'Projects', href: '/projects' },
        { label: projectName(), href: `/projects/${params.id}` },
        { label: 'Pipelines', href: `/projects/${params.id}/pipelines` },
        { label: pipeline()?.name ?? 'Pipeline' },
      ]}
      actions={
        <div class="flex items-center gap-2">
          <Show when={pipeline()}>
            <Badge variant={pipeline()!.is_active ? 'success' : 'default'} dot>
              {pipeline()!.is_active ? 'Active' : 'Disabled'}
            </Badge>
          </Show>
          <Button size="sm" variant="outline" onClick={() => setShowTrigger(true)}
            icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M6.3 2.841A1.5 1.5 0 004 4.11V15.89a1.5 1.5 0 002.3 1.269l9.344-5.89a1.5 1.5 0 000-2.538L6.3 2.84z" /></svg>}
          >
            Trigger
          </Button>
        </div>
      }
    >
      {/* Error state */}
      <Show when={data.error}>
        <div class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30 flex items-center justify-between">
          <p class="text-sm text-red-400">Failed to load pipeline: {(data.error as Error)?.message}</p>
          <Button size="sm" variant="outline" onClick={refetch}>Retry</Button>
        </div>
      </Show>

      <Tabs tabs={tabs()} activeTab={activeTab()} onTabChange={(tab) => { setActiveTab(tab); if (tab === 'settings') syncSettings(); }} class="mb-6" />

      <Show when={!data.loading} fallback={
        <div class="h-96 bg-[var(--color-bg-secondary)] rounded-xl animate-pulse" />
      }>
        <Switch>
          {/* ---- Configuration (YAML Viewer) ---- */}
          <Match when={activeTab() === 'configuration'}>
            <div class="grid grid-cols-1 lg:grid-cols-4 gap-6">
              <div class="lg:col-span-3">
                <Card title="Pipeline Configuration" description={`Version ${pipeline()?.config_version ?? '-'} · Source: ${pipeline()?.config_source === 'repo' ? pipeline()?.config_path : 'Database'}`}>
                  <Show when={pipeline()?.config_content} fallback={
                    <div class="text-center py-8 text-[var(--color-text-tertiary)]">
                      <p class="text-sm">No configuration content available.</p>
                      <p class="text-xs mt-1">
                        {pipeline()?.config_source === 'repo' ? 'Configuration is read from the repository.' : 'Edit the pipeline configuration below.'}
                      </p>
                    </div>
                  }>
                    <pre class="bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] rounded-lg p-4 overflow-auto max-h-[600px] text-sm font-mono leading-relaxed">
                      <code class="text-[var(--color-text-secondary)]">{pipeline()!.config_content}</code>
                    </pre>
                  </Show>
                </Card>
              </div>

              <div class="space-y-4">
                <Card title="Pipeline Info">
                  <div class="space-y-3 text-sm">
                    <div class="flex justify-between"><span class="text-[var(--color-text-tertiary)]">ID</span><span class="font-mono text-[var(--color-text-secondary)] text-xs">{pipeline()?.id}</span></div>
                    <div class="flex justify-between"><span class="text-[var(--color-text-tertiary)]">Version</span><span class="text-[var(--color-text-secondary)]">{pipeline()?.config_version}</span></div>
                    <div class="flex justify-between"><span class="text-[var(--color-text-tertiary)]">Source</span><span class="text-[var(--color-text-secondary)] capitalize">{pipeline()?.config_source}</span></div>
                    <Show when={pipeline()?.created_at}>
                      <div class="flex justify-between"><span class="text-[var(--color-text-tertiary)]">Created</span><span class="text-[var(--color-text-secondary)]">{formatRelativeTime(pipeline()!.created_at)}</span></div>
                    </Show>
                    <Show when={pipeline()?.updated_at}>
                      <div class="flex justify-between"><span class="text-[var(--color-text-tertiary)]">Updated</span><span class="text-[var(--color-text-secondary)]">{formatRelativeTime(pipeline()!.updated_at)}</span></div>
                    </Show>
                  </div>
                </Card>

                <Card title="Quick Stats">
                  <div class="space-y-2 text-sm">
                    <div class="flex justify-between"><span class="text-[var(--color-text-tertiary)]">Total Runs</span><span class="text-[var(--color-text-primary)] font-medium">{totalRuns()}</span></div>
                    <div class="flex justify-between"><span class="text-[var(--color-text-tertiary)]">Success Rate</span><span class="text-emerald-400 font-medium">{successRate()}%</span></div>
                    <div class="flex justify-between"><span class="text-[var(--color-text-tertiary)]">Avg Duration</span><span class="text-[var(--color-text-primary)] font-medium">{formatDuration(avgDuration())}</span></div>
                  </div>
                </Card>
              </div>
            </div>
          </Match>

          {/* ---- Runs tab ---- */}
          <Match when={activeTab() === 'runs'}>
            <Card padding={false}>
              <Table columns={runColumns} data={runs()} emptyMessage="No runs yet" />
            </Card>
          </Match>

          {/* ---- Triggers tab ---- */}
          <Match when={activeTab() === 'triggers'}>
            <div class="space-y-4">
              <Show when={pushTrigger()}>
                <Card title="Push Trigger">
                  <div class="space-y-3">
                    <Show when={(pushTrigger() as any)?.branches}>
                      <div>
                        <p class="text-sm font-medium text-[var(--color-text-primary)] mb-1">Branches</p>
                        <div class="flex flex-wrap gap-2">
                          <For each={(pushTrigger() as any).branches as string[]}>
                            {(branch) => (
                              <span class="text-xs font-mono px-2 py-1 rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] border border-[var(--color-border-primary)]">{branch}</span>
                            )}
                          </For>
                        </div>
                      </div>
                    </Show>
                    <Show when={(pushTrigger() as any)?.paths}>
                      <div>
                        <p class="text-sm font-medium text-[var(--color-text-primary)] mb-1">Path Filters</p>
                        <div class="flex flex-wrap gap-2">
                          <For each={(pushTrigger() as any).paths as string[]}>
                            {(path) => (
                              <span class="text-xs font-mono px-2 py-1 rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] border border-[var(--color-border-primary)]">{path}</span>
                            )}
                          </For>
                        </div>
                      </div>
                    </Show>
                  </div>
                </Card>
              </Show>

              <Show when={prTrigger()}>
                <Card title="Pull Request Trigger">
                  <div class="space-y-3">
                    <Show when={(prTrigger() as any)?.types}>
                      <div>
                        <p class="text-sm font-medium text-[var(--color-text-primary)] mb-1">Events</p>
                        <div class="flex flex-wrap gap-2">
                          <For each={(prTrigger() as any).types as string[]}>
                            {(evt) => <Badge variant="info" size="sm">{evt}</Badge>}
                          </For>
                        </div>
                      </div>
                    </Show>
                    <Show when={(prTrigger() as any)?.branches}>
                      <div>
                        <p class="text-sm font-medium text-[var(--color-text-primary)] mb-1">Target Branches</p>
                        <div class="flex flex-wrap gap-2">
                          <For each={(prTrigger() as any).branches as string[]}>
                            {(branch) => (
                              <span class="text-xs font-mono px-2 py-1 rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] border border-[var(--color-border-primary)]">{branch}</span>
                            )}
                          </For>
                        </div>
                      </div>
                    </Show>
                  </div>
                </Card>
              </Show>

              <Show when={scheduleTrigger()}>
                <Card title="Schedule Trigger">
                  <div class="space-y-2">
                    <For each={scheduleTrigger() as any[]}>
                      {(sched) => (
                        <div class="flex items-center gap-3 text-sm">
                          <span class="font-mono text-xs px-2 py-1 rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] border border-[var(--color-border-primary)]">{sched.cron}</span>
                          <Show when={sched.timezone}>
                            <span class="text-[var(--color-text-tertiary)]">{sched.timezone}</span>
                          </Show>
                        </div>
                      )}
                    </For>
                  </div>
                </Card>
              </Show>

              <Card title="Manual Trigger">
                <p class="text-sm text-[var(--color-text-secondary)] mb-3">
                  {manualTrigger() ? 'This pipeline can be triggered manually from the UI or API.' : 'Manual triggering is available for all pipelines.'}
                </p>
                <Button onClick={() => setShowTrigger(true)} icon={
                  <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M6.3 2.841A1.5 1.5 0 004 4.11V15.89a1.5 1.5 0 002.3 1.269l9.344-5.89a1.5 1.5 0 000-2.538L6.3 2.84z" /></svg>
                }>Trigger Run</Button>
              </Card>

              <Show when={!pushTrigger() && !prTrigger() && !scheduleTrigger() && !manualTrigger()}>
                <div class="text-center py-8 text-[var(--color-text-tertiary)]">
                  <p class="text-sm">No triggers configured for this pipeline.</p>
                </div>
              </Show>
            </div>
          </Match>

          {/* ---- Versions tab ---- */}
          <Match when={activeTab() === 'versions'}>
            <Card title="Version History" padding={false}>
              <Show when={versions().length > 0} fallback={
                <div class="text-center py-8 text-[var(--color-text-tertiary)]">
                  <p class="text-sm">No version history available.</p>
                </div>
              }>
                <div class="divide-y divide-[var(--color-border-primary)]">
                  <For each={versions()}>
                    {(version) => (
                      <div class="flex items-center justify-between px-5 py-4 hover:bg-[var(--color-bg-hover)]">
                        <div class="flex items-center gap-4">
                          <div class="w-8 h-8 rounded-full bg-indigo-500/10 flex items-center justify-center">
                            <span class="text-xs font-bold text-indigo-400">v{version.version}</span>
                          </div>
                          <div>
                            <p class="text-sm font-medium text-[var(--color-text-primary)]">{version.message || 'No message'}</p>
                            <p class="text-xs text-[var(--color-text-tertiary)] mt-0.5">
                              <Show when={version.created_by}>by {version.created_by} · </Show>
                              {formatRelativeTime(version.created_at)}
                            </p>
                          </div>
                        </div>
                        <div class="flex items-center gap-2">
                          <Show when={version.version === pipeline()?.config_version}>
                            <Badge variant="success" size="sm">Current</Badge>
                          </Show>
                        </div>
                      </div>
                    )}
                  </For>
                </div>
              </Show>
            </Card>
          </Match>

          {/* ---- Settings tab ---- */}
          <Match when={activeTab() === 'settings'}>
            <div class="space-y-6 max-w-2xl">
              <Card title="Pipeline Settings">
                <div class="space-y-4">
                  <Input label="Pipeline Name" value={editName()} onInput={(e) => setEditName(e.currentTarget.value)} />
                  <Input label="Description" value={editDesc()} onInput={(e) => setEditDesc(e.currentTarget.value)} />
                  <div class="flex items-center justify-between p-3 rounded-lg bg-[var(--color-bg-tertiary)]">
                    <div>
                      <p class="text-sm font-medium text-[var(--color-text-primary)]">Active</p>
                      <p class="text-xs text-[var(--color-text-tertiary)]">Pipeline will be triggered based on configured rules</p>
                    </div>
                    <button
                      class={`w-10 h-6 rounded-full relative cursor-pointer transition-colors ${editActive() ? 'bg-indigo-500' : 'bg-gray-600'}`}
                      onClick={() => setEditActive(!editActive())}
                    >
                      <div class={`absolute top-1 w-4 h-4 rounded-full bg-white transition-transform ${editActive() ? 'left-5' : 'left-1'}`} />
                    </button>
                  </div>
                  <Button onClick={handleSaveSettings} loading={saving()}>Save Changes</Button>
                </div>
              </Card>
              <Card title="Danger Zone">
                <div class="flex items-center justify-between p-4 rounded-lg border border-red-500/30 bg-red-500/5">
                  <div>
                    <p class="text-sm font-medium text-red-400">Delete Pipeline</p>
                    <p class="text-xs text-[var(--color-text-tertiary)] mt-1">All run history and configuration will be permanently deleted.</p>
                  </div>
                  <Button variant="danger" size="sm" onClick={() => setShowDeleteConfirm(true)}>Delete</Button>
                </div>
              </Card>
            </div>
          </Match>
        </Switch>
      </Show>

      {/* Trigger Modal */}
      <Show when={showTrigger()}>
        <Modal open={showTrigger()} onClose={() => setShowTrigger(false)} title="Trigger Pipeline" description="Manually start a new pipeline run" footer={
          <>
            <Button variant="ghost" onClick={() => setShowTrigger(false)}>Cancel</Button>
            <Button onClick={handleTrigger} loading={triggering()}>Trigger Run</Button>
          </>
        }>
          <div class="space-y-4">
            <Input label="Branch" value={triggerBranch()} onInput={(e) => setTriggerBranch(e.currentTarget.value)} placeholder="main" />
            <p class="text-xs text-[var(--color-text-tertiary)]">The pipeline will execute against the latest commit on this branch.</p>
          </div>
        </Modal>
      </Show>

      {/* Delete Confirmation Modal */}
      <Show when={showDeleteConfirm()}>
        <Modal open={showDeleteConfirm()} onClose={() => setShowDeleteConfirm(false)} title="Delete Pipeline" footer={
          <>
            <Button variant="ghost" onClick={() => setShowDeleteConfirm(false)}>Cancel</Button>
            <Button variant="danger" onClick={handleDelete} loading={deleting()}>Delete Pipeline</Button>
          </>
        }>
          <p class="text-sm text-[var(--color-text-secondary)]">
            Are you sure you want to delete <strong class="text-[var(--color-text-primary)]">{pipeline()?.name}</strong>? This will permanently delete all run history, versions, and configuration. This action cannot be undone.
          </p>
        </Modal>
      </Show>
    </PageContainer>
  );
};

export default PipelineDetailPage;
