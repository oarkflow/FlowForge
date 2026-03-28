import { For, Show, createMemo } from 'solid-js';
import type { Component } from 'solid-js';
import type { PipelineDAG } from '../../types';

interface StageNode {
	id: string;
	name: string;
	status: string;
	position: number;
}

interface JobNode {
	id: string;
	name: string;
	status: string;
	stageId: string;
}

interface PipelineGraphProps {
	stages: StageNode[];
	jobs: JobNode[];
	dag?: PipelineDAG | null;
	onJobClick?: (jobId: string) => void;
}

const statusColors: Record<string, { bg: string; border: string; text: string; dot: string }> = {
	pending: { bg: '#1c1c1c', border: '#444', text: '#888', dot: '#666' },
	queued: { bg: '#1c1c1c', border: '#444', text: '#888', dot: '#666' },
	running: { bg: '#0c2d48', border: '#1d6fb8', text: '#58a6ff', dot: '#58a6ff' },
	success: { bg: '#0d2818', border: '#238636', text: '#3fb950', dot: '#3fb950' },
	failure: { bg: '#2d1117', border: '#da3633', text: '#ff7b72', dot: '#ff7b72' },
	cancelled: { bg: '#2d2013', border: '#d29922', text: '#e3b341', dot: '#d29922' },
	skipped: { bg: '#1c1c1c', border: '#444', text: '#888', dot: '#666' },
};

/**
 * PipelineGraph - Visual DAG representation of pipeline stages and jobs.
 * Supports both sequential and parallel (DAG-based) layouts.
 */
