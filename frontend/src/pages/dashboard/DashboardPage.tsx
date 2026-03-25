import type { Component } from 'solid-js';
import { createSignal, createResource, onMount, onCleanup, Show, For, Switch, Match } from 'solid-js';
import { A, useNavigate } from '@solidjs/router';
import PageContainer from '../../components/layout/PageContainer';
import Card from '../../components/ui/Card';
import Badge from '../../components/ui/Badge';
import Button from '../../components/ui/Button';
import Table, { type TableColumn } from '../../components/ui/Table';
import Modal from '../../components/ui/Modal';
import { toast } from '../../components/ui/Toast';
import { api } from '../../api/client';
import { getEventSocket } from '../../api/websocket';
import type { Project, Agent, PipelineRun, RunStatus, TriggerType, AgentStatus, SystemHealth, Deployment, Approval, DashboardWidgetConfig } from '../../types';
import { formatDuration, formatRelativeTime, getStatusBgColor, getAgentStatusVariant } from '../../utils/helpers';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
interface StatItem { label: string; value: string; trend: number; icon: string; }
interface RunWithMeta extends PipelineRun { pipeline_name: string; project_id: string; project_name: string; }
interface ActivityEvent { id: string; type: string; message: string; actor: string; time: string; }

// ---------------------------------------------------------------------------
// Default widget definitions
// ---------------------------------------------------------------------------
type WidgetId = 'stats' | 'recent_runs' | 'recent_deployments' | 'pending_approvals' | 'agent_status' | 'quick_actions' | 'pipeline_health' | 'activity_feed';

interface WidgetDef {
	id: WidgetId;
	label: string;
	description: string;
	defaultSize: 'full' | 'half';
}

const WIDGET_DEFS: WidgetDef[] = [
	{ id: 'stats', label: 'Stats Overview', description: 'Total pipelines, runs, success rate, and more', defaultSize: 'full' },
	{ id: 'recent_runs', label: 'Recent Runs', description: 'Latest pipeline executions', defaultSize: 'full' },
	{ id: 'recent_deployments', label: 'Recent Deployments', description: 'Latest deployments across environments', defaultSize: 'full' },
	{ id: 'pending_approvals', label: 'Pending Approvals', description: 'Approval requests awaiting response', defaultSize: 'full' },
	{ id: 'agent_status', label: 'Agent Status', description: 'Online/offline/busy agent counts', defaultSize: 'half' },
	{ id: 'quick_actions', label: 'Quick Actions', description: 'Create project, trigger pipeline, etc.', defaultSize: 'half' },
	{ id: 'pipeline_health', label: 'Pipeline Health', description: 'Top failing pipelines', defaultSize: 'half' },
	{ id: 'activity_feed', label: 'Live Activity', description: 'Real-time events from WebSocket', defaultSize: 'full' },
];

function getDefaultLayout(): DashboardWidgetConfig[] {
	return WIDGET_DEFS.map((def, i) => ({
		id: def.id,
		visible: true,
		size: def.defaultSize,
		order: i,
	}));
}

