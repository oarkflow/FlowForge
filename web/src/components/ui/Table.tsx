import type { JSX, Component } from 'solid-js';
import { For, Show } from 'solid-js';

// ---------------------------------------------------------------------------
// Table types
// ---------------------------------------------------------------------------
export interface TableColumn<T> {
  key: string;
  header: string | JSX.Element;
  render: (row: T, index: number) => JSX.Element;
  width?: string;
  align?: 'left' | 'center' | 'right';
}

interface TableProps<T> {
  columns: TableColumn<T>[];
  data: T[];
  emptyMessage?: string;
  loading?: boolean;
  onRowClick?: (row: T) => void;
  class?: string;
}

function Table<T>(props: TableProps<T>): JSX.Element {
  const alignClass = (align?: string) => {
    if (align === 'center') return 'text-center';
    if (align === 'right') return 'text-right';
    return 'text-left';
  };

  return (
    <div class={`overflow-x-auto ${props.class ?? ''}`}>
      <table class="w-full">
        <thead>
          <tr class="border-b border-[var(--color-border-primary)]">
            <For each={props.columns}>
              {(col) => (
                <th
                  class={`px-4 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] ${alignClass(col.align)}`}
                  style={col.width ? { width: col.width } : {}}
                >
                  {col.header}
                </th>
              )}
            </For>
          </tr>
        </thead>
        <tbody>
          <Show
            when={!props.loading && props.data.length > 0}
            fallback={
              <tr>
                <td
                  colspan={props.columns.length}
                  class="px-4 py-12 text-center text-sm text-[var(--color-text-tertiary)]"
                >
                  {props.loading ? (
                    <div class="flex items-center justify-center gap-2">
                      <svg class="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                      </svg>
                      Loading...
                    </div>
                  ) : (
                    props.emptyMessage ?? 'No data'
                  )}
                </td>
              </tr>
            }
          >
            <For each={props.data}>
              {(row, index) => (
                <tr
                  class={`border-b border-[var(--color-border-primary)] last:border-b-0 transition-colors ${
                    props.onRowClick
                      ? 'cursor-pointer hover:bg-[var(--color-bg-hover)]'
                      : ''
                  }`}
                  onClick={() => props.onRowClick?.(row)}
                >
                  <For each={props.columns}>
                    {(col) => (
                      <td
                        class={`px-4 py-3 text-sm ${alignClass(col.align)}`}
                        style={col.width ? { width: col.width } : {}}
                      >
                        {col.render(row, index())}
                      </td>
                    )}
                  </For>
                </tr>
              )}
            </For>
          </Show>
        </tbody>
      </table>
    </div>
  );
}

export default Table;
