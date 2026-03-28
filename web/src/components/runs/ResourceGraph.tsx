import type { Component } from 'solid-js';
import { createSignal, createMemo, For, Show, onMount, onCleanup } from 'solid-js';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
export interface StepResourcePoint {
	timestamp: number; // ms since run start
	cpu_percent: number;
	memory_mb: number;
	step_name: string;
}

export interface StepResourceSummary {
	step_name: string;
	step_id: string;
	avg_cpu: number;
	max_cpu: number;
	avg_memory: number;
	max_memory: number;
	duration_ms: number;
	started_at?: string;
}

interface ResourceGraphProps {
	data: StepResourcePoint[];
	steps: StepResourceSummary[];
	totalDurationMs?: number;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const ResourceGraph: Component<ResourceGraphProps> = (props) => {
	let canvasRef: HTMLCanvasElement | undefined;
	let chart: any;
	const [loaded, setLoaded] = createSignal(false);
	const [activeMetric, setActiveMetric] = createSignal<'cpu' | 'memory'>('cpu');

	const maxCpu = createMemo(() => Math.max(...props.steps.map(s => s.max_cpu), 10));
	const maxMem = createMemo(() => Math.max(...props.steps.map(s => s.max_memory), 64));

	onMount(async () => {
		try {
			const { Chart, registerables } = await import('chart.js');
			Chart.register(...registerables);
			Chart.defaults.color = '#484f58';
			Chart.defaults.borderColor = '#21262d';
			setLoaded(true);
			requestAnimationFrame(() => createChart(Chart));
		} catch {
			// chart.js not available
		}
	});

	const createChart = (Chart: any) => {
		if (!canvasRef || !props.data.length) return;

		const data = props.data;
		const labels = data.map(d => {
			const sec = Math.round(d.timestamp / 1000);
			const min = Math.floor(sec / 60);
			const s = sec % 60;
			return `${min}:${s.toString().padStart(2, '0')}`;
		});

		chart = new Chart(canvasRef, {
			type: 'line',
			data: {
				labels,
				datasets: [
					{
						label: 'CPU %',
						data: data.map(d => d.cpu_percent),
						borderColor: '#58a6ff',
						backgroundColor: 'rgba(88, 166, 255, 0.05)',
						fill: true,
						tension: 0.2,
						borderWidth: 1.5,
						pointRadius: 0,
						pointHitRadius: 10,
						yAxisID: 'y',
					},
					{
						label: 'Memory (MB)',
						data: data.map(d => d.memory_mb),
						borderColor: '#bc8cff',
						backgroundColor: 'rgba(188, 140, 255, 0.05)',
						fill: true,
						tension: 0.2,
						borderWidth: 1.5,
						pointRadius: 0,
						pointHitRadius: 10,
						yAxisID: 'y1',
					},
				],
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				interaction: { mode: 'index', intersect: false },
				plugins: {
					legend: {
						position: 'bottom',
						labels: { boxWidth: 12, padding: 16, usePointStyle: true, pointStyle: 'line' },
					},
					tooltip: {
						backgroundColor: '#161b22',
						borderColor: '#30363d',
						borderWidth: 1,
						titleColor: '#c9d1d9',
						bodyColor: '#c9d1d9',
						callbacks: {
							title: (items: any[]) => {
								if (!items[0]) return '';
								const idx = items[0].dataIndex;
								const point = data[idx];
								return `${items[0].label} - ${point.step_name}`;
							},
						},
					},
				},
				scales: {
					y: {
						type: 'linear',
						position: 'left',
						min: 0,
						ticks: { callback: (v: number) => v + '%' },
						grid: { color: '#21262d' },
						title: { display: true, text: 'CPU %', color: '#58a6ff' },
					},
					y1: {
						type: 'linear',
						position: 'right',
						min: 0,
						grid: { display: false },
						ticks: { callback: (v: number) => v + 'MB' },
						title: { display: true, text: 'Memory', color: '#bc8cff' },
					},
					x: {
						grid: { display: false },
						title: { display: true, text: 'Elapsed Time' },
					},
				},
			},
		});
	};

	onCleanup(() => {
		chart?.destroy();
	});

	const formatMem = (mb: number): string => mb >= 1024 ? `${(mb / 1024).toFixed(1)} GB` : `${Math.round(mb)} MB`;

	const isEmpty = () => props.steps.length === 0;

	return (
		<div class="space-y-4">
			<Show when={isEmpty()}>
				<div class="flex items-center justify-center py-8 text-sm text-[var(--color-text-tertiary)]">
					No resource utilization data available for this run.
				</div>
			</Show>

			<Show when={!isEmpty()}>
				{/* Step resource sparklines */}
				<div class="grid grid-cols-1 gap-2">
					<For each={props.steps}>
						{(step) => {
							const cpuWidth = () => Math.min((step.max_cpu / maxCpu()) * 100, 100);
							const memWidth = () => Math.min((step.max_memory / maxMem()) * 100, 100);
							return (
								<div class="flex items-center gap-3 p-3 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
									<div class="flex-1 min-w-0">
										<div class="text-sm font-medium text-[var(--color-text-primary)] truncate">{step.step_name}</div>
										<div class="text-xs text-[var(--color-text-tertiary)] mt-0.5">
											{step.duration_ms ? `${Math.round(step.duration_ms / 1000)}s` : '-'}
										</div>
									</div>
									{/* CPU sparkline */}
									<div class="w-24">
										<div class="flex items-center justify-between mb-0.5">
											<span class="text-[10px] text-blue-400">CPU</span>
											<span class="text-[10px] text-[var(--color-text-tertiary)] tabular-nums">{step.max_cpu.toFixed(0)}%</span>
										</div>
										<div class="h-1 rounded-full bg-[var(--color-bg-tertiary)] overflow-hidden">
											<div class="h-full rounded-full bg-blue-500/60 transition-all" style={{ width: `${cpuWidth()}%` }} />
										</div>
									</div>
									{/* Memory sparkline */}
									<div class="w-24">
										<div class="flex items-center justify-between mb-0.5">
											<span class="text-[10px] text-violet-400">Mem</span>
											<span class="text-[10px] text-[var(--color-text-tertiary)] tabular-nums">{formatMem(step.max_memory)}</span>
										</div>
										<div class="h-1 rounded-full bg-[var(--color-bg-tertiary)] overflow-hidden">
											<div class="h-full rounded-full bg-violet-500/60 transition-all" style={{ width: `${memWidth()}%` }} />
										</div>
									</div>
								</div>
							);
						}}
					</For>
				</div>

				{/* Timeline chart */}
				<Show when={props.data.length > 0}>
					<div class="rounded-xl border border-[var(--color-border-primary)] overflow-hidden">
						<div class="px-3 py-2 bg-[#161b22] border-b border-[var(--color-border-primary)]">
							<span class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Resource Timeline</span>
						</div>
						<div style={{ height: '250px' }} class="bg-[#0d1117] p-2">
							<Show when={loaded()} fallback={
								<div class="flex items-center justify-center h-full text-sm text-[var(--color-text-tertiary)]">
									Loading chart...
								</div>
							}>
								<canvas ref={canvasRef} />
							</Show>
						</div>
					</div>
				</Show>
			</Show>
		</div>
	);
};

export default ResourceGraph;
