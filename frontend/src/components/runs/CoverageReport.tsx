import type { Component } from 'solid-js';
import { createMemo, For, Show } from 'solid-js';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
export interface FileCoverage {
	file: string;
	lines: number;
	covered: number;
	percentage: number;
}

export interface CoverageData {
	totalPercentage: number;
	totalLines: number;
	coveredLines: number;
	threshold?: number;
	files: FileCoverage[];
}

interface CoverageReportProps {
	data?: CoverageData;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function getCoverageColor(percentage: number, threshold?: number): string {
	const t = threshold ?? 80;
	if (percentage >= t) return 'emerald';
	if (percentage >= t * 0.75) return 'amber';
	return 'red';
}

function getCoverageLabel(percentage: number, threshold?: number): string {
	const t = threshold ?? 80;
	if (percentage >= t) return 'Good';
	if (percentage >= t * 0.75) return 'Needs Work';
	return 'Below Threshold';
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const CoverageReport: Component<CoverageReportProps> = (props) => {
	const color = createMemo(() => getCoverageColor(props.data?.totalPercentage ?? 0, props.data?.threshold));
	const label = createMemo(() => getCoverageLabel(props.data?.totalPercentage ?? 0, props.data?.threshold));
	const threshold = () => props.data?.threshold ?? 80;

	const sortedFiles = createMemo(() => {
		if (!props.data?.files) return [];
		return [...props.data.files].sort((a, b) => a.percentage - b.percentage);
	});

	return (
		<Show when={props.data} fallback={
			<div class="flex items-center justify-center py-8 text-sm text-[var(--color-text-tertiary)]">
				No coverage data available.
			</div>
		}>
			{(data) => (
				<div class="space-y-4">
					{/* Overall coverage */}
					<div class="flex items-center gap-6 p-4 rounded-xl bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
						{/* Circular progress indicator */}
						<div class="relative w-24 h-24 flex-shrink-0">
							<svg class="w-24 h-24 -rotate-90" viewBox="0 0 100 100">
								{/* Background circle */}
								<circle
									cx="50" cy="50" r="42"
									fill="none" stroke="#21262d" stroke-width="8"
								/>
								{/* Progress circle */}
								<circle
									cx="50" cy="50" r="42"
									fill="none"
									stroke={color() === 'emerald' ? '#3fb950' : color() === 'amber' ? '#d29922' : '#da3633'}
									stroke-width="8"
									stroke-linecap="round"
									stroke-dasharray={`${(data().totalPercentage / 100) * 264} 264`}
									class="transition-all duration-500"
								/>
								{/* Threshold marker */}
								<circle
									cx="50" cy="50" r="42"
									fill="none"
									stroke="#484f58"
									stroke-width="1"
									stroke-dasharray={`1 ${264 - 1}`}
									stroke-dashoffset={`${-(threshold() / 100) * 264}`}
								/>
							</svg>
							<div class="absolute inset-0 flex flex-col items-center justify-center">
								<span class={`text-xl font-bold tabular-nums text-${color()}-400`}>
									{data().totalPercentage.toFixed(1)}%
								</span>
							</div>
						</div>

						{/* Stats */}
						<div class="flex-1">
							<div class="flex items-center gap-2 mb-2">
								<span class={`text-sm font-medium text-${color()}-400`}>{label()}</span>
								<span class="text-xs text-[var(--color-text-tertiary)]">
									(threshold: {threshold()}%)
								</span>
							</div>
							<div class="grid grid-cols-2 gap-3">
								<div>
									<div class="text-xs text-[var(--color-text-tertiary)]">Covered Lines</div>
									<div class="text-sm font-medium text-[var(--color-text-primary)] tabular-nums">
										{data().coveredLines.toLocaleString()} / {data().totalLines.toLocaleString()}
									</div>
								</div>
								<div>
									<div class="text-xs text-[var(--color-text-tertiary)]">Files</div>
									<div class="text-sm font-medium text-[var(--color-text-primary)] tabular-nums">
										{data().files.length}
									</div>
								</div>
							</div>

							{/* Overall bar */}
							<div class="mt-3 relative">
								<div class="h-2 rounded-full bg-[var(--color-bg-tertiary)] overflow-hidden">
									<div
										class={`h-full rounded-full transition-all duration-500 bg-${color()}-500`}
										style={{ width: `${data().totalPercentage}%` }}
									/>
								</div>
								{/* Threshold indicator */}
								<div
									class="absolute top-0 w-0.5 h-2 bg-[var(--color-text-tertiary)]"
									style={{ left: `${threshold()}%` }}
									title={`Threshold: ${threshold()}%`}
								/>
							</div>
						</div>
					</div>

					{/* File-level breakdown */}
					<Show when={sortedFiles().length > 0}>
						<div class="rounded-xl border border-[var(--color-border-primary)] overflow-hidden">
							<div class="px-4 py-3 bg-[var(--color-bg-secondary)] border-b border-[var(--color-border-primary)]">
								<h4 class="text-sm font-semibold text-[var(--color-text-primary)]">File Coverage</h4>
							</div>
							<div class="divide-y divide-[var(--color-border-primary)]">
								<For each={sortedFiles()}>
									{(file) => {
										const fileColor = getCoverageColor(file.percentage, threshold());
										return (
											<div class="flex items-center gap-3 px-4 py-2.5 hover:bg-[var(--color-bg-hover)] transition-colors">
												<div class="flex-1 min-w-0">
													<span class="text-sm text-[var(--color-text-primary)] font-mono truncate block">{file.file}</span>
												</div>
												<div class="flex items-center gap-3 flex-shrink-0">
													<div class="w-24 h-1.5 rounded-full bg-[var(--color-bg-tertiary)] overflow-hidden">
														<div
															class={`h-full rounded-full bg-${fileColor}-500`}
															style={{ width: `${file.percentage}%` }}
														/>
													</div>
													<span class={`text-xs tabular-nums font-medium w-12 text-right text-${fileColor}-400`}>
														{file.percentage.toFixed(1)}%
													</span>
													<span class="text-xs tabular-nums text-[var(--color-text-tertiary)] w-20 text-right">
														{file.covered}/{file.lines}
													</span>
												</div>
											</div>
										);
									}}
								</For>
							</div>
						</div>
					</Show>
				</div>
			)}
		</Show>
	);
};

export default CoverageReport;
