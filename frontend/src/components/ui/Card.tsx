import type { Component, JSX } from 'solid-js';
import { Show, splitProps } from 'solid-js';

interface CardProps {
  title?: string;
  description?: string;
  actions?: JSX.Element;
  children: JSX.Element;
  class?: string;
  padding?: boolean;
}

const Card: Component<CardProps> = (props) => {
  const [local, _rest] = splitProps(props, ['title', 'description', 'actions', 'children', 'class', 'padding']);

  const showHeader = () => local.title || local.description || local.actions;
  const pad = () => local.padding !== false;

  return (
    <div
      class={`bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl overflow-hidden ${local.class ?? ''}`}
    >
      <Show when={showHeader()}>
        <div class="flex items-center justify-between px-5 py-4 border-b border-[var(--color-border-primary)]">
          <div>
            <Show when={local.title}>
              <h3 class="text-sm font-semibold text-[var(--color-text-primary)]">{local.title}</h3>
            </Show>
            <Show when={local.description}>
              <p class="text-xs text-[var(--color-text-secondary)] mt-0.5">{local.description}</p>
            </Show>
          </div>
          <Show when={local.actions}>
            <div class="flex items-center gap-2">{local.actions}</div>
          </Show>
        </div>
      </Show>
      <div class={pad() ? 'p-5' : ''}>{local.children}</div>
    </div>
  );
};

export default Card;
