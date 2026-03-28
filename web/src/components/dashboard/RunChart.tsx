import { Component, For, createMemo, Show } from 'solid-js';

interface RunChartData {
  date: string;
  success: number;
  failure: number;
  total: number;
}

interface RunChartProps {
  data: RunChartData[];
  height?: number;
}

/**
 * RunChart - Simple SVG bar chart showing run trends per day.
 * Uses pure SVG to avoid Chart.js dependency.
 */
export const RunChart: Component<RunChartProps> = (props) => {
  const chartHeight = () => props.height || 200;
  const barWidth = 24;
  const gap = 8;
  const padding = { top: 20, right: 20, bottom: 40, left: 40 };

  const maxValue = createMemo(() => {
    if (props.data.length === 0) return 10;
    return Math.max(...props.data.map((d) => d.total), 1);
  });

  const chartWidth = createMemo(() => {
    return padding.left + padding.right + props.data.length * (barWidth + gap);
  });

  const scale = (value: number) => {
    const innerHeight = chartHeight() - padding.top - padding.bottom;
    return (value / maxValue()) * innerHeight;
  };

  const yTicks = createMemo(() => {
    const max = maxValue();
    const step = Math.max(1, Math.ceil(max / 5));
    const ticks: number[] = [];
    for (let i = 0; i <= max; i += step) {
      ticks.push(i);
    }
    return ticks;
  });

  return (
    <Show when={props.data.length > 0} fallback={
      <div class="flex items-center justify-center py-8 text-sm" style="color: var(--text-tertiary);">
        No run data available
      </div>
    }>
      <div class="overflow-x-auto">
        <svg
          width={chartWidth()}
          height={chartHeight()}
          class="w-full"
          viewBox={`0 0 ${chartWidth()} ${chartHeight()}`}
        >
          {/* Y-axis grid lines and labels */}
          <For each={yTicks()}>
            {(tick) => {
              const y = chartHeight() - padding.bottom - scale(tick);
              return (
                <>
                  <line
                    x1={padding.left}
                    y1={y}
                    x2={chartWidth() - padding.right}
                    y2={y}
                    stroke="var(--border-primary)"
                    stroke-dasharray="4,4"
                    opacity="0.5"
                  />
                  <text
                    x={padding.left - 8}
                    y={y + 4}
                    text-anchor="end"
                    fill="var(--text-tertiary)"
                    font-size="11"
                  >
                    {tick}
                  </text>
                </>
              );
            }}
          </For>

          {/* Bars */}
          <For each={props.data}>
            {(item, i) => {
              const x = padding.left + i() * (barWidth + gap);
              const baseY = chartHeight() - padding.bottom;
              const successHeight = scale(item.success);
              const failureHeight = scale(item.failure);

              return (
                <g>
                  {/* Success bar (bottom) */}
                  <rect
                    x={x}
                    y={baseY - successHeight}
                    width={barWidth}
                    height={Math.max(successHeight, 0)}
                    rx="3"
                    fill="#238636"
                    opacity="0.8"
                  >
                    <title>{`${item.date}: ${item.success} successful`}</title>
                  </rect>

                  {/* Failure bar (stacked on top) */}
                  <Show when={failureHeight > 0}>
                    <rect
                      x={x}
                      y={baseY - successHeight - failureHeight}
                      width={barWidth}
                      height={failureHeight}
                      rx="3"
                      fill="#da3633"
                      opacity="0.8"
                    >
                      <title>{`${item.date}: ${item.failure} failed`}</title>
                    </rect>
                  </Show>

                  {/* Date label */}
                  <text
                    x={x + barWidth / 2}
                    y={chartHeight() - padding.bottom + 16}
                    text-anchor="middle"
                    fill="var(--text-tertiary)"
                    font-size="10"
                    transform={`rotate(-45, ${x + barWidth / 2}, ${chartHeight() - padding.bottom + 16})`}
                  >
                    {item.date.slice(5)} {/* Show MM-DD */}
                  </text>
                </g>
              );
            }}
          </For>
        </svg>

        {/* Legend */}
        <div class="flex items-center justify-center gap-4 mt-2">
          <div class="flex items-center gap-1.5">
            <div class="w-3 h-3 rounded-sm" style="background: #238636;" />
            <span class="text-xs" style="color: var(--text-tertiary);">Success</span>
          </div>
          <div class="flex items-center gap-1.5">
            <div class="w-3 h-3 rounded-sm" style="background: #da3633;" />
            <span class="text-xs" style="color: var(--text-tertiary);">Failed</span>
          </div>
        </div>
      </div>
    </Show>
  );
};
