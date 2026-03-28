import type { Component } from 'solid-js';
import { createSignal, createEffect, onMount, onCleanup, Show } from 'solid-js';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
interface DayData {
	date: string;
	success: number;
	failure: number;
	cancelled: number;
}

interface PipelineHealthChartProps {
	data: DayData[];
}

// ---------------------------------------------------------------------------
// Component - Uses Chart.js loaded dynamically
// ---------------------------------------------------------------------------
const PipelineHealthChart: Component<PipelineHealthChartProps> = (props) => {
	let lineCanvasRef: HTMLCanvasElement | undefined;
	let barCanvasRef: HTMLCanvasElement | undefined;
	let pieCanvasRef: HTMLCanvasElement | undefined;
	let lineChart: any;
	let barChart: any;
	let pieChart: any;
	const [loaded, setLoaded] = createSignal(false);
	const [activeView, setActiveView] = createSignal<'rate' | 'duration' | 'distribution'>('rate');

	onMount(async () => {
		try {
			const { Chart, registerables } = await import('chart.js');
			Chart.register(...registerables);

			// Common chart defaults
			Chart.defaults.color = '#484f58';
			Chart.defaults.borderColor = '#21262d';
			Chart.defaults.font.family = "'Inter', system-ui, sans-serif";
			Chart.defaults.font.size = 11;

			setLoaded(true);

			// We need to wait a tick for the canvas refs to be available
			requestAnimationFrame(() => {
				createCharts(Chart);
			});
		} catch (err) {
			console.warn('Chart.js not available:', err);
		}
	});

	const createCharts = (Chart: any) => {
		const data = props.data;
		if (!data || data.length === 0) return;

		const labels = data.map(d => d.date.slice(5)); // MM-DD

		// Line chart - Pass/Fail rate over time
		if (lineCanvasRef) {
			const successRates = data.map(d => {
				const total = d.success + d.failure + d.cancelled;
				return total > 0 ? Math.round((d.success / total) * 100) : 0;
			});

			lineChart = new Chart(lineCanvasRef, {
				type: 'line',
				data: {
					labels,
					datasets: [{
						label: 'Success Rate %',
						data: successRates,
						borderColor: '#3fb950',
						backgroundColor: 'rgba(63, 185, 80, 0.1)',
						fill: true,
						tension: 0.3,
						borderWidth: 2,
						pointRadius: 3,
						pointBackgroundColor: '#3fb950',
						pointBorderColor: '#0d1117',
						pointBorderWidth: 2,
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
							padding: 8,
						},
					},
					scales: {
						y: {
							min: 0,
							max: 100,
							ticks: { callback: (v: number) => v + '%' },
							grid: { color: '#21262d' },
						},
						x: {
							grid: { display: false },
						},
					},
				},
			});
		}

		// Bar chart - Build counts
		if (barCanvasRef) {
			barChart = new Chart(barCanvasRef, {
				type: 'bar',
				data: {
					labels,
					datasets: [
						{
							label: 'Success',
							data: data.map(d => d.success),
							backgroundColor: '#238636',
							borderRadius: 3,
						},
						{
							label: 'Failed',
							data: data.map(d => d.failure),
							backgroundColor: '#da3633',
							borderRadius: 3,
						},
						{
							label: 'Cancelled',
							data: data.map(d => d.cancelled),
							backgroundColor: '#d29922',
							borderRadius: 3,
						},
					],
				},
				options: {
					responsive: true,
					maintainAspectRatio: false,
					plugins: {
						legend: {
							position: 'bottom',
							labels: { boxWidth: 12, padding: 16, usePointStyle: true, pointStyle: 'rectRounded' },
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
							stacked: true,
							beginAtZero: true,
							grid: { color: '#21262d' },
							ticks: { precision: 0 },
						},
						x: {
							stacked: true,
							grid: { display: false },
						},
					},
				},
			});
		}

		// Pie chart - Distribution
		if (pieCanvasRef) {
			const totals = data.reduce((acc, d) => ({
				success: acc.success + d.success,
				failure: acc.failure + d.failure,
				cancelled: acc.cancelled + d.cancelled,
			}), { success: 0, failure: 0, cancelled: 0 });

			pieChart = new Chart(pieCanvasRef, {
				type: 'doughnut',
				data: {
					labels: ['Success', 'Failed', 'Cancelled'],
					datasets: [{
						data: [totals.success, totals.failure, totals.cancelled],
						backgroundColor: ['#238636', '#da3633', '#d29922'],
						borderColor: '#0d1117',
						borderWidth: 2,
					}],
				},
				options: {
					responsive: true,
					maintainAspectRatio: false,
					cutout: '65%',
					plugins: {
						legend: {
							position: 'bottom',
							labels: { boxWidth: 12, padding: 16, usePointStyle: true, pointStyle: 'circle' },
						},
						tooltip: {
							backgroundColor: '#161b22',
							borderColor: '#30363d',
							borderWidth: 1,
							titleColor: '#c9d1d9',
							bodyColor: '#c9d1d9',
						},
					},
				},
			});
		}
	};

	// Update charts when data changes
	createEffect(() => {
		const data = props.data;
		if (!loaded() || !data || data.length === 0) return;

		const labels = data.map(d => d.date.slice(5));

		if (lineChart) {
			const successRates = data.map(d => {
				const total = d.success + d.failure + d.cancelled;
				return total > 0 ? Math.round((d.success / total) * 100) : 0;
			});
			lineChart.data.labels = labels;
			lineChart.data.datasets[0].data = successRates;
			lineChart.update('none');
		}

		if (barChart) {
			barChart.data.labels = labels;
			barChart.data.datasets[0].data = data.map(d => d.success);
			barChart.data.datasets[1].data = data.map(d => d.failure);
			barChart.data.datasets[2].data = data.map(d => d.cancelled);
			barChart.update('none');
		}

		if (pieChart) {
			const totals = data.reduce((acc, d) => ({
				success: acc.success + d.success,
				failure: acc.failure + d.failure,
				cancelled: acc.cancelled + d.cancelled,
			}), { success: 0, failure: 0, cancelled: 0 });
			pieChart.data.datasets[0].data = [totals.success, totals.failure, totals.cancelled];
			pieChart.update('none');
		}
	});

	onCleanup(() => {
		lineChart?.destroy();
		barChart?.destroy();
		pieChart?.destroy();
	});

	const isEmpty = () => !props.data || props.data.length === 0;

	return (
		<div class="space-y-4">
			{/* Tab bar */}
			<div class="flex gap-1 bg-[var(--color-bg-tertiary)] p-1 rounded-lg w-fit">
				{(['rate', 'duration', 'distribution'] as const).map(view => (
					<button
						class={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors cursor-pointer ${
							activeView() === view
								? 'bg-[var(--color-bg-secondary)] text-[var(--color-text-primary)] shadow-sm'
								: 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'
						}`}
						onClick={() => setActiveView(view)}
					>
						{view === 'rate' ? 'Success Rate' : view === 'duration' ? 'Build Counts' : 'Distribution'}
					</button>
				))}
			</div>

			{/* Charts */}
			<Show when={isEmpty()}>
				<div class="flex items-center justify-center py-12 text-sm text-[var(--color-text-tertiary)]">
					No build data available for the last 30 days.
				</div>
			</Show>

			<Show when={!isEmpty()}>
				{/* Success Rate */}
				<div class={activeView() === 'rate' ? '' : 'hidden'} style={{ height: '220px' }}>
					<canvas ref={lineCanvasRef} />
				</div>

				{/* Build Counts */}
				<div class={activeView() === 'duration' ? '' : 'hidden'} style={{ height: '220px' }}>
					<canvas ref={barCanvasRef} />
				</div>

				{/* Distribution */}
				<div class={activeView() === 'distribution' ? '' : 'hidden'} style={{ height: '220px' }}>
					<canvas ref={pieCanvasRef} />
				</div>
			</Show>
		</div>
	);
};

export default PipelineHealthChart;
