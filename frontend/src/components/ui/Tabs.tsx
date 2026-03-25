import type { Component, JSX } from 'solid-js';
import { For } from 'solid-js';

interface Tab {
  id: string;
  label: string | JSX.Element;
  icon?: JSX.Element;
  disabled?: boolean;
}

interface TabsProps {
  tabs: Tab[];
  activeTab: string;
  onTabChange: (id: string) => void;
  class?: string;
}

const Tabs: Component<TabsProps> = (props) => {
  return (
    <div
      class={`flex border-b border-[var(--color-border-primary)] ${props.class ?? ''}`}
      role="tablist"
    >
      <For each={props.tabs}>
        {(tab) => {
          const isActive = () => props.activeTab === tab.id;
          return (
            <button
              role="tab"
              aria-selected={isActive()}
              disabled={tab.disabled}
              onClick={() => props.onTabChange(tab.id)}
              class={`relative flex items-center gap-2 px-4 py-2.5 text-sm font-medium transition-colors whitespace-nowrap disabled:opacity-40 disabled:cursor-not-allowed ${
                isActive()
                  ? 'text-[var(--color-text-primary)]'
                  : 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'
              }`}
            >
              {tab.icon}
              {tab.label}
              {/* Active indicator */}
              {isActive() && (
                <span class="absolute bottom-0 left-0 right-0 h-0.5 bg-indigo-500 rounded-t" />
              )}
            </button>
          );
        }}
      </For>
    </div>
  );
};

export default Tabs;