function mergeWithDefaults(saved: DashboardWidgetConfig[]): DashboardWidgetConfig[] {
	const savedMap = new Map(saved.map(w => [w.id, w]));
	const merged: DashboardWidgetConfig[] = [];
	// Keep saved order for known widgets
	for (const w of saved) {
		if (WIDGET_DEFS.some(d => d.id === w.id)) {
			merged.push(w);
		}
	}
	// Add any new widgets that aren't in saved layout
	for (const def of WIDGET_DEFS) {
		if (!savedMap.has(def.id)) {
			merged.push({ id: def.id, visible: true, size: def.defaultSize, order: merged.length });
		}
	}
	return merged.map((w, i) => ({ ...w, order: i }));
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
const agentDotColor = (s: AgentStatus) => s === 'online' ? 'bg-emerald-400' : s === 'busy' ? 'bg-violet-400 animate-pulse' : s === 'draining' ? 'bg-amber-400' : 'bg-gray-500';
const activityColor = (t: string) => t === 'success' ? 'bg-emerald-400' : t === 'failure' ? 'bg-red-400' : t === 'deploy' ? 'bg-blue-400' : t === 'agent' ? 'bg-violet-400' : 'bg-amber-400';

const SkeletonCard: Component = () => (
	<div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-5 animate-pulse">
		<div class="flex items-center justify-between mb-3"><div class="h-3 w-24 bg-[var(--color-bg-tertiary)] rounded" /><div class="h-5 w-5 bg-[var(--color-bg-tertiary)] rounded" /></div>
		<div class="h-7 w-16 bg-[var(--color-bg-tertiary)] rounded mb-2" /><div class="h-3 w-20 bg-[var(--color-bg-tertiary)] rounded" />
	</div>
);

// ---------------------------------------------------------------------------
// Data fetcher
// ---------------------------------------------------------------------------
async function fetchDashboardData() {
	const [projectsRes, agents, health] = await Promise.all([
		api.projects.list({ page: '1', per_page: '100' }),
		api.agents.list(),
		api.system.health(),
	]);

	const projects = projectsRes.data;

	// Fetch recent runs across all projects and their first pipeline
	const recentRuns: RunWithMeta[] = [];
	const pipelinesByProject = new Map<string, { pipeline_id: string; pipeline_name: string; project_name: string }[]>();

	for (const project of projects.slice(0, 5)) {
		try {
			const pipelines = await api.pipelines.list(project.id);
			const entries = pipelines.map(p => ({ pipeline_id: p.id, pipeline_name: p.name, project_name: project.name }));
			pipelinesByProject.set(project.id, entries);

			for (const pipeline of pipelines.slice(0, 3)) {
				try {
					const runsRes = await api.runs.list(project.id, pipeline.id, { page: '1', per_page: '3' });
					for (const run of runsRes.data) {
						recentRuns.push({ ...run, pipeline_name: pipeline.name, project_id: project.id, project_name: project.name });
					}
				} catch { /* pipeline may have no runs */ }
			}
		} catch { /* project may have no pipelines */ }
	}

	recentRuns.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());

	// Compute stats
	const totalPipelines = Array.from(pipelinesByProject.values()).reduce((sum, arr) => sum + arr.length, 0);
	const onlineAgents = agents.filter(a => a.status === 'online' || a.status === 'busy').length;
	const queuedRuns = recentRuns.filter(r => r.status === 'queued').length;
	const completedRuns = recentRuns.filter(r => r.status === 'success' || r.status === 'failure');
	const successRate = completedRuns.length > 0
		? Math.round(completedRuns.filter(r => r.status === 'success').length / completedRuns.length * 100)
		: 0;
	const avgDuration = completedRuns.length > 0
		? Math.round(completedRuns.reduce((sum, r) => sum + (r.duration_ms || 0), 0) / completedRuns.length)
		: 0;

	const stats: StatItem[] = [
		{ label: 'Total Pipelines', value: String(totalPipelines), trend: 0, icon: 'M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z' },
		{ label: 'Recent Runs', value: String(recentRuns.length), trend: 0, icon: 'M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z' },
		{ label: 'Success Rate', value: `${successRate}%`, trend: 0, icon: 'M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z' },
		{ label: 'Avg Duration', value: formatDuration(avgDuration), trend: 0, icon: 'M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z' },
		{ label: 'Active Agents', value: `${onlineAgents} / ${agents.length}`, trend: 0, icon: 'M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2' },
		{ label: 'Queued Jobs', value: String(queuedRuns), trend: 0, icon: 'M3.75 12h16.5m-16.5 3.75h16.5M3.75 19.5h16.5M5.625 4.5h12.75a1.875 1.875 0 010 3.75H5.625a1.875 1.875 0 010-3.75z' },
	];

	// Compute pipeline health (top 5 failing)
	const pipelineFailures = new Map<string, { name: string; failures: number; total: number }>();
	for (const run of recentRuns) {
		const key = run.pipeline_id;
		if (!pipelineFailures.has(key)) {
			pipelineFailures.set(key, { name: run.pipeline_name, failures: 0, total: 0 });
		}
		const entry = pipelineFailures.get(key)!;
		entry.total++;
		if (run.status === 'failure') entry.failures++;
	}
	const pipelineHealth = Array.from(pipelineFailures.values())
		.filter(p => p.failures > 0)
		.sort((a, b) => b.failures - a.failures)
		.slice(0, 5);

	// Fetch recent deployments
	let recentDeployments: Deployment[] = [];
	try {
		recentDeployments = await api.deployments.recent();
	} catch { /* deployments endpoint may not be available */ }

	// Fetch pending approvals
	let pendingApprovals: Approval[] = [];
	try {
		pendingApprovals = await api.approvals.listPending();
	} catch { /* approvals endpoint may not be available */ }

	return { stats, runs: recentRuns.slice(0, 10), agents, health, projects, recentDeployments, pendingApprovals, pipelineHealth };
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const DashboardPage: Component = () => {
	const navigate = useNavigate();
	const [data, { refetch }] = createResource(fetchDashboardData);
	const [activity, setActivity] = createSignal<ActivityEvent[]>([]);
	const [layout, setLayout] = createSignal<DashboardWidgetConfig[]>(getDefaultLayout());
	const [showCustomize, setShowCustomize] = createSignal(false);
	const [editLayout, setEditLayout] = createSignal<DashboardWidgetConfig[]>([]);
	const [savingLayout, setSavingLayout] = createSignal(false);

	// Load dashboard preferences
	onMount(async () => {
		try {
			const pref = await api.dashboardPrefs.get();
			if (pref && pref.layout) {
				const parsed = JSON.parse(pref.layout) as DashboardWidgetConfig[];
				if (Array.isArray(parsed) && parsed.length > 0) {
					setLayout(mergeWithDefaults(parsed));
				}
			}
		} catch {
			// Use defaults
		}
	});

	// Subscribe to real-time events
	onMount(() => {
		const ws = getEventSocket();
		if (!ws.isConnected) ws.connect();

		const unsub1 = ws.on('run.completed', (payload: any) => {
			const status = payload?.status || 'success';
			const type = status === 'success' ? 'success' : 'failure';
			setActivity(prev => [{
				id: `evt-${Date.now()}`, type,
				message: `${payload?.pipeline_name || 'Pipeline'} #${payload?.number || '?'} ${status === 'success' ? 'completed successfully' : 'failed'}`,
				actor: payload?.author || 'system',
				time: new Date().toISOString(),
			}, ...prev].slice(0, 20));
			refetch();
		});

		const unsub2 = ws.on('agent.status', (payload: any) => {
			setActivity(prev => [{
				id: `evt-${Date.now()}`, type: 'agent',
				message: `${payload?.name || 'Agent'} is now ${payload?.status || 'unknown'}`,
				actor: 'system',
				time: new Date().toISOString(),
			}, ...prev].slice(0, 20));
			refetch();
		});

		onCleanup(() => { unsub1(); unsub2(); });
	});

	const handleRefresh = () => {
		refetch();
		toast.info('Refreshing dashboard...');
	};

	// --- Customize handlers ---
	const openCustomize = () => {
		setEditLayout([...layout()]);
		setShowCustomize(true);
	};

	const toggleWidgetVisible = (id: string) => {
		setEditLayout(prev => prev.map(w => w.id === id ? { ...w, visible: !w.visible } : w));
	};

	const toggleWidgetSize = (id: string) => {
		setEditLayout(prev => prev.map(w => w.id === id ? { ...w, size: w.size === 'full' ? 'half' : 'full' } : w));
	};

	const moveWidget = (id: string, direction: 'up' | 'down') => {
		setEditLayout(prev => {
			const idx = prev.findIndex(w => w.id === id);
			if (idx < 0) return prev;
			const target = direction === 'up' ? idx - 1 : idx + 1;
			if (target < 0 || target >= prev.length) return prev;
			const arr = [...prev];
			[arr[idx], arr[target]] = [arr[target], arr[idx]];
			return arr.map((w, i) => ({ ...w, order: i }));
		});
	};

	const handleSaveLayout = async () => {
		const newLayout = editLayout();
		setSavingLayout(true);
		try {
			await api.dashboardPrefs.update({ layout: JSON.stringify(newLayout), theme: 'default' });
			setLayout(newLayout);
			setShowCustomize(false);
			toast.success('Dashboard layout saved');
		} catch {
			toast.error('Failed to save layout');
		} finally {
			setSavingLayout(false);
		}
	};

	const handleResetLayout = () => {
		setEditLayout(getDefaultLayout());
	};

	const runColumns: TableColumn<RunWithMeta>[] = [
		{ key: 'status', header: '', width: '40px', align: 'center', render: (row) => <span class={`w-2.5 h-2.5 rounded-full inline-block ${getStatusBgColor(row.status)} ${row.status === 'running' ? 'animate-pulse' : ''}`} /> },
		{ key: 'pipeline', header: 'Pipeline', render: (row) => (<div><div class="flex items-center gap-2"><span class="font-medium text-[var(--color-text-primary)]">{row.pipeline_name}</span><span class="text-[var(--color-text-tertiary)]">#{row.number}</span></div><div class="text-xs text-[var(--color-text-tertiary)] mt-0.5 truncate max-w-xs">{row.commit_message}</div></div>) },
		{ key: 'branch', header: 'Branch', render: (row) => row.branch ? <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-[var(--color-bg-tertiary)] text-xs text-[var(--color-text-secondary)] font-mono">{row.branch}</span> : <span class="text-[var(--color-text-tertiary)]">-</span> },
		{ key: 'trigger', header: 'Trigger', render: (row) => <span class="text-xs text-[var(--color-text-secondary)] capitalize">{row.trigger_type.replace('_', ' ')}</span> },
		{ key: 'duration', header: 'Duration', align: 'right', render: (row) => <span class="text-xs font-mono text-[var(--color-text-secondary)]">{formatDuration(row.duration_ms)}</span> },
		{ key: 'time', header: 'Started', align: 'right', render: (row) => <span class="text-xs text-[var(--color-text-tertiary)]">{formatRelativeTime(row.started_at ?? row.created_at)}</span> },
	];

	// Get visible widgets in order
	const visibleWidgets = () => layout().filter(w => w.visible).sort((a, b) => a.order - b.order);

	// Render widget by ID
	const renderWidget = (widgetId: string, size: 'full' | 'half') => {
		const sizeClass = size === 'full' ? 'col-span-full' : 'col-span-1';

		switch (widgetId) {
			case 'stats':
				return (
					<div class={`${sizeClass}`}>
						<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4">
							<Show when={!data.loading} fallback={<><SkeletonCard /><SkeletonCard /><SkeletonCard /><SkeletonCard /><SkeletonCard /><SkeletonCard /></>}>
								<For each={data()?.stats ?? []}>{(stat) => (
									<div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-5 hover:border-[var(--color-border-secondary)] transition-colors">
										<div class="flex items-center justify-between mb-3">
											<span class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">{stat.label}</span>
											<div class="p-1.5 rounded-lg bg-[var(--color-bg-tertiary)]"><svg class="w-4 h-4 text-[var(--color-text-tertiary)]" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d={stat.icon} /></svg></div>
										</div>
										<div class="text-2xl font-bold text-[var(--color-text-primary)] tabular-nums">{stat.value}</div>
									</div>
								)}</For>
							</Show>
						</div>
					</div>
				);

			case 'recent_runs':
				return (
					<div class={`${sizeClass}`}>
						<Card title="Recent Pipeline Runs" description="Latest pipeline executions" padding={false} actions={<A href="/runs" class="text-xs text-indigo-400 hover:text-indigo-300">View all</A>}>
							<Show when={!data.loading} fallback={<div class="p-6 space-y-3">{Array.from({ length: 5 }).map(() => <div class="h-12 bg-[var(--color-bg-tertiary)] rounded animate-pulse" />)}</div>}>
								<Table
									columns={runColumns}
									data={data()?.runs ?? []}
									emptyMessage="No pipeline runs yet. Create a project and trigger a pipeline to get started."
									onRowClick={(row) => {
										navigate(`/projects/${row.project_id}/pipelines/${row.pipeline_id}/runs/${row.id}`);
									}}
								/>
							</Show>
						</Card>
					</div>
				);

			case 'agent_status':
				return (
					<div class={`${sizeClass}`}>
						<Card title="Agent Status" padding={false} actions={<A href="/agents" class="text-xs text-indigo-400 hover:text-indigo-300">Manage</A>}>
							<Show when={!data.loading} fallback={<div class="p-4 space-y-3">{Array.from({ length: 3 }).map(() => <div class="h-10 bg-[var(--color-bg-tertiary)] rounded animate-pulse" />)}</div>}>
								<Show when={(data()?.agents ?? []).length > 0} fallback={
									<div class="p-6 text-center">
										<p class="text-sm text-[var(--color-text-tertiary)]">No agents registered yet.</p>
										<A href="/agents" class="text-xs text-indigo-400 hover:text-indigo-300 mt-1 inline-block">Register an agent</A>
									</div>
								}>
									<div class="divide-y divide-[var(--color-border-primary)]">
										<For each={data()?.agents ?? []}>{(agent) => (
											<div class="flex items-center justify-between px-5 py-3 hover:bg-[var(--color-bg-hover)] transition-colors">
												<div class="flex items-center gap-3">
													<span class={`w-2 h-2 rounded-full shrink-0 ${agentDotColor(agent.status)}`} />
													<div><div class="text-sm font-medium text-[var(--color-text-primary)]">{agent.name}</div><div class="text-xs text-[var(--color-text-tertiary)]">{agent.executor} · {agent.os}/{agent.arch}</div></div>
												</div>
												<Badge variant={getAgentStatusVariant(agent.status)} size="sm">{agent.status}</Badge>
											</div>
										)}</For>
									</div>
								</Show>
							</Show>
						</Card>
					</div>
				);

			case 'quick_actions':
				return (
					<div class={`${sizeClass}`}>
						<Card title="Quick Actions">
							<div class="grid grid-cols-2 gap-2">
								{[
									{ label: 'New Project', color: 'text-indigo-400', href: '/projects', icon: 'M12 4.5v15m7.5-7.5h-15' },
									{ label: 'Add Agent', color: 'text-emerald-400', href: '/agents', icon: 'M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2' },
									{ label: 'Add Secret', color: 'text-amber-400', href: '/settings', icon: 'M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z' },
									{ label: 'View Logs', color: 'text-blue-400', href: '/runs', icon: 'M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z' },
								].map(({ label, color, href, icon }) => (
									<button onClick={() => navigate(href)} class="flex flex-col items-center gap-2 p-3 rounded-lg bg-[var(--color-bg-tertiary)] hover:bg-[var(--color-bg-hover)] border border-[var(--color-border-primary)] transition-colors cursor-pointer">
										<svg class={`w-5 h-5 ${color}`} fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d={icon} /></svg>
										<span class="text-xs font-medium text-[var(--color-text-secondary)]">{label}</span>
									</button>
								))}
							</div>
						</Card>
					</div>
				);

			case 'pending_approvals':
				return (
					<Show when={(data()?.pendingApprovals ?? []).length > 0}>
						<div class={`${sizeClass}`}>
							<Card title="Pending Approvals" description="Deployments awaiting your approval" padding={false} actions={<A href="/approvals" class="text-xs text-indigo-400 hover:text-indigo-300">View all</A>}>
								<div class="divide-y divide-[var(--color-border-primary)]">
									<For each={(data()?.pendingApprovals ?? []).slice(0, 5)}>{(approval) => (
										<div class="flex items-center justify-between px-5 py-3 hover:bg-[var(--color-bg-hover)] transition-colors">
											<div class="flex items-center gap-3">
												<span class="w-2.5 h-2.5 rounded-full shrink-0 bg-amber-400 animate-pulse" />
												<div>
													<div class="flex items-center gap-2">
														<span class="text-sm font-medium text-[var(--color-text-primary)]">
															{approval.type === 'deployment' ? 'Deployment' : 'Pipeline Run'} Approval
														</span>
														<Badge variant="warning" size="sm">pending</Badge>
													</div>
													<div class="text-xs text-[var(--color-text-tertiary)] mt-0.5">
														<span>Requested by {approval.requested_by} · </span>
														<span>{approval.current_approvals}/{approval.min_approvals} approvals · </span>
														<span>{formatRelativeTime(approval.created_at)}</span>
													</div>
												</div>
											</div>
											<A href={`/approvals`} class="text-xs text-indigo-400 hover:text-indigo-300">Review</A>
										</div>
									)}</For>
								</div>
							</Card>
						</div>
					</Show>
				);

			case 'recent_deployments':
				return (
					<Show when={(data()?.recentDeployments ?? []).length > 0}>
						<div class={`${sizeClass}`}>
							<Card title="Recent Deployments" description="Latest deployments across all environments" padding={false}>
								<div class="divide-y divide-[var(--color-border-primary)]">
									<For each={data()?.recentDeployments ?? []}>{(dep) => (
										<div class="flex items-center justify-between px-5 py-3 hover:bg-[var(--color-bg-hover)] transition-colors">
											<div class="flex items-center gap-3">
												<span class={`w-2.5 h-2.5 rounded-full shrink-0 ${dep.status === 'live' ? 'bg-emerald-400' :
													dep.status === 'deploying' ? 'bg-violet-400 animate-pulse' :
														dep.status === 'failed' ? 'bg-red-400' :
															dep.status === 'rolled_back' ? 'bg-amber-400' :
																'bg-gray-400'
													}`} />
												<div>
													<div class="flex items-center gap-2">
														<span class="text-sm font-medium text-[var(--color-text-primary)]">{dep.version || dep.commit_sha?.substring(0, 7) || dep.id.substring(0, 8)}</span>
														<Badge variant={
															dep.status === 'live' ? 'success' :
																dep.status === 'deploying' ? 'running' :
																	dep.status === 'failed' ? 'error' :
																		dep.status === 'rolled_back' ? 'warning' :
																			'default'
														} size="sm">{dep.status}</Badge>
													</div>
													<div class="text-xs text-[var(--color-text-tertiary)] mt-0.5">
														{dep.deployed_by && <span>{dep.deployed_by} · </span>}
														{dep.image_tag && <span class="font-mono">{dep.image_tag} · </span>}
														<span>{formatRelativeTime(dep.created_at)}</span>
													</div>
												</div>
											</div>
											<Show when={dep.commit_sha}>
												<span class="text-xs font-mono text-[var(--color-text-tertiary)]">{dep.commit_sha?.substring(0, 7)}</span>
											</Show>
										</div>
									)}</For>
								</div>
							</Card>
						</div>
					</Show>
				);

			case 'pipeline_health':
				return (
					<div class={`${sizeClass}`}>
						<Card title="Pipeline Health" description="Top failing pipelines">
							<Show when={(data()?.pipelineHealth ?? []).length > 0} fallback={
								<div class="text-center py-6 text-[var(--color-text-tertiary)]">
									<p class="text-sm">No pipeline failures to report 🎉</p>
								</div>
							}>
								<div class="space-y-3">
									<For each={data()?.pipelineHealth ?? []}>{(p) => {
										const failRate = Math.round(p.failures / p.total * 100);
										return (
											<div class="flex items-center justify-between">
												<div class="flex-1 min-w-0">
													<p class="text-sm font-medium text-[var(--color-text-primary)] truncate">{p.name}</p>
													<div class="flex items-center gap-2 mt-1">
														<div class="flex-1 h-1.5 bg-[var(--color-bg-tertiary)] rounded-full overflow-hidden">
															<div class="h-full bg-red-500 rounded-full" style={{ width: `${failRate}%` }} />
														</div>
														<span class="text-xs text-[var(--color-text-tertiary)] whitespace-nowrap">{p.failures}/{p.total}</span>
													</div>
												</div>
												<span class={`ml-3 text-sm font-medium ${failRate > 50 ? 'text-red-400' : 'text-amber-400'}`}>{failRate}%</span>
											</div>
										);
									}}</For>
								</div>
							</Show>
						</Card>
					</div>
				);

			case 'activity_feed':
				return (
					<Show when={activity().length > 0}>
						<div class={`${sizeClass}`}>
							<Card title="Live Activity" padding={false}>
								<div class="divide-y divide-[var(--color-border-primary)]">
									<For each={activity()}>{(event) => (
										<div class="flex items-start gap-3 px-5 py-3.5 hover:bg-[var(--color-bg-hover)] transition-colors">
											<span class={`w-2 h-2 rounded-full mt-1.5 shrink-0 ${activityColor(event.type)}`} />
											<div class="flex-1 min-w-0">
												<p class="text-sm text-[var(--color-text-primary)]">{event.message}</p>
												<div class="flex items-center gap-2 mt-0.5"><span class="text-xs text-[var(--color-text-tertiary)]">{event.actor}</span><span class="text-xs text-[var(--color-text-tertiary)]">·</span><span class="text-xs text-[var(--color-text-tertiary)]">{formatRelativeTime(event.time)}</span></div>
											</div>
										</div>
									)}</For>
								</div>
							</Card>
						</div>
					</Show>
				);

			default:
				return null;
		}
	};

	return (
		<PageContainer title="Dashboard" description="Overview of your CI/CD platform" actions={
			<div class="flex gap-2">
				<Button variant="outline" size="sm" onClick={openCustomize}
					icon={<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M10.5 6h9.75M10.5 6a1.5 1.5 0 11-3 0m3 0a1.5 1.5 0 10-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-9.75 0h9.75" /></svg>}
				>Customize</Button>
				<Button variant="outline" size="sm" onClick={handleRefresh} loading={data.loading}>Refresh</Button>
				<Button size="sm" onClick={() => navigate('/projects')}>New Project</Button>
			</div>
		}>
			{/* Error state */}
			<Show when={data.error}>
				<div class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30">
					<div class="flex items-center justify-between">
						<p class="text-sm text-red-400">Failed to load dashboard data: {(data.error as Error)?.message || 'Unknown error'}</p>
						<Button size="sm" variant="outline" onClick={handleRefresh}>Retry</Button>
					</div>
				</div>
			</Show>

			{/* Dynamic widget grid */}
			<div class="grid grid-cols-1 xl:grid-cols-2 gap-6">
				<For each={visibleWidgets()}>
					{(widget) => renderWidget(widget.id, widget.size)}
				</For>
			</div>

			{/* Customize Modal */}
			<Show when={showCustomize()}>
				<Modal
					open={showCustomize()}
					onClose={() => setShowCustomize(false)}
					title="Customize Dashboard"
					description="Show, hide, reorder, and resize widgets."
					footer={
						<>
							<Button variant="ghost" onClick={handleResetLayout}>Reset to Default</Button>
							<Button variant="ghost" onClick={() => setShowCustomize(false)}>Cancel</Button>
							<Button onClick={handleSaveLayout} loading={savingLayout()}>Save Layout</Button>
						</>
					}
				>
					<div class="space-y-2 max-h-[60vh] overflow-y-auto">
						<For each={editLayout()}>
							{(widget, idx) => {
								const def = WIDGET_DEFS.find(d => d.id === widget.id);
								if (!def) return null;
								return (
									<div class={`flex items-center gap-3 p-3 rounded-lg border transition-colors ${widget.visible ? 'bg-[var(--color-bg-secondary)] border-[var(--color-border-primary)]' : 'bg-[var(--color-bg-primary)] border-[var(--color-border-primary)] opacity-60'}`}>
										{/* Visibility checkbox */}
										<input
											type="checkbox"
											checked={widget.visible}
											onChange={() => toggleWidgetVisible(widget.id)}
											class="w-4 h-4 rounded border-[var(--color-border-primary)] bg-[var(--color-bg-secondary)] text-indigo-500 focus:ring-indigo-500/40"
										/>

										{/* Widget info */}
										<div class="flex-1 min-w-0">
											<p class="text-sm font-medium text-[var(--color-text-primary)]">{def.label}</p>
											<p class="text-xs text-[var(--color-text-tertiary)]">{def.description}</p>
										</div>

										{/* Size toggle */}
										<button
											class={`px-2 py-1 text-[10px] font-medium rounded border transition-colors cursor-pointer ${widget.size === 'full' ? 'bg-indigo-500/20 border-indigo-500/40 text-indigo-400' : 'bg-[var(--color-bg-tertiary)] border-[var(--color-border-primary)] text-[var(--color-text-tertiary)]'}`}
											onClick={() => toggleWidgetSize(widget.id)}
											title={widget.size === 'full' ? 'Full width — click for half' : 'Half width — click for full'}
										>
											{widget.size === 'full' ? 'Full' : 'Half'}
										</button>

										{/* Up/down buttons */}
										<div class="flex flex-col gap-0.5">
											<button
												class="p-0.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] transition-colors cursor-pointer disabled:opacity-30"
												onClick={() => moveWidget(widget.id, 'up')}
												disabled={idx() === 0}
											>
												<svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M14.77 12.79a.75.75 0 01-1.06-.02L10 8.832 6.29 12.77a.75.75 0 11-1.08-1.04l4.25-4.5a.75.75 0 011.08 0l4.25 4.5a.75.75 0 01-.02 1.06z" clip-rule="evenodd" /></svg>
											</button>
											<button
												class="p-0.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] transition-colors cursor-pointer disabled:opacity-30"
												onClick={() => moveWidget(widget.id, 'down')}
												disabled={idx() === editLayout().length - 1}
											>
												<svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" /></svg>
											</button>
										</div>
									</div>
								);
							}}
						</For>
					</div>
				</Modal>
			</Show>
		</PageContainer>
	);
};

export default DashboardPage;
