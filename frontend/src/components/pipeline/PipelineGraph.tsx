import { Component, For, Show, createMemo } from 'solid-js';

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
 */
export const PipelineGraph: Component<PipelineGraphProps> = (props) => {
  const sortedStages = createMemo(() =>
    [...props.stages].sort((a, b) => a.position - b.position)
  );

  const jobsByStage = createMemo(() => {
    const map = new Map<string, JobNode[]>();
    for (const job of props.jobs) {
      const existing = map.get(job.stageId) || [];
      existing.push(job);
      map.set(job.stageId, existing);
    }
    return map;
  });

  const getColors = (status: string) => statusColors[status] || statusColors.pending;

  return (
    <div class="overflow-x-auto py-4">
      <div class="flex items-start gap-2 min-w-max px-4">
        <For each={sortedStages()}>
          {(stage, i) => {
            const stageJobs = () => jobsByStage().get(stage.id) || [];
            const colors = () => getColors(stage.status);

            return (
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

                {/* Stage column */}
                <div class="flex flex-col gap-2 min-w-[180px]">
                  {/* Stage header */}
                  <div
                    class="text-xs font-medium px-3 py-1.5 rounded-md text-center uppercase tracking-wider"
                    style={`background: ${colors().bg}; border: 1px solid ${colors().border}; color: ${colors().text};`}
                  >
                    <span class="inline-block w-2 h-2 rounded-full mr-1.5" style={`background: ${colors().dot}; ${stage.status === 'running' ? 'animation: pulse-dot 1.5s infinite;' : ''}`} />
                    {stage.name}
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
              </>
            );
          }}
        </For>
      </div>
    </div>
  );
};