export const PipelineGraph: Component<PipelineGraphProps> = (props) => {
	const stagesByName = createMemo(() => {
		const map = new Map<string, StageNode>();
		for (const s of props.stages) {
			map.set(s.name, s);
		}
		return map;
	});

	const jobsByStage = createMemo(() => {
		const map = new Map<string, JobNode[]>();
		for (const job of props.jobs) {
			const existing = map.get(job.stageId) || [];
			existing.push(job);
			map.set(job.stageId, existing);
		}
		return map;
	});

	// Check if we have a DAG with multiple levels (parallel execution)
	const isDAGMode = createMemo(() => {
		const dag = props.dag;
		if (!dag || dag.has_cycle) return false;
		// DAG mode if any level has more than one stage
		return dag.levels.some(level => level.length > 1);
	});

	// For sequential mode, sort stages by position
	const sortedStages = createMemo(() =>
		[...props.stages].sort((a, b) => a.position - b.position)
	);

	const getColors = (status: string) => statusColors[status] || statusColors.pending;

	// Render a single stage column
	const StageColumn = (stageProps: { stage: StageNode; showParallelIndicator?: boolean }) => {
		const stage = stageProps.stage;
		const stageJobs = () => jobsByStage().get(stage.id) || [];
		const colors = () => getColors(stage.status);

		return (
			<div class="flex flex-col gap-2 min-w-[180px]">
				{/* Stage header */}
				<div
					class="text-xs font-medium px-3 py-1.5 rounded-md text-center uppercase tracking-wider relative"
					style={`background: ${colors().bg}; border: 1px solid ${colors().border}; color: ${colors().text};`}
				>
					<span class="inline-block w-2 h-2 rounded-full mr-1.5" style={`background: ${colors().dot}; ${stage.status === 'running' ? 'animation: pulse-dot 1.5s infinite;' : ''}`} />
					{stage.name}
					<Show when={stageProps.showParallelIndicator}>
						<span class="absolute -top-1.5 -right-1.5 w-4 h-4 bg-indigo-500 rounded-full flex items-center justify-center" title="Runs in parallel">
							<svg class="w-2.5 h-2.5 text-white" viewBox="0 0 16 16" fill="currentColor">
								<path d="M2 4h4v2H2V4zm0 6h4v2H2v-2zm8-6h4v2h-4V4zm0 6h4v2h-4v-2z" />
							</svg>
						</span>
					</Show>
				</div>

				{/* Jobs in stage */}
				<For each={stageJobs()}>
					{(job) => {
						const jc = () => getColors(job.status);
						return (
							<button
								class="text-left px-3 py-2 rounded-md text-sm transition-all hover:brightness-125 cursor-pointer"
								style={`background: ${jc().bg}; border: 1px solid ${jc().border}; color: ${jc().text};`}
								onClick={() => props.onJobClick?.(job.id)}
							>
								<div class="flex items-center gap-2">
									<span
										class="w-2 h-2 rounded-full shrink-0"
										style={`background: ${jc().dot}; ${job.status === 'running' ? 'animation: pulse-dot 1.5s infinite;' : ''}`}
									/>
									<span class="truncate">{job.name}</span>
								</div>
							</button>
						);
					}}
				</For>

				<Show when={stageJobs().length === 0}>
					<div class="text-xs text-center py-2" style="color: var(--text-tertiary);">
						No jobs
					</div>
				</Show>
			</div>
		);
	};

	return (
		<div class="overflow-x-auto py-4">
			<Show when={isDAGMode()} fallback={
				/* Sequential layout (backward compatible) */
				<div class="flex items-start gap-2 min-w-max px-4">
					<For each={sortedStages()}>
						{(stage, i) => (
							<>
								{/* Connector line between stages */}
								<Show when={i() > 0}>
									<div class="flex items-center self-center pt-2">
										<div class="w-8 h-0.5" style="background: var(--border-primary);" />
										<svg width="8" height="12" viewBox="0 0 8 12" fill="var(--border-primary)">
											<path d="M0 0L8 6L0 12Z" />
										</svg>
									</div>
								</Show>
								<StageColumn stage={stage} />
							</>
						)}
					</For>
				</div>
			}>
				{/* DAG-based parallel layout */}
				<div class="flex items-start gap-2 min-w-max px-4">
					<For each={props.dag!.levels}>
						{(level, levelIdx) => {
							const isParallel = () => level.length > 1;
							return (
								<>
									{/* Connector between levels */}
									<Show when={levelIdx() > 0}>
										<div class="flex items-center self-center pt-2">
											<div class="w-8 h-0.5" style="background: var(--border-primary);" />
											<svg width="8" height="12" viewBox="0 0 8 12" fill="var(--border-primary)">
												<path d="M0 0L8 6L0 12Z" />
											</svg>
										</div>
									</Show>

									<Show when={isParallel()} fallback={
										/* Single stage at this level */
										<Show when={stagesByName().get(level[0])}>
											{(stage) => <StageColumn stage={stage()} />}
										</Show>
									}>
										{/* Multiple parallel stages at this level */}
										<div class="flex flex-col gap-3 relative">
											{/* Level label */}
											<div class="text-[10px] font-medium text-center px-2 py-0.5 rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 self-center">
												Level {levelIdx()} · {level.length} parallel
											</div>
											{/* Parallel group container */}
											<div class="flex flex-col gap-3 px-2 py-2 rounded-lg border border-dashed border-indigo-500/30 bg-indigo-500/5">
												<For each={level}>
													{(stageName) => {
														const stage = () => stagesByName().get(stageName);
														return (
															<Show when={stage()}>
																{(s) => <StageColumn stage={s()} showParallelIndicator={true} />}
															</Show>
														);
													}}
												</For>
											</div>
										</div>
									</Show>
								</>
							);
						}}
					</For>
				</div>

				{/* DAG dependency info */}
				<Show when={props.dag && Object.keys(props.dag!.nodes).length > 0}>
					<div class="mt-4 px-4">
						<div class="text-xs text-[var(--color-text-tertiary)] flex items-center gap-4 flex-wrap">
							<span class="flex items-center gap-1.5">
								<span class="w-3 h-3 rounded-full bg-indigo-500/20 border border-indigo-500/40 inline-flex items-center justify-center">
									<svg class="w-2 h-2 text-indigo-400" viewBox="0 0 16 16" fill="currentColor">
										<path d="M2 4h4v2H2V4zm0 6h4v2H2v-2zm8-6h4v2h-4V4zm0 6h4v2h-4v-2z" />
									</svg>
								</span>
								Parallel stages
							</span>
							<span class="flex items-center gap-1.5">
								<span class="w-6 h-0.5 bg-[var(--color-border-primary)] inline-block" />
								<svg width="6" height="10" viewBox="0 0 8 12" fill="var(--color-border-primary)" class="inline-block">
									<path d="M0 0L8 6L0 12Z" />
								</svg>
								Dependency flow
							</span>
							<span>{props.dag!.levels.length} execution level{props.dag!.levels.length !== 1 ? 's' : ''}</span>
							<span>{Object.keys(props.dag!.nodes).length} stage{Object.keys(props.dag!.nodes).length !== 1 ? 's' : ''}</span>
						</div>
					</div>
				</Show>
			</Show>
		</div>
	);
};
