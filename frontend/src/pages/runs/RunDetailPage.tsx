import type { Component } from 'solid-js';
import { createSignal, createResource, For, Show, onMount, onCleanup, createEffect } from 'solid-js';
import { useParams, A } from '@solidjs/router';
import PageContainer from '../../components/layout/PageContainer';
import Badge from '../../components/ui/Badge';
import Button from '../../components/ui/Button';
import { toast } from '../../components/ui/Toast';
import { api, ApiRequestError, type RunDetail } from '../../api/client';
import { createRunLogSocket } from '../../api/websocket';
import type { StageRun, JobRun, StepRun, RunStatus, Artifact, LogLine } from '../../types';
import { formatDuration, formatRelativeTime, getStatusBadgeVariant, truncateCommitSha, formatBytes } from '../../utils/helpers';

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

// Get CSS class for a log line based on its stream type and content
function streamClass(stream: string, content?: string): string {
	switch (stream) {
		case 'stderr': return 'text-red-400';
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
		case 'stderr': return <span class="w-1.5 h-1.5 rounded-full bg-red-400 flex-shrink-0 mt-2" />;
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
	const [selectedStep, setSelectedStep] = createSignal<string>('__all__');
	const [expandedStages, setExpandedStages] = createSignal<Set<string>>(new Set());
	const [expandedJobs, setExpandedJobs] = createSignal<Set<string>>(new Set());
	const [showArtifacts, setShowArtifacts] = createSignal(false);
	const [cancelling, setCancelling] = createSignal(false);
	const [rerunning, setRerunning] = createSignal(false);
	const [approving, setApproving] = createSignal(false);
	const [liveLogLines, setLiveLogLines] = createSignal<LogEntry[]>([]);
	let logContainerRef: HTMLDivElement | undefined;

	// Project + pipeline names for breadcrumb
	const [projectName, setProjectName] = createSignal('Project');
	const [pipelineName, setPipelineName] = createSignal('Pipeline');

	onMount(async () => {
		try {
			const [project, pipeline] = await Promise.all([
				api.projects.get(params.id),
				api.pipelines.get(params.id, params.pid),
			]);
			setProjectName(project.name);
			setPipelineName(pipeline.name);
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

	// Expand all stages/jobs on load
	createEffect(() => {
		const s = stages();
		const j = jobs();
		if (s.length > 0) {
			setExpandedStages(new Set(s.map(st => st.id)));
			setExpandedJobs(new Set(j.map(jb => jb.id)));
		}
	});

	// WebSocket for live log streaming
	let logSocket: ReturnType<typeof createRunLogSocket> | null = null;
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
		const r = run();
		if (r && (r.status === 'running' || r.status === 'queued' || r.status === 'pending')) {
			logSocket = createRunLogSocket(r.id);
			logSocket.on('log', (payload: unknown) => {
				const line = payload as { step_run_id?: string; content: string; stream?: string };
				setLiveLogLines(prev => [...prev, {
					step_run_id: line.step_run_id ?? '',
					content: line.content,
					stream: line.stream ?? 'stdout',
				}]);
			});
			logSocket.on('status', (payload: unknown) => {
				const status = payload as { status?: string };
				refetch();
				if (status.status === 'success' || status.status === 'failure' || status.status === 'cancelled') {
					logSocket?.disconnect();
					logSocket = null;
				}
			});
			logSocket.on('status_change', () => {
				debouncedRefetch();
			});
			logSocket.on('replay_complete', () => {
				// Replay done
			});
			logSocket.connect();
		}
	});

	onCleanup(() => {
		if (logSocket) {
			logSocket.disconnect();
			logSocket = null;
		}
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

	// Get logs for selected step or all logs — returns LogEntry[] with stream info
	const selectedStepLogEntries = (): LogEntry[] => {
		const stepId = selectedStep();
		if (!stepId) return [];

		if (stepId === '__all__') {
			const historicalEntries: LogEntry[] = allLogs().map(l => ({
				content: l.content,
				stream: l.stream || 'stdout',
				step_run_id: l.step_run_id ?? '',
			}));
			const liveEntries = liveLogLines();
			return [...historicalEntries, ...liveEntries];
		}

		const stepLogs: LogEntry[] = allLogs()
			.filter(l => l.step_run_id === stepId)
			.map(l => ({
				content: l.content,
				stream: l.stream || 'stdout',
				step_run_id: l.step_run_id ?? '',
			}));
		const liveLinesForStep = liveLogLines().filter(l => l.step_run_id === stepId);
		return [...stepLogs, ...liveLinesForStep];
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
			refetch();
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
		<PageContainer
			title={run() ? `Run #${run()!.number}` : 'Run Detail'}
			breadcrumbs={[
				{ label: 'Projects', href: '/projects' },
				{ label: projectName(), href: `/projects/${params.id}` },
				{ label: 'Pipelines', href: `/projects/${params.id}/pipelines` },
				{ label: pipelineName(), href: `/projects/${params.id}/pipelines/${params.pid}` },
				{ label: run() ? `Run #${run()!.number}` : 'Run' },
			]}
			actions={
				<div class="flex items-center gap-2">
					<A href="/runs" class="text-xs text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] mr-2">
						&larr; All Builds
					</A>
					<Show when={prevRun()}>
						<A
							href={`/projects/${params.id}/pipelines/${params.pid}/runs/${prevRun()!.id}`}
							class="px-2 py-1 text-xs rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] border border-[var(--color-border-primary)] transition-colors"
						>
							&larr; #{prevRun()!.number}
						</A>
					</Show>
					<Show when={nextRun()}>
						<A
							href={`/projects/${params.id}/pipelines/${params.pid}/runs/${nextRun()!.id}`}
							class="px-2 py-1 text-xs rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] border border-[var(--color-border-primary)] transition-colors"
						>
							#{nextRun()!.number} &rarr;
						</A>
					</Show>
					<Show when={run()}>
						<Badge variant={getStatusBadgeVariant(run()!.status)} dot>
							{statusLabel[run()!.status]}
						</Badge>
						<Show when={run()!.status === 'running' || run()!.status === 'queued'}>
							<Button size="sm" variant="danger" onClick={handleCancel} loading={cancelling()}>Cancel</Button>
						</Show>
						<Show when={run()!.status === 'waiting_approval'}>
							<Button size="sm" onClick={handleApprove} loading={approving()}>Approve</Button>
						</Show>
						<Show when={run()!.status === 'failure' || run()!.status === 'cancelled'}>
							<Button size="sm" variant="outline" onClick={handleRerun} loading={rerunning()}>Rerun</Button>
						</Show>
					</Show>
					<Button size="sm" variant="ghost" onClick={() => setShowArtifacts(!showArtifacts())}>
						<svg class="w-4 h-4 mr-1" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 2.75a.75.75 0 00-1.5 0v8.614L6.295 8.235a.75.75 0 10-1.09 1.03l4.25 4.5a.75.75 0 001.09 0l4.25-4.5a.75.75 0 00-1.09-1.03l-2.955 3.129V2.75z" /><path d="M3.5 12.75a.75.75 0 00-1.5 0v2.5A2.75 2.75 0 004.75 18h10.5A2.75 2.75 0 0018 15.25v-2.5a.75.75 0 00-1.5 0v2.5c0 .69-.56 1.25-1.25 1.25H4.75c-.69 0-1.25-.56-1.25-1.25v-2.5z" /></svg>
						Artifacts ({artifacts().length})
					</Button>
				</div>
			}
		>
			{/* Error state */}
			<Show when={data.error}>
				<div class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30 flex items-center justify-between">
					<p class="text-sm text-red-400">Failed to load run: {(data.error as Error)?.message}</p>
					<Button size="sm" variant="outline" onClick={refetch}>Retry</Button>
				</div>
			</Show>

			<Show when={!data.loading} fallback={
				<div class="flex gap-6">
					<div class="w-80 space-y-3">
						<For each={[1, 2, 3]}>{() => <div class="h-20 bg-[var(--color-bg-secondary)] rounded-lg animate-pulse" />}</For>
					</div>
					<div class="flex-1 h-[500px] bg-[var(--color-bg-secondary)] rounded-lg animate-pulse" />
				</div>
			}>
				{/* Run info bar */}
				<Show when={run()}>
					<div class="flex flex-wrap items-center gap-4 mb-6 p-4 bg-[var(--color-bg-secondary)] rounded-xl border border-[var(--color-border-primary)]">
						<Show when={run()!.commit_message}>
							<div class="flex items-center gap-2 text-sm">
								<svg class="w-4 h-4 text-[var(--color-text-tertiary)]" viewBox="0 0 20 20" fill="currentColor"><path d="M3.505 2.365A41.369 41.369 0 019 2c1.863 0 3.697.124 5.495.365 1.247.167 2.18 1.108 2.435 2.268a4.45 4.45 0 00-.577-.069 43.141 43.141 0 00-4.706 0C9.229 4.696 7.5 6.727 7.5 8.998v2.24a23.269 23.269 0 01-1.943-.178l-2.46 2.46A.75.75 0 012 12.945V5.147a2.778 2.778 0 011.505-2.782z" /><path d="M10.647 4.563a41.612 41.612 0 00-4.294 0C5.025 4.68 4 5.865 4 7.222v6.195c0 .573.224 1.122.623 1.528l.119.116 2.36-2.36A22.288 22.288 0 009 12.998a22.288 22.288 0 003.898-.297l2.36 2.36.119-.116c.399-.406.623-.955.623-1.528V7.222c0-1.357-1.025-2.542-2.353-2.659z" /></svg>
								<span class="text-[var(--color-text-secondary)]">{run()!.commit_message}</span>
							</div>
						</Show>
						<div class="flex items-center gap-2 text-xs text-[var(--color-text-tertiary)]">
							<Show when={run()!.commit_sha}>
								<span class="font-mono bg-[var(--color-bg-tertiary)] px-2 py-0.5 rounded border border-[var(--color-border-primary)]">{truncateCommitSha(run()!.commit_sha)}</span>
							</Show>
							<Show when={run()!.branch}>
								<span>on</span>
								<span class="font-mono bg-[var(--color-bg-tertiary)] px-2 py-0.5 rounded border border-[var(--color-border-primary)]">{run()!.branch}</span>
							</Show>
							<Show when={run()!.author}>
								<span>by {run()!.author}</span>
							</Show>
							<span>·</span>
							<span>{formatDuration(run()!.duration_ms)}</span>
							<span>·</span>
							<span>{formatRelativeTime(run()!.created_at)}</span>
						</div>
					</div>
				</Show>

				{/* Artifacts panel */}
				<Show when={showArtifacts() && artifacts().length > 0}>
					<div class="mb-6 p-4 bg-[var(--color-bg-secondary)] rounded-xl border border-[var(--color-border-primary)]">
						<h3 class="text-sm font-semibold text-[var(--color-text-primary)] mb-3">Build Artifacts</h3>
						<div class="space-y-2">
							<For each={artifacts()}>
								{(artifact) => (
									<div class="flex items-center justify-between p-3 rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)]">
										<div class="flex items-center gap-3">
											<svg class="w-5 h-5 text-[var(--color-text-tertiary)]" viewBox="0 0 20 20" fill="currentColor">
												<path d="M3 3.5A1.5 1.5 0 014.5 2h6.879a1.5 1.5 0 011.06.44l4.122 4.12A1.5 1.5 0 0117 7.622V16.5a1.5 1.5 0 01-1.5 1.5h-11A1.5 1.5 0 013 16.5v-13z" />
											</svg>
											<div>
												<p class="text-sm font-medium text-[var(--color-text-primary)]">{artifact.name}</p>
												<p class="text-xs text-[var(--color-text-tertiary)]">{artifact.path} · {formatBytes(artifact.size_bytes)}</p>
											</div>
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

				{/* Main content: stage tree + log viewer */}
				<div class="flex gap-6 h-[calc(100vh-380px)] min-h-[500px]">
					{/* Stage/Job/Step tree sidebar */}
					<div class="w-80 flex-shrink-0 overflow-y-auto bg-[var(--color-bg-secondary)] rounded-xl border border-[var(--color-border-primary)]">
						<div class="p-3">
							<h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] mb-3 px-2">Pipeline Stages</h3>
							{/* "All Logs" button */}
							<button
								class={`w-full flex items-center gap-2 px-2 py-2 rounded-lg transition-colors text-left mb-2 ${isAllLogsView()
										? 'bg-indigo-500/10 border border-indigo-500/30'
										: 'hover:bg-[var(--color-bg-hover)]'
									}`}
								onClick={() => setSelectedStep('__all__')}
							>
								<svg class="w-4 h-4 text-[var(--color-text-tertiary)]" viewBox="0 0 20 20" fill="currentColor">
									<path fill-rule="evenodd" d="M4.5 2A1.5 1.5 0 003 3.5v13A1.5 1.5 0 004.5 18h11a1.5 1.5 0 001.5-1.5V7.621a1.5 1.5 0 00-.44-1.06l-4.12-4.122A1.5 1.5 0 0011.378 2H4.5zm2.25 8.5a.75.75 0 000 1.5h6.5a.75.75 0 000-1.5h-6.5zm0 3a.75.75 0 000 1.5h6.5a.75.75 0 000-1.5h-6.5z" clip-rule="evenodd" />
								</svg>
								<span class={`text-sm font-medium flex-1 ${isAllLogsView() ? 'text-indigo-400' : 'text-[var(--color-text-primary)]'}`}>All Logs</span>
								<span class="text-xs text-[var(--color-text-tertiary)]">{allLogs().length + liveLogLines().length}</span>
							</button>

							{/* Stream legend */}
							<div class="flex items-center gap-3 px-2 mb-3 text-[10px] text-[var(--color-text-tertiary)]">
								<span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-emerald-400/50" /> stdout</span>
								<span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-red-400" /> stderr</span>
								<span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-blue-400" /> system</span>
							</div>

							<Show when={stages().length > 0} fallback={
								<p class="text-xs text-[var(--color-text-tertiary)] px-2">No stage data available.</p>
							}>
								<For each={stages()}>
									{(stage) => {
										const stageJobs = () => jobs().filter(j => j.stage_run_id === stage.id);
										const isExpanded = () => expandedStages().has(stage.id);

										return (
											<div class="mb-1">
												<button
													class="w-full flex items-center gap-2 px-2 py-2 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors text-left"
													onClick={() => toggleStage(stage.id)}
												>
													<svg class={`w-3 h-3 text-[var(--color-text-tertiary)] transition-transform ${isExpanded() ? 'rotate-90' : ''}`} viewBox="0 0 20 20" fill="currentColor">
														<path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
													</svg>
													{statusIcon(stage.status)}
													<span class="text-sm font-medium text-[var(--color-text-primary)] capitalize flex-1">{stage.name}</span>
													<span class="text-xs text-[var(--color-text-tertiary)]">
														{stage.started_at && stage.finished_at ? formatDuration(new Date(stage.finished_at).getTime() - new Date(stage.started_at).getTime()) : '-'}
													</span>
												</button>

												<Show when={isExpanded()}>
													<div class="ml-4 pl-3 border-l border-[var(--color-border-primary)]">
														<For each={stageJobs()}>
															{(job) => {
																const jobSteps = () => steps().filter(s => s.job_run_id === job.id);
																const isJobExpanded = () => expandedJobs().has(job.id);

																return (
																	<div class="mb-0.5">
																		<button
																			class="w-full flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors text-left"
																			onClick={() => toggleJob(job.id)}
																		>
																			<svg class={`w-3 h-3 text-[var(--color-text-tertiary)] transition-transform ${isJobExpanded() ? 'rotate-90' : ''}`} viewBox="0 0 20 20" fill="currentColor">
																				<path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
																			</svg>
																			{statusIcon(job.status)}
																			<span class="text-xs font-medium text-[var(--color-text-secondary)] flex-1 truncate">{job.name}</span>
																		</button>

																		<Show when={isJobExpanded()}>
																			<div class="ml-4 pl-3 border-l border-[var(--color-border-primary)]">
																				<For each={jobSteps()}>
																					{(step) => (
																						<button
																							class={`w-full flex items-center gap-2 px-2 py-1.5 rounded-lg transition-colors text-left ${selectedStep() === step.id
																								? 'bg-indigo-500/10 border border-indigo-500/30'
																								: 'hover:bg-[var(--color-bg-hover)]'
																								}`}
																							onClick={() => setSelectedStep(step.id)}
																						>
																							{statusIcon(step.status)}
																							<span class={`text-xs flex-1 truncate ${selectedStep() === step.id ? 'text-indigo-400 font-medium' : 'text-[var(--color-text-tertiary)]'}`}>
																								{step.name}
																							</span>
																							<span class="text-xs text-[var(--color-text-tertiary)]">{formatDuration(step.duration_ms)}</span>
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
						<div class="flex items-center justify-between px-4 py-2 bg-[#161b22] border-b border-[var(--color-border-primary)]">
							<div class="flex items-center gap-2">
								<Show when={isAllLogsView()}>
									<svg class="w-4 h-4 text-[var(--color-text-tertiary)]" viewBox="0 0 20 20" fill="currentColor">
										<path fill-rule="evenodd" d="M4.5 2A1.5 1.5 0 003 3.5v13A1.5 1.5 0 004.5 18h11a1.5 1.5 0 001.5-1.5V7.621a1.5 1.5 0 00-.44-1.06l-4.12-4.122A1.5 1.5 0 0011.378 2H4.5zm2.25 8.5a.75.75 0 000 1.5h6.5a.75.75 0 000-1.5h-6.5zm0 3a.75.75 0 000 1.5h6.5a.75.75 0 000-1.5h-6.5z" clip-rule="evenodd" />
									</svg>
									<span class="text-sm font-medium text-[var(--color-text-primary)]">All Logs</span>
									<span class="text-xs text-[var(--color-text-tertiary)]">({selectedStepLogEntries().length} lines)</span>
								</Show>
								<Show when={!isAllLogsView() && selectedStepData()}>
									{statusIcon(selectedStepData()!.status)}
									<span class="text-sm font-medium text-[var(--color-text-primary)]">{selectedStepData()!.name}</span>
									<Show when={selectedStepData()!.exit_code !== undefined && selectedStepData()!.exit_code !== null}>
										<span class={`text-xs px-1.5 py-0.5 rounded ${selectedStepData()!.exit_code === 0 ? 'bg-emerald-500/10 text-emerald-400' : 'bg-red-500/10 text-red-400'}`}>
											exit {selectedStepData()!.exit_code}
										</span>
									</Show>
								</Show>
							</div>
							<div class="flex items-center gap-2">
								<Show when={isAllLogsView() && run()}>
									<span class="text-xs text-[var(--color-text-tertiary)]">{formatDuration(run()!.duration_ms)}</span>
								</Show>
								<Show when={!isAllLogsView() && selectedStepData()?.duration_ms}>
									<span class="text-xs text-[var(--color-text-tertiary)]">{formatDuration(selectedStepData()?.duration_ms)}</span>
								</Show>
							</div>
						</div>

						{/* Log content */}
						<div
							ref={logContainerRef}
							class="flex-1 overflow-y-auto p-4 font-mono text-sm leading-6"
						>
							<Show when={selectedStepLogEntries().length > 0} fallback={
								<div class="flex items-center justify-center h-full text-[var(--color-text-tertiary)]">
									<Show when={run()?.status === 'running' || run()?.status === 'queued' || run()?.status === 'pending'}
										fallback={isAllLogsView() ? "No logs available for this run." : "No logs available for this step."}>
										<div class="flex flex-col items-center gap-2">
											<svg class="w-6 h-6 animate-spin text-violet-400" viewBox="0 0 24 24" fill="none">
												<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
												<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
											</svg>
											<span>Waiting for logs...</span>
										</div>
									</Show>
								</div>
							}>
								<For each={selectedStepLogEntries()}>
									{(entry, i) => (
										<div class={`flex hover:bg-white/[0.03] group ${entry.stream === 'stderr' ? 'bg-red-500/[0.03]' : ''}`}>
											<span class="w-10 flex-shrink-0 text-right pr-3 text-[var(--color-text-tertiary)] select-none opacity-40 group-hover:opacity-100">
												{i() + 1}
											</span>
											{streamIndicator(entry.stream, entry.content)}
											<span class={`flex-1 whitespace-pre-wrap break-all ml-2 ${streamClass(entry.stream, entry.content)}`} innerHTML={ansiToHtml(entry.content)} />
										</div>
									)}
								</For>
							</Show>
						</div>
					</div>
				</div>
			</Show>
		</PageContainer>
	);
};

export default RunDetailPage;
