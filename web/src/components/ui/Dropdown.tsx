import type { Component, JSX } from 'solid-js';
import { createSignal, Show, onMount, onCleanup } from 'solid-js';
import { Portal } from 'solid-js/web';

interface DropdownProps {
  trigger: JSX.Element;
  children: JSX.Element;
  align?: 'left' | 'right';
  class?: string;
}

const Dropdown: Component<DropdownProps> = (props) => {
  const [open, setOpen] = createSignal(false);
  let containerRef: HTMLDivElement | undefined;

  const align = () => props.align ?? 'left';

  // Close on click outside
  const handleClickOutside = (e: MouseEvent) => {
    if (containerRef && !containerRef.contains(e.target as Node)) {
      setOpen(false);
    }
  };

  // Close on Escape
  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'Escape') setOpen(false);
  };

  onMount(() => {
    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleKeyDown);
  });

  onCleanup(() => {
    document.removeEventListener('mousedown', handleClickOutside);
    document.removeEventListener('keydown', handleKeyDown);
  });

  return (
    <div ref={containerRef} class={`relative inline-block ${props.class ?? ''}`}>
      <div onClick={() => setOpen(!open())}>{props.trigger}</div>

      <Show when={open()}>
        <div
          class={`absolute z-50 mt-1 min-w-[200px] bg-[var(--color-bg-elevated)] border border-[var(--color-border-primary)] rounded-lg shadow-xl py-1 animate-slide-down ${
            align() === 'right' ? 'right-0' : 'left-0'
          }`}
          onClick={() => setOpen(false)}
        >
          {props.children}
        </div>
      </Show>
    </div>
  );
};

// ---------------------------------------------------------------------------
// DropdownItem helper
// ---------------------------------------------------------------------------
interface DropdownItemProps {
  onClick?: () => void;
  icon?: JSX.Element;
  danger?: boolean;
  children: JSX.Element;
  disabled?: boolean;
}

export const DropdownItem: Component<DropdownItemProps> = (props) => {
  return (
    <button
      onClick={props.onClick}
      disabled={props.disabled}
      class={`w-full flex items-center gap-2 px-3 py-2 text-sm transition-colors disabled:opacity-40 disabled:cursor-not-allowed ${
        props.danger
          ? 'text-red-400 hover:bg-red-500/10'
          : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)]'
      }`}
    >
      {props.icon}
      {props.children}
    </button>
  );
};

export const DropdownSeparator: Component = () => (
  <div class="my-1 border-t border-[var(--color-border-primary)]" />
);

export default Dropdown;
