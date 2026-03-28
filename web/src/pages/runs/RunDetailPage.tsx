import type { Component } from 'solid-js';
import { createSignal, createResource, For, Show, onMount, onCleanup, createEffect, lazy, Suspense } from 'solid-js';
import { useParams, A, useNavigate } from '@solidjs/router';
import Badge from '../../components/ui/Badge';
import Button from '../../components/ui/Button';
import { toast } from '../../components/ui/Toast';
import { api, ApiRequestError, type RunDetail } from '../../api/client';
import { createRunLogSocket } from '../../api/websocket';
import type { RunStatus, Artifact, LogLine } from '../../types';
import { formatDuration, formatRelativeTime, getStatusBadgeVariant, truncateCommitSha, formatBytes } from '../../utils/helpers';

const TestResults = lazy(() => import('../../components/runs/TestResults'));
const CoverageReport = lazy(() => import('../../components/runs/CoverageReport'));
const ResourceGraph = lazy(() => import('../../components/runs/ResourceGraph'));

// ---------------------------------------------------------------------------
// Status icon
// ---------------------------------------------------------------------------
const statusIcon = (status: RunStatus) => {
	switch (status) {
		case 'success': return (
			<svg class="w-4 h-4 text-emerald-400" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" />
			</svg>
		);
		case 'failure': return (
			<svg class="w-4 h-4 text-red-400" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clip-rule="evenodd" />
			</svg>
		);
		case 'running': return (
			<svg class="w-4 h-4 text-violet-400 animate-spin" viewBox="0 0 24 24" fill="none">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
				<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
			</svg>
		);
		case 'queued': case 'pending': return (
			<div class="w-4 h-4 rounded-full border-2 border-gray-500" />
		);
		case 'cancelled': return (
			<svg class="w-4 h-4 text-gray-500" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM6.75 9.25a.75.75 0 000 1.5h6.5a.75.75 0 000-1.5h-6.5z" clip-rule="evenodd" />
			</svg>
		);
		case 'skipped': return (
			<svg class="w-4 h-4 text-gray-500" viewBox="0 0 20 20" fill="currentColor">
				<path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
			</svg>
		);
		default: return <div class="w-4 h-4 rounded-full border-2 border-amber-400" />;
	}
};

const statusLabel: Record<RunStatus, string> = {
	success: 'Passed', failure: 'Failed', cancelled: 'Cancelled', running: 'Running',
	queued: 'Queued', pending: 'Pending', skipped: 'Skipped', waiting_approval: 'Awaiting Approval',
};

