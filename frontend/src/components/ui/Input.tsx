import type { JSX, Component } from 'solid-js';
import { splitProps, Show } from 'solid-js';

interface InputProps extends JSX.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  hint?: string;
  icon?: JSX.Element;
  rightIcon?: JSX.Element;
}

const Input: Component<InputProps> = (props) => {
  const [local, rest] = splitProps(props, [
    'label',
    'error',
    'hint',
    'icon',
    'rightIcon',
    'class',
    'id',
  ]);

  const inputId = () => local.id ?? local.label?.toLowerCase().replace(/\s+/g, '-');

  return (
    <div class={`flex flex-col gap-1.5 ${local.class ?? ''}`}>
      <Show when={local.label}>
        <label
          for={inputId()}
          class="text-sm font-medium text-[var(--color-text-secondary)]"
        >
          {local.label}
        </label>
      </Show>

      <div class="relative">
        <Show when={local.icon}>
          <div class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--color-text-tertiary)]">
            {local.icon}
          </div>
        </Show>

        <input
          {...rest}
          id={inputId()}
          class={`w-full rounded-lg border bg-[var(--color-bg-secondary)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] transition-colors duration-150 focus:outline-none focus:ring-2 focus:ring-indigo-500/40 focus:border-[var(--color-border-focus)] ${
            local.error
              ? 'border-red-500/60 focus:ring-red-500/40 focus:border-red-500'
              : 'border-[var(--color-border-primary)]'
          } ${local.icon ? 'pl-10' : 'pl-3'} ${local.rightIcon ? 'pr-10' : 'pr-3'} py-2 text-sm`}
        />

        <Show when={local.rightIcon}>
          <div class="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--color-text-tertiary)]">
            {local.rightIcon}
          </div>
        </Show>
      </div>

      <Show when={local.error}>
        <p class="text-xs text-red-400">{local.error}</p>
      </Show>
      <Show when={local.hint && !local.error}>
        <p class="text-xs text-[var(--color-text-tertiary)]">{local.hint}</p>
      </Show>
    </div>
  );
};

export default Input;
