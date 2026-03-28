import type { Component } from 'solid-js';
import { Show } from 'solid-js';
import type { SetStoreFunction } from 'solid-js/store';
import Button from '../ui/Button';
import Input from '../ui/Input';
import Select from '../ui/Select';
import type { WizardState } from './ImportWizard';

interface Props {
  state: WizardState;
  setState: SetStoreFunction<WizardState>;
  onCreate: () => void;
  onBack: () => void;
}

const StepProjectCreate: Component<Props> = (props) => {
  const updateName = (name: string) => {
    props.setState({
      projectName: name,
      projectSlug: name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, ''),
    });
  };

  const canCreate = () => !!props.state.projectName.trim();

  return (
    <div>
      <h2 class="text-xl font-semibold text-[var(--color-text-primary)] mb-2">
        Create Project
      </h2>
      <p class="text-sm text-[var(--color-text-secondary)] mb-6">
        Configure your new project and confirm creation.
      </p>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Form */}
        <div class="space-y-4">
          <Input
            label="Project Name"
            placeholder="My Awesome Project"
            value={props.state.projectName}
            onInput={(e) => updateName(e.currentTarget.value)}
          />
          <Show when={props.state.projectName}>
            <p class="text-xs text-[var(--color-text-tertiary)] -mt-2">
              Slug: <span class="font-mono">{props.state.projectSlug}</span>
            </p>
          </Show>

          <Input
            label="Description"
            placeholder="Brief description of the project..."
            value={props.state.projectDescription}
            onInput={(e) => props.setState('projectDescription', e.currentTarget.value)}
          />

          <Select
            label="Visibility"
            value={props.state.visibility}
            onChange={(e) => props.setState('visibility', e.currentTarget.value as WizardState['visibility'])}
            options={[
              { value: 'private', label: 'Private — Only team members' },
              { value: 'internal', label: 'Internal — Organization members' },
              { value: 'public', label: 'Public — Everyone' },
            ]}
          />

          <Show when={props.state.sourceType === 'github' || props.state.sourceType === 'gitlab' || props.state.sourceType === 'bitbucket'}>
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={props.state.setupWebhook}
                onChange={(e) => props.setState('setupWebhook', e.currentTarget.checked)}
                class="w-4 h-4 rounded border-[var(--color-border-primary)] bg-[var(--color-bg-tertiary)] text-indigo-600 focus:ring-indigo-500"
              />
              <span class="text-sm text-[var(--color-text-secondary)]">
                Set up webhook for automatic triggers
              </span>
            </label>
          </Show>
        </div>

        {/* Summary */}
        <div class="p-5 rounded-xl bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
          <h3 class="text-sm font-medium text-[var(--color-text-primary)] mb-4">Summary</h3>
          <dl class="space-y-3 text-sm">
            <div>
              <dt class="text-[var(--color-text-tertiary)]">Source</dt>
              <dd class="text-[var(--color-text-primary)] font-medium capitalize">{props.state.sourceType}</dd>
            </div>
            <Show when={props.state.selectedRepo}>
              <div>
                <dt class="text-[var(--color-text-tertiary)]">Repository</dt>
                <dd class="text-[var(--color-text-primary)] font-mono text-xs">{props.state.selectedRepo!.full_name}</dd>
              </div>
            </Show>
            <Show when={props.state.gitUrl}>
              <div>
                <dt class="text-[var(--color-text-tertiary)]">Git URL</dt>
                <dd class="text-[var(--color-text-primary)] font-mono text-xs break-all">{props.state.gitUrl}</dd>
              </div>
            </Show>
            <Show when={props.state.defaultBranch}>
              <div>
                <dt class="text-[var(--color-text-tertiary)]">Branch</dt>
                <dd class="text-[var(--color-text-primary)]">{props.state.defaultBranch}</dd>
              </div>
            </Show>
            <Show when={props.state.detections.length > 0}>
              <div>
                <dt class="text-[var(--color-text-tertiary)]">Detected Stack</dt>
                <dd class="text-[var(--color-text-primary)]">
                  {props.state.detections.map(d =>
                    d.framework ? `${d.language}/${d.framework}` : d.language
                  ).join(', ')}
                </dd>
              </div>
            </Show>
            <Show when={props.state.editedPipeline}>
              <div>
                <dt class="text-[var(--color-text-tertiary)]">Pipeline</dt>
                <dd class="text-green-400 text-xs">Auto-generated pipeline will be created</dd>
              </div>
            </Show>
          </dl>
        </div>
      </div>

      <div class="flex justify-between mt-6">
        <Button variant="ghost" onClick={props.onBack}>Back</Button>
        <Button
          onClick={props.onCreate}
          loading={props.state.loading}
          disabled={!canCreate() || props.state.loading}
        >
          Create Project
        </Button>
      </div>
    </div>
  );
};

export default StepProjectCreate;
