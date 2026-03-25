import type { Component } from 'solid-js';
import { Show } from 'solid-js';
import { authStore } from '../../stores/auth';
import Dropdown, { DropdownItem, DropdownSeparator } from '../ui/Dropdown';

const TopBar: Component = () => {
  const user = authStore.user;

  return (
    <header
      class="fixed top-0 right-0 z-20 flex items-center justify-between px-6 bg-[var(--color-bg-secondary)]/80 backdrop-blur-md border-b border-[var(--color-border-primary)]"
      style={{
        left: 'var(--sidebar-width)',
        height: 'var(--topbar-height)',
      }}
    >
      {/* Left: breadcrumb slot or search */}
      <div class="flex items-center gap-3">
        <div class="relative">
          <svg
            class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--color-text-tertiary)]"
            viewBox="0 0 20 20"
            fill="currentColor"
          >
            <path
              fill-rule="evenodd"
              d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z"
              clip-rule="evenodd"
            />
          </svg>
          <input
            type="text"
            placeholder="Search pipelines, projects..."
            class="w-64 pl-9 pr-3 py-1.5 text-sm rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40 focus:border-[var(--color-border-focus)] transition-colors"
          />
        </div>
      </div>

      {/* Right: notifications + user */}
      <div class="flex items-center gap-3">
        {/* Notification bell */}
        <button class="relative p-2 rounded-lg text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors">
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M14.857 17.082a23.848 23.848 0 005.454-1.31A8.967 8.967 0 0118 9.75V9A6 6 0 006 9v.75a8.967 8.967 0 01-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 01-5.714 0m5.714 0a3 3 0 11-5.714 0" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          {/* Notification dot */}
          <span class="absolute top-1.5 right-1.5 w-2 h-2 rounded-full bg-red-500" />
        </button>

        {/* User dropdown */}
        <Show when={user()}>
          {(u) => (
            <Dropdown
              align="right"
              trigger={
                <button class="flex items-center gap-2 p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
                  <div class="w-8 h-8 rounded-full bg-indigo-600/30 border border-indigo-500/30 flex items-center justify-center text-sm font-medium text-indigo-400">
                    {u().display_name?.[0]?.toUpperCase() ?? u().email[0].toUpperCase()}
                  </div>
                  <div class="text-left hidden sm:block">
                    <p class="text-sm font-medium text-[var(--color-text-primary)] leading-tight">
                      {u().display_name ?? u().username}
                    </p>
                    <p class="text-xs text-[var(--color-text-tertiary)] leading-tight">
                      {u().role}
                    </p>
                  </div>
                  <svg class="w-4 h-4 text-[var(--color-text-tertiary)]" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
                  </svg>
                </button>
              }
            >
              <div class="px-3 py-2 border-b border-[var(--color-border-primary)]">
                <p class="text-sm font-medium text-[var(--color-text-primary)]">{u().email}</p>
              </div>
              <DropdownItem
                icon={
                  <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
                    <path d="M10 8a3 3 0 100-6 3 3 0 000 6zM3.465 14.493a1.23 1.23 0 00.41 1.412A9.957 9.957 0 0010 18c2.31 0 4.438-.784 6.131-2.1.43-.333.604-.903.408-1.41a7.002 7.002 0 00-13.074.003z" />
                  </svg>
                }
              >
                Profile
              </DropdownItem>
              <DropdownItem
                icon={
                  <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M7.84 1.804A1 1 0 018.82 1h2.36a1 1 0 01.98.804l.331 1.652a6.993 6.993 0 011.929 1.115l1.598-.54a1 1 0 011.186.447l1.18 2.044a1 1 0 01-.205 1.251l-1.267 1.113a7.047 7.047 0 010 2.228l1.267 1.113a1 1 0 01.206 1.25l-1.18 2.045a1 1 0 01-1.187.447l-1.598-.54a6.993 6.993 0 01-1.929 1.115l-.33 1.652a1 1 0 01-.98.804H8.82a1 1 0 01-.98-.804l-.331-1.652a6.993 6.993 0 01-1.929-1.115l-1.598.54a1 1 0 01-1.186-.447l-1.18-2.044a1 1 0 01.205-1.251l1.267-1.114a7.05 7.05 0 010-2.227L1.821 7.773a1 1 0 01-.206-1.25l1.18-2.045a1 1 0 011.187-.447l1.598.54A6.993 6.993 0 017.51 3.456l.33-1.652zM10 13a3 3 0 100-6 3 3 0 000 6z" clip-rule="evenodd" />
                  </svg>
                }
              >
                Settings
              </DropdownItem>
              <DropdownSeparator />
              <DropdownItem
                danger
                onClick={() => authStore.logout()}
                icon={
                  <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M3 4.25A2.25 2.25 0 015.25 2h5.5A2.25 2.25 0 0113 4.25v2a.75.75 0 01-1.5 0v-2a.75.75 0 00-.75-.75h-5.5a.75.75 0 00-.75.75v11.5c0 .414.336.75.75.75h5.5a.75.75 0 00.75-.75v-2a.75.75 0 011.5 0v2A2.25 2.25 0 0110.75 18h-5.5A2.25 2.25 0 013 15.75V4.25z" clip-rule="evenodd" />
                    <path fill-rule="evenodd" d="M19 10a.75.75 0 00-.75-.75H8.704l1.048-.943a.75.75 0 10-1.004-1.114l-2.5 2.25a.75.75 0 000 1.114l2.5 2.25a.75.75 0 101.004-1.114l-1.048-.943h9.546A.75.75 0 0019 10z" clip-rule="evenodd" />
                  </svg>
                }
              >
                Sign out
              </DropdownItem>
            </Dropdown>
          )}
        </Show>
      </div>
    </header>
  );
};

export default TopBar;
