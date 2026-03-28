import type { Component } from 'solid-js';
import { createSignal, onMount, onCleanup, Show } from 'solid-js';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
interface AgentMetricPoint {
	timestamp: string;
	cpu_percent: number;
	memory_percent: number;
	queue_depth: number;
}

interface AgentUtilizationChartProps {
	data: AgentMetricPoint[];
	totalAgents?: number;
	onlineAgents?: number;
	busyAgents?: number;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const AgentUtilizationChart: Component<AgentUtilizationChartProps> = (props) => {
	let cpuCanvasRef: HTMLCanvasElement | undefined;
	let queueCanvasRef: HTMLCanvasElement | undefined;
	let cpuChart: any;
	let queueChart: any;
	const [loaded, setLoaded] = createSignal(false);
	const [activeView, setActiveView] = createSignal<'utilization' | 'queue'>('utilization');

	onMount(async () => {
		try {
			const { Chart, registerables } = await import('chart.js');
			Chart.register(...registerables);
			Chart.defaults.color = '#484f58';
			Chart.defaults.borderColor = '#21262d';
			Chart.defaults.font.family = "'Inter', system-ui, sans-serif";
			Chart.defaults.font.size = 11;

			setLoaded(true);
			requestAnimationFrame(() => createCharts(Chart));
		} catch {
			console.warn('Chart.js not available');
		}
	});

	const createCharts = (Chart: any) => {
		const data = props.data;
		if (!data || data.length === 0) return;

		const labels = data.map(d => {
			const date = new Date(d.timestamp);
			return `${date.getHours()}:${date.getMinutes().toString().padStart(2, '0')}`;
		});

		// CPU/Memory usage chart
		if (cpuCanvasRef) {
			cpuChart = new Chart(cpuCanvasRef, {
				type: 'line',
				data: {
					labels,
					datasets: [
						{
							label: 'CPU %',
							data: data.map(d => d.cpu_percent),
							borderColor: '#58a6ff',
							backgroundColor: 'rgba(88, 166, 255, 0.1)',
							fill: true,
							tension: 0.3,
							borderWidth: 2,
							pointRadius: 0,
							pointHitRadius: 10,
						},
						{
							label: 'Memory %',
							data: data.map(d => d.memory_percent),
							borderColor: '#bc8cff',
							backgroundColor: 'rgba(188, 140, 255, 0.1)',
							fill: true,
							tension: 0.3,
							borderWidth: 2,
							pointRadius: 0,
							pointHitRadius: 10,
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
						},
					},
					scales: {
						y: {
							min: 0,
							max: 100,
							ticks: { callback: (v: number) => v + '%' },
							grid: { color: '#21262d' },
						},
						x: { grid: { display: false } },
					},
				},
			});
		}

		// Queue depth chart
		if (queueCanvasRef) {
			queueChart = new Chart(queueCanvasRef, {
				type: 'bar',
				data: {
					labels,
					datasets: [{
						label: 'Queue Depth',
						data: data.map(d => d.queue_depth),
						backgroundColor: data.map(d =>
							d.queue_depth > 10 ? '#da3633' : d.queue_depth > 5 ? '#d29922' : '#238636'
						),
						borderRadius: 3,
						barPercentage: 0.6,
					}],
				},
				options: {
					responsive: true,
					maintainAspectRatio: false,
					plugins: {
						legend: { display: false },
						tooltip: {
							backgroundColor: '#161b22',
							borderColor: '#30363d',
							borderWidth: 1,
							titleColor: '#c9d1d9',
							bodyColor: '#c9d1d9',
						},
					},
					scales: {
						y: {
							beginAtZero: true,
							grid: { color: '#21262d' },
							ticks: { precision: 0 },
						},
						x: { grid: { display: false } },
					},
				},
			});
		}
	};

	onCleanup(() => {
		cpuChart?.destroy();
		queueChart?.destroy();
	});

	const isEmpty = () => !props.data || props.data.length === 0;

	return (
		<div class="space-y-4">
			{/* Agent summary stats */}
			<div class="grid grid-cols-3 gap-3">
				<div class="px-3 py-2 rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)]">
					<div class="text-xs text-[var(--color-text-tertiary)]">Total</div>
					<div class="text-lg font-bold text-[var(--color-text-primary)] tabular-nums">{props.totalAgents ?? 0}</div>
				</div>
				<div class="px-3 py-2 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
					<div class="text-xs text-emerald-400/80">Online</div>
					<div class="text-lg font-bold text-emerald-400 tabular-nums">{props.onlineAgents ?? 0}</div>
				</div>
				<div class="px-3 py-2 rounded-lg bg-violet-500/10 border border-violet-500/20">
					<div class="text-xs text-violet-400/80">Busy</div>
					<div class="text-lg font-bold text-violet-400 tabular-nums">{props.busyAgents ?? 0}</div>
				</div>
			</div>

			{/* Tab bar */}
			<div class="flex gap-1 bg-[var(--color-bg-tertiary)] p-1 rounded-lg w-fit">
				{(['utilization', 'queue'] as const).map(view => (
					<button
						class={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors cursor-pointer ${
							activeView() === view
								? 'bg-[var(--color-bg-secondary)] text-[var(--color-text-primary)] shadow-sm'
								: 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'
						}`}
						onClick={() => setActiveView(view)}
					>
						{view === 'utilization' ? 'CPU / Memory' : 'Queue Depth'}
					</button>
				))}
			</div>

			<Show when={isEmpty()}>
				<div class="flex items-center justify-center py-8 text-sm text-[var(--color-text-tertiary)]">
					No agent metrics available yet.
				</div>
			</Show>

			<Show when={!isEmpty()}>
				<div class={activeView() === 'utilization' ? '' : 'hidden'} style={{ height: '200px' }}>
					<canvas ref={cpuCanvasRef} />
				</div>
				<div class={activeView() === 'queue' ? '' : 'hidden'} style={{ height: '200px' }}>
					<canvas ref={queueCanvasRef} />
				</div>
			</Show>
		</div>
	);
};

export default AgentUtilizationChart;
