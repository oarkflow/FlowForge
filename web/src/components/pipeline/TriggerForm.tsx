import { Component, createSignal, For, Show } from 'solid-js';

interface TriggerFormProps {
  onTrigger: (params: { branch: string; env: Record<string, string> }) => void;
  loading?: boolean;
  defaultBranch?: string;
}

/**
 * TriggerForm - Form for manually triggering a pipeline run.
 */
export const TriggerForm: Component<TriggerFormProps> = (props) => {
  const [branch, setBranch] = createSignal(props.defaultBranch || 'main');
  const [envVars, setEnvVars] = createSignal<{ key: string; value: string }[]>([]);

  const addEnvVar = () => {
    setEnvVars([...envVars(), { key: '', value: '' }]);
  };

  const removeEnvVar = (index: number) => {
    setEnvVars(envVars().filter((_, i) => i !== index));
  };

  const updateEnvVar = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...envVars()];
    updated[index] = { ...updated[index], [field]: value };
    setEnvVars(updated);
  };

  const handleSubmit = (e: Event) => {
    e.preventDefault();
    const env: Record<string, string> = {};
    for (const v of envVars()) {
      if (v.key.trim()) {
        env[v.key.trim()] = v.value;
      }
    }
    props.onTrigger({ branch: branch(), env });
  };

  return (
    <form onSubmit={handleSubmit} class="space-y-4">
      <div>
        <label class="block text-sm font-medium mb-1" style="color: var(--text-secondary);">
          Branch / Tag
        </label>
        <input
          type="text"
          value={branch()}
          onInput={(e) => setBranch(e.currentTarget.value)}
          class="w-full px-3 py-2 rounded-md text-sm"
          style="background: var(--bg-tertiary); border: 1px solid var(--border-primary); color: var(--text-primary);"
          placeholder="main"
        />
      </div>

      <div>
        <div class="flex items-center justify-between mb-2">
          <label class="text-sm font-medium" style="color: var(--text-secondary);">
            Environment Variables
          </label>
          <button
            type="button"
            onClick={addEnvVar}
            class="text-xs px-2 py-1 rounded"
            style="background: var(--bg-tertiary); color: var(--accent-primary);"
          >
            + Add Variable
          </button>
        </div>

        <For each={envVars()}>
          {(envVar, i) => (
            <div class="flex gap-2 mb-2">
              <input
                type="text"
                value={envVar.key}
                onInput={(e) => updateEnvVar(i(), 'key', e.currentTarget.value)}
                class="flex-1 px-3 py-2 rounded-md text-sm"
                style="background: var(--bg-tertiary); border: 1px solid var(--border-primary); color: var(--text-primary);"
                placeholder="KEY"
              />
              <input
                type="text"
                value={envVar.value}
                onInput={(e) => updateEnvVar(i(), 'value', e.currentTarget.value)}
                class="flex-1 px-3 py-2 rounded-md text-sm"
                style="background: var(--bg-tertiary); border: 1px solid var(--border-primary); color: var(--text-primary);"
                placeholder="value"
              />
              <button
                type="button"
                onClick={() => removeEnvVar(i())}
                class="px-2 text-red-400 hover:text-red-300"
              >
                ×
              </button>
            </div>
          )}
        </For>
      </div>

      <button
        type="submit"
        disabled={props.loading}
        class="w-full py-2 rounded-md text-sm font-medium transition-colors"
        style="background: var(--accent-primary); color: white;"
      >
        {props.loading ? 'Triggering...' : 'Trigger Pipeline'}
      </button>
    </form>
  );
};
