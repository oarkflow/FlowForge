import type { Component } from 'solid-js';
import { createSignal, createResource, createMemo, For, Show, Switch, Match } from 'solid-js';
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
import KeyValueEditor, { type KeyValuePair } from '../../components/ui/KeyValueEditor';
import EnvironmentsTab from '../../components/environments/EnvironmentsTab';
import RegistriesTab from '../../components/registries/RegistriesTab';
import DeploymentProvidersTab from '../../components/providers/DeploymentProvidersTab';
import { toast } from '../../components/ui/Toast';
import { api, ApiRequestError } from '../../api/client';
import type { Pipeline, PipelineRun, Repository, Secret, EnvVar, NotificationChannel, RunStatus } from '../../types';
import { formatRelativeTime, formatDuration, getStatusBadgeVariant, truncateCommitSha } from '../../utils/helpers';

const statusLabel: Record<RunStatus, string> = {
	success: 'Success', failure: 'Failed', cancelled: 'Cancelled', running: 'Running',
	queued: 'Queued', pending: 'Pending', skipped: 'Skipped', waiting_approval: 'Awaiting Approval',
};

// ---------------------------------------------------------------------------
// Fetchers
// ---------------------------------------------------------------------------
async function fetchProject(id: string) {
	const [project, pipelines, repos, secrets, notifications, envVars] = await Promise.all([
		api.projects.get(id),
		api.pipelines.list(id).catch(() => [] as Pipeline[]),
		api.repositories.list(id).catch(() => [] as Repository[]),
		api.secrets.list(id).catch(() => [] as Secret[]),
		api.notifications.list(id).catch(() => [] as NotificationChannel[]),
		api.envVars.list(id).catch(() => [] as EnvVar[]),
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

	return { project, pipelines, repos, secrets, notifications, envVars, runs: runs.slice(0, 10) };
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const ProjectDetailPage: Component = () => {
	const params = useParams<{ id: string }>();
	const navigate = useNavigate();
	const [activeTab, setActiveTab] = createSignal('overview');
	const [data, { refetch }] = createResource(() => params.id, fetchProject);

	// Run table columns (inside component for access to params.id)
	const runColumns = createMemo((): TableColumn<PipelineRun>[] => [
		{
			key: 'number', header: '#', width: '70px', render: (row) => (
				<A href={`/projects/${params.id}/pipelines/${row.pipeline_id}/runs/${row.id}`}
					class="text-sm font-mono font-medium text-indigo-400 hover:text-indigo-300">
					#{row.number}
				</A>
			)
		},
		{
			key: 'status', header: 'Status', width: '140px', render: (row) => (
				<div>
					<Badge variant={getStatusBadgeVariant(row.status)} dot size="sm">{statusLabel[row.status]}</Badge>
					{row.status === 'failure' && row.error_summary ? (
						<p class="text-xs text-red-400/80 mt-1 truncate max-w-[200px]" title={row.error_summary}>{row.error_summary}</p>
					) : null}
				</div>
			)
		},
		{
			key: 'commit', header: 'Commit', render: (row) => (
				<div class="min-w-0">
					<p class="text-sm text-[var(--color-text-primary)] truncate max-w-sm">{row.commit_message || '-'}</p>
					<div class="flex items-center gap-2 mt-0.5">
						<span class="text-xs font-mono text-[var(--color-text-tertiary)]">{truncateCommitSha(row.commit_sha)}</span>
						<Show when={row.branch}><span class="text-xs text-[var(--color-text-tertiary)]">on {row.branch}</span></Show>
					</div>
				</div>
			)
		},
		{ key: 'duration', header: 'Duration', width: '100px', align: 'right' as const, render: (row) => <span class="text-xs font-mono text-[var(--color-text-secondary)]">{formatDuration(row.duration_ms)}</span> },
		{ key: 'time', header: 'Started', width: '90px', align: 'right' as const, render: (row) => <span class="text-xs text-[var(--color-text-tertiary)]">{formatRelativeTime(row.created_at)}</span> },
	]);

	// Environment variables state
	const [envVarItems, setEnvVarItems] = createSignal<KeyValuePair[]>([]);
	const [savingEnvVars, setSavingEnvVars] = createSignal(false);
	const [envVarsDirty, setEnvVarsDirty] = createSignal(false);

	const initEnvVars = () => {
		const vars = data()?.envVars ?? [];
		setEnvVarItems(vars.map(v => ({ id: v.id, key: v.key, value: v.value })));
		setEnvVarsDirty(false);
	};

	const handleEnvVarChange = (items: KeyValuePair[]) => {
		setEnvVarItems(items);
		setEnvVarsDirty(true);
	};

	const handleSaveEnvVars = async () => {
		setSavingEnvVars(true);
		try {
			const vars = envVarItems().filter(v => v.key.trim());
			await api.envVars.bulkSave(params.id, vars.map(v => ({ key: v.key, value: v.value })));
			toast.success('Environment variables saved');
			setEnvVarsDirty(false);
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to save environment variables');
		} finally {
			setSavingEnvVars(false);
		}
	};

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

	// Edit repository modal
	const [editingRepo, setEditingRepo] = createSignal<Repository | null>(null);
	const [repoFullName, setRepoFullName] = createSignal('');
	const [repoCloneUrl, setRepoCloneUrl] = createSignal('');
	const [repoSshUrl, setRepoSshUrl] = createSignal('');
	const [repoBranch, setRepoBranch] = createSignal('');
	const [repoProvider, setRepoProvider] = createSignal('');
	const [savingRepo, setSavingRepo] = createSignal(false);

	// Create pipeline modal
	const [showCreatePipeline, setShowCreatePipeline] = createSignal(false);
	const [newPipelineName, setNewPipelineName] = createSignal('');
	const [newPipelineDesc, setNewPipelineDesc] = createSignal('');
	const [newPipelineSource, setNewPipelineSource] = createSignal('db');
	const [newPipelineConfigPath, setNewPipelineConfigPath] = createSignal('.flowforge.yml');
	const [creatingPipeline, setCreatingPipeline] = createSignal(false);

	// Trigger pipeline modal
	const [showTriggerPipeline, setShowTriggerPipeline] = createSignal(false);
	const [triggerPipelineId, setTriggerPipelineId] = createSignal('');
	const [triggerBranch, setTriggerBranch] = createSignal('main');
	const [triggeringPipeline, setTriggeringPipeline] = createSignal(false);

	const openEditRepo = (repo: Repository) => {
		setEditingRepo(repo);
		setRepoFullName(repo.full_name);
		setRepoCloneUrl(repo.clone_url);
		setRepoSshUrl(repo.ssh_url || '');
		setRepoBranch(repo.default_branch);
		setRepoProvider(repo.provider);
	};

	const handleSaveRepo = async () => {
		const repo = editingRepo();
		if (!repo) return;
		setSavingRepo(true);
		try {
			await api.repositories.update(params.id, repo.id, {
				full_name: repoFullName().trim(),
				clone_url: repoCloneUrl().trim(),
				ssh_url: repoSshUrl().trim() || undefined,
				default_branch: repoBranch().trim(),
				provider: repoProvider() as Repository['provider'],
			});
			toast.success('Repository updated');
			setEditingRepo(null);
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to update repository');
		} finally {
			setSavingRepo(false);
		}
	};

	const handleCreatePipeline = async () => {
		if (!newPipelineName().trim()) return;
		setCreatingPipeline(true);
		try {
			await api.pipelines.create(params.id, {
				name: newPipelineName().trim(),
				description: newPipelineDesc().trim() || undefined,
				config_source: newPipelineSource() as 'db' | 'repo',
				config_path: newPipelineSource() === 'repo' ? newPipelineConfigPath().trim() : undefined,
			});
			toast.success('Pipeline created');
			setShowCreatePipeline(false);
			setNewPipelineName('');
			setNewPipelineDesc('');
			setNewPipelineSource('db');
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to create pipeline');
		} finally {
			setCreatingPipeline(false);
		}
	};

	const handleTriggerPipeline = async () => {
		const pid = triggerPipelineId();
		if (!pid) return;
		setTriggeringPipeline(true);
		try {
			const newRun = await api.pipelines.trigger(params.id, pid, { branch: triggerBranch().trim() || 'main' });
			toast.success('Pipeline triggered');
			setShowTriggerPipeline(false);
			navigate(`/projects/${params.id}/pipelines/${pid}/runs/${newRun.id}`);
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to trigger pipeline');
		} finally {
			setTriggeringPipeline(false);
		}
	};

	const openTriggerModal = (pipelineId?: string) => {
		const pips = data()?.pipelines ?? [];
		setTriggerPipelineId(pipelineId || pips[0]?.id || '');
		setTriggerBranch('main');
		setShowTriggerPipeline(true);
	};

	const handleDeletePipeline = async (pipelineId: string, name: string) => {
		try {
			await api.pipelines.delete(params.id, pipelineId);
			toast.success(`Pipeline "${name}" deleted`);
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to delete pipeline');
		}
	};

	// Initialize settings form when data loads
	const initSettings = () => {
		const p = data()?.project;
		if (p) { setEditName(p.name); setEditDesc(p.description || ''); setEditVis(p.visibility); }
	};

	const project = () => data()?.project;

	const tabs = () => [
		{ id: 'overview', label: 'Overview' },
		{ id: 'pipelines', label: `Pipelines (${data()?.pipelines?.length ?? 0})` },
		{ id: 'environments', label: 'Environments' },
		{ id: 'registries', label: 'Registries' },
		{ id: 'providers', label: 'Deployment Providers' },
		{ id: 'runs', label: 'Runs' },
		{ id: 'repositories', label: 'Repositories' },
		{ id: 'environment', label: `Environment (${data()?.envVars?.length ?? 0})` },
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
					<Button size="sm" variant="outline" onClick={() => openTriggerModal()}
						icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M6.3 2.841A1.5 1.5 0 004 4.11V15.89a1.5 1.5 0 002.3 1.269l9.344-5.89a1.5 1.5 0 000-2.538L6.3 2.84z" /></svg>}
					>Trigger</Button>
					<Button size="sm" onClick={() => setShowCreatePipeline(true)}
						icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>}
					>New Pipeline</Button>
					<A href={`/projects/${params.id}/pipelines`}><Button size="sm" variant="ghost">All Pipelines</Button></A>
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
				<Tabs tabs={tabs()} activeTab={activeTab()} onTabChange={(id) => { setActiveTab(id); if (id === 'settings') initSettings(); if (id === 'environment') initEnvVars(); }} class="mb-6" />

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
									<Table columns={runColumns()} data={data()?.runs ?? []} emptyMessage="No runs yet"
										onRowClick={(row) => navigate(`/projects/${params.id}/pipelines/${row.pipeline_id}/runs/${row.id}`)} />
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
						<div class="flex items-center justify-between mb-4">
							<p class="text-sm text-[var(--color-text-tertiary)]">{(data()?.pipelines ?? []).length} pipeline{(data()?.pipelines ?? []).length !== 1 ? 's' : ''}</p>
							<div class="flex items-center gap-2">
								<Button size="sm" variant="outline" onClick={() => openTriggerModal()}
									icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M6.3 2.841A1.5 1.5 0 004 4.11V15.89a1.5 1.5 0 002.3 1.269l9.344-5.89a1.5 1.5 0 000-2.538L6.3 2.84z" /></svg>}
								>Trigger Run</Button>
								<Button size="sm" onClick={() => setShowCreatePipeline(true)}
									icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>}
								>New Pipeline</Button>
							</div>
						</div>
						<div class="space-y-4">
							<For each={data()?.pipelines ?? []} fallback={
								<div class="text-center py-12">
									<svg class="w-12 h-12 mx-auto text-[var(--color-text-tertiary)] mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25H12" /></svg>
									<p class="text-[var(--color-text-secondary)] mb-2">No pipelines yet</p>
									<p class="text-sm text-[var(--color-text-tertiary)] mb-4">Create your first pipeline to automate your builds.</p>
									<Button onClick={() => setShowCreatePipeline(true)}>Create Pipeline</Button>
								</div>
							}>
								{(pipeline) => (
									<Card>
										<div class="flex items-center justify-between">
											<A href={`/projects/${params.id}/pipelines/${pipeline.id}`} class="flex items-center gap-4 flex-1 min-w-0">
												<div class={`w-3 h-3 rounded-full shrink-0 ${pipeline.is_active ? 'bg-emerald-400' : 'bg-gray-500'}`} />
												<div class="min-w-0">
													<h3 class="text-sm font-semibold text-[var(--color-text-primary)]">{pipeline.name}</h3>
													<p class="text-xs text-[var(--color-text-tertiary)] mt-0.5">{pipeline.description || 'No description'}</p>
												</div>
											</A>
											<div class="flex items-center gap-3 shrink-0 ml-4">
												<p class="text-xs text-[var(--color-text-tertiary)]">v{pipeline.config_version} · {formatRelativeTime(pipeline.updated_at)}</p>
												<Badge variant={pipeline.is_active ? 'success' : 'default'} size="sm">{pipeline.is_active ? 'Active' : 'Disabled'}</Badge>
												<Button size="sm" variant="outline" disabled={!pipeline.is_active} title={!pipeline.is_active ? 'Enable pipeline to trigger runs' : 'Trigger a run'} onClick={(e: Event) => { e.preventDefault(); e.stopPropagation(); openTriggerModal(pipeline.id); }}>Trigger</Button>
												<A href={`/projects/${params.id}/pipelines/${pipeline.id}`}><Button size="sm" variant="ghost">Edit</Button></A>
												<Button size="sm" variant="danger" onClick={(e: Event) => { e.preventDefault(); e.stopPropagation(); handleDeletePipeline(pipeline.id, pipeline.name); }}>Delete</Button>
											</div>
										</div>
									</Card>
								)}
							</For>
						</div>
					</Match>

					{/* Runs */}
					<Match when={activeTab() === 'runs'}>
						<Card padding={false}>
							<Table columns={runColumns()} data={data()?.runs ?? []} emptyMessage="No pipeline runs yet"
								onRowClick={(row) => navigate(`/projects/${params.id}/pipelines/${row.pipeline_id}/runs/${row.id}`)} />
						</Card>
					</Match>

					{/* Environments */}
					<Match when={activeTab() === 'environments'}>
						<EnvironmentsTab projectId={params.id} pipelines={(data()?.pipelines ?? []).map(p => ({ id: p.id, name: p.name }))} />
					</Match>

					{/* Registries */}
					<Match when={activeTab() === 'registries'}>
						<RegistriesTab projectId={params.id} />
					</Match>

					{/* Deployment Providers */}
					<Match when={activeTab() === 'providers'}>
						<DeploymentProvidersTab projectId={params.id} />
					</Match>

					{/* Repositories */}
					<Match when={activeTab() === 'repositories'}>
						<Card title="Connected Repositories" actions={<Button size="sm" variant="outline">Connect Repository</Button>}>
							<Show when={(data()?.repos ?? []).length > 0} fallback={<p class="text-sm text-[var(--color-text-tertiary)]">No repositories connected.</p>}>
								<For each={data()?.repos ?? []}>
									{(repo) => (
										<div class="flex items-center justify-between p-4 border-b border-[var(--color-border-primary)] last:border-b-0">
											<div class="min-w-0 flex-1">
												<p class="text-sm font-semibold text-[var(--color-text-primary)]">{repo.full_name}</p>
												<div class="flex items-center gap-3 mt-1 text-xs text-[var(--color-text-tertiary)]">
													<span>Branch: {repo.default_branch}</span>
													<span>Provider: {repo.provider}</span>
													<Show when={repo.clone_url}><span class="truncate max-w-xs" title={repo.clone_url}>{repo.clone_url}</span></Show>
													<Show when={repo.last_sync_at}><span>Synced: {formatRelativeTime(repo.last_sync_at!)}</span></Show>
												</div>
											</div>
											<div class="flex items-center gap-2">
												<Button size="sm" variant="outline" onClick={() => openEditRepo(repo)}>Edit</Button>
												<Button size="sm" variant="ghost" onClick={() => handleSyncRepo(repo.id)}>Sync</Button>
												<Button size="sm" variant="danger" onClick={() => handleDisconnectRepo(repo.id)}>Disconnect</Button>
											</div>
										</div>
									)}
								</For>
							</Show>
						</Card>
						<Show when={editingRepo()}>
							<Modal open={!!editingRepo()} onClose={() => setEditingRepo(null)} title="Edit Repository" footer={
								<><Button variant="ghost" onClick={() => setEditingRepo(null)}>Cancel</Button><Button onClick={handleSaveRepo} loading={savingRepo()} disabled={!repoFullName().trim() || !repoCloneUrl().trim()}>Save Changes</Button></>
							}>
								<div class="space-y-4">
									<Input label="Full Name" placeholder="owner/repo" value={repoFullName()} onInput={(e) => setRepoFullName(e.currentTarget.value)} />
									<Input label="Clone URL" placeholder="https://github.com/owner/repo.git" value={repoCloneUrl()} onInput={(e) => setRepoCloneUrl(e.currentTarget.value)} />
									<Input label="SSH URL (optional)" placeholder="git@github.com:owner/repo.git" value={repoSshUrl()} onInput={(e) => setRepoSshUrl(e.currentTarget.value)} />
									<Input label="Default Branch" placeholder="main" value={repoBranch()} onInput={(e) => setRepoBranch(e.currentTarget.value)} />
									<Select label="Provider" value={repoProvider()} onChange={(e) => setRepoProvider(e.currentTarget.value)} options={[
										{ value: 'github', label: 'GitHub' },
										{ value: 'gitlab', label: 'GitLab' },
										{ value: 'bitbucket', label: 'Bitbucket' },
									]} />
								</div>
							</Modal>
						</Show>
					</Match>

					{/* Environment Variables */}
					<Match when={activeTab() === 'environment'}>
						<Card title="Environment Variables" description="Non-secret key-value pairs available to all pipelines in this project. For sensitive values, use the Secrets tab instead.">
							<KeyValueEditor
								items={envVarItems()}
								onChange={handleEnvVarChange}
								keyPlaceholder="VARIABLE_NAME"
								valuePlaceholder="value"
							/>
							<Show when={envVarItems().length > 0 || envVarsDirty()}>
								<div class="flex items-center justify-between mt-6 pt-4 border-t border-[var(--color-border-primary)]">
									<p class="text-xs text-[var(--color-text-tertiary)]">
										{envVarsDirty() ? 'You have unsaved changes.' : 'All changes saved.'}
									</p>
									<Button onClick={handleSaveEnvVars} loading={savingEnvVars()} disabled={!envVarsDirty()}>
										Save Environment Variables
									</Button>
								</div>
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

			{/* Create Pipeline Modal */}
			<Show when={showCreatePipeline()}>
				<Modal open={showCreatePipeline()} onClose={() => setShowCreatePipeline(false)} title="Create Pipeline" description="Set up a new CI/CD pipeline" footer={
					<><Button variant="ghost" onClick={() => setShowCreatePipeline(false)}>Cancel</Button><Button onClick={handleCreatePipeline} loading={creatingPipeline()} disabled={!newPipelineName().trim()}>Create Pipeline</Button></>
				}>
					<div class="space-y-4">
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
			<Show when={showTriggerPipeline()}>
				<Modal open={showTriggerPipeline()} onClose={() => setShowTriggerPipeline(false)} title="Trigger Pipeline" description="Manually start a new pipeline run" footer={
					<><Button variant="ghost" onClick={() => setShowTriggerPipeline(false)}>Cancel</Button><Button onClick={handleTriggerPipeline} loading={triggeringPipeline()}>Trigger Run</Button></>
				}>
					<div class="space-y-4">
						<Show when={(data()?.pipelines ?? []).length > 1}>
							<Select label="Pipeline" value={triggerPipelineId()} onChange={(e) => setTriggerPipelineId(e.currentTarget.value)} options={
								(data()?.pipelines ?? []).filter(p => p.is_active).map(p => ({ value: p.id, label: p.name }))
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

export default ProjectDetailPage;
