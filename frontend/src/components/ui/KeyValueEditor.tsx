import type { Component } from 'solid-js';
import { createSignal, For, Show } from 'solid-js';

export interface KeyValuePair {
  id?: string;
  key: string;
  value: string;
  isNew?: boolean; // true for rows not yet saved
}

interface KeyValueEditorProps {
  items: KeyValuePair[];
  onChange: (items: KeyValuePair[]) => void;
  secretMode?: boolean; // masks values, shows password inputs
  keyPlaceholder?: string;
  valuePlaceholder?: string;
  readOnlyKeys?: boolean; // prevent editing keys of existing items
}

const KeyValueEditor: Component<KeyValueEditorProps> = (props) => {
  const [editingValue, setEditingValue] = createSignal<string | null>(null);

  const addRow = () => {
    props.onChange([...props.items, { key: '', value: '', isNew: true }]);
  };

  const updateKey = (index: number, key: string) => {
    const updated = [...props.items];
    updated[index] = { ...updated[index], key: key.toUpperCase().replace(/[^A-Z0-9_]/g, '') };
    props.onChange(updated);
  };

  const updateValue = (index: number, value: string) => {
    const updated = [...props.items];
    updated[index] = { ...updated[index], value };
    props.onChange(updated);
  };

  const removeRow = (index: number) => {
    const updated = props.items.filter((_, i) => i !== index);
    props.onChange(updated);
  };

  return (
    <div class="space-y-2">
      {/* Header */}
      <Show when={props.items.length > 0}>
        <div class="grid gap-3 px-1 pb-1" style={{ 'grid-template-columns': '1fr 1fr 36px' }}>
          <span class="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)]">Key</span>
          <span class="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)]">Value</span>
          <span />
        </div>
      </Show>

      {/* Rows */}
      <For each={props.items}>
        {(item, index) => (
          <div class="grid gap-3 items-center" style={{ 'grid-template-columns': '1fr 1fr 36px' }}>
            <input
              type="text"
              value={item.key}
              onInput={(e) => updateKey(index(), e.currentTarget.value)}
              placeholder={props.keyPlaceholder ?? 'KEY_NAME'}
              readOnly={props.readOnlyKeys && !item.isNew && !!item.id}
              class="w-full px-3 py-2 text-sm font-mono bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] rounded-lg text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500/50"
            />
            <Show when={props.secretMode && !item.isNew && item.id && editingValue() !== item.id}>
              <div
                class="w-full px-3 py-2 text-sm font-mono bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] rounded-lg text-[var(--color-text-tertiary)] cursor-pointer hover:border-indigo-500/50 transition-colors"
                onClick={() => setEditingValue(item.id!)}
                title="Click to edit value"
              >
                ••••••••
              </div>
            </Show>
            <Show when={!props.secretMode || item.isNew || !item.id || editingValue() === item.id}>
              <input
                type={props.secretMode ? 'password' : 'text'}
                value={item.value}
                onInput={(e) => updateValue(index(), e.currentTarget.value)}
                onBlur={() => { if (props.secretMode) setEditingValue(null); }}
                placeholder={props.valuePlaceholder ?? 'value'}
                class="w-full px-3 py-2 text-sm font-mono bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] rounded-lg text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500/50"
              />
            </Show>
            <button
              type="button"
              onClick={() => removeRow(index())}
              class="w-9 h-9 flex items-center justify-center rounded-lg text-[var(--color-text-tertiary)] hover:text-red-400 hover:bg-red-500/10 transition-colors"
              title="Remove"
            >
              <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
              </svg>
            </button>
          </div>
        )}
      </For>

      {/* Add button */}
      <button
        type="button"
        onClick={addRow}
        class="flex items-center gap-2 px-3 py-2 text-sm text-indigo-400 hover:text-indigo-300 hover:bg-indigo-500/10 rounded-lg transition-colors"
      >
        <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
          <path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" />
        </svg>
        Add {props.secretMode ? 'Secret' : 'Variable'}
      </button>

      <Show when={props.items.length === 0}>
        <p class="text-sm text-[var(--color-text-tertiary)] py-4 text-center">
          No {props.secretMode ? 'secrets' : 'environment variables'} configured. Click the button above to add one.
        </p>
      </Show>
    </div>
  );
};

export default KeyValueEditor;
