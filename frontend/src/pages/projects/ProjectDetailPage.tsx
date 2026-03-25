import type { Component } from 'solid-js';
import { createSignal, createResource, For, Show, Switch, Match } from 'solid-js';
import { useParams, A, useNavigate } from '@solidjs/router';
import PageContainer from '../../components/layout/PageContainer';
import Card from '../../components/ui/Card';
import Badge from '../../components/ui/Badge';
import Button from '../../components/ui/Button';
import Input from '../../components/ui/Input';
import Modal from '../../components/ui/Modal';
import Select from '../../components/ui/Select';
import Tabs from '../../components/ui/Tabs';
import Table, { type TableColumn } from '../../components/ui/Table';
import { toast } from '../../components/ui/Toast';
import { api, ApiRequestError } from '../../api/client';
import type { Pipeline, PipelineRun, Repository, Secret, NotificationChannel, RunStatus } from '../../types';
import { formatRelativeTime, formatDuration, getStatusBadgeVariant, truncateCommitSha } from '../../utils/helpers';

const statusLabel: Record<RunStatus, string> = {
  success: 'Success', failure: 'Failed', cancelled: 'Cancelled', running: 'Running',
  queued: 'Queued', pending: 'Pending', skipped: 'Skipped', waiting_approval: 'Awaiting Approval',
};

// ---------------------------------------------------------------------------
// Fetchers
// ---------------------------------------------------------------------------
async function fetchProject(id: string) {
  const [project, pipelines, repos, secrets, notifications] = await Promise.all([
    api.projects.get(id),
    api.pipelines.list(id).catch(() => [] as Pipeline[]),
    api.repositories.list(id).catch(() => [] as Repository[]),
    api.secrets.list(id).catch(() => [] as Secret[]),
    api.notifications.list(id).catch(() => [] as NotificationChannel[]),
  ]);

  // Fetch recent runs across all pipelines
  const runs: PipelineRun[] = [];
  for (const pipeline of pipelines.slice(0, 5)) {
    try {
      const res = await api.runs.list(id, pipeline.id, { page: '1', per_page: '5' });
      runs.push(...res.data);
    } catch { /* ignore */ }
  }
  runs.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());

  return { project, pipelines, repos, secrets, notifications, runs: runs.slice(0, 10) };
}

