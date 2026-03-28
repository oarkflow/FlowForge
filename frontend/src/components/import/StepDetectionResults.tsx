import type { Component } from 'solid-js';
import { For, Show, createMemo } from 'solid-js';
import Button from '../ui/Button';
import Badge from '../ui/Badge';
import type { DetectionResult, ExtractedVariable } from '../../types';

interface Props {
  detections: DetectionResult[];
  generatedPipeline: string;
  editedPipeline: string;
  onEditPipeline: (yaml: string) => void;
  extractedEnvVars: ExtractedVariable[];
  extractedSecrets: ExtractedVariable[];
  onNext: () => void;
  onBack: () => void;
}

function confidenceVariant(c: number): 'success' | 'warning' | 'error' {
  if (c >= 0.8) return 'success';
  if (c >= 0.5) return 'warning';
  return 'error';
}

function confidenceLabel(c: number): string {
  return `${Math.round(c * 100)}%`;
}

const StepDetectionResults: Component<Props> = (props) => {
  const needsConfigEnvVars = createMemo(() =>
    props.extractedEnvVars.filter(v => !v.has_value)
  );

  const hasRequiredConfig = createMemo(() =>
    props.extractedSecrets.length > 0 || needsConfigEnvVars().length > 0
  );

  return (
    <div>
      <h2 class="text-xl font-semibold text-[var(--color-text-primary)] mb-2">
        Stack Detection
      </h2>
      <p class="text-sm text-[var(--color-text-secondary)] mb-6">
        Review detected technologies and the generated pipeline.
      </p>

      {/* Detection results */}
      <div class="mb-6">
        <h3 class="text-sm font-medium text-[var(--color-text-primary)] mb-3">
          Detected Technologies
        </h3>
        <Show when={props.detections.length > 0} fallback={
          <div class="p-4 rounded-xl bg-yellow-500/10 border border-yellow-500/30">
            <p class="text-sm text-yellow-400">
              No technologies were detected. A generic pipeline has been generated.
            </p>
          </div>
        }>
          <div class="space-y-2">
            <For each={props.detections}>{(d) => (
              <div class="flex items-center justify-between p-3 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
                <div class="flex items-center gap-3">
                  <div class="w-8 h-8 rounded-lg bg-[var(--color-bg-tertiary)] flex items-center justify-center">
                    <span class="text-xs font-bold text-[var(--color-text-secondary)]">
                      {d.language.slice(0, 2).toUpperCase()}
                    </span>
                  </div>
                  <div>
                    <span class="text-sm font-medium text-[var(--color-text-primary)]">
                      {d.language}
                    </span>
                    <Show when={d.framework}>
                      <span class="text-sm text-[var(--color-text-tertiary)]"> / {d.framework}</span>
                    </Show>
                    <Show when={d.build_tool || d.runtime_version}>
                      <p class="text-xs text-[var(--color-text-tertiary)]">
                        {[d.build_tool, d.runtime_version].filter(Boolean).join(' · ')}
                      </p>
                    </Show>
                  </div>
                </div>
                <Badge variant={confidenceVariant(d.confidence)} size="sm">
                  {confidenceLabel(d.confidence)}
                </Badge>
              </div>
            )}</For>
          </div>
        </Show>
      </div>

      {/* Pipeline editor */}
      <div class="mb-6">
        <h3 class="text-sm font-medium text-[var(--color-text-primary)] mb-3">
          Generated Pipeline
        </h3>
        <p class="text-xs text-[var(--color-text-tertiary)] mb-2">
          Review and edit the auto-generated pipeline configuration.
        </p>
        <textarea
          class="w-full h-80 px-4 py-3 rounded-xl bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] font-mono resize-y focus:outline-none focus:border-indigo-500 leading-relaxed"
          value={props.editedPipeline}
          onInput={(e) => props.onEditPipeline(e.currentTarget.value)}
          spellcheck={false}
        />
      </div>

      {/* Required Configuration */}
      <Show when={hasRequiredConfig()}>
        <div class="mb-6 p-4 rounded-xl bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
          <h3 class="text-sm font-medium text-[var(--color-text-primary)] mb-3">
            Required Configuration
          </h3>
          <p class="text-xs text-[var(--color-text-tertiary)] mb-3">
            Your pipeline references variables and secrets that will need to be configured after project creation.
          </p>

          <Show when={props.extractedSecrets.length > 0}>
            <div class="mb-3">
              <p class="text-xs font-medium text-amber-400 mb-2">
                Secrets ({props.extractedSecrets.length})
              </p>
              <div class="flex flex-wrap gap-2">
                <For each={props.extractedSecrets}>{(s) => (
                  <span class="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-mono bg-amber-500/10 text-amber-400 border border-amber-500/20">
                    {s.name}
                  </span>
                )}</For>
              </div>
            </div>
          </Show>

          <Show when={needsConfigEnvVars().length > 0}>
            <div>
              <p class="text-xs font-medium text-blue-400 mb-2">
                Environment Variables ({needsConfigEnvVars().length})
              </p>
              <div class="flex flex-wrap gap-2">
                <For each={needsConfigEnvVars()}>{(v) => (
                  <span class="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-mono bg-blue-500/10 text-blue-400 border border-blue-500/20">
                    {v.name}
                  </span>
                )}</For>
              </div>
            </div>
          </Show>
        </div>
      </Show>

      <div class="flex justify-between">
        <Button variant="ghost" onClick={props.onBack}>Back</Button>
        <Button onClick={props.onNext}>Continue</Button>
      </div>
    </div>
  );
};

export default StepDetectionResults;
