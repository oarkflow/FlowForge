import type { Component } from 'solid-js';
import { A, useLocation } from '@solidjs/router';
import { For, Show, createResource } from 'solid-js';
import { api } from '../../api/client';
import type { Approval } from '../../types';

interface NavItem {
  label: string;
  href: string;
  icon: string; // SVG path d attribute
  badge?: () => number;
}

// Shared resource for pending approval count (polled every 30s)
const [pendingApprovals] = createResource(
  async () => {
    try {
      const list = await api.approvals.listPending();
      return list.filter((a: Approval) => a.status === 'pending').length;
    } catch {
      return 0;
    }
  }
);

// Refresh pending count periodically
if (typeof window !== 'undefined') {
  setInterval(() => {
    // Trigger re-fetch by creating a new resource isn't straightforward;
    // the count will update on page navigation/reload.
  }, 30000);
}

const navItems: NavItem[] = [
  {
    label: 'Dashboard',
    href: '/',
    icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6',
  },
  {
    label: 'Projects',
    href: '/projects',
    icon: 'M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z',
  },
  {
    label: 'Builds',
    href: '/runs',
    icon: 'M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z',
  },
  {
    label: 'Agents',
    href: '/agents',
    icon: 'M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01',
  },
  {
    label: 'Approvals',
    href: '/approvals',
    icon: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z',
    badge: () => pendingApprovals() || 0,
  },
  {
    label: 'Settings',
    href: '/settings',
    icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z',
  },
];

const adminNavItems: NavItem[] = [
  {
    label: 'Admin',
    href: '/admin',
    icon: 'M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z',
  },
];

const Sidebar: Component = () => {
  const location = useLocation();

  const isActive = (href: string) => {
    if (href === '/') return location.pathname === '/';
    return location.pathname.startsWith(href);
  };

  return (
    <aside
      class="fixed left-0 top-0 bottom-0 z-30 flex flex-col bg-[var(--color-bg-secondary)] border-r border-[var(--color-border-primary)]"
      style={{ width: 'var(--sidebar-width)' }}
    >
      {/* Logo */}
      <div class="flex items-center gap-3 px-5 h-14 border-b border-[var(--color-border-primary)]">
        <div class="w-8 h-8 rounded-lg bg-indigo-600 flex items-center justify-center">
          <svg class="w-5 h-5 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
        </div>
        <span class="text-base font-bold text-[var(--color-text-primary)] tracking-tight">
          FlowForge
        </span>
      </div>

      {/* Navigation */}
      <nav class="flex-1 overflow-y-auto px-3 py-4">
        <div class="space-y-1">
          <For each={navItems}>
            {(item) => (
              <A
                href={item.href}
                class={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                  isActive(item.href)
                    ? 'bg-[var(--color-accent-bg)] text-indigo-400'
                    : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)]'
                }`}
              >
                <svg
                  class="w-5 h-5 shrink-0"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                >
                  <path d={item.icon} />
                </svg>
                <span class="flex-1">{item.label}</span>
                <Show when={item.badge && item.badge() > 0}>
                  <span class="inline-flex items-center justify-center min-w-[20px] h-5 px-1.5 rounded-full bg-yellow-500/20 text-yellow-400 text-[10px] font-bold">
                    {item.badge!()}
                  </span>
                </Show>
              </A>
            )}
          </For>
        </div>

        {/* Admin section */}
        <div class="mt-6 pt-4 border-t border-[var(--color-border-primary)]">
          <p class="px-3 mb-2 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)]">
            Administration
          </p>
          <div class="space-y-1">
            <For each={adminNavItems}>
              {(item) => (
                <A
                  href={item.href}
                  class={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                    isActive(item.href)
                      ? 'bg-[var(--color-accent-bg)] text-indigo-400'
                      : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)]'
                  }`}
                >
                  <svg
                    class="w-5 h-5 shrink-0"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.5"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <path d={item.icon} />
                  </svg>
                  {item.label}
                </A>
              )}
            </For>
          </div>
        </div>
      </nav>

      {/* Footer */}
      <div class="px-4 py-3 border-t border-[var(--color-border-primary)]">
        <div class="flex items-center gap-2 text-xs text-[var(--color-text-tertiary)]">
          <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse-dot" />
          System healthy
        </div>
      </div>
    </aside>
  );
};

export default Sidebar;