// Simple ANSI to HTML converter
function ansiToHtml(text: string): string {
	return text
		.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
		.replace(/\x1b\[36m/g, '<span class="text-cyan-400">')
		.replace(/\x1b\[32m/g, '<span class="text-emerald-400">')
		.replace(/\x1b\[33m/g, '<span class="text-amber-400">')
		.replace(/\x1b\[31m/g, '<span class="text-red-400">')
		.replace(/\x1b\[90m/g, '<span class="text-gray-500">')
		.replace(/\x1b\[1m/g, '<span class="font-bold">')
		.replace(/\x1b\[0m/g, '</span>')
		.replace(/\x1b\[\d+m/g, '');
}

// Detect whether a stderr line is truly an error or just informational output.
// Many tools (sbt, npm, gradle, docker, cargo) write progress, notices, and
// diagnostics to stderr even when nothing is wrong.
function isStderrError(content: string): boolean {
	const lower = content.toLowerCase();
	// Definitive error indicators
	if (lower.includes('[error]') || lower.includes('error:') || lower.includes('fatal:') ||
		lower.includes('exception') || lower.includes('failed') || lower.includes('failure') ||
		lower.includes('cannot find') || lower.includes('not found') || lower.includes('no such file') ||
		lower.includes('compilation failed') || lower.includes('exit code') ||
		lower.includes('panic:') || lower.includes('segfault') || lower.includes('traceback'))
		return true;
	// Definitive non-error indicators (informational stderr)
	if (lower.includes('[info]') || lower.includes('[launcher]') || lower.includes('[warn]') ||
		lower.includes('npm notice') || lower.includes('npm warn') ||
		lower.includes('downloading') || lower.includes('resolving') ||
		lower.includes('getting ') || lower.includes('compiling ') ||
		lower.includes('new major version') || lower.includes('changelog:') ||
		lower.includes('to update run:'))
		return false;
	return false; // Default: treat unknown stderr as informational, not error
}

// Get CSS class for a log line based on its stream type and content
function streamClass(stream: string, content?: string): string {
	switch (stream) {
		case 'stderr': {
			if (content && isStderrError(content)) return 'text-red-400';
			if (content) {
				const lower = content.toLowerCase();
				if (lower.includes('[warn]') || lower.includes('npm warn') || lower.includes('warning:')) return 'text-amber-400';
			}
			return 'text-[#8b949e]'; // Dimmed gray for informational stderr
		}
		case 'system': {
			if (content) {
				const lower = content.toLowerCase();
				if (lower.includes('— success') || lower.includes('- success') || lower.includes('— passed') || lower.includes('checked out')) return 'text-emerald-400';
				if (lower.includes('— fail') || lower.includes('- fail') || lower.includes('— error')) return 'text-red-400';
				if (lower.includes('warning:')) return 'text-amber-400';
			}
			return 'text-blue-400/80';
		}
		default: return 'text-[#c9d1d9]'; // stdout — default terminal foreground
	}
}

// Stream indicator icon (small colored dot/marker)
function streamIndicator(stream: string, content?: string) {
	switch (stream) {
		case 'stderr': {
			if (content && isStderrError(content)) return <span class="w-1.5 h-1.5 rounded-full bg-red-400 flex-shrink-0 mt-2" />;
			if (content) {
				const lower = content.toLowerCase();
				if (lower.includes('[warn]') || lower.includes('npm warn') || lower.includes('warning:')) return <span class="w-1.5 h-1.5 rounded-full bg-amber-400 flex-shrink-0 mt-2" />;
			}
			return <span class="w-1.5 h-1.5 rounded-full bg-gray-400 flex-shrink-0 mt-2" />;
		}
		case 'system': {
			if (content) {
				const lower = content.toLowerCase();
				if (lower.includes('— success') || lower.includes('- success') || lower.includes('— passed') || lower.includes('checked out')) return <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 flex-shrink-0 mt-2" />;
				if (lower.includes('— fail') || lower.includes('- fail') || lower.includes('— error')) return <span class="w-1.5 h-1.5 rounded-full bg-red-400 flex-shrink-0 mt-2" />;
			}
			return <span class="w-1.5 h-1.5 rounded-full bg-blue-400 flex-shrink-0 mt-2" />;
		}
		default: return <span class="w-1.5 h-1.5 rounded-full bg-emerald-400/50 flex-shrink-0 mt-2" />;
	}
}

// Internal log line with stream info for color coding
interface LogEntry {
	content: string;
	stream: string;
	step_run_id: string;
}

function toLogEntry(line: { content: string; stream?: string; step_run_id?: string | null }): LogEntry {
	return {
		content: line.content,
		stream: line.stream || 'stdout',
		step_run_id: line.step_run_id ?? '',
	};
}

function logEntryKey(entry: LogEntry): string {
	return `${entry.step_run_id}\u0000${entry.stream}\u0000${entry.content}`;
}

function mergeHistoricalAndLiveLogs(historical: LogEntry[], live: LogEntry[]): LogEntry[] {
	if (historical.length === 0) return live;
	if (live.length === 0) return historical;

	const historicalCounts = new Map<string, number>();
	for (const entry of historical) {
		const key = logEntryKey(entry);
		historicalCounts.set(key, (historicalCounts.get(key) ?? 0) + 1);
	}

	const merged = [...historical];
	for (const entry of live) {
		const key = logEntryKey(entry);
		const remaining = historicalCounts.get(key) ?? 0;
		if (remaining > 0) {
			historicalCounts.set(key, remaining - 1);
			continue;
		}
		merged.push(entry);
	}

	return merged;
}

function isActiveRunStatus(status?: RunStatus): boolean {
	return status === 'running' || status === 'queued' || status === 'pending';
}

// ---------------------------------------------------------------------------
// Fetcher
// ---------------------------------------------------------------------------
interface RunPageData {
	run: RunDetail;
	artifacts: Artifact[];
	logs: LogLine[];
}

async function fetchRunData(ids: { projectId: string; pipelineId: string; runId: string }): Promise<RunPageData> {
	const [run, artifacts, logs] = await Promise.all([
		api.runs.get(ids.projectId, ids.pipelineId, ids.runId),
		api.runs.getArtifacts(ids.projectId, ids.pipelineId, ids.runId).catch(() => [] as Artifact[]),
		api.runs.getLogs(ids.projectId, ids.pipelineId, ids.runId).catch(() => [] as LogLine[]),
	]);
	return { run, artifacts, logs };
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const RunDetailPage: Component = () => {
	const params = useParams<{ id: string; pid: string; rid: string }>();
	const navigate = useNavigate();
	const [selectedStep, setSelectedStep] = createSignal<string>('__all__');
	const [logsCopied, setLogsCopied] = createSignal(false);
	const [expandedStages, setExpandedStages] = createSignal<Set<string>>(new Set());
	const [expandedJobs, setExpandedJobs] = createSignal<Set<string>>(new Set());
	const [showArtifacts, setShowArtifacts] = createSignal(false);
	const [cancelling, setCancelling] = createSignal(false);
	const [rerunning, setRerunning] = createSignal(false);
	const [approving, setApproving] = createSignal(false);
	const [liveLogLines, setLiveLogLines] = createSignal<LogEntry[]>([]);
	const [runViewTab, setRunViewTab] = createSignal<'logs' | 'tests' | 'coverage' | 'resources'>('logs');
	let logContainerRef: HTMLDivElement | undefined;

	// Project + pipeline names for breadcrumb
	const [, setProjectName] = createSignal('Project');
	const [pipelineName, setPipelineName] = createSignal('Pipeline');
	const [pipelineActive, setPipelineActive] = createSignal(true);

	onMount(async () => {
		try {
			const [project, pipeline] = await Promise.all([
				api.projects.get(params.id),
				api.pipelines.get(params.id, params.pid),
			]);
			setProjectName(project.name);
			setPipelineName(pipeline.name);
			setPipelineActive(Boolean(pipeline.is_active));
		} catch { /* fallback */ }
	});

	// Fetch run data
	const [data, { refetch }] = createResource(
		() => ({ projectId: params.id, pipelineId: params.pid, runId: params.rid }),
		fetchRunData
	);

	const run = () => data()?.run;
	const artifacts = () => data()?.artifacts ?? [];
	const allLogs = () => data()?.logs ?? [];
	const stages = () => run()?.stages ?? [];
	const jobs = () => run()?.jobs ?? [];
	const steps = () => run()?.steps ?? [];

	// Fetch test results, coverage, and resource metrics lazily when tabs are activated
	const [testResultXml, setTestResultXml] = createSignal<string | undefined>(undefined);
	const [coverageData, setCoverageData] = createSignal<any>(undefined);
	const [resourceData, setResourceData] = createSignal<{ points: any[]; steps: any[] }>({ points: [], steps: [] });
	const [tabDataLoading, setTabDataLoading] = createSignal(false);

	const loadTabData = async (tab: 'tests' | 'coverage' | 'resources') => {
		setTabDataLoading(true);
		try {
			if (tab === 'tests' && testResultXml() === undefined) {
				const result = await api.runMetrics.testResults(params.id, params.pid, params.rid).catch(() => ({ xml: '' }));
				setTestResultXml(result.xml || '');
			} else if (tab === 'coverage' && coverageData() === undefined) {
				const result = await api.runMetrics.coverage(params.id, params.pid, params.rid).catch(() => null);
				setCoverageData(result);
			} else if (tab === 'resources' && resourceData().points.length === 0 && resourceData().steps.length === 0) {
				const result = await api.runMetrics.resources(params.id, params.pid, params.rid).catch(() => ({ points: [], steps: [] }));
				setResourceData(result);
			}
		} catch { /* handled by .catch above */ }
		finally { setTabDataLoading(false); }
	};

	const handleRunViewTab = (tab: 'logs' | 'tests' | 'coverage' | 'resources') => {
		setRunViewTab(tab);
		if (tab !== 'logs') loadTabData(tab);
	};

	// Fetch sibling runs for prev/next navigation
	const [siblingRuns] = createResource(
		() => ({ projectId: params.id, pipelineId: params.pid }),
		(ids) => api.runs.list(ids.projectId, ids.pipelineId, { per_page: '50' }).then(res => res.data)
	);

	const prevRun = () => {
		const siblings = siblingRuns() ?? [];
		const currentIdx = siblings.findIndex(r => r.id === params.rid);
		if (currentIdx < 0 || currentIdx >= siblings.length - 1) return null;
		return siblings[currentIdx + 1]; // sorted desc by number, so next index = prev run
	};

	const nextRun = () => {
		const siblings = siblingRuns() ?? [];
		const currentIdx = siblings.findIndex(r => r.id === params.rid);
		if (currentIdx <= 0) return null;
		return siblings[currentIdx - 1]; // sorted desc by number, so prev index = next run
	};

	// Stages and jobs start collapsed by default (initial state is empty Set).
	// User clicks to expand what they need.

	let statusChangeTimer: ReturnType<typeof setTimeout> | null = null;

	// Debounced refetch to avoid excessive API calls during rapid status changes
	const debouncedRefetch = () => {
		if (statusChangeTimer) clearTimeout(statusChangeTimer);
		statusChangeTimer = setTimeout(() => {
			refetch();
			statusChangeTimer = null;
		}, 300);
	};

	createEffect(() => {
		params.rid;
		setLiveLogLines([]);
	});

	createEffect(() => {
		const r = run();
		if (!r || !isActiveRunStatus(r.status)) return;

		const socket = createRunLogSocket(r.id);
		const offLog = socket.on('log', (payload: unknown) => {
			const line = payload as { step_run_id?: string; content: string; stream?: string };
			setLiveLogLines(prev => [...prev, toLogEntry(line)]);
		});
		const offStatus = socket.on('status', (payload: unknown) => {
			const status = payload as { status?: string };
			if (status.status === 'success' || status.status === 'failure' || status.status === 'cancelled') {
				setLiveLogLines([]);
				refetch();
				socket.disconnect();
				return;
			}
			debouncedRefetch();
		});
		const offStatusChange = socket.on('status_change', () => {
			debouncedRefetch();
		});

		socket.connect();

		onCleanup(() => {
			offLog();
			offStatus();
			offStatusChange();
			socket.disconnect();
		});
	});

	createEffect(() => {
		const r = run();
		if (!r || !isActiveRunStatus(r.status)) return;

		const interval = window.setInterval(() => {
			refetch();
		}, 2500);

		onCleanup(() => {
			window.clearInterval(interval);
		});
	});

	onCleanup(() => {
		if (statusChangeTimer) {
			clearTimeout(statusChangeTimer);
			statusChangeTimer = null;
		}
	});

	// Auto-scroll logs when step changes or live logs arrive
	createEffect(() => {
		selectedStep();
		liveLogLines();
		if (logContainerRef) {
			logContainerRef.scrollTop = logContainerRef.scrollHeight;
		}
	});

	const toggleStage = (id: string) => {
		setExpandedStages(prev => {
			const next = new Set(prev);
			if (next.has(id)) next.delete(id); else next.add(id);
			return next;
		});
	};

	const toggleJob = (id: string) => {
		setExpandedJobs(prev => {
			const next = new Set(prev);
			if (next.has(id)) next.delete(id); else next.add(id);
			return next;
		});
	};

	const selectedStepData = () => {
		const id = selectedStep();
		if (!id || id === '__all__') return undefined;
		return steps().find(s => s.id === id);
	};

	const isAllLogsView = () => selectedStep() === '__all__';

	// Get logs for selected step or all logs — returns LogEntry[] with stream info.
	// Two sources: historical (REST API / DB) and live (WebSocket).
	// On every refetch, liveLogLines is cleared so we never merge both.
	// During streaming, live lines accumulate; on refetch, DB takes over.
	const selectedStepLogEntries = (): LogEntry[] => {
		const stepId = selectedStep();
		if (!stepId) return [];

		const live = liveLogLines();
		const historical = allLogs().map(toLogEntry);
		const source = mergeHistoricalAndLiveLogs(historical, live);

		if (stepId === '__all__') {
			return source;
		}
		return source.filter(l => l.step_run_id === stepId);
	};

	const handleCancel = async () => {
		setCancelling(true);
		try {
			await api.runs.cancel(params.id, params.pid, params.rid);
			toast.success('Run cancelled');
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to cancel run');
		} finally {
			setCancelling(false);
		}
	};

	const handleRerun = async () => {
		setRerunning(true);
		try {
			const newRun = await api.runs.rerun(params.id, params.pid, params.rid);
			toast.success(`Re-run started as #${newRun.number}`);
			navigate(`/projects/${params.id}/pipelines/${params.pid}/runs/${newRun.id}`);
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to rerun');
		} finally {
			setRerunning(false);
		}
	};

	const handleApprove = async () => {
		setApproving(true);
		try {
			await api.runs.approve(params.id, params.pid, params.rid);
			toast.success('Run approved');
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to approve run');
		} finally {
			setApproving(false);
		}
	};

	return (
		<div class="animate-fade-in">
			{/* Compact header */}
			<div class="flex items-center gap-2 mb-3 text-sm flex-wrap">
				{/* Breadcrumb-style nav */}
				<A href={`/projects/${params.id}/pipelines/${params.pid}`} class="text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] transition-colors truncate max-w-[200px]">
					{pipelineName()}
				</A>
				<svg class="w-3.5 h-3.5 text-[var(--color-text-tertiary)] flex-shrink-0" viewBox="0 0 20 20" fill="currentColor">
					<path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
				</svg>
				<Show when={run()}>
					<span class="font-semibold text-[var(--color-text-primary)]">#{run()!.number}</span>
					<Badge variant={getStatusBadgeVariant(run()!.status)} dot>
						{statusLabel[run()!.status]}
					</Badge>
				</Show>

				{/* Metadata pills */}
				<Show when={run()}>
					<div class="flex items-center gap-1.5 text-xs text-[var(--color-text-tertiary)]">
						<Show when={run()!.branch}>
							<span class="font-mono bg-[var(--color-bg-tertiary)] px-1.5 py-0.5 rounded border border-[var(--color-border-primary)]">{run()!.branch}</span>
						</Show>
						<span class="capitalize">{run()!.trigger_type.replace('_', ' ')}</span>
						<span>·</span>
						<span>{formatDuration(run()!.duration_ms)}</span>
						<span>·</span>
						<span>{formatRelativeTime(run()!.created_at)}</span>
					</div>
				</Show>

				{/* Spacer */}
				<div class="flex-1" />

				{/* Actions */}
				<div class="flex items-center gap-1.5 flex-shrink-0">
					<Show when={prevRun()}>
						<A
							href={`/projects/${params.id}/pipelines/${params.pid}/runs/${prevRun()!.id}`}
							class="px-1.5 py-0.5 text-xs rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] border border-[var(--color-border-primary)] transition-colors"
						>
							&larr; #{prevRun()!.number}
						</A>
					</Show>
					<Show when={nextRun()}>
						<A
							href={`/projects/${params.id}/pipelines/${params.pid}/runs/${nextRun()!.id}`}
							class="px-1.5 py-0.5 text-xs rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] border border-[var(--color-border-primary)] transition-colors"
						>
							#{nextRun()!.number} &rarr;
						</A>
					</Show>
					<Show when={run()}>
						<Show when={run()!.status === 'running' || run()!.status === 'queued'}>
							<Button size="sm" variant="danger" onClick={handleCancel} loading={cancelling()}>Cancel</Button>
						</Show>
						<Show when={run()!.status === 'waiting_approval'}>
							<Button size="sm" onClick={handleApprove} loading={approving()}>Approve</Button>
						</Show>
						<Show when={run()!.status === 'failure' || run()!.status === 'cancelled'}>
							<Button size="sm" variant="outline" onClick={handleRerun} loading={rerunning()} disabled={!pipelineActive()} title={!pipelineActive() ? 'Pipeline is disabled' : 'Rerun this pipeline'}>Rerun</Button>
						</Show>
					</Show>
					<Button size="sm" variant="ghost" onClick={() => setShowArtifacts(!showArtifacts())}>
						<svg class="w-3.5 h-3.5 mr-1" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 2.75a.75.75 0 00-1.5 0v8.614L6.295 8.235a.75.75 0 10-1.09 1.03l4.25 4.5a.75.75 0 001.09 0l4.25-4.5a.75.75 0 00-1.09-1.03l-2.955 3.129V2.75z" /><path d="M3.5 12.75a.75.75 0 00-1.5 0v2.5A2.75 2.75 0 004.75 18h10.5A2.75 2.75 0 0018 15.25v-2.5a.75.75 0 00-1.5 0v2.5c0 .69-.56 1.25-1.25 1.25H4.75c-.69 0-1.25-.56-1.25-1.25v-2.5z" /></svg>
						{artifacts().length}
					</Button>
				</div>
			</div>

			{/* Inline status + deploy URL row */}
			<Show when={run()}>
				<div class="flex items-center gap-3 mb-3 flex-wrap">
					{/* Compact status indicator */}
					<Show when={run()!.status === 'success'}>
						{(() => {
							const completedSteps = steps().filter(s => s.status === 'success');
							const totalDuration = steps().reduce((sum, s) => sum + (s.duration_ms ?? 0), 0);
							return (
								<div class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-xs">
									<svg class="w-3.5 h-3.5 text-emerald-400 flex-shrink-0" viewBox="0 0 20 20" fill="currentColor">
										<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" />
									</svg>
									<span class="text-emerald-400 font-medium">{completedSteps.length} steps passed</span>
									<span class="text-emerald-400/50">{formatDuration(totalDuration)}</span>
								</div>
							);
						})()}
					</Show>
					<Show when={run()!.status === 'running' || run()!.status === 'queued' || run()!.status === 'pending'}>
						{(() => {
							const activeSteps = steps().filter(s => s.status === 'running');
							const completedSteps = steps().filter(s => s.status === 'success');
							const totalSteps = steps().length;
							return (
								<div class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-violet-500/10 border border-violet-500/20 text-xs">
									<svg class="w-3.5 h-3.5 text-violet-400 flex-shrink-0 animate-spin" viewBox="0 0 24 24" fill="none">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
									</svg>
									<Show when={activeSteps.length > 0} fallback={
										<span class="text-violet-400">{run()!.status === 'queued' ? 'Queued' : 'Preparing...'}</span>
									}>
										<span class="text-violet-400 font-medium">{activeSteps.map(s => s.name).join(', ')}</span>
									</Show>
									<Show when={totalSteps > 0}>
										<span class="text-violet-400/50">{completedSteps.length}/{totalSteps}</span>
									</Show>
								</div>
							);
						})()}
					</Show>
					<Show when={run()!.status === 'failure'}>
						{(() => {
							const failedSteps = steps().filter(s => s.status === 'failure');
							return (
								<Show when={failedSteps.length > 0}>
									<div class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-red-500/10 border border-red-500/20 text-xs">
										<svg class="w-3.5 h-3.5 text-red-400 flex-shrink-0" viewBox="0 0 20 20" fill="currentColor">
											<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clip-rule="evenodd" />
										</svg>
										<For each={failedSteps}>
											{(step) => (
												<span
													class="text-red-400 font-mono cursor-pointer hover:text-red-300 transition-colors"
													onClick={() => setSelectedStep(step.id)}
												>
													{step.name}
												</span>
											)}
										</For>
									</div>
								</Show>
							);
						})()}
					</Show>
					<Show when={run()!.status === 'cancelled'}>
						<div class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-gray-500/10 border border-gray-500/20 text-xs">
							<svg class="w-3.5 h-3.5 text-gray-400 flex-shrink-0" viewBox="0 0 20 20" fill="currentColor">
								<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM6.75 9.25a.75.75 0 000 1.5h6.5a.75.75 0 000-1.5h-6.5z" clip-rule="evenodd" />
							</svg>
							<span class="text-gray-400">Cancelled</span>
						</div>
					</Show>

					{/* Deploy URL inline */}
					<Show when={run()?.deploy_url}>
						<a
							href={run()!.deploy_url!}
							target="_blank"
							rel="noopener noreferrer"
							class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-xs text-emerald-400 hover:bg-emerald-500/15 transition-colors"
						>
							<svg class="w-3.5 h-3.5 flex-shrink-0" viewBox="0 0 20 20" fill="currentColor">
								<path fill-rule="evenodd" d="M4.25 5.5a.75.75 0 00-.75.75v8.5c0 .414.336.75.75.75h8.5a.75.75 0 00.75-.75v-4a.75.75 0 011.5 0v4A2.25 2.25 0 0112.75 17h-8.5A2.25 2.25 0 012 14.75v-8.5A2.25 2.25 0 014.25 4h5a.75.75 0 010 1.5h-5z" clip-rule="evenodd" />
								<path fill-rule="evenodd" d="M6.194 12.753a.75.75 0 001.06.053L16.5 4.44v2.81a.75.75 0 001.5 0v-4.5a.75.75 0 00-.75-.75h-4.5a.75.75 0 000 1.5h2.553l-9.056 8.194a.75.75 0 00-.053 1.06z" clip-rule="evenodd" />
							</svg>
							{run()!.deploy_url}
						</a>
					</Show>

					{/* Commit SHA if present */}
					<Show when={run()!.commit_sha}>
						<span class="font-mono text-xs bg-[var(--color-bg-tertiary)] px-1.5 py-0.5 rounded border border-[var(--color-border-primary)] text-[var(--color-text-tertiary)]">{truncateCommitSha(run()!.commit_sha)}</span>
					</Show>
				</div>
			</Show>

			{/* Error state */}
			<Show when={data.error}>
				<div class="mb-3 p-3 rounded-lg bg-red-500/10 border border-red-500/30 flex items-center justify-between">
					<p class="text-sm text-red-400">Failed to load run: {(data.error as Error)?.message}</p>
					<Button size="sm" variant="outline" onClick={refetch}>Retry</Button>
				</div>
			</Show>

			{/* Artifacts panel */}
			<Show when={showArtifacts() && artifacts().length > 0}>
				<div class="mb-3 p-3 bg-[var(--color-bg-secondary)] rounded-lg border border-[var(--color-border-primary)]">
					<div class="space-y-1.5">
						<For each={artifacts()}>
							{(artifact) => (
								<div class="flex items-center justify-between p-2 rounded bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)]">
									<div class="flex items-center gap-2">
										<svg class="w-4 h-4 text-[var(--color-text-tertiary)]" viewBox="0 0 20 20" fill="currentColor">
											<path d="M3 3.5A1.5 1.5 0 014.5 2h6.879a1.5 1.5 0 011.06.44l4.122 4.12A1.5 1.5 0 0117 7.622V16.5a1.5 1.5 0 01-1.5 1.5h-11A1.5 1.5 0 013 16.5v-13z" />
										</svg>
										<span class="text-sm text-[var(--color-text-primary)]">{artifact.name}</span>
										<span class="text-xs text-[var(--color-text-tertiary)]">{formatBytes(artifact.size_bytes)}</span>
									</div>
									<a href={api.artifacts.downloadUrl(artifact.id)} target="_blank" rel="noopener noreferrer">
										<Button size="sm" variant="outline">Download</Button>
									</a>
								</div>
							)}
						</For>
					</div>
				</div>
			</Show>

			{/* Run view tabs */}
			<div class="flex items-center gap-1 mb-3 bg-[var(--color-bg-tertiary)] p-1 rounded-lg w-fit">
				<For each={[
					{ id: 'logs' as const, label: 'Logs', icon: 'M4.5 2A1.5 1.5 0 003 3.5v13A1.5 1.5 0 004.5 18h11a1.5 1.5 0 001.5-1.5V7.621a1.5 1.5 0 00-.44-1.06l-4.12-4.122A1.5 1.5 0 0011.378 2H4.5z' },
					{ id: 'tests' as const, label: 'Tests', icon: 'M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z' },
					{ id: 'coverage' as const, label: 'Coverage', icon: 'M10 1a6 6 0 00-3.815 10.631C7.237 12.5 8 13.443 8 14.5v.5h4v-.5c0-1.057.763-2 1.815-2.869A6 6 0 0010 1zM8 17.5a.75.75 0 01.75-.75h2.5a.75.75 0 010 1.5h-2.5a.75.75 0 01-.75-.75z' },
					{ id: 'resources' as const, label: 'Resources', icon: 'M15.312 11.424a5.5 5.5 0 01-9.201 2.466l-.312-.311h2.433a.75.75 0 000-1.5H4.233a.75.75 0 00-.75.75v4a.75.75 0 001.5 0v-2.146a7 7 0 0011.712-3.138.75.75 0 00-1.383-.575zm-9.624-3.848a5.5 5.5 0 019.201-2.466l.312.31h-2.433a.75.75 0 000 1.5H16.767a.75.75 0 00.75-.75v-4a.75.75 0 00-1.5 0v2.146A7 7 0 004.305 7.454a.75.75 0 001.383.575z' },
				]}>
					{(tab) => (
						<button
							class={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md transition-colors cursor-pointer ${
								runViewTab() === tab.id
									? 'bg-[var(--color-bg-secondary)] text-[var(--color-text-primary)] shadow-sm'
									: 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'
							}`}
							onClick={() => handleRunViewTab(tab.id)}
						>
							<svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
								<path fill-rule="evenodd" d={tab.icon} clip-rule="evenodd" />
							</svg>
							{tab.label}
						</button>
					)}
				</For>
			</div>

			<Show when={!data.loading} fallback={
				<div class="flex gap-4">
					<div class="w-72 space-y-2">
						<For each={[1, 2, 3]}>{() => <div class="h-16 bg-[var(--color-bg-secondary)] rounded-lg animate-pulse" />}</For>
					</div>
					<div class="flex-1 h-[500px] bg-[var(--color-bg-secondary)] rounded-lg animate-pulse" />
				</div>
			}>
				{/* Logs tab: stage tree + log viewer */}
				<Show when={runViewTab() === 'logs'}>
				<div class="flex gap-4 h-[calc(100vh-240px)] min-h-[500px]">
					{/* Stage/Job/Step tree sidebar */}
					<div class="w-72 flex-shrink-0 overflow-y-auto bg-[var(--color-bg-secondary)] rounded-xl border border-[var(--color-border-primary)]">
						<div class="p-2">
							{/* "All Logs" button */}
							<button
								class={`w-full flex items-center gap-2 px-2 py-1.5 rounded-lg transition-colors text-left mb-1 ${isAllLogsView()
										? 'bg-indigo-500/10 border border-indigo-500/30'
										: 'hover:bg-[var(--color-bg-hover)]'
									}`}
								onClick={() => setSelectedStep('__all__')}
							>
								<svg class="w-3.5 h-3.5 text-[var(--color-text-tertiary)]" viewBox="0 0 20 20" fill="currentColor">
									<path fill-rule="evenodd" d="M4.5 2A1.5 1.5 0 003 3.5v13A1.5 1.5 0 004.5 18h11a1.5 1.5 0 001.5-1.5V7.621a1.5 1.5 0 00-.44-1.06l-4.12-4.122A1.5 1.5 0 0011.378 2H4.5zm2.25 8.5a.75.75 0 000 1.5h6.5a.75.75 0 000-1.5h-6.5zm0 3a.75.75 0 000 1.5h6.5a.75.75 0 000-1.5h-6.5z" clip-rule="evenodd" />
								</svg>
								<span class={`text-xs font-medium flex-1 ${isAllLogsView() ? 'text-indigo-400' : 'text-[var(--color-text-primary)]'}`}>All Logs</span>
								<span class="text-[10px] text-[var(--color-text-tertiary)]">{allLogs().length || liveLogLines().length}</span>
							</button>

							{/* Stream legend */}
							<div class="flex items-center gap-2 px-2 mb-2 text-[10px] text-[var(--color-text-tertiary)]">
								<span class="flex items-center gap-1"><span class="w-1 h-1 rounded-full bg-emerald-400/50" /> out</span>
								<span class="flex items-center gap-1"><span class="w-1 h-1 rounded-full bg-gray-400" /> err</span>
								<span class="flex items-center gap-1"><span class="w-1 h-1 rounded-full bg-red-400" /> error</span>
								<span class="flex items-center gap-1"><span class="w-1 h-1 rounded-full bg-blue-400" /> sys</span>
							</div>

							<Show when={stages().length > 0} fallback={
								<p class="text-xs text-[var(--color-text-tertiary)] px-2">No stage data available.</p>
							}>
								<For each={stages()}>
									{(stage) => {
										const stageJobs = () => jobs().filter(j => j.stage_run_id === stage.id);
										const isExpanded = () => expandedStages().has(stage.id);

										return (
											<div class="mb-0.5">
												<button
													class="w-full flex items-center gap-1.5 px-2 py-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors text-left"
													onClick={() => toggleStage(stage.id)}
												>
													<svg class={`w-2.5 h-2.5 text-[var(--color-text-tertiary)] transition-transform ${isExpanded() ? 'rotate-90' : ''}`} viewBox="0 0 20 20" fill="currentColor">
														<path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
													</svg>
													{statusIcon(stage.status)}
													<span class="text-xs font-medium text-[var(--color-text-primary)] capitalize flex-1 truncate">{stage.name}</span>
													<span class="text-[10px] text-[var(--color-text-tertiary)]">
														{stage.started_at && stage.finished_at ? formatDuration(new Date(stage.finished_at).getTime() - new Date(stage.started_at).getTime()) : ''}
													</span>
												</button>

												<Show when={isExpanded()}>
													<div class="ml-3 pl-2.5 border-l border-[var(--color-border-primary)]">
														<For each={stageJobs()}>
															{(job) => {
																const jobSteps = () => steps().filter(s => s.job_run_id === job.id);
																const isJobExpanded = () => expandedJobs().has(job.id);

																return (
																	<div class="mb-0.5">
																		<button
																			class="w-full flex items-center gap-1.5 px-2 py-1 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors text-left"
																			onClick={() => toggleJob(job.id)}
																		>
																			<svg class={`w-2.5 h-2.5 text-[var(--color-text-tertiary)] transition-transform ${isJobExpanded() ? 'rotate-90' : ''}`} viewBox="0 0 20 20" fill="currentColor">
																				<path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
																			</svg>
																			{statusIcon(job.status)}
																			<span class="text-xs text-[var(--color-text-secondary)] flex-1 truncate">{job.name}</span>
																		</button>

																		<Show when={isJobExpanded()}>
																			<div class="ml-3 pl-2.5 border-l border-[var(--color-border-primary)]">
																				<For each={jobSteps()}>
																					{(step) => (
																						<button
																							class={`w-full flex items-center gap-1.5 px-2 py-1 rounded-lg transition-colors text-left ${selectedStep() === step.id
																								? 'bg-indigo-500/10 border border-indigo-500/30'
																								: 'hover:bg-[var(--color-bg-hover)]'
																								}`}
																							onClick={() => setSelectedStep(step.id)}
																						>
																							{statusIcon(step.status)}
																							<span class={`text-xs flex-1 truncate ${selectedStep() === step.id ? 'text-indigo-400 font-medium' : 'text-[var(--color-text-tertiary)]'}`}>
																								{step.name}
																							</span>
																							<span class="text-[10px] text-[var(--color-text-tertiary)]">{formatDuration(step.duration_ms)}</span>
																						</button>
																					)}
																				</For>
																			</div>
																		</Show>
																	</div>
																);
															}}
														</For>
													</div>
												</Show>
											</div>
										);
									}}
								</For>
							</Show>
						</div>
					</div>

					{/* Log viewer */}
					<div class="flex-1 flex flex-col bg-[#0d1117] rounded-xl border border-[var(--color-border-primary)] overflow-hidden">
						{/* Log header */}
						<div class="flex items-center justify-between px-3 py-1.5 bg-[#161b22] border-b border-[var(--color-border-primary)]">
							<div class="flex items-center gap-2">
								<Show when={isAllLogsView()}>
									<span class="text-xs font-medium text-[var(--color-text-primary)]">All Logs</span>
									<span class="text-[10px] text-[var(--color-text-tertiary)]">{selectedStepLogEntries().length}</span>
								</Show>
								<Show when={!isAllLogsView() && selectedStepData()}>
									{statusIcon(selectedStepData()!.status)}
									<span class="text-xs font-medium text-[var(--color-text-primary)]">{selectedStepData()!.name}</span>
									<Show when={selectedStepData()!.exit_code !== undefined && selectedStepData()!.exit_code !== null}>
										<span class={`text-[10px] px-1 py-0.5 rounded ${selectedStepData()!.exit_code === 0 ? 'bg-emerald-500/10 text-emerald-400' : 'bg-red-500/10 text-red-400'}`}>
											exit {selectedStepData()!.exit_code}
										</span>
									</Show>
								</Show>
							</div>
							<div class="flex items-center gap-2">
								<Show when={isAllLogsView() && run()}>
									<span class="text-[10px] text-[var(--color-text-tertiary)]">{formatDuration(run()!.duration_ms)}</span>
								</Show>
								<Show when={!isAllLogsView() && selectedStepData()?.duration_ms}>
									<span class="text-[10px] text-[var(--color-text-tertiary)]">{formatDuration(selectedStepData()?.duration_ms)}</span>
								</Show>
								<Show when={selectedStepLogEntries().length > 0}>
									<button
										class="flex items-center gap-1 px-1.5 py-0.5 text-[10px] rounded border border-[var(--color-border-primary)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-secondary)] transition-colors"
										onClick={() => {
											const text = selectedStepLogEntries()
												.map((l) => l.content)
												.join('');
											navigator.clipboard.writeText(text).then(() => {
												setLogsCopied(true);
												setTimeout(() => setLogsCopied(false), 2000);
											});
										}}
									>
										<Show when={logsCopied()} fallback={
											<>
												<svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor">
													<path d="M7 3.5A1.5 1.5 0 018.5 2h3.879a1.5 1.5 0 011.06.44l3.122 3.12A1.5 1.5 0 0117 6.622V12.5a1.5 1.5 0 01-1.5 1.5h-1v-3.379a3 3 0 00-.879-2.121L10.5 5.379A3 3 0 008.379 4.5H7v-1z" />
													<path d="M4.5 6A1.5 1.5 0 003 7.5v9A1.5 1.5 0 004.5 18h7a1.5 1.5 0 001.5-1.5v-5.879a1.5 1.5 0 00-.44-1.06L9.44 6.439A1.5 1.5 0 008.378 6H4.5z" />
												</svg>
												Copy
											</>
										}>
											<>
												<svg class="w-3 h-3 text-emerald-400" viewBox="0 0 20 20" fill="currentColor">
													<path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
												</svg>
												Copied
											</>
										</Show>
									</button>
								</Show>
							</div>
						</div>

						{/* Log content */}
						<div
							ref={logContainerRef}
							class="flex-1 overflow-y-auto p-3 font-mono text-xs leading-5"
						>
							<Show when={selectedStepLogEntries().length > 0} fallback={
								<div class="flex items-center justify-center h-full text-[var(--color-text-tertiary)]">
									<Show when={run()?.status === 'running' || run()?.status === 'queued' || run()?.status === 'pending'}
										fallback={isAllLogsView() ? "No logs available." : "No logs for this step."}>
										<div class="flex items-center gap-2">
											<svg class="w-4 h-4 animate-spin text-violet-400" viewBox="0 0 24 24" fill="none">
												<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
												<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
											</svg>
											<span class="text-xs">Waiting for logs...</span>
										</div>
									</Show>
								</div>
							}>
								<For each={selectedStepLogEntries()}>
									{(entry, i) => (
										<div class={`flex hover:bg-white/[0.03] group ${entry.stream === 'stderr' && isStderrError(entry.content) ? 'bg-red-500/[0.03]' : ''}`}>
											<span class="w-8 flex-shrink-0 text-right pr-2 text-[var(--color-text-tertiary)] select-none opacity-30 group-hover:opacity-100 text-[10px]">
												{i() + 1}
											</span>
											{streamIndicator(entry.stream, entry.content)}
											<span class={`flex-1 whitespace-pre-wrap break-all ml-1.5 ${streamClass(entry.stream, entry.content)}`} innerHTML={ansiToHtml(entry.content)} />
										</div>
									)}
								</For>
							</Show>
						</div>
					</div>
				</div>
				</Show>

				{/* Tests tab */}
				<Show when={runViewTab() === 'tests'}>
					<div class="bg-[var(--color-bg-secondary)] rounded-xl border border-[var(--color-border-primary)] p-4">
						<Show when={!tabDataLoading()} fallback={
							<div class="flex items-center justify-center py-16 text-[var(--color-text-tertiary)]">
								<div class="flex items-center gap-2">
									<svg class="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
									</svg>
									<span class="text-sm">Loading test results...</span>
								</div>
							</div>
						}>
							<Suspense fallback={<div class="text-center py-8 text-sm text-[var(--color-text-tertiary)]">Loading component...</div>}>
								<Show when={testResultXml() && testResultXml()!.length > 0} fallback={
									<div class="flex flex-col items-center justify-center py-16 text-[var(--color-text-tertiary)]">
										<svg class="w-10 h-10 mb-3 opacity-40" viewBox="0 0 20 20" fill="currentColor">
											<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" />
										</svg>
										<p class="text-sm">No test results available for this run.</p>
										<p class="text-xs mt-1 text-[var(--color-text-tertiary)]">Test results are generated from JUnit XML reports.</p>
									</div>
								}>
									<TestResults xmlContent={testResultXml()} />
								</Show>
							</Suspense>
						</Show>
					</div>
				</Show>

				{/* Coverage tab */}
				<Show when={runViewTab() === 'coverage'}>
					<div class="bg-[var(--color-bg-secondary)] rounded-xl border border-[var(--color-border-primary)] p-4">
						<Show when={!tabDataLoading()} fallback={
							<div class="flex items-center justify-center py-16 text-[var(--color-text-tertiary)]">
								<div class="flex items-center gap-2">
									<svg class="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
									</svg>
									<span class="text-sm">Loading coverage data...</span>
								</div>
							</div>
						}>
							<Suspense fallback={<div class="text-center py-8 text-sm text-[var(--color-text-tertiary)]">Loading component...</div>}>
								<Show when={coverageData()} fallback={
									<div class="flex flex-col items-center justify-center py-16 text-[var(--color-text-tertiary)]">
										<svg class="w-10 h-10 mb-3 opacity-40" viewBox="0 0 20 20" fill="currentColor">
											<path d="M10 1a6 6 0 00-3.815 10.631C7.237 12.5 8 13.443 8 14.5v.5h4v-.5c0-1.057.763-2 1.815-2.869A6 6 0 0010 1zM8 17.5a.75.75 0 01.75-.75h2.5a.75.75 0 010 1.5h-2.5a.75.75 0 01-.75-.75z" />
										</svg>
										<p class="text-sm">No coverage data available for this run.</p>
										<p class="text-xs mt-1 text-[var(--color-text-tertiary)]">Coverage data is generated from test coverage reports.</p>
									</div>
								}>
									<CoverageReport data={coverageData()} />
								</Show>
							</Suspense>
						</Show>
					</div>
				</Show>

				{/* Resources tab */}
				<Show when={runViewTab() === 'resources'}>
					<div class="bg-[var(--color-bg-secondary)] rounded-xl border border-[var(--color-border-primary)] p-4">
						<Show when={!tabDataLoading()} fallback={
							<div class="flex items-center justify-center py-16 text-[var(--color-text-tertiary)]">
								<div class="flex items-center gap-2">
									<svg class="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
									</svg>
									<span class="text-sm">Loading resource data...</span>
								</div>
							</div>
						}>
							<Suspense fallback={<div class="text-center py-8 text-sm text-[var(--color-text-tertiary)]">Loading component...</div>}>
								<ResourceGraph
									data={resourceData().points}
									steps={resourceData().steps}
									totalDurationMs={run()?.duration_ms}
								/>
							</Suspense>
						</Show>
					</div>
				</Show>
			</Show>
		</div>
	);
};

export default RunDetailPage;
