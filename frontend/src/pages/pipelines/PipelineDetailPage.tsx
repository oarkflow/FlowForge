import type { Component } from 'solid-js';
import { createSignal, createResource, createEffect, For, Show, Switch, Match, onMount } from 'solid-js';
import { useParams, useNavigate, A } from '@solidjs/router';
import * as yaml from 'js-yaml';
import PageContainer from '../../components/layout/PageContainer';
import Card from '../../components/ui/Card';
import Badge from '../../components/ui/Badge';
import Button from '../../components/ui/Button';
import Tabs from '../../components/ui/Tabs';
import Table, { type TableColumn } from '../../components/ui/Table';
import Modal from '../../components/ui/Modal';
import Input from '../../components/ui/Input';
import { toast } from '../../components/ui/Toast';
import PipelineBuilder from '../../components/pipeline/PipelineBuilder';
import { api, ApiRequestError, type RunDetail } from '../../api/client';
import type { Pipeline, PipelineRun, PipelineVersion, PipelineSchedule, PipelineLink, PipelineDAG, RunStatus } from '../../types';
import { formatRelativeTime, formatDuration, getStatusBadgeVariant, truncateCommitSha, describeCron, formatFutureRelativeTime, COMMON_TIMEZONES, copyToClipboard } from '../../utils/helpers';

const statusLabel: Record<RunStatus, string> = {
	success: 'Success', failure: 'Failed', cancelled: 'Cancelled', running: 'Running',
	queued: 'Queued', pending: 'Pending', skipped: 'Skipped', waiting_approval: 'Awaiting Approval',
};

// ---------------------------------------------------------------------------
// Cron Builder helper
// ---------------------------------------------------------------------------
type CronFrequency = 'every_minute' | 'hourly' | 'daily' | 'weekly' | 'monthly' | 'custom';