// Run table columns
const runColumns: TableColumn<PipelineRun>[] = [
  { key: 'number', header: '#', width: '70px', render: (row) => <span class="text-sm font-mono font-medium text-[var(--color-text-primary)]">#{row.number}</span> },
  { key: 'status', header: 'Status', width: '140px', render: (row) => <Badge variant={getStatusBadgeVariant(row.status)} dot size="sm">{statusLabel[row.status]}</Badge> },
  { key: 'commit', header: 'Commit', render: (row) => (
    <div class="min-w-0">
      <p class="text-sm text-[var(--color-text-primary)] truncate max-w-sm">{row.commit_message || '-'}</p>
      <div class="flex items-center gap-2 mt-0.5">
        <span class="text-xs font-mono text-[var(--color-text-tertiary)]">{truncateCommitSha(row.commit_sha)}</span>
        <Show when={row.branch}><span class="text-xs text-[var(--color-text-tertiary)]">on {row.branch}</span></Show>
      </div>
    </div>
  )},
  { key: 'duration', header: 'Duration', width: '100px', align: 'right' as const, render: (row) => <span class="text-xs font-mono text-[var(--color-text-secondary)]">{formatDuration(row.duration_ms)}</span> },
  { key: 'time', header: 'Started', width: '90px', align: 'right' as const, render: (row) => <span class="text-xs text-[var(--color-text-tertiary)]">{formatRelativeTime(row.created_at)}</span> },
];

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const ProjectDetailPage: Component = () => {
  const params = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = createSignal('overview');
  const [data, { refetch }] = createResource(() => params.id, fetchProject);

  // Secret form
  const [showAddSecret, setShowAddSecret] = createSignal(false);
  const [newSecretKey, setNewSecretKey] = createSignal('');
  const [newSecretValue, setNewSecretValue] = createSignal('');
  const [savingSecret, setSavingSecret] = createSignal(false);

  // Confirm delete
  const [showDeleteConfirm, setShowDeleteConfirm] = createSignal(false);
  const [deleting, setDeleting] = createSignal(false);

  // Settings form
  const [editName, setEditName] = createSignal('');
  const [editDesc, setEditDesc] = createSignal('');
  const [editVis, setEditVis] = createSignal('');
  const [savingSettings, setSavingSettings] = createSignal(false);

  // Initialize settings form when data loads
  const initSettings = () => {
    const p = data()?.project;
    if (p) { setEditName(p.name); setEditDesc(p.description || ''); setEditVis(p.visibility); }
  };

  const project = () => data()?.project;

  const tabs = () => [
    { id: 'overview', label: 'Overview' },
    { id: 'pipelines', label: `Pipelines (${data()?.pipelines?.length ?? 0})` },
    { id: 'runs', label: 'Runs' },
    { id: 'repositories', label: 'Repositories' },
    { id: 'secrets', label: `Secrets (${data()?.secrets?.length ?? 0})` },
    { id: 'notifications', label: 'Notifications' },
    { id: 'settings', label: 'Settings' },
  ];

  const handleAddSecret = async () => {
    if (!newSecretKey().trim() || !newSecretValue().trim()) return;
    setSavingSecret(true);
    try {
      await api.secrets.create(params.id, newSecretKey().trim(), newSecretValue());
      toast.success(`Secret "${newSecretKey()}" added`);
      setShowAddSecret(false);
      setNewSecretKey('');
      setNewSecretValue('');
      refetch();
    } catch (err) {
      toast.error(err instanceof ApiRequestError ? err.message : 'Failed to add secret');
    } finally {
      setSavingSecret(false);
    }
  };

  const handleDeleteSecret = async (secretId: string, key: string) => {
    try {
      await api.secrets.delete(params.id, secretId);
      toast.success(`Secret "${key}" deleted`);
      refetch();
    } catch (err) {
      toast.error(err instanceof ApiRequestError ? err.message : 'Failed to delete secret');
    }
  };

  const handleSyncRepo = async (repoId: string) => {
    try {
      await api.repositories.sync(params.id, repoId);
      toast.success('Repository sync started');
      refetch();
    } catch (err) {
      toast.error(err instanceof ApiRequestError ? err.message : 'Failed to sync repository');
    }
  };

  const handleDisconnectRepo = async (repoId: string) => {
    try {
      await api.repositories.disconnect(params.id, repoId);
      toast.success('Repository disconnected');
      refetch();
    } catch (err) {
      toast.error(err instanceof ApiRequestError ? err.message : 'Failed to disconnect');
    }
  };

  const handleSaveSettings = async () => {
    setSavingSettings(true);
    try {
      await api.projects.update(params.id, {
        name: editName().trim(),
        description: editDesc().trim() || undefined,
        visibility: editVis() as 'private' | 'internal' | 'public',
      });
      toast.success('Project settings saved');
      refetch();
    } catch (err) {
      toast.error(err instanceof ApiRequestError ? err.message : 'Failed to save');
    } finally {
      setSavingSettings(false);
    }
  };

  const handleDeleteProject = async () => {
    setDeleting(true);
    try {
      await api.projects.delete(params.id);
      toast.success('Project deleted');
      navigate('/projects');
    } catch (err) {
      toast.error(err instanceof ApiRequestError ? err.message : 'Failed to delete project');
      setDeleting(false);
    }
  };

  const handleDeleteNotification = async (channelId: string) => {
    try {
      await api.notifications.delete(params.id, channelId);
      toast.success('Notification channel deleted');
      refetch();
    } catch (err) {
      toast.error(err instanceof ApiRequestError ? err.message : 'Failed to delete');
    }
  };

  return (
    <PageContainer
      title={project()?.name ?? 'Loading...'}
      description={project()?.description}
      breadcrumbs={[{ label: 'Projects', href: '/projects' }, { label: project()?.name ?? '...' }]}
      actions={
        <div class="flex items-center gap-2">
          <A href={`/projects/${params.id}/pipelines`}><Button size="sm">View Pipelines</Button></A>
        </div>
      }
    >
      {/* Error */}
      <Show when={data.error}>
        <div class="p-4 rounded-xl bg-red-500/10 border border-red-500/30 flex items-center justify-between mb-6">
          <p class="text-sm text-red-400">Failed to load project: {(data.error as Error)?.message}</p>
          <Button size="sm" variant="outline" onClick={refetch}>Retry</Button>
        </div>
      </Show>

      <Show when={!data.loading && data()} fallback={
        <div class="space-y-4"><div class="h-10 bg-[var(--color-bg-secondary)] rounded animate-pulse" /><div class="h-64 bg-[var(--color-bg-secondary)] rounded-xl animate-pulse" /></div>
      }>
        <Tabs tabs={tabs()} activeTab={activeTab()} onTabChange={(id) => { setActiveTab(id); if (id === 'settings') initSettings(); }} class="mb-6" />

        <Switch>
          {/* Overview */}
          <Match when={activeTab() === 'overview'}>
            <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
              <div class="lg:col-span-2 space-y-6">
                <Card title="Pipelines">
                  <Show when={(data()?.pipelines ?? []).length > 0} fallback={
                    <p class="text-sm text-[var(--color-text-tertiary)]">No pipelines configured. <A href={`/projects/${params.id}/pipelines`} class="text-indigo-400">Create one</A></p>
                  }>
                    <div class="space-y-3">
                      <For each={data()?.pipelines ?? []}>
                        {(pipeline) => (
                          <A href={`/projects/${params.id}/pipelines/${pipeline.id}`} class="block">
                            <div class="flex items-center justify-between p-3 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
                              <div class="flex items-center gap-3">
                                <div class={`w-2 h-2 rounded-full ${pipeline.is_active ? 'bg-emerald-400' : 'bg-gray-500'}`} />
                                <div>
                                  <p class="text-sm font-medium text-[var(--color-text-primary)]">{pipeline.name}</p>
                                  <p class="text-xs text-[var(--color-text-tertiary)]">{pipeline.description}</p>
                                </div>
                              </div>
                              <Badge variant={pipeline.is_active ? 'success' : 'default'} size="sm">{pipeline.is_active ? 'Active' : 'Disabled'}</Badge>
                            </div>
                          </A>
                        )}
                      </For>
                    </div>
                  </Show>
                </Card>
                <Card title="Recent Runs" padding={false}>
                  <Table columns={runColumns} data={data()?.runs ?? []} emptyMessage="No runs yet" />
                </Card>
              </div>
              <div class="space-y-6">
                <Card title="Details">
                  <div class="space-y-3 text-sm">
                    <div class="flex justify-between"><span class="text-[var(--color-text-tertiary)]">Slug</span><span class="font-mono text-[var(--color-text-secondary)]">{project()?.slug}</span></div>
                    <div class="flex justify-between"><span class="text-[var(--color-text-tertiary)]">Visibility</span><span class="capitalize text-[var(--color-text-secondary)]">{project()?.visibility}</span></div>
                    <div class="flex justify-between"><span class="text-[var(--color-text-tertiary)]">Pipelines</span><span class="text-[var(--color-text-secondary)]">{data()?.pipelines?.length ?? 0}</span></div>
                    <div class="flex justify-between"><span class="text-[var(--color-text-tertiary)]">Secrets</span><span class="text-[var(--color-text-secondary)]">{data()?.secrets?.length ?? 0}</span></div>
                    <div class="flex justify-between"><span class="text-[var(--color-text-tertiary)]">Created</span><span class="text-[var(--color-text-secondary)]">{formatRelativeTime(project()!.created_at)}</span></div>
                  </div>
                </Card>
                <Card title="Repositories">
                  <Show when={(data()?.repos ?? []).length > 0} fallback={<p class="text-sm text-[var(--color-text-tertiary)]">No repositories connected.</p>}>
                    <For each={data()?.repos ?? []}>
                      {(repo) => (
                        <div class="flex items-center gap-3 p-2">
                          <div class="min-w-0 flex-1">
                            <p class="text-sm font-medium text-[var(--color-text-primary)] truncate">{repo.full_name}</p>
                            <p class="text-xs text-[var(--color-text-tertiary)]">{repo.default_branch}{repo.last_sync_at ? ` · Synced ${formatRelativeTime(repo.last_sync_at)}` : ''}</p>
                          </div>
                          <Badge variant={repo.is_active ? 'success' : 'default'} size="sm">{repo.is_active ? 'Active' : 'Inactive'}</Badge>
                        </div>
                      )}
                    </For>
                  </Show>
                </Card>
              </div>
            </div>
          </Match>

          {/* Pipelines */}
          <Match when={activeTab() === 'pipelines'}>
            <div class="space-y-4">
              <For each={data()?.pipelines ?? []} fallback={<p class="text-sm text-[var(--color-text-tertiary)] text-center py-8">No pipelines. <A href={`/projects/${params.id}/pipelines`} class="text-indigo-400">Create one</A></p>}>
                {(pipeline) => (
                  <A href={`/projects/${params.id}/pipelines/${pipeline.id}`} class="block">
                    <Card>
                      <div class="flex items-center justify-between">
                        <div class="flex items-center gap-4">
                          <div class={`w-3 h-3 rounded-full ${pipeline.is_active ? 'bg-emerald-400' : 'bg-gray-500'}`} />
                          <div>
                            <h3 class="text-sm font-semibold text-[var(--color-text-primary)]">{pipeline.name}</h3>
                            <p class="text-xs text-[var(--color-text-tertiary)] mt-0.5">{pipeline.description}</p>
                          </div>
                        </div>
                        <div class="flex items-center gap-4">
                          <p class="text-xs text-[var(--color-text-tertiary)]">v{pipeline.config_version} · {formatRelativeTime(pipeline.updated_at)}</p>
                          <Badge variant={pipeline.is_active ? 'success' : 'default'} size="sm">{pipeline.is_active ? 'Active' : 'Disabled'}</Badge>
                        </div>
                      </div>
                    </Card>
                  </A>
                )}
              </For>
            </div>
          </Match>

          {/* Runs */}
          <Match when={activeTab() === 'runs'}>
            <Card padding={false}>
              <Table columns={runColumns} data={data()?.runs ?? []} emptyMessage="No pipeline runs yet" />
            </Card>
          </Match>

          {/* Repositories */}
          <Match when={activeTab() === 'repositories'}>
            <Card title="Connected Repositories" actions={<Button size="sm" variant="outline">Connect Repository</Button>}>
              <Show when={(data()?.repos ?? []).length > 0} fallback={<p class="text-sm text-[var(--color-text-tertiary)]">No repositories connected.</p>}>
                <For each={data()?.repos ?? []}>
                  {(repo) => (
                    <div class="flex items-center justify-between p-4 border-b border-[var(--color-border-primary)] last:border-b-0">
                      <div>
                        <p class="text-sm font-semibold text-[var(--color-text-primary)]">{repo.full_name}</p>
                        <div class="flex items-center gap-3 mt-1 text-xs text-[var(--color-text-tertiary)]">
                          <span>Branch: {repo.default_branch}</span>
                          <span>Provider: {repo.provider}</span>
                          <Show when={repo.last_sync_at}><span>Synced: {formatRelativeTime(repo.last_sync_at!)}</span></Show>
                        </div>
                      </div>
                      <div class="flex items-center gap-2">
                        <Button size="sm" variant="ghost" onClick={() => handleSyncRepo(repo.id)}>Sync</Button>
                        <Button size="sm" variant="danger" onClick={() => handleDisconnectRepo(repo.id)}>Disconnect</Button>
                      </div>
                    </div>
                  )}
                </For>
              </Show>
            </Card>
          </Match>

          {/* Secrets */}
          <Match when={activeTab() === 'secrets'}>
            <Card title="Project Secrets" description="Encrypted secrets available to all pipelines" actions={
              <Button size="sm" onClick={() => setShowAddSecret(true)}>Add Secret</Button>
            } padding={false}>
              <Show when={(data()?.secrets ?? []).length > 0} fallback={
                <div class="p-6 text-center text-sm text-[var(--color-text-tertiary)]">No secrets configured.</div>
              }>
                <table class="w-full">
                  <thead><tr class="border-b border-[var(--color-border-primary)]">
                    <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Key</th>
                    <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Value</th>
                    <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Updated</th>
                    <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-right">Actions</th>
                  </tr></thead>
                  <tbody>
                    <For each={data()?.secrets ?? []}>
                      {(secret) => (
                        <tr class="border-b border-[var(--color-border-primary)] last:border-b-0 hover:bg-[var(--color-bg-hover)]">
                          <td class="px-5 py-3"><span class="text-sm font-mono font-medium text-[var(--color-text-primary)]">{secret.key}</span></td>
                          <td class="px-5 py-3"><span class="text-sm font-mono text-[var(--color-text-tertiary)]">••••••••</span></td>
                          <td class="px-5 py-3 text-xs text-[var(--color-text-tertiary)]">{formatRelativeTime(secret.updated_at)}</td>
                          <td class="px-5 py-3 text-right">
                            <Button size="sm" variant="danger" onClick={() => handleDeleteSecret(secret.id, secret.key)}>Delete</Button>
                          </td>
                        </tr>
                      )}
                    </For>
                  </tbody>
                </table>
              </Show>
            </Card>
            <Show when={showAddSecret()}>
              <Modal open={showAddSecret()} onClose={() => setShowAddSecret(false)} title="Add Secret" footer={
                <><Button variant="ghost" onClick={() => setShowAddSecret(false)}>Cancel</Button><Button onClick={handleAddSecret} loading={savingSecret()} disabled={!newSecretKey().trim() || !newSecretValue().trim()}>Add Secret</Button></>
              }>
                <div class="space-y-4">
                  <Input label="Key" placeholder="MY_SECRET_KEY" value={newSecretKey()} onInput={(e) => setNewSecretKey(e.currentTarget.value.toUpperCase().replace(/[^A-Z0-9_]/g, ''))} />
                  <Input label="Value" type="password" placeholder="Enter secret value..." value={newSecretValue()} onInput={(e) => setNewSecretValue(e.currentTarget.value)} />
                  <p class="text-xs text-[var(--color-text-tertiary)]">Secrets are encrypted with AES-256-GCM and masked in build logs.</p>
                </div>
              </Modal>
            </Show>
          </Match>

          {/* Notifications */}
          <Match when={activeTab() === 'notifications'}>
            <Card title="Notification Channels" actions={<Button size="sm">Add Channel</Button>}>
              <Show when={(data()?.notifications ?? []).length > 0} fallback={
                <p class="text-sm text-[var(--color-text-tertiary)]">No notification channels configured.</p>
              }>
                <div class="space-y-3">
                  <For each={data()?.notifications ?? []}>
                    {(channel) => (
                      <div class="flex items-center justify-between p-3 rounded-lg border border-[var(--color-border-primary)]">
                        <div class="flex items-center gap-3">
                          <div class="w-8 h-8 rounded-lg bg-[var(--color-bg-tertiary)] flex items-center justify-center text-xs font-bold text-[var(--color-text-secondary)] uppercase">{channel.type[0]}</div>
                          <div>
                            <p class="text-sm font-medium text-[var(--color-text-primary)]">{channel.name}</p>
                            <p class="text-xs text-[var(--color-text-tertiary)] capitalize">{channel.type}</p>
                          </div>
                        </div>
                        <div class="flex items-center gap-2">
                          <Badge variant={channel.is_active ? 'success' : 'default'} size="sm">{channel.is_active ? 'Active' : 'Disabled'}</Badge>
                          <Button size="sm" variant="danger" onClick={() => handleDeleteNotification(channel.id)}>Delete</Button>
                        </div>
                      </div>
                    )}
                  </For>
                </div>
              </Show>
            </Card>
          </Match>

          {/* Settings */}
          <Match when={activeTab() === 'settings'}>
            <div class="space-y-6">
              <Card title="General Settings">
                <div class="space-y-4 max-w-lg">
                  <Input label="Project Name" value={editName()} onInput={(e) => setEditName(e.currentTarget.value)} />
                  <Input label="Description" value={editDesc()} onInput={(e) => setEditDesc(e.currentTarget.value)} />
                  <Select label="Visibility" value={editVis()} onChange={(e) => setEditVis(e.currentTarget.value)} options={[
                    { value: 'private', label: 'Private' }, { value: 'internal', label: 'Internal' }, { value: 'public', label: 'Public' },
                  ]} />
                  <Button onClick={handleSaveSettings} loading={savingSettings()}>Save Changes</Button>
                </div>
              </Card>
              <Card title="Danger Zone">
                <div class="flex items-center justify-between p-4 rounded-lg border border-red-500/30 bg-red-500/5">
                  <div>
                    <p class="text-sm font-medium text-red-400">Delete Project</p>
                    <p class="text-xs text-[var(--color-text-tertiary)] mt-1">This will permanently delete the project and all associated data.</p>
                  </div>
                  <Button variant="danger" size="sm" onClick={() => setShowDeleteConfirm(true)}>Delete Project</Button>
                </div>
              </Card>
            </div>
            <Show when={showDeleteConfirm()}>
              <Modal open={showDeleteConfirm()} onClose={() => setShowDeleteConfirm(false)} title="Delete Project" size="sm" footer={
                <><Button variant="ghost" onClick={() => setShowDeleteConfirm(false)}>Cancel</Button><Button variant="danger" onClick={handleDeleteProject} loading={deleting()}>Delete Permanently</Button></>
              }>
                <p class="text-sm text-[var(--color-text-secondary)]">Are you sure you want to delete <strong>{project()?.name}</strong>? This action cannot be undone. All pipelines, runs, secrets, and artifacts will be permanently deleted.</p>
              </Modal>
            </Show>
          </Match>
        </Switch>
      </Show>
    </PageContainer>
  );
};

export default ProjectDetailPage;
