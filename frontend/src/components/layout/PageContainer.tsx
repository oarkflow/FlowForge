import type { Component, JSX } from 'solid-js';
import { Show, For } from 'solid-js';
import { A } from '@solidjs/router';

interface Breadcrumb {
  label: string;
  href?: string;
}

interface PageContainerProps {
  title?: string;
  description?: string;
  breadcrumbs?: Breadcrumb[];
  actions?: JSX.Element;
  children: JSX.Element;
  class?: string;
}

const PageContainer: Component<PageContainerProps> = (props) => {
  return (
    <div class={`animate-fade-in ${props.class ?? ''}`}>
      {/* Breadcrumbs */}
      <Show when={props.breadcrumbs && props.breadcrumbs.length > 0}>
        <nav class="flex items-center gap-1.5 mb-4 text-sm">
          <For each={props.breadcrumbs}>
            {(crumb, index) => (
              <>
                <Show when={index() > 0}>
                  <svg class="w-3.5 h-3.5 text-[var(--color-text-tertiary)]" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
                  </svg>
                </Show>
                <Show
                  when={crumb.href}
                  fallback={
                    <span class="text-[var(--color-text-primary)] font-medium">
                      {crumb.label}
                    </span>
                  }
                >
                  <A
                    href={crumb.href!}
                    class="text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] transition-colors"
                  >
                    {crumb.label}
                  </A>
                </Show>
              </>
            )}
          </For>
        </nav>
      </Show>

      {/* Page header */}
      <Show when={props.title}>
        <div class="flex items-start justify-between mb-6">
          <div>
            <h1 class="text-2xl font-bold text-[var(--color-text-primary)] tracking-tight">
              {props.title}
            </h1>
            <Show when={props.description}>
              <p class="text-sm text-[var(--color-text-secondary)] mt-1">
                {props.description}
              </p>
            </Show>
          </div>
          <Show when={props.actions}>
            <div class="flex items-center gap-2 shrink-0">{props.actions}</div>
          </Show>
        </div>
      </Show>

      {/* Content */}
      {props.children}
    </div>
  );
};

export default PageContainer;