function buildCronExpression(
	freq: CronFrequency,
	minute: number,
	hour: number,
	dayOfMonth: number,
	dayOfWeek: number,
): string {
	switch (freq) {
		case 'every_minute': return '* * * * *';
		case 'hourly': return `${minute} * * * *`;
		case 'daily': return `${minute} ${hour} * * *`;
		case 'weekly': return `${minute} ${hour} * * ${dayOfWeek}`;
		case 'monthly': return `${minute} ${hour} ${dayOfMonth} * *`;
		default: return '* * * * *';
	}
}

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
	const [savingConfig, setSavingConfig] = createSignal(false);
	const [togglingActive, setTogglingActive] = createSignal(false);

	// Settings form
	const [editName, setEditName] = createSignal('');
	const [editDesc, setEditDesc] = createSignal('');
	const [editActive, setEditActive] = createSignal(true);

	// Schedule state
	const [schedules, setSchedules] = createSignal<PipelineSchedule[]>([]);
	const [schedulesLoading, setSchedulesLoading] = createSignal(false);
	const [showScheduleModal, setShowScheduleModal] = createSignal(false);
	const [editingSchedule, setEditingSchedule] = createSignal<PipelineSchedule | null>(null);
	const [savingSchedule, setSavingSchedule] = createSignal(false);
	const [deletingScheduleId, setDeletingScheduleId] = createSignal<string | null>(null);
	const [scheduleNextRuns, setScheduleNextRuns] = createSignal<string[]>([]);
	const [loadingNextRuns, setLoadingNextRuns] = createSignal(false);

	// Schedule form state
	const [schedFreq, setSchedFreq] = createSignal<CronFrequency>('daily');
	const [schedMinute, setSchedMinute] = createSignal(0);
	const [schedHour, setSchedHour] = createSignal(0);
	const [schedDom, setSchedDom] = createSignal(1);
	const [schedDow, setSchedDow] = createSignal(1);
	const [schedCron, setSchedCron] = createSignal('0 0 * * *');
	const [schedTimezone, setSchedTimezone] = createSignal('UTC');
	const [schedBranch, setSchedBranch] = createSignal('main');
	const [schedDescription, setSchedDescription] = createSignal('');
	const [schedVarKeys, setSchedVarKeys] = createSignal<string[]>([]);
	const [schedVarVals, setSchedVarVals] = createSignal<string[]>([]);

	// Composition state
	const [pipelineLinks, setPipelineLinks] = createSignal<PipelineLink[]>([]);
	const [linksLoading, setLinksLoading] = createSignal(false);
	const [showLinkModal, setShowLinkModal] = createSignal(false);
	const [editingLink, setEditingLink] = createSignal<PipelineLink | null>(null);
	const [savingLink, setSavingLink] = createSignal(false);
	const [deletingLinkId, setDeletingLinkId] = createSignal<string | null>(null);
	const [linkTargetId, setLinkTargetId] = createSignal('');
	const [linkType, setLinkType] = createSignal<'trigger' | 'fan_out' | 'fan_in'>('trigger');
	const [linkCondition, setLinkCondition] = createSignal('success');
	const [linkPassVars, setLinkPassVars] = createSignal(false);
	const [linkEnabled, setLinkEnabled] = createSignal(true);
	const [pipelineDAG, setPipelineDAG] = createSignal<PipelineDAG | null>(null);

	// Monorepo config state
	const [pathFilters, setPathFilters] = createSignal('');
	const [ignorePaths, setIgnorePaths] = createSignal('');
	const [savingMonorepo, setSavingMonorepo] = createSignal(false);

	// Update cron expression when builder values change
	createEffect(() => {
		const freq = schedFreq();
		if (freq !== 'custom') {
			const expr = buildCronExpression(freq, schedMinute(), schedHour(), schedDom(), schedDow());
			setSchedCron(expr);
		}
	});

	// Fetch next runs preview when cron changes
	createEffect(() => {
		const cron = schedCron();
		// Only preview when modal is open and cron looks valid
		if (showScheduleModal() && cron && cron.trim().split(/\s+/).length === 5) {
			// Preview runs from a create/edit schedule — we'll call the API only if editing existing
			const editing = editingSchedule();
			if (editing) {
				setLoadingNextRuns(true);
				api.schedules.getNextRuns(editing.id, 5)
					.then(runs => setScheduleNextRuns(runs))
					.catch(() => setScheduleNextRuns([]))
					.finally(() => setLoadingNextRuns(false));
			} else {
				setScheduleNextRuns([]);
			}
		}
	});

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

	// Load schedules
	const loadSchedules = async () => {
		setSchedulesLoading(true);
		try {
			const result = await api.schedules.listByPipeline(params.pid);
			setSchedules(result ?? []);
		} catch {
			setSchedules([]);
		} finally {
			setSchedulesLoading(false);
		}
	};

	// Load pipeline links (composition)
	const loadLinks = async () => {
		setLinksLoading(true);
		try {
			const result = await api.pipelineLinks.list(params.pid);
			setPipelineLinks(result ?? []);
		} catch {
			setPipelineLinks([]);
		} finally {
			setLinksLoading(false);
		}
	};

	// Load DAG
	const loadDAG = async () => {
		try {
			const dag = await api.pipelineLinks.getDAG(params.pid);
			setPipelineDAG(dag);
		} catch {
			setPipelineDAG(null);
		}
	};

	// Sync settings form when data loads
	const syncSettings = () => {
		const p = pipeline();
		if (p) {
			setEditName(p.name);
			setEditDesc(p.description || '');
			setEditActive(p.is_active);
			setPathFilters((p as any).path_filters || '');
			setIgnorePaths((p as any).ignore_paths || '');
		}
	};

	// Run columns
	const runColumns: TableColumn<PipelineRun>[] = [
		{
			key: 'number', header: '#', width: '70px', render: (row) => (
				<A href={`/projects/${params.id}/pipelines/${params.pid}/runs/${row.id}`} class="text-sm font-mono font-medium text-indigo-400 hover:text-indigo-300">#{row.number}</A>
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
			)
		},
		{
			key: 'branch', header: 'Branch', render: (row) => (
				<Show when={row.branch}>
					<span class="inline-flex items-center gap-1 text-xs font-mono px-2 py-0.5 rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] border border-[var(--color-border-primary)]">
						{row.branch}
					</span>
				</Show>
			)
		},
		{
			key: 'duration', header: 'Duration', width: '100px', align: 'right' as const, render: (row) => (
				<span class="text-xs font-mono text-[var(--color-text-secondary)]">
					{row.status === 'running' ? (
						<span class="text-violet-400 flex items-center gap-1 justify-end">
							<svg class="animate-spin h-3 w-3" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
							running
						</span>
					) : row.status === 'queued' ? 'queued' : formatDuration(row.duration_ms)}
				</span>
			)
		},
		{
			key: 'time', header: 'Started', width: '90px', align: 'right' as const, render: (row) => (
				<span class="text-xs text-[var(--color-text-tertiary)]">{formatRelativeTime(row.created_at)}</span>
			)
		},
	];

	const tabs = () => [
		{ id: 'configuration', label: 'Configuration' },
		{ id: 'runs', label: `Runs (${totalRuns()})` },
		{ id: 'triggers', label: 'Triggers' },
		{ id: 'composition', label: `Composition (${pipelineLinks().length})` },
		{ id: 'schedules', label: `Schedules (${schedules().length})` },
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

	const handleToggleActive = async () => {
		const p = pipeline();
		if (!p || togglingActive()) return;
		setTogglingActive(true);
		const newActive = !p.is_active;
		try {
			await api.pipelines.update(params.id, params.pid, { is_active: newActive ? 1 : 0 } as any);
			toast.success(newActive ? 'Pipeline enabled' : 'Pipeline disabled');
			setEditActive(newActive);
			refetch();
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to update pipeline';
			toast.error(msg);
		} finally {
			setTogglingActive(false);
		}
	};

	const handleSaveSettings = async () => {
		setSaving(true);
		try {
			await api.pipelines.update(params.id, params.pid, {
				name: editName().trim(),
				description: editDesc().trim() || undefined,
				is_active: editActive() ? 1 : 0,
			} as any);
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

	const handleSaveConfig = async (yamlContent: string) => {
		setSavingConfig(true);
		try {
			await api.pipelines.update(params.id, params.pid, {
				config_content: yamlContent,
			});
			toast.success('Pipeline configuration saved');
			refetch();
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to save configuration';
			toast.error(msg);
		} finally {
			setSavingConfig(false);
		}
	};

	// Extract triggers — prefer parsing from YAML config_content (on: section),
	// fall back to the triggers column which may be stale or empty for older pipelines.
	const triggers = (): Record<string, unknown> => {
		// First try parsing from YAML config
		const configContent = pipeline()?.config_content;
		if (configContent) {
			try {
				const spec = yaml.load(configContent) as Record<string, unknown> | null;
				if (spec?.on && typeof spec.on === 'object') {
					return spec.on as Record<string, unknown>;
				}
			} catch { /* fall through */ }
		}
		// Fall back to triggers column
		const raw = pipeline()?.triggers;
		if (!raw) return {};
		if (typeof raw === 'string') {
			try { return JSON.parse(raw) || {}; } catch { return {}; }
		}
		return raw;
	};
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

	// ---- Schedule handlers ----
	const openCreateSchedule = () => {
		setEditingSchedule(null);
		setSchedFreq('daily');
		setSchedMinute(0);
		setSchedHour(0);
		setSchedDom(1);
		setSchedDow(1);
		setSchedCron('0 0 * * *');
		setSchedTimezone('UTC');
		setSchedBranch('main');
		setSchedDescription('');
		setSchedVarKeys([]);
		setSchedVarVals([]);
		setScheduleNextRuns([]);
		setShowScheduleModal(true);
	};

	const openEditSchedule = (schedule: PipelineSchedule) => {
		setEditingSchedule(schedule);
		setSchedCron(schedule.cron_expression);
		setSchedTimezone(schedule.timezone);
		setSchedBranch(schedule.branch);
		setSchedDescription(schedule.description);
		setSchedFreq('custom');
		setSchedMinute(0);
		setSchedHour(0);
		setSchedDom(1);
		setSchedDow(1);

		// Parse variables
		try {
			const vars = JSON.parse(schedule.variables || '{}');
			const keys = Object.keys(vars);
			setSchedVarKeys(keys);
			setSchedVarVals(keys.map(k => vars[k]));
		} catch {
			setSchedVarKeys([]);
			setSchedVarVals([]);
		}

		setScheduleNextRuns([]);
		setShowScheduleModal(true);

		// Load next runs preview
		setLoadingNextRuns(true);
		api.schedules.getNextRuns(schedule.id, 5)
			.then(runs => setScheduleNextRuns(runs))
			.catch(() => setScheduleNextRuns([]))
			.finally(() => setLoadingNextRuns(false));
	};

	const handleSaveSchedule = async () => {
		const cron = schedCron().trim();
		if (!cron) { toast.error('Cron expression is required'); return; }

		// Build variables object
		const vars: Record<string, string> = {};
		schedVarKeys().forEach((key, i) => {
			const k = key.trim();
			if (k) vars[k] = schedVarVals()[i] || '';
		});

		setSavingSchedule(true);
		try {
			const editing = editingSchedule();
			if (editing) {
				await api.schedules.update(editing.id, {
					cron_expression: cron,
					timezone: schedTimezone(),
					branch: schedBranch().trim() || 'main',
					description: schedDescription().trim(),
					variables: JSON.stringify(vars),
				} as Partial<PipelineSchedule>);
				toast.success('Schedule updated');
			} else {
				await api.schedules.create(params.pid, {
					cron_expression: cron,
					timezone: schedTimezone(),
					branch: schedBranch().trim() || 'main',
					description: schedDescription().trim(),
					variables: vars,
				});
				toast.success('Schedule created');
			}
			setShowScheduleModal(false);
			loadSchedules();
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to save schedule';
			toast.error(msg);
		} finally {
			setSavingSchedule(false);
		}
	};

	const handleDeleteSchedule = async (id: string) => {
		setDeletingScheduleId(id);
		try {
			await api.schedules.delete(id);
			toast.success('Schedule deleted');
			loadSchedules();
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to delete schedule';
			toast.error(msg);
		} finally {
			setDeletingScheduleId(null);
		}
	};

	const handleToggleSchedule = async (schedule: PipelineSchedule) => {
		try {
			if (schedule.enabled) {
				await api.schedules.disable(schedule.id);
				toast.success('Schedule disabled');
			} else {
				await api.schedules.enable(schedule.id);
				toast.success('Schedule enabled');
			}
			loadSchedules();
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to toggle schedule';
			toast.error(msg);
		}
	};

	const addVariable = () => {
		setSchedVarKeys([...schedVarKeys(), '']);
		setSchedVarVals([...schedVarVals(), '']);
	};

	const removeVariable = (index: number) => {
		setSchedVarKeys(schedVarKeys().filter((_, i) => i !== index));
		setSchedVarVals(schedVarVals().filter((_, i) => i !== index));
	};

	const updateVarKey = (index: number, value: string) => {
		const keys = [...schedVarKeys()];
		keys[index] = value;
		setSchedVarKeys(keys);
	};

	const updateVarVal = (index: number, value: string) => {
		const vals = [...schedVarVals()];
		vals[index] = value;
		setSchedVarVals(vals);
	};

	// ---- Composition handlers ----
	const openCreateLink = () => {
		setEditingLink(null);
		setLinkTargetId('');
		setLinkType('trigger');
		setLinkCondition('success');
		setLinkPassVars(false);
		setLinkEnabled(true);
		setShowLinkModal(true);
	};

	const openEditLink = (link: PipelineLink) => {
		setEditingLink(link);
		setLinkTargetId(link.target_pipeline_id);
		setLinkType(link.link_type);
		setLinkCondition(link.condition || 'success');
		setLinkPassVars(link.pass_variables);
		setLinkEnabled(link.enabled);
		setShowLinkModal(true);
	};

	const handleSaveLink = async () => {
		if (!linkTargetId().trim()) { toast.error('Target pipeline ID is required'); return; }
		setSavingLink(true);
		try {
			const editing = editingLink();
			if (editing) {
				await api.pipelineLinks.update(editing.id, {
					target_pipeline_id: linkTargetId().trim(),
					link_type: linkType(),
					condition: linkCondition(),
					pass_variables: linkPassVars(),
					enabled: linkEnabled(),
				});
				toast.success('Pipeline link updated');
			} else {
				await api.pipelineLinks.create(params.pid, {
					target_pipeline_id: linkTargetId().trim(),
					link_type: linkType(),
					condition: linkCondition(),
					pass_variables: linkPassVars(),
					enabled: linkEnabled(),
				});
				toast.success('Pipeline link created');
			}
			setShowLinkModal(false);
			loadLinks();
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to save pipeline link';
			toast.error(msg);
		} finally {
			setSavingLink(false);
		}
	};

	const handleDeleteLink = async (id: string) => {
		setDeletingLinkId(id);
		try {
			await api.pipelineLinks.delete(id);
			toast.success('Pipeline link deleted');
			loadLinks();
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to delete link';
			toast.error(msg);
		} finally {
			setDeletingLinkId(null);
		}
	};

	const handleSaveMonorepo = async () => {
		setSavingMonorepo(true);
		try {
			await api.pipelines.update(params.id, params.pid, {
				path_filters: pathFilters().trim(),
				ignore_paths: ignorePaths().trim(),
			} as any);
			toast.success('Path filters saved');
			refetch();
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to save path filters';
			toast.error(msg);
		} finally {
			setSavingMonorepo(false);
		}
	};

	const linkTypeLabel = (type: string) => {
		switch (type) {
			case 'trigger': return 'Trigger';
			case 'fan_out': return 'Fan Out';
			case 'fan_in': return 'Fan In';
			default: return type;
		}
	};

	const linkTypeBadge = (type: string) => {
		switch (type) {
			case 'trigger': return 'info' as const;
			case 'fan_out': return 'warning' as const;
			case 'fan_in': return 'success' as const;
			default: return 'default' as const;
		}
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
						<button
							type="button"
							onClick={handleToggleActive}
							disabled={togglingActive()}
							class="cursor-pointer disabled:opacity-50"
							title={pipeline()!.is_active ? 'Click to disable pipeline' : 'Click to enable pipeline'}
						>
							<Badge variant={pipeline()!.is_active ? 'success' : 'default'} dot>
								{togglingActive() ? '...' : pipeline()!.is_active ? 'Active' : 'Disabled'}
							</Badge>
						</button>
					</Show>
					<Button size="sm" variant="outline" onClick={() => setShowTrigger(true)}
						disabled={!pipeline()?.is_active}
						title={!pipeline()?.is_active ? 'Enable the pipeline to trigger runs' : 'Trigger a new run'}
						icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M6.3 2.841A1.5 1.5 0 004 4.11V15.89a1.5 1.5 0 002.3 1.269l9.344-5.89a1.5 1.5 0 000-2.538L6.3 2.84z" /></svg>}
					>
						Trigger
					</Button>
					<Button size="sm" variant="danger" onClick={() => setShowDeleteConfirm(true)}
						icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.519.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM9.5 3.25a1.25 1.25 0 00-1.25 1.25v.1c.832-.07 1.671-.1 2.516-.1h.468c.845 0 1.684.03 2.516.1v-.1a1.25 1.25 0 00-1.25-1.25h-3z" clip-rule="evenodd" /></svg>}
					>
						Delete
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

			<Tabs tabs={tabs()} activeTab={activeTab()} onTabChange={(tab) => {
				setActiveTab(tab);
				if (tab === 'settings') syncSettings();
				if (tab === 'schedules') loadSchedules();
				if (tab === 'composition') { loadLinks(); loadDAG(); }
			}} class="mb-6" />

			<Show when={!data.loading} fallback={
				<div class="h-96 bg-[var(--color-bg-secondary)] rounded-xl animate-pulse" />
			}>
				<Switch>
					{/* ---- Configuration (Pipeline Builder) ---- */}
					<Match when={activeTab() === 'configuration'}>
						<div class="grid grid-cols-1 lg:grid-cols-4 gap-6">
							<div class="lg:col-span-3">
								<Show when={pipeline()?.config_source === 'repo'}>
									<Card title="Pipeline Configuration" description={`Source: ${pipeline()?.config_path ?? '.flowforge.yml'} (read from repository)`}>
										<Show when={pipeline()?.config_content} fallback={
											<div class="text-center py-8 text-[var(--color-text-tertiary)]">
												<p class="text-sm">Configuration is read from the repository.</p>
												<p class="text-xs mt-1">Commit changes to <code class="px-1 py-0.5 rounded bg-[var(--color-bg-tertiary)]">{pipeline()?.config_path}</code> to update the pipeline.</p>
											</div>
										}>
											<pre class="bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] rounded-lg p-4 overflow-auto max-h-[600px] text-sm font-mono leading-relaxed">
												<code class="text-[var(--color-text-secondary)]">{pipeline()!.config_content}</code>
											</pre>
										</Show>
									</Card>
								</Show>
								<Show when={pipeline()?.config_source !== 'repo'}>
									<PipelineBuilder
										initialYaml={pipeline()?.config_content ?? ''}
										projectId={params.id}
										pipelineId={params.pid}
										pipelineName={pipeline()?.name ?? 'pipeline'}
										onSave={handleSaveConfig}
										saving={savingConfig()}
									/>
								</Show>
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
										<Show when={(pushTrigger() as any)?.tags}>
											<div>
												<p class="text-sm font-medium text-[var(--color-text-primary)] mb-1">Tags</p>
												<div class="flex flex-wrap gap-2">
													<For each={(pushTrigger() as any).tags as string[]}>
														{(tag) => (
															<span class="text-xs font-mono px-2 py-1 rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">{tag}</span>
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
										<Show when={(pushTrigger() as any)?.ignore_paths}>
											<div>
												<p class="text-sm font-medium text-[var(--color-text-primary)] mb-1">Ignored Paths</p>
												<div class="flex flex-wrap gap-2">
													<For each={(pushTrigger() as any).ignore_paths as string[]}>
														{(path) => (
															<span class="text-xs font-mono px-2 py-1 rounded bg-red-500/10 text-red-400 border border-red-500/20">{path}</span>
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
								<Show when={!pipeline()?.is_active}>
									<p class="text-xs text-amber-400 mb-2">Pipeline is disabled. Enable it to trigger runs.</p>
								</Show>
								<Button onClick={() => setShowTrigger(true)} disabled={!pipeline()?.is_active} icon={
									<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M6.3 2.841A1.5 1.5 0 004 4.11V15.89a1.5 1.5 0 002.3 1.269l9.344-5.89a1.5 1.5 0 000-2.538L6.3 2.84z" /></svg>
								}>Trigger Run</Button>
							</Card>

							<Show when={!pushTrigger() && !prTrigger() && !scheduleTrigger() && !manualTrigger()}>
								<div class="text-center py-8 text-[var(--color-text-tertiary)]">
									<p class="text-sm">No triggers configured — pipeline triggers on all events.</p>
									<p class="text-xs mt-1">Add trigger rules in the pipeline configuration to filter events.</p>
								</div>
							</Show>
						</div>
					</Match>

					{/* ---- Composition tab ---- */}
					<Match when={activeTab() === 'composition'}>
						<div class="space-y-6">
							{/* DAG Visualization */}
							<Show when={pipelineDAG()}>
								<Card title="Stage Execution Graph" description="Visual representation of stage dependencies and parallel execution levels">
									<div class="space-y-4">
										<Show when={pipelineDAG()!.has_cycle}>
											<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30">
												<p class="text-sm text-red-400 font-medium">⚠ Cycle detected in stage dependencies</p>
												<p class="text-xs text-red-400/70 mt-1">The pipeline DAG contains circular dependencies. Please fix the stage <code>needs</code> configuration.</p>
											</div>
										</Show>
										<Show when={!pipelineDAG()!.has_cycle && pipelineDAG()!.levels.length > 0}>
											<div class="space-y-3">
												<For each={pipelineDAG()!.levels}>
													{(level, levelIdx) => (
														<div>
															<p class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider mb-2">Level {levelIdx()}{level.length > 1 ? ' (parallel)' : ''}</p>
															<div class="flex flex-wrap gap-2">
																<For each={level}>
																	{(stageName) => {
																		const node = pipelineDAG()!.nodes[stageName];
																		return (
																			<div class={`px-4 py-3 rounded-lg border ${level.length > 1 ? 'bg-indigo-500/5 border-indigo-500/20' : 'bg-[var(--color-bg-tertiary)] border-[var(--color-border-primary)]'}`}>
																				<p class="text-sm font-medium text-[var(--color-text-primary)]">{stageName}</p>
																				<Show when={node && node.dependencies.length > 0}>
																					<p class="text-xs text-[var(--color-text-tertiary)] mt-1">
																						needs: {node!.dependencies.join(', ')}
																					</p>
																				</Show>
																				<Show when={node && node.dependents.length > 0}>
																					<p class="text-xs text-[var(--color-text-tertiary)] mt-0.5">
																						blocks: {node!.dependents.join(', ')}
																					</p>
																				</Show>
																			</div>
																		);
																	}}
																</For>
															</div>
														</div>
													)}
												</For>
											</div>
										</Show>
										<Show when={!pipelineDAG()!.has_cycle && pipelineDAG()!.levels.length === 0}>
											<p class="text-sm text-[var(--color-text-tertiary)] text-center py-4">No stages with explicit dependencies. Stages run sequentially.</p>
										</Show>
									</div>
								</Card>
							</Show>

							{/* Pipeline Links */}
							<div class="flex items-center justify-between">
								<div>
									<h3 class="text-lg font-semibold text-[var(--color-text-primary)]">Pipeline Links</h3>
									<p class="text-sm text-[var(--color-text-tertiary)]">Cross-pipeline triggers, fan-out, and fan-in composition.</p>
								</div>
								<Button onClick={openCreateLink} icon={
									<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>
								}>
									Add Link
								</Button>
							</div>

							<Show when={linksLoading()}>
								<div class="h-48 bg-[var(--color-bg-secondary)] rounded-xl animate-pulse" />
							</Show>

							<Show when={!linksLoading()}>
								<Show when={pipelineLinks().length > 0} fallback={
									<Card>
										<div class="text-center py-12">
											<svg class="w-12 h-12 mx-auto text-[var(--color-text-tertiary)] mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
												<path stroke-linecap="round" stroke-linejoin="round" d="M13.19 8.688a4.5 4.5 0 011.242 7.244l-4.5 4.5a4.5 4.5 0 01-6.364-6.364l1.757-1.757m13.35-.622l1.757-1.757a4.5 4.5 0 00-6.364-6.364l-4.5 4.5a4.5 4.5 0 001.242 7.244" />
											</svg>
											<p class="text-sm text-[var(--color-text-tertiary)] mb-2">No pipeline links configured</p>
											<p class="text-xs text-[var(--color-text-tertiary)] mb-4">Link pipelines together to create composition workflows — trigger downstream pipelines, fan-out to multiple targets, or fan-in from multiple sources.</p>
											<Button size="sm" onClick={openCreateLink}>Create Link</Button>
										</div>
									</Card>
								}>
									<Card padding={false}>
										<div class="overflow-x-auto">
											<table class="w-full text-sm">
												<thead>
													<tr class="border-b border-[var(--color-border-primary)]">
														<th class="text-left px-4 py-3 text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Type</th>
														<th class="text-left px-4 py-3 text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Target Pipeline</th>
														<th class="text-left px-4 py-3 text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Condition</th>
														<th class="text-center px-4 py-3 text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Pass Vars</th>
														<th class="text-center px-4 py-3 text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Enabled</th>
														<th class="text-right px-4 py-3 text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Actions</th>
													</tr>
												</thead>
												<tbody class="divide-y divide-[var(--color-border-primary)]">
													<For each={pipelineLinks()}>
														{(link) => (
															<tr class="hover:bg-[var(--color-bg-hover)]">
																<td class="px-4 py-3">
																	<Badge variant={linkTypeBadge(link.link_type)} size="sm">{linkTypeLabel(link.link_type)}</Badge>
																</td>
																<td class="px-4 py-3">
																	<span class="text-xs font-mono text-[var(--color-text-secondary)]">{link.target_pipeline_id}</span>
																</td>
																<td class="px-4 py-3">
																	<span class="text-xs px-2 py-0.5 rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] border border-[var(--color-border-primary)]">
																		{link.condition || 'success'}
																	</span>
																</td>
																<td class="px-4 py-3 text-center">
																	<span class={`text-xs ${link.pass_variables ? 'text-emerald-400' : 'text-[var(--color-text-tertiary)]'}`}>
																		{link.pass_variables ? 'Yes' : 'No'}
																	</span>
																</td>
																<td class="px-4 py-3 text-center">
																	<Badge variant={link.enabled ? 'success' : 'default'} size="sm" dot>
																		{link.enabled ? 'Active' : 'Disabled'}
																	</Badge>
																</td>
																<td class="px-4 py-3 text-right">
																	<div class="flex items-center justify-end gap-1">
																		<button
																			class="p-1 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] transition-colors cursor-pointer"
																			onClick={() => openEditLink(link)}
																			title="Edit link"
																		>
																			<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10" /></svg>
																		</button>
																		<button
																			class="p-1 rounded hover:bg-red-500/10 text-[var(--color-text-tertiary)] hover:text-red-400 transition-colors cursor-pointer"
																			onClick={() => handleDeleteLink(link.id)}
																			disabled={deletingLinkId() === link.id}
																			title="Delete link"
																		>
																			<Show when={deletingLinkId() === link.id} fallback={
																				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>
																			}>
																				<svg class="animate-spin w-4 h-4" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
																			</Show>
																		</button>
																	</div>
																</td>
															</tr>
														)}
													</For>
												</tbody>
											</table>
										</div>
									</Card>
								</Show>
							</Show>
						</div>
					</Match>

					{/* ---- Schedules tab ---- */}
					<Match when={activeTab() === 'schedules'}>
						<div class="space-y-4">
							<div class="flex items-center justify-between">
								<div>
									<h3 class="text-lg font-semibold text-[var(--color-text-primary)]">Pipeline Schedules</h3>
									<p class="text-sm text-[var(--color-text-tertiary)]">Configure cron-based schedules to automatically trigger this pipeline.</p>
								</div>
								<Button onClick={openCreateSchedule} icon={
									<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>
								}>
									New Schedule
								</Button>
							</div>

							<Show when={schedulesLoading()}>
								<div class="h-48 bg-[var(--color-bg-secondary)] rounded-xl animate-pulse" />
							</Show>

							<Show when={!schedulesLoading()}>
								<Show when={schedules().length > 0} fallback={
									<Card>
										<div class="text-center py-12">
											<svg class="w-12 h-12 mx-auto text-[var(--color-text-tertiary)] mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
												<path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
											</svg>
											<p class="text-sm text-[var(--color-text-tertiary)] mb-2">No schedules configured</p>
											<p class="text-xs text-[var(--color-text-tertiary)] mb-4">Create a schedule to automatically trigger this pipeline on a recurring basis.</p>
											<Button size="sm" onClick={openCreateSchedule}>Create Schedule</Button>
										</div>
									</Card>
								}>
									<Card padding={false}>
										<div class="overflow-x-auto">
											<table class="w-full text-sm">
												<thead>
													<tr class="border-b border-[var(--color-border-primary)]">
														<th class="text-left px-4 py-3 text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Schedule</th>
														<th class="text-left px-4 py-3 text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Branch</th>
														<th class="text-left px-4 py-3 text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Timezone</th>
														<th class="text-left px-4 py-3 text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Next Run</th>
														<th class="text-left px-4 py-3 text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Last Run</th>
														<th class="text-right px-4 py-3 text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Runs</th>
														<th class="text-center px-4 py-3 text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Enabled</th>
														<th class="text-right px-4 py-3 text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Actions</th>
													</tr>
												</thead>
												<tbody class="divide-y divide-[var(--color-border-primary)]">
													<For each={schedules()}>
														{(schedule) => (
															<tr class="hover:bg-[var(--color-bg-hover)]">
																<td class="px-4 py-3">
																	<div>
																		<p class="text-sm text-[var(--color-text-primary)]">{describeCron(schedule.cron_expression)}</p>
																		<p class="text-xs font-mono text-[var(--color-text-tertiary)] mt-0.5">{schedule.cron_expression}</p>
																		<Show when={schedule.description}>
																			<p class="text-xs text-[var(--color-text-tertiary)] mt-0.5 italic">{schedule.description}</p>
																		</Show>
																	</div>
																</td>
																<td class="px-4 py-3">
																	<span class="text-xs font-mono px-2 py-0.5 rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] border border-[var(--color-border-primary)]">
																		{schedule.branch}
																	</span>
																</td>
																<td class="px-4 py-3">
																	<span class="text-xs text-[var(--color-text-secondary)]">{schedule.timezone}</span>
																</td>
																<td class="px-4 py-3">
																	<Show when={schedule.next_run_at} fallback={<span class="text-xs text-[var(--color-text-tertiary)]">—</span>}>
																		<span class="text-xs text-[var(--color-text-secondary)]" title={schedule.next_run_at!}>
																			{formatFutureRelativeTime(schedule.next_run_at!)}
																		</span>
																	</Show>
																</td>
																<td class="px-4 py-3">
																	<Show when={schedule.last_run_at} fallback={<span class="text-xs text-[var(--color-text-tertiary)]">Never</span>}>
																		<div class="flex items-center gap-2">
																			<Show when={schedule.last_run_status}>
																				<Badge variant={schedule.last_run_status === 'success' ? 'success' : schedule.last_run_status === 'failure' ? 'error' : 'default'} size="sm">
																					{schedule.last_run_status}
																				</Badge>
																			</Show>
																			<span class="text-xs text-[var(--color-text-tertiary)]">{formatRelativeTime(schedule.last_run_at!)}</span>
																		</div>
																	</Show>
																</td>
																<td class="px-4 py-3 text-right">
																	<span class="text-sm font-medium text-[var(--color-text-secondary)]">{schedule.run_count}</span>
																</td>
																<td class="px-4 py-3 text-center">
																	<button
																		class={`w-9 h-5 rounded-full relative cursor-pointer transition-colors ${schedule.enabled ? 'bg-indigo-500' : 'bg-gray-600'}`}
																		onClick={() => handleToggleSchedule(schedule)}
																		title={schedule.enabled ? 'Disable schedule' : 'Enable schedule'}
																	>
																		<div class={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${schedule.enabled ? 'left-4' : 'left-0.5'}`} />
																	</button>
																</td>
																<td class="px-4 py-3 text-right">
																	<div class="flex items-center justify-end gap-1">
																		<button
																			class="p-1 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] transition-colors cursor-pointer"
																			onClick={() => openEditSchedule(schedule)}
																			title="Edit schedule"
																		>
																			<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10" /></svg>
																		</button>
																		<button
																			class="p-1 rounded hover:bg-red-500/10 text-[var(--color-text-tertiary)] hover:text-red-400 transition-colors cursor-pointer"
																			onClick={() => handleDeleteSchedule(schedule.id)}
																			disabled={deletingScheduleId() === schedule.id}
																			title="Delete schedule"
																		>
																			<Show when={deletingScheduleId() === schedule.id} fallback={
																				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>
																			}>
																				<svg class="animate-spin w-4 h-4" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
																			</Show>
																		</button>
																	</div>
																</td>
															</tr>
														)}
													</For>
												</tbody>
											</table>
										</div>
									</Card>
								</Show>
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

							{/* Monorepo / Path Filters */}
							<Card title="Monorepo Path Filters" description="Control which file changes trigger this pipeline. Useful for monorepo setups where multiple pipelines live in one repository.">
								<div class="space-y-4">
									<div>
										<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-1">Path Filters</label>
										<input
											type="text"
											class="w-full px-3 py-2 text-sm font-mono rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
											value={pathFilters()}
											onInput={(e) => setPathFilters(e.currentTarget.value)}
											placeholder="services/api/**, shared/**, libs/common/**"
										/>
										<p class="text-xs text-[var(--color-text-tertiary)] mt-1">
											Comma-separated glob patterns. Only changes matching these paths will trigger the pipeline. Leave empty to match all files.
										</p>
									</div>
									<div>
										<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-1">Ignore Paths</label>
										<input
											type="text"
											class="w-full px-3 py-2 text-sm font-mono rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
											value={ignorePaths()}
											onInput={(e) => setIgnorePaths(e.currentTarget.value)}
											placeholder="docs/**, *.md, .gitignore"
										/>
										<p class="text-xs text-[var(--color-text-tertiary)] mt-1">
											Comma-separated glob patterns to exclude. Changes matching these paths will be ignored even if they match path filters.
										</p>
									</div>

									{/* Preview of included/excluded directories */}
									<Show when={pathFilters().trim() || ignorePaths().trim()}>
										<div class="p-3 rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)]">
											<p class="text-xs font-medium text-[var(--color-text-primary)] mb-2">Preview</p>
											<Show when={pathFilters().trim()}>
												<div class="mb-2">
													<p class="text-xs text-emerald-400 mb-1">✓ Included paths:</p>
													<div class="flex flex-wrap gap-1">
														<For each={pathFilters().split(',').map(p => p.trim()).filter(Boolean)}>
															{(pattern) => (
																<span class="text-xs font-mono px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">{pattern}</span>
															)}
														</For>
													</div>
												</div>
											</Show>
											<Show when={ignorePaths().trim()}>
												<div>
													<p class="text-xs text-red-400 mb-1">✗ Excluded paths:</p>
													<div class="flex flex-wrap gap-1">
														<For each={ignorePaths().split(',').map(p => p.trim()).filter(Boolean)}>
															{(pattern) => (
																<span class="text-xs font-mono px-2 py-0.5 rounded bg-red-500/10 text-red-400 border border-red-500/20">{pattern}</span>
															)}
														</For>
													</div>
												</div>
											</Show>
										</div>
									</Show>

									<Button onClick={handleSaveMonorepo} loading={savingMonorepo()}>Save Path Filters</Button>
								</div>
							</Card>

							{/* Pipeline Status Badge */}
							<Card title="Status Badge" description="Embed a live status badge for this pipeline in your README or documentation.">
								<div class="space-y-4">
									{/* Badge preview */}
									<div>
										<p class="text-sm font-medium text-[var(--color-text-primary)] mb-2">Preview</p>
										<div class="p-4 rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] flex items-center gap-3">
											<img
												src={`${window.location.origin}/api/v1/badges/pipeline/${params.pid}`}
												alt="Pipeline Status Badge"
												class="h-5"
											/>
										</div>
									</div>

									{/* Badge URL */}
									<div>
										<p class="text-sm font-medium text-[var(--color-text-primary)] mb-1">Badge URL</p>
										<div class="flex items-center gap-2">
											<code class="flex-1 text-xs font-mono bg-[var(--color-bg-tertiary)] px-3 py-2 rounded border border-[var(--color-border-primary)] text-[var(--color-text-secondary)] break-all">
												{`${window.location.origin}/api/v1/badges/pipeline/${params.pid}`}
											</code>
											<Button size="sm" variant="outline" onClick={() => { copyToClipboard(`${window.location.origin}/api/v1/badges/pipeline/${params.pid}`); toast.success('Copied!'); }}>Copy</Button>
										</div>
									</div>

									{/* Markdown snippet */}
									<div>
										<p class="text-sm font-medium text-[var(--color-text-primary)] mb-1">Markdown</p>
										<div class="flex items-center gap-2">
											<code class="flex-1 text-xs font-mono bg-[var(--color-bg-tertiary)] px-3 py-2 rounded border border-[var(--color-border-primary)] text-[var(--color-text-secondary)] break-all">
												{`![Pipeline Status](${window.location.origin}/api/v1/badges/pipeline/${params.pid})`}
											</code>
											<Button size="sm" variant="outline" onClick={() => { copyToClipboard(`![Pipeline Status](${window.location.origin}/api/v1/badges/pipeline/${params.pid})`); toast.success('Copied!'); }}>Copy</Button>
										</div>
									</div>

									{/* HTML snippet */}
									<div>
										<p class="text-sm font-medium text-[var(--color-text-primary)] mb-1">HTML</p>
										<div class="flex items-center gap-2">
											<code class="flex-1 text-xs font-mono bg-[var(--color-bg-tertiary)] px-3 py-2 rounded border border-[var(--color-border-primary)] text-[var(--color-text-secondary)] break-all">
												{`<img src="${window.location.origin}/api/v1/badges/pipeline/${params.pid}" alt="Pipeline Status" />`}
											</code>
											<Button size="sm" variant="outline" onClick={() => { copyToClipboard(`<img src="${window.location.origin}/api/v1/badges/pipeline/${params.pid}" alt="Pipeline Status" />`); toast.success('Copied!'); }}>Copy</Button>
										</div>
									</div>
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

			{/* Schedule Create/Edit Modal */}
			<Show when={showScheduleModal()}>
				<Modal
					open={showScheduleModal()}
					onClose={() => setShowScheduleModal(false)}
					title={editingSchedule() ? 'Edit Schedule' : 'Create Schedule'}
					description={editingSchedule() ? 'Modify this scheduled pipeline run.' : 'Configure a new cron-based schedule for this pipeline.'}
					footer={
						<>
							<Button variant="ghost" onClick={() => setShowScheduleModal(false)}>Cancel</Button>
							<Button onClick={handleSaveSchedule} loading={savingSchedule()}>
								{editingSchedule() ? 'Update Schedule' : 'Create Schedule'}
							</Button>
						</>
					}
				>
					<div class="space-y-5 max-h-[70vh] overflow-y-auto pr-1">
						{/* Cron Builder */}
						<div>
							<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-2">Frequency</label>
							<div class="grid grid-cols-3 gap-2">
								{(['every_minute', 'hourly', 'daily', 'weekly', 'monthly', 'custom'] as CronFrequency[]).map(f => (
									<button
										class={`px-3 py-2 text-xs rounded-lg border transition-colors cursor-pointer ${schedFreq() === f
											? 'bg-indigo-500/20 border-indigo-500/40 text-indigo-400'
											: 'bg-[var(--color-bg-tertiary)] border-[var(--color-border-primary)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-secondary)]'
											}`}
										onClick={() => setSchedFreq(f)}
									>
										{f === 'every_minute' ? 'Every Minute' : f === 'custom' ? 'Custom' : f.charAt(0).toUpperCase() + f.slice(1)}
									</button>
								))}
							</div>
						</div>

						{/* Builder inputs based on frequency */}
						<Show when={schedFreq() === 'hourly'}>
							<div>
								<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-1">At minute</label>
								<select
									class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)]"
									value={schedMinute()}
									onChange={(e) => setSchedMinute(parseInt(e.currentTarget.value))}
								>
									<For each={Array.from({ length: 60 }, (_, i) => i)}>
										{(m) => <option value={m}>{m.toString().padStart(2, '0')}</option>}
									</For>
								</select>
							</div>
						</Show>

						<Show when={schedFreq() === 'daily' || schedFreq() === 'weekly' || schedFreq() === 'monthly'}>
							<div class="grid grid-cols-2 gap-3">
								<div>
									<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-1">Hour</label>
									<select
										class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)]"
										value={schedHour()}
										onChange={(e) => setSchedHour(parseInt(e.currentTarget.value))}
									>
										<For each={Array.from({ length: 24 }, (_, i) => i)}>
											{(h) => <option value={h}>{h.toString().padStart(2, '0')}:00</option>}
										</For>
									</select>
								</div>
								<div>
									<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-1">Minute</label>
									<select
										class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)]"
										value={schedMinute()}
										onChange={(e) => setSchedMinute(parseInt(e.currentTarget.value))}
									>
										<For each={Array.from({ length: 60 }, (_, i) => i)}>
											{(m) => <option value={m}>{m.toString().padStart(2, '0')}</option>}
										</For>
									</select>
								</div>
							</div>
						</Show>

						<Show when={schedFreq() === 'weekly'}>
							<div>
								<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-1">Day of Week</label>
								<select
									class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)]"
									value={schedDow()}
									onChange={(e) => setSchedDow(parseInt(e.currentTarget.value))}
								>
									{['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'].map((name, i) => (
										<option value={i}>{name}</option>
									))}
								</select>
							</div>
						</Show>

						<Show when={schedFreq() === 'monthly'}>
							<div>
								<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-1">Day of Month</label>
								<select
									class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)]"
									value={schedDom()}
									onChange={(e) => setSchedDom(parseInt(e.currentTarget.value))}
								>
									<For each={Array.from({ length: 31 }, (_, i) => i + 1)}>
										{(d) => <option value={d}>{d}</option>}
									</For>
								</select>
							</div>
						</Show>

						{/* Cron expression (always shown) */}
						<div>
							<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-1">
								Cron Expression
								<Show when={schedFreq() !== 'custom'}>
									<span class="text-xs font-normal text-[var(--color-text-tertiary)] ml-2">(auto-generated)</span>
								</Show>
							</label>
							<input
								type="text"
								class="w-full px-3 py-2 text-sm font-mono rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
								value={schedCron()}
								onInput={(e) => {
									setSchedCron(e.currentTarget.value);
									if (schedFreq() !== 'custom') setSchedFreq('custom');
								}}
								placeholder="* * * * *"
							/>
							<p class="text-xs text-[var(--color-text-tertiary)] mt-1">
								Format: minute hour day-of-month month day-of-week
							</p>
							<Show when={schedCron()}>
								<p class="text-xs text-indigo-400 mt-1">
									→ {describeCron(schedCron())}
								</p>
							</Show>
						</div>

						{/* Timezone */}
						<div>
							<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-1">Timezone</label>
							<select
								class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)]"
								value={schedTimezone()}
								onChange={(e) => setSchedTimezone(e.currentTarget.value)}
							>
								<For each={COMMON_TIMEZONES}>
									{(tz) => <option value={tz}>{tz.replace(/_/g, ' ')}</option>}
								</For>
							</select>
						</div>

						{/* Branch */}
						<div>
							<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-1">Branch</label>
							<input
								type="text"
								class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
								value={schedBranch()}
								onInput={(e) => setSchedBranch(e.currentTarget.value)}
								placeholder="main"
							/>
						</div>

						{/* Description */}
						<div>
							<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-1">Description (optional)</label>
							<input
								type="text"
								class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
								value={schedDescription()}
								onInput={(e) => setSchedDescription(e.currentTarget.value)}
								placeholder="Nightly build, weekly release, etc."
							/>
						</div>

						{/* Variables */}
						<div>
							<div class="flex items-center justify-between mb-2">
								<label class="text-sm font-medium text-[var(--color-text-primary)]">Variables (optional)</label>
								<button
									class="text-xs text-indigo-400 hover:text-indigo-300 cursor-pointer"
									onClick={addVariable}
								>
									+ Add Variable
								</button>
							</div>
							<Show when={schedVarKeys().length > 0}>
								<div class="space-y-2">
									<For each={schedVarKeys()}>
										{(_, i) => (
											<div class="flex items-center gap-2">
												<input
													type="text"
													class="flex-1 px-3 py-1.5 text-xs font-mono rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
													value={schedVarKeys()[i()]}
													onInput={(e) => updateVarKey(i(), e.currentTarget.value)}
													placeholder="KEY"
												/>
												<input
													type="text"
													class="flex-1 px-3 py-1.5 text-xs font-mono rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
													value={schedVarVals()[i()]}
													onInput={(e) => updateVarVal(i(), e.currentTarget.value)}
													placeholder="value"
												/>
												<button
													class="p-1 rounded hover:bg-red-500/10 text-[var(--color-text-tertiary)] hover:text-red-400 cursor-pointer"
													onClick={() => removeVariable(i())}
												>
													<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
												</button>
											</div>
										)}
									</For>
								</div>
							</Show>
						</div>

						{/* Next runs preview */}
						<Show when={editingSchedule() && scheduleNextRuns().length > 0}>
							<div>
								<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-2">Next Scheduled Runs</label>
								<Show when={loadingNextRuns()}>
									<div class="h-20 bg-[var(--color-bg-tertiary)] rounded-lg animate-pulse" />
								</Show>
								<Show when={!loadingNextRuns()}>
									<div class="space-y-1 bg-[var(--color-bg-tertiary)] rounded-lg p-3">
										<For each={scheduleNextRuns()}>
											{(run, i) => (
												<div class="flex items-center justify-between text-xs">
													<span class="text-[var(--color-text-tertiary)]">#{i() + 1}</span>
													<span class="text-[var(--color-text-secondary)]">
														{new Date(run).toLocaleString()}
													</span>
													<span class="text-[var(--color-text-tertiary)]">
														{formatFutureRelativeTime(run)}
													</span>
												</div>
											)}
										</For>
									</div>
								</Show>
							</div>
						</Show>
					</div>
				</Modal>
			</Show>

			{/* Pipeline Link Create/Edit Modal */}
			<Show when={showLinkModal()}>
				<Modal
					open={showLinkModal()}
					onClose={() => setShowLinkModal(false)}
					title={editingLink() ? 'Edit Pipeline Link' : 'Create Pipeline Link'}
					description={editingLink() ? 'Modify this pipeline composition link.' : 'Link this pipeline to another for cross-pipeline triggering.'}
					footer={
						<>
							<Button variant="ghost" onClick={() => setShowLinkModal(false)}>Cancel</Button>
							<Button onClick={handleSaveLink} loading={savingLink()}>
								{editingLink() ? 'Update Link' : 'Create Link'}
							</Button>
						</>
					}
				>
					<div class="space-y-4">
						{/* Target Pipeline ID */}
						<div>
							<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-1">Target Pipeline ID</label>
							<input
								type="text"
								class="w-full px-3 py-2 text-sm font-mono rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
								value={linkTargetId()}
								onInput={(e) => setLinkTargetId(e.currentTarget.value)}
								placeholder="Enter target pipeline ID"
							/>
							<p class="text-xs text-[var(--color-text-tertiary)] mt-1">The pipeline that will be triggered when this pipeline completes.</p>
						</div>

						{/* Link Type */}
						<div>
							<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-1">Link Type</label>
							<div class="grid grid-cols-3 gap-2">
								{(['trigger', 'fan_out', 'fan_in'] as const).map(t => (
									<button
										class={`px-3 py-2 text-xs rounded-lg border transition-colors cursor-pointer ${linkType() === t
											? 'bg-indigo-500/20 border-indigo-500/40 text-indigo-400'
											: 'bg-[var(--color-bg-tertiary)] border-[var(--color-border-primary)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-secondary)]'
											}`}
										onClick={() => setLinkType(t)}
									>
										{linkTypeLabel(t)}
									</button>
								))}
							</div>
							<p class="text-xs text-[var(--color-text-tertiary)] mt-1">
								{linkType() === 'trigger' ? 'Trigger the target pipeline after this one completes.' :
									linkType() === 'fan_out' ? 'Fan out to multiple downstream pipelines in parallel.' :
										'Wait for multiple upstream pipelines before triggering.'}
							</p>
						</div>

						{/* Condition */}
						<div>
							<label class="block text-sm font-medium text-[var(--color-text-primary)] mb-1">Condition</label>
							<select
								class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)]"
								value={linkCondition()}
								onChange={(e) => setLinkCondition(e.currentTarget.value)}
							>
								<option value="success">On Success</option>
								<option value="failure">On Failure</option>
								<option value="always">Always</option>
							</select>
							<p class="text-xs text-[var(--color-text-tertiary)] mt-1">When should the target pipeline be triggered?</p>
						</div>

						{/* Pass Variables */}
						<div class="flex items-center justify-between p-3 rounded-lg bg-[var(--color-bg-tertiary)]">
							<div>
								<p class="text-sm font-medium text-[var(--color-text-primary)]">Pass Variables</p>
								<p class="text-xs text-[var(--color-text-tertiary)]">Forward pipeline variables to the target pipeline</p>
							</div>
							<button
								class={`w-10 h-6 rounded-full relative cursor-pointer transition-colors ${linkPassVars() ? 'bg-indigo-500' : 'bg-gray-600'}`}
								onClick={() => setLinkPassVars(!linkPassVars())}
							>
								<div class={`absolute top-1 w-4 h-4 rounded-full bg-white transition-transform ${linkPassVars() ? 'left-5' : 'left-1'}`} />
							</button>
						</div>

						{/* Enabled */}
						<div class="flex items-center justify-between p-3 rounded-lg bg-[var(--color-bg-tertiary)]">
							<div>
								<p class="text-sm font-medium text-[var(--color-text-primary)]">Enabled</p>
								<p class="text-xs text-[var(--color-text-tertiary)]">Link is active and will trigger downstream pipelines</p>
							</div>
							<button
								class={`w-10 h-6 rounded-full relative cursor-pointer transition-colors ${linkEnabled() ? 'bg-indigo-500' : 'bg-gray-600'}`}
								onClick={() => setLinkEnabled(!linkEnabled())}
							>
								<div class={`absolute top-1 w-4 h-4 rounded-full bg-white transition-transform ${linkEnabled() ? 'left-5' : 'left-1'}`} />
							</button>
						</div>
					</div>
				</Modal>
			</Show>
		</PageContainer>
	);
};

export default PipelineDetailPage;
