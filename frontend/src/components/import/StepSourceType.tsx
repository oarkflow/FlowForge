import type { Component } from 'solid-js';
import type { SourceType } from './ImportWizard';

interface Props {
  onSelect: (type: SourceType) => void;
}

const sources: { type: SourceType; label: string; description: string; icon: string }[] = [
  {
    type: 'git',
    label: 'Git URL',
    description: 'Clone from any HTTPS or SSH Git URL',
    icon: 'M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.166 6.839 9.489.5.092.682-.217.682-.482 0-.237-.009-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.115 2.504.337 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.579.688.481C19.138 20.161 22 16.416 22 12c0-5.523-4.477-10-10-10z',
  },
  {
    type: 'github',
    label: 'GitHub',
    description: 'Browse and import from GitHub',
    icon: 'M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.166 6.839 9.489.5.092.682-.217.682-.482 0-.237-.009-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.115 2.504.337 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.579.688.481C19.138 20.161 22 16.416 22 12c0-5.523-4.477-10-10-10z',
  },
  {
    type: 'gitlab',
    label: 'GitLab',
    description: 'Browse and import from GitLab',
    icon: 'M21.94 12.865l-1.066-3.28-2.12-6.52a.443.443 0 00-.842 0l-2.12 6.52H8.208l-2.12-6.52a.443.443 0 00-.842 0l-2.12 6.52-1.066 3.28a.87.87 0 00.316.973l9.624 6.993 9.624-6.993a.87.87 0 00.316-.973z',
  },
  {
    type: 'bitbucket',
    label: 'Bitbucket',
    description: 'Browse and import from Bitbucket',
    icon: 'M2.65 3C2.29 3 2 3.31 2 3.67l.02.12 2.73 16.5c.07.42.44.73.87.73h13.05c.32 0 .6-.24.65-.56L22 3.79l-.01-.12c0-.36-.29-.67-.65-.67H2.65zm10.08 12.51H11.3L9.94 9.5h4.14l-1.35 5.99z',
  },
  {
    type: 'local',
    label: 'Local Directory',
    description: 'Point to a directory on the server',
    icon: 'M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z',
  },
  {
    type: 'upload',
    label: 'Upload Archive',
    description: 'Upload a .zip or .tar.gz file',
    icon: 'M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12',
  },
];

const StepSourceType: Component<Props> = (props) => {
  return (
    <div>
      <h2 class="text-xl font-semibold text-[var(--color-text-primary)] mb-2">
        Choose a Source
      </h2>
      <p class="text-sm text-[var(--color-text-secondary)] mb-6">
        How would you like to import your project?
      </p>

      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {sources.map((source) => (
          <button
            type="button"
            onClick={() => props.onSelect(source.type)}
            class="group text-left p-5 rounded-xl border border-[var(--color-border-primary)] bg-[var(--color-bg-secondary)] hover:border-indigo-500/50 hover:bg-[var(--color-bg-tertiary)]/50 transition-all"
          >
            <div class="w-10 h-10 rounded-lg bg-[var(--color-bg-tertiary)] flex items-center justify-center mb-3 group-hover:bg-indigo-500/10 transition-colors">
              <svg
                class="w-5 h-5 text-[var(--color-text-secondary)] group-hover:text-indigo-400 transition-colors"
                viewBox="0 0 24 24"
                fill="currentColor"
              >
                <path d={source.icon} />
              </svg>
            </div>
            <h3 class="text-sm font-semibold text-[var(--color-text-primary)] mb-1 group-hover:text-indigo-400 transition-colors">
              {source.label}
            </h3>
            <p class="text-xs text-[var(--color-text-tertiary)]">
              {source.description}
            </p>
          </button>
        ))}
      </div>
    </div>
  );
};

export default StepSourceType;
