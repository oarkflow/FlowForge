import type { Component, JSX } from 'solid-js';
import { createSignal, createEffect, createMemo, For, Show, onMount, onCleanup, batch } from 'solid-js';
import * as yaml from 'js-yaml';
import Button from '../ui/Button';
import Input from '../ui/Input';
import Modal from '../ui/Modal';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
interface DagJob {
	id: string;
	name: string;
	stage: string;
	image: string;
	executor: string;
	needs: string[];
	steps: { name: string; run: string }[];
	x: number;
	y: number;
}

interface DagEdge {
	from: string;
	to: string;
}

interface DagBuilderProps {
	initialYaml?: string;
	onExport?: (yamlContent: string) => void;
	readOnly?: boolean;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------
const NODE_WIDTH = 180;
const NODE_HEIGHT = 64;
const GRID_SIZE = 20;
const MIN_ZOOM = 0.3;
const MAX_ZOOM = 2;

const EXECUTOR_COLORS: Record<string, { bg: string; border: string; text: string }> = {
	docker: { bg: '#0c2d48', border: '#1d6fb8', text: '#58a6ff' },
	local: { bg: '#0d2818', border: '#238636', text: '#3fb950' },
	kubernetes: { bg: '#2d1f4e', border: '#8b5cf6', text: '#a78bfa' },
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
const uid = () => crypto.randomUUID().slice(0, 8);

function snapToGrid(val: number): number {
	return Math.round(val / GRID_SIZE) * GRID_SIZE;
}

function parseYamlToJobs(yamlContent: string): DagJob[] {
	try {
		const spec = yaml.load(yamlContent) as Record<string, any>;
		if (!spec?.jobs) return [];

		const jobs: DagJob[] = [];
		const stagePositions = new Map<string, number>();
		let stageIdx = 0;

		// Collect stages for positioning
		const stages = spec.stages as string[] | undefined;
		if (stages) {
			stages.forEach((s: string, i: number) => stagePositions.set(s, i));
		}

		let jobIdx = 0;
		for (const [key, val] of Object.entries(spec.jobs)) {
			const job = val as Record<string, any>;
			const stage = job.stage || 'default';
			if (!stagePositions.has(stage)) {
				stagePositions.set(stage, stageIdx++);
			}
			const stagePos = stagePositions.get(stage) ?? 0;

			// Count jobs per stage for vertical offset
			const sameStageJobs = jobs.filter(j => j.stage === stage).length;

			jobs.push({
				id: uid(),
				name: key,
				stage,
				image: job.image || '',
				executor: job.executor || 'docker',
				needs: Array.isArray(job.needs) ? job.needs : [],
				steps: Array.isArray(job.steps) ? job.steps.map((s: any) => ({
					name: s.name || s.uses || 'Step',
					run: s.run || s.uses || '',
				})) : [],
				x: stagePos * (NODE_WIDTH + 100) + 60,
				y: sameStageJobs * (NODE_HEIGHT + 40) + 60,
			});
			jobIdx++;
		}

		return jobs;
	} catch {
		return [];
	}
}

function jobsToYaml(jobs: DagJob[], existingYaml?: string): string {
	let spec: Record<string, any> = {};
	try {
		if (existingYaml) spec = (yaml.load(existingYaml) as Record<string, any>) || {};
	} catch { /* ignore parse errors */ }

	// Collect stages
	const stages = [...new Set(jobs.map(j => j.stage))];
	spec.stages = stages;

	// Build jobs
	const jobsObj: Record<string, any> = {};
	for (const job of jobs) {
		const jobDef: Record<string, any> = { stage: job.stage };
		if (job.executor) jobDef.executor = job.executor;
		if (job.image) jobDef.image = job.image;
		if (job.needs.length > 0) jobDef.needs = job.needs;
		if (job.steps.length > 0) {
			jobDef.steps = job.steps.map(s => {
				const step: Record<string, any> = {};
				if (s.name) step.name = s.name;
				if (s.run) step.run = s.run;
				return step;
			});
		}
		jobsObj[job.name] = jobDef;
	}
	spec.jobs = jobsObj;

	return yaml.dump(spec, { indent: 2, lineWidth: 120, noRefs: true });
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const DagBuilder: Component<DagBuilderProps> = (props) => {
	let svgRef: SVGSVGElement | undefined;
	const [jobs, setJobs] = createSignal<DagJob[]>([]);
	const [selectedJobId, setSelectedJobId] = createSignal<string | null>(null);
	const [dragging, setDragging] = createSignal<{ id: string; offsetX: number; offsetY: number } | null>(null);
	const [panning, setPanning] = createSignal<{ startX: number; startY: number; originX: number; originY: number } | null>(null);
	const [viewOffset, setViewOffset] = createSignal({ x: 0, y: 0 });
	const [zoom, setZoom] = createSignal(1);
	const [showAddJob, setShowAddJob] = createSignal(false);
	const [showConfig, setShowConfig] = createSignal(false);
	const [connecting, setConnecting] = createSignal<{ fromId: string; mouseX: number; mouseY: number } | null>(null);

	// Add job form
	const [newJobName, setNewJobName] = createSignal('');
	const [newJobStage, setNewJobStage] = createSignal('build');
	const [newJobExecutor, setNewJobExecutor] = createSignal('docker');
	const [newJobImage, setNewJobImage] = createSignal('ubuntu:22.04');

	// Config panel form
	const [configName, setConfigName] = createSignal('');
	const [configStage, setConfigStage] = createSignal('');
	const [configImage, setConfigImage] = createSignal('');
	const [configExecutor, setConfigExecutor] = createSignal('');
	const [configStepName, setConfigStepName] = createSignal('');
	const [configStepRun, setConfigStepRun] = createSignal('');

	// Parse initial YAML
	onMount(() => {
		if (props.initialYaml) {
			setJobs(parseYamlToJobs(props.initialYaml));
		}
	});

	createEffect(() => {
		if (props.initialYaml) {
			const parsed = parseYamlToJobs(props.initialYaml);
			if (parsed.length > 0) setJobs(parsed);
		}
	});

	// Compute edges from job dependencies
	const edges = createMemo((): DagEdge[] => {
		const allJobs = jobs();
		const result: DagEdge[] = [];
		for (const job of allJobs) {
			for (const dep of job.needs) {
				const depJob = allJobs.find(j => j.name === dep);
				if (depJob) {
					result.push({ from: depJob.id, to: job.id });
				}
			}
		}
		return result;
	});

	const selectedJob = createMemo(() => {
		const id = selectedJobId();
		return id ? jobs().find(j => j.id === id) : undefined;
	});

	const stages = createMemo(() => [...new Set(jobs().map(j => j.stage))]);

	// ---------------------------------------------------------------------------
	// Mouse handlers
	// ---------------------------------------------------------------------------
	const svgPoint = (clientX: number, clientY: number): { x: number; y: number } => {
		if (!svgRef) return { x: clientX, y: clientY };
		const rect = svgRef.getBoundingClientRect();
		return {
			x: (clientX - rect.left - viewOffset().x) / zoom(),
			y: (clientY - rect.top - viewOffset().y) / zoom(),
		};
	};

	const handleMouseDown = (e: MouseEvent) => {
		if (e.button !== 0) return;
		// Start panning if clicking on background
		if (e.target === svgRef || (e.target as SVGElement).classList.contains('dag-bg')) {
			setPanning({ startX: e.clientX, startY: e.clientY, originX: viewOffset().x, originY: viewOffset().y });
			setSelectedJobId(null);
		}
	};

	const handleMouseMove = (e: MouseEvent) => {
		const pan = panning();
		if (pan) {
			setViewOffset({
				x: pan.originX + (e.clientX - pan.startX),
				y: pan.originY + (e.clientY - pan.startY),
			});
			return;
		}

		const drag = dragging();
		if (drag) {
			const pt = svgPoint(e.clientX, e.clientY);
			setJobs(prev => prev.map(j =>
				j.id === drag.id
					? { ...j, x: snapToGrid(pt.x - drag.offsetX), y: snapToGrid(pt.y - drag.offsetY) }
					: j
			));
			return;
		}

		const conn = connecting();
		if (conn && svgRef) {
			const rect = svgRef.getBoundingClientRect();
			setConnecting({ ...conn, mouseX: (e.clientX - rect.left - viewOffset().x) / zoom(), mouseY: (e.clientY - rect.top - viewOffset().y) / zoom() });
		}
	};

	const handleMouseUp = (e: MouseEvent) => {
		if (panning()) setPanning(null);
		if (dragging()) setDragging(null);
		if (connecting()) {
			// Check if we dropped on a job node
			const conn = connecting()!;
			const pt = svgPoint(e.clientX, e.clientY);
			const targetJob = jobs().find(j =>
				j.id !== conn.fromId &&
				pt.x >= j.x && pt.x <= j.x + NODE_WIDTH &&
				pt.y >= j.y && pt.y <= j.y + NODE_HEIGHT
			);
			if (targetJob) {
				// Add dependency
				setJobs(prev => prev.map(j =>
					j.id === targetJob.id && !j.needs.includes(jobs().find(jj => jj.id === conn.fromId)?.name ?? '')
						? { ...j, needs: [...j.needs, jobs().find(jj => jj.id === conn.fromId)?.name ?? ''] }
						: j
				));
			}
			setConnecting(null);
		}
	};

	const handleWheel = (e: WheelEvent) => {
		e.preventDefault();
		const delta = e.deltaY > 0 ? -0.1 : 0.1;
		setZoom(prev => Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, prev + delta)));
	};

	const handleNodeMouseDown = (e: MouseEvent, jobId: string) => {
		e.stopPropagation();
		const pt = svgPoint(e.clientX, e.clientY);
		const job = jobs().find(j => j.id === jobId);
		if (!job) return;
		setDragging({ id: jobId, offsetX: pt.x - job.x, offsetY: pt.y - job.y });
		setSelectedJobId(jobId);
	};

	const handleConnectorMouseDown = (e: MouseEvent, jobId: string) => {
		e.stopPropagation();
		const job = jobs().find(j => j.id === jobId);
		if (!job) return;
		setConnecting({ fromId: jobId, mouseX: job.x + NODE_WIDTH, mouseY: job.y + NODE_HEIGHT / 2 });
	};

	// ---------------------------------------------------------------------------
	// Job CRUD
	// ---------------------------------------------------------------------------
	const addJob = () => {
		if (!newJobName().trim()) return;
		const stageJobs = jobs().filter(j => j.stage === newJobStage());
		const maxX = stageJobs.length > 0 ? Math.max(...stageJobs.map(j => j.x)) : 60;
		const maxY = stageJobs.length > 0 ? Math.max(...stageJobs.map(j => j.y)) + NODE_HEIGHT + 40 : 60;
		setJobs(prev => [...prev, {
			id: uid(),
			name: newJobName().trim(),
			stage: newJobStage(),
			image: newJobImage(),
			executor: newJobExecutor(),
			needs: [],
			steps: [{ name: 'Run', run: 'echo "Hello"' }],
			x: maxX,
			y: maxY,
		}]);
		batch(() => {
			setNewJobName('');
			setShowAddJob(false);
		});
	};

	const removeJob = (id: string) => {
		const jobName = jobs().find(j => j.id === id)?.name;
		setJobs(prev => prev
			.filter(j => j.id !== id)
			.map(j => ({ ...j, needs: j.needs.filter(n => n !== jobName) }))
		);
		if (selectedJobId() === id) setSelectedJobId(null);
	};

	const openConfig = (id: string) => {
		const job = jobs().find(j => j.id === id);
		if (!job) return;
		setConfigName(job.name);
		setConfigStage(job.stage);
		setConfigImage(job.image);
		setConfigExecutor(job.executor);
		setConfigStepName(job.steps[0]?.name || '');
		setConfigStepRun(job.steps[0]?.run || '');
		setSelectedJobId(id);
		setShowConfig(true);
	};

	const saveConfig = () => {
		const id = selectedJobId();
		if (!id) return;
		setJobs(prev => prev.map(j =>
			j.id === id ? {
				...j,
				name: configName().trim() || j.name,
				stage: configStage() || j.stage,
				image: configImage(),
				executor: configExecutor(),
				steps: [{ name: configStepName(), run: configStepRun() }],
			} : j
		));
		setShowConfig(false);
	};

	const removeEdge = (from: string, to: string) => {
		const fromJob = jobs().find(j => j.id === from);
		if (!fromJob) return;
		setJobs(prev => prev.map(j =>
			j.id === to ? { ...j, needs: j.needs.filter(n => n !== fromJob.name) } : j
		));
	};

	const handleExport = () => {
		const yamlContent = jobsToYaml(jobs(), props.initialYaml);
		props.onExport?.(yamlContent);
	};

	// ---------------------------------------------------------------------------
	// Edge path computation
	// ---------------------------------------------------------------------------
	const computeEdgePath = (fromJob: DagJob, toJob: DagJob): string => {
		const x1 = fromJob.x + NODE_WIDTH;
		const y1 = fromJob.y + NODE_HEIGHT / 2;
		const x2 = toJob.x;
		const y2 = toJob.y + NODE_HEIGHT / 2;
		const midX = (x1 + x2) / 2;
		return `M ${x1} ${y1} C ${midX} ${y1}, ${midX} ${y2}, ${x2} ${y2}`;
	};

	// ---------------------------------------------------------------------------
	// Keyboard handler
	// ---------------------------------------------------------------------------
	const handleKeyDown = (e: KeyboardEvent) => {
		if (e.key === 'Delete' || e.key === 'Backspace') {
			const id = selectedJobId();
			if (id && !showConfig() && !showAddJob()) {
				removeJob(id);
			}
		}
		if (e.key === 'Escape') {
			setSelectedJobId(null);
			setConnecting(null);
		}
	};

	onMount(() => {
		document.addEventListener('keydown', handleKeyDown);
	});

	onCleanup(() => {
		document.removeEventListener('keydown', handleKeyDown);
	});

	// ---------------------------------------------------------------------------
	// Render
	// ---------------------------------------------------------------------------
	return (
		<div class="flex flex-col rounded-xl overflow-hidden border border-[var(--color-border-primary)] bg-[#0d1117]">
			{/* Toolbar */}
			<div class="flex items-center justify-between px-3 py-2 bg-[#161b22] border-b border-[var(--color-border-primary)]">
				<div class="flex items-center gap-2">
					<span class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">DAG Builder</span>
					<span class="px-1.5 py-0.5 text-[10px] rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] border border-[var(--color-border-primary)]">
						{jobs().length} jobs
					</span>
					<span class="px-1.5 py-0.5 text-[10px] rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] border border-[var(--color-border-primary)]">
						{edges().length} edges
					</span>
				</div>
				<div class="flex items-center gap-2">
					<Show when={!props.readOnly}>
						<Button size="sm" variant="outline" onClick={() => setShowAddJob(true)}
							icon={<svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>}
						>Add Job</Button>
					</Show>
					<Button size="sm" variant="outline" onClick={() => setZoom(1)}>
						{Math.round(zoom() * 100)}%
					</Button>
					<Button size="sm" variant="outline" onClick={handleExport}
						icon={<svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path d="M13.75 7h-3v5.296l1.943-2.048a.75.75 0 011.114 1.004l-3.25 3.5a.75.75 0 01-1.114 0l-3.25-3.5a.75.75 0 111.114-1.004l1.943 2.048V7h1.5V1.75a.75.75 0 00-1.5 0V7h-3A2.25 2.25 0 003 9.25v7.5A2.25 2.25 0 005.25 19h9.5A2.25 2.25 0 0017 16.75v-7.5A2.25 2.25 0 0014.75 7h-1z" /></svg>}
					>Export YAML</Button>
				</div>
			</div>

			{/* Canvas */}
			<svg
				ref={svgRef}
				class="w-full cursor-grab active:cursor-grabbing"
				style={{ height: '500px' }}
				onMouseDown={handleMouseDown}
				onMouseMove={handleMouseMove}
				onMouseUp={handleMouseUp}
				onWheel={handleWheel}
			>
				{/* Background grid */}
				<defs>
					<pattern id="dag-grid" width={GRID_SIZE} height={GRID_SIZE} patternUnits="userSpaceOnUse">
						<circle cx="1" cy="1" r="0.5" fill="#21262d" />
					</pattern>
				</defs>
				<rect class="dag-bg" width="100%" height="100%" fill="url(#dag-grid)" />

				{/* Transform group for zoom/pan */}
				<g transform={`translate(${viewOffset().x}, ${viewOffset().y}) scale(${zoom()})`}>
					{/* Stage backgrounds */}
					<For each={stages()}>
						{(stage) => {
							const stageJobs = () => jobs().filter(j => j.stage === stage);
							if (stageJobs().length === 0) return null;
							const minX = () => Math.min(...stageJobs().map(j => j.x)) - 10;
							const minY = () => Math.min(...stageJobs().map(j => j.y)) - 30;
							const maxX = () => Math.max(...stageJobs().map(j => j.x + NODE_WIDTH)) + 10;
							const maxY = () => Math.max(...stageJobs().map(j => j.y + NODE_HEIGHT)) + 10;
							return (
								<g>
									<rect
										x={minX()} y={minY()}
										width={maxX() - minX()} height={maxY() - minY()}
										rx="8" fill="#21262d" opacity="0.3"
										stroke="#30363d" stroke-width="1" stroke-dasharray="4 4"
									/>
									<text x={minX() + 8} y={minY() + 16} fill="#484f58" font-size="11" font-weight="500">
										{stage}
									</text>
								</g>
							);
						}}
					</For>

					{/* Edges */}
					<For each={edges()}>
						{(edge) => {
							const fromJob = () => jobs().find(j => j.id === edge.from);
							const toJob = () => jobs().find(j => j.id === edge.to);
							return (
								<Show when={fromJob() && toJob()}>
									<g>
										<path
											d={computeEdgePath(fromJob()!, toJob()!)}
											fill="none" stroke="#484f58" stroke-width="2"
											class="transition-colors"
										/>
										{/* Arrow at end */}
										{(() => {
											const to = toJob()!;
											return (
												<polygon
													points={`${to.x - 6},${to.y + NODE_HEIGHT / 2 - 4} ${to.x},${to.y + NODE_HEIGHT / 2} ${to.x - 6},${to.y + NODE_HEIGHT / 2 + 4}`}
													fill="#484f58"
												/>
											);
										})()}
										{/* Remove edge button (shown on hover via CSS would be ideal, using always-visible small X) */}
										<Show when={!props.readOnly}>
											{(() => {
												const from = fromJob()!;
												const to = toJob()!;
												const midX = (from.x + NODE_WIDTH + to.x) / 2;
												const midY = (from.y + NODE_HEIGHT / 2 + to.y + NODE_HEIGHT / 2) / 2;
												return (
													<g
														class="cursor-pointer opacity-0 hover:opacity-100 transition-opacity"
														onClick={(e) => { e.stopPropagation(); removeEdge(edge.from, edge.to); }}
													>
														<circle cx={midX} cy={midY} r="8" fill="#da3633" opacity="0.9" />
														<text x={midX} y={midY + 4} text-anchor="middle" fill="white" font-size="10" font-weight="bold">x</text>
													</g>
												);
											})()}
										</Show>
									</g>
								</Show>
							);
						}}
					</For>

					{/* Connecting line (drag to connect) */}
					<Show when={connecting()}>
						{(() => {
							const conn = connecting()!;
							const fromJob = jobs().find(j => j.id === conn.fromId);
							if (!fromJob) return null;
							return (
								<line
									x1={fromJob.x + NODE_WIDTH} y1={fromJob.y + NODE_HEIGHT / 2}
									x2={conn.mouseX} y2={conn.mouseY}
									stroke="#58a6ff" stroke-width="2" stroke-dasharray="6 3"
								/>
							);
						})()}
					</Show>

					{/* Job nodes */}
					<For each={jobs()}>
						{(job) => {
							const isSelected = () => selectedJobId() === job.id;
							const colors = () => EXECUTOR_COLORS[job.executor] || EXECUTOR_COLORS.docker;
							return (
								<g
									class="cursor-pointer"
									onMouseDown={(e) => handleNodeMouseDown(e, job.id)}
									onDblClick={(e) => { e.stopPropagation(); openConfig(job.id); }}
								>
									{/* Node background */}
									<rect
										x={job.x} y={job.y}
										width={NODE_WIDTH} height={NODE_HEIGHT}
										rx="8"
										fill={colors().bg}
										stroke={isSelected() ? '#58a6ff' : colors().border}
										stroke-width={isSelected() ? 2 : 1}
									/>

									{/* Status dot */}
									<circle cx={job.x + 14} cy={job.y + 22} r="4" fill={colors().text} />

									{/* Job name */}
									<text
										x={job.x + 26} y={job.y + 26}
										fill={colors().text} font-size="12" font-weight="600"
										class="select-none"
									>
										{job.name.length > 16 ? job.name.slice(0, 16) + '...' : job.name}
									</text>

									{/* Stage label */}
									<text
										x={job.x + 14} y={job.y + 46}
										fill="#484f58" font-size="10"
										class="select-none"
									>
										{job.stage} | {job.executor}
									</text>

									{/* Steps count */}
									<text
										x={job.x + NODE_WIDTH - 14} y={job.y + 46}
										fill="#484f58" font-size="10" text-anchor="end"
										class="select-none"
									>
										{job.steps.length} step{job.steps.length !== 1 ? 's' : ''}
									</text>

									{/* Connector (right side, for creating edges) */}
									<Show when={!props.readOnly}>
										<circle
											cx={job.x + NODE_WIDTH} cy={job.y + NODE_HEIGHT / 2}
											r="6" fill="#30363d" stroke="#484f58" stroke-width="1"
											class="cursor-crosshair hover:fill-indigo-500 hover:stroke-indigo-400 transition-colors"
											onMouseDown={(e) => handleConnectorMouseDown(e, job.id)}
										/>
									</Show>

									{/* Input connector (left side) */}
									<Show when={job.needs.length > 0}>
										<circle
											cx={job.x} cy={job.y + NODE_HEIGHT / 2}
											r="4" fill="#484f58"
										/>
									</Show>
								</g>
							);
						}}
					</For>
				</g>
			</svg>

			{/* Help bar */}
			<div class="flex items-center justify-between px-3 py-1.5 bg-[#161b22] border-t border-[var(--color-border-primary)] text-[10px] text-[var(--color-text-tertiary)]">
				<div class="flex items-center gap-3">
					<span>Drag nodes to move</span>
					<span>Double-click to configure</span>
					<span>Drag connectors to link</span>
					<span>Scroll to zoom</span>
					<span>Delete key to remove</span>
				</div>
			</div>

			{/* Add Job Modal */}
			<Show when={showAddJob()}>
				<Modal
					open={showAddJob()}
					onClose={() => setShowAddJob(false)}
					title="Add Job"
					footer={
						<>
							<Button variant="ghost" onClick={() => setShowAddJob(false)}>Cancel</Button>
							<Button onClick={addJob} disabled={!newJobName().trim()}>Add Job</Button>
						</>
					}
				>
					<div class="space-y-4">
						<Input label="Job Name" value={newJobName()} onInput={(e) => setNewJobName(e.currentTarget.value)} placeholder="e.g., build-app" />
						<Input label="Stage" value={newJobStage()} onInput={(e) => setNewJobStage(e.currentTarget.value)} placeholder="e.g., build" />
						<div>
							<label class="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">Executor</label>
							<select
								value={newJobExecutor()}
								onChange={(e) => setNewJobExecutor(e.currentTarget.value)}
								class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
							>
								<option value="docker">Docker</option>
								<option value="local">Local</option>
								<option value="kubernetes">Kubernetes</option>
							</select>
						</div>
						<Input label="Image" value={newJobImage()} onInput={(e) => setNewJobImage(e.currentTarget.value)} placeholder="e.g., ubuntu:22.04" />
					</div>
				</Modal>
			</Show>

			{/* Config Side Panel */}
			<Show when={showConfig()}>
				<Modal
					open={showConfig()}
					onClose={() => setShowConfig(false)}
					title={`Configure: ${selectedJob()?.name}`}
					footer={
						<>
							<Show when={!props.readOnly}>
								<Button variant="danger" onClick={() => { removeJob(selectedJobId()!); setShowConfig(false); }}>Delete Job</Button>
							</Show>
							<div class="flex-1" />
							<Button variant="ghost" onClick={() => setShowConfig(false)}>Cancel</Button>
							<Show when={!props.readOnly}>
								<Button onClick={saveConfig}>Save</Button>
							</Show>
						</>
					}
				>
					<div class="space-y-4">
						<Input label="Job Name" value={configName()} onInput={(e) => setConfigName(e.currentTarget.value)} readOnly={props.readOnly} />
						<Input label="Stage" value={configStage()} onInput={(e) => setConfigStage(e.currentTarget.value)} readOnly={props.readOnly} />
						<div>
							<label class="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">Executor</label>
							<select
								value={configExecutor()}
								onChange={(e) => setConfigExecutor(e.currentTarget.value)}
								disabled={props.readOnly}
								class="w-full px-3 py-2 text-sm rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
							>
								<option value="docker">Docker</option>
								<option value="local">Local</option>
								<option value="kubernetes">Kubernetes</option>
							</select>
						</div>
						<Input label="Image" value={configImage()} onInput={(e) => setConfigImage(e.currentTarget.value)} readOnly={props.readOnly} />
						<div class="border-t border-[var(--color-border-primary)] pt-4">
							<label class="block text-xs font-medium text-[var(--color-text-secondary)] mb-2">Dependencies</label>
							<div class="flex flex-wrap gap-1">
								<Show when={selectedJob()?.needs.length} fallback={<span class="text-xs text-[var(--color-text-tertiary)]">No dependencies</span>}>
									<For each={selectedJob()?.needs ?? []}>
										{(dep) => (
											<span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-[var(--color-bg-tertiary)] text-xs text-[var(--color-text-secondary)] border border-[var(--color-border-primary)]">
												{dep}
												<Show when={!props.readOnly}>
													<button
														class="text-red-400 hover:text-red-300 cursor-pointer"
														onClick={() => {
															setJobs(prev => prev.map(j =>
																j.id === selectedJobId() ? { ...j, needs: j.needs.filter(n => n !== dep) } : j
															));
														}}
													>x</button>
												</Show>
											</span>
										)}
									</For>
								</Show>
							</div>
						</div>
						<div class="border-t border-[var(--color-border-primary)] pt-4">
							<label class="block text-xs font-medium text-[var(--color-text-secondary)] mb-2">Step</label>
							<Input label="Step Name" value={configStepName()} onInput={(e) => setConfigStepName(e.currentTarget.value)} readOnly={props.readOnly} />
							<div class="mt-2">
								<label class="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">Run Command</label>
								<textarea
									value={configStepRun()}
									onInput={(e) => setConfigStepRun(e.currentTarget.value)}
									readOnly={props.readOnly}
									rows={4}
									class="w-full px-3 py-2 text-sm font-mono rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40 resize-none"
								/>
							</div>
						</div>
					</div>
				</Modal>
			</Show>
		</div>
	);
};

export default DagBuilder;
