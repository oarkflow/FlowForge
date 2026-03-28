import type { JSX, Component } from 'solid-js';
import { splitProps, Show, For } from 'solid-js';

interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

interface SelectProps extends Omit<JSX.SelectHTMLAttributes<HTMLSelectElement>, 'children'> {
  label?: string;
  error?: string;
  options: SelectOption[];
  placeholder?: string;
}

const Select: Component<SelectProps> = (props) => {
  const [local, rest] = splitProps(props, [
    'label',
    'error',
    'options',
    'placeholder',
    'class',
    'id',
  ]);

  const selectId = () => local.id ?? local.label?.toLowerCase().replace(/\s+/g, '-');

  return (
    <div class={`flex flex-col gap-1.5 ${local.class ?? ''}`}>
      <Show when={local.label}>
        <label
          for={selectId()}
          class="text-sm font-medium text-[var(--color-text-secondary)]"
        >
          {local.label}
        </label>
      </Show>

      <div class="relative">
        <select
          {...rest}
          id={selectId()}
          class={`w-full rounded-lg border bg-[var(--color-bg-secondary)] text-[var(--color-text-primary)] transition-colors duration-150 focus:outline-none focus:ring-2 focus:ring-indigo-500/40 focus:border-[var(--color-border-focus)] appearance-none px-3 py-2 pr-9 text-sm cursor-pointer ${
            local.error
              ? 'border-red-500/60'
              : 'border-[var(--color-border-primary)]'
          }`}
        >
          <Show when={local.placeholder}>
            <option value="" disabled>
              {local.placeholder}
            </option>
          </Show>
          <For each={local.options}>
            {(opt) => (
              <option value={opt.value} disabled={opt.disabled}>
                {opt.label}
              </option>
            )}
          </For>
        </select>

        {/* Chevron icon */}
        <div class="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-[var(--color-text-tertiary)]">
          <svg
            class="w-4 h-4"
            viewBox="0 0 20 20"
            fill="currentColor"
          >
            <path
              fill-rule="evenodd"
              d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z"
              clip-rule="evenodd"
            />
          </svg>
        </div>
      </div>

      <Show when={local.error}>
        <p class="text-xs text-red-400">{local.error}</p>
      </Show>
    </div>
  );
};

export default Select;
