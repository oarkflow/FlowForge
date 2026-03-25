import type { Component, JSX } from 'solid-js';
import { createSignal, For } from 'solid-js';
import { Portal } from 'solid-js/web';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface ToastItem {
  id: number;
  type: ToastType;
  message: string;
  duration?: number;
}

// ---------------------------------------------------------------------------
// Global toast state
// ---------------------------------------------------------------------------
const [toasts, setToasts] = createSignal<ToastItem[]>([]);
let nextId = 0;

export function addToast(type: ToastType, message: string, duration = 5000): void {
  const id = nextId++;
  setToasts((prev) => [...prev, { id, type, message, duration }]);

  if (duration > 0) {
    setTimeout(() => removeToast(id), duration);
  }
}

export function removeToast(id: number): void {
  setToasts((prev) => prev.filter((t) => t.id !== id));
}

// Convenience functions
export const toast = {
  success: (msg: string, dur?: number) => addToast('success', msg, dur),
  error: (msg: string, dur?: number) => addToast('error', msg, dur),
  warning: (msg: string, dur?: number) => addToast('warning', msg, dur),
  info: (msg: string, dur?: number) => addToast('info', msg, dur),
};

// ---------------------------------------------------------------------------
// Icons & colors per type
// ---------------------------------------------------------------------------
const typeConfig: Record<ToastType, { bg: string; border: string; icon: JSX.Element }> = {
  success: {
    bg: 'bg-emerald-500/10',
    border: 'border-emerald-500/30',
    icon: (
      <svg class="w-5 h-5 text-emerald-400" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" />
      </svg>
    ),
  },
  error: {
    bg: 'bg-red-500/10',
    border: 'border-red-500/30',
    icon: (
      <svg class="w-5 h-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clip-rule="evenodd" />
      </svg>
    ),
  },
  warning: {
    bg: 'bg-amber-500/10',
    border: 'border-amber-500/30',
    icon: (
      <svg class="w-5 h-5 text-amber-400" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" />
      </svg>
    ),
  },
  info: {
    bg: 'bg-blue-500/10',
    border: 'border-blue-500/30',
    icon: (
      <svg class="w-5 h-5 text-blue-400" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z" clip-rule="evenodd" />
      </svg>
    ),
  },
};

// ---------------------------------------------------------------------------
// Toast Container Component
// ---------------------------------------------------------------------------
const ToastContainer: Component = () => {
  return (
    <Portal>
      <div class="fixed top-4 right-4 z-[100] flex flex-col gap-2 max-w-sm w-full pointer-events-none">
        <For each={toasts()}>
          {(item) => {
            const cfg = typeConfig[item.type];
            return (
              <div
                class={`${cfg.bg} ${cfg.border} border rounded-lg p-3 flex items-start gap-3 shadow-lg pointer-events-auto animate-slide-down`}
              >
                <div class="shrink-0 mt-0.5">{cfg.icon}</div>
                <p class="text-sm text-[var(--color-text-primary)] flex-1">{item.message}</p>
                <button
                  onClick={() => removeToast(item.id)}
                  class="shrink-0 p-0.5 rounded text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] transition-colors"
                >
                  <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
                    <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
                  </svg>
                </button>
              </div>
            );
          }}
        </For>
      </div>
    </Portal>
  );
};

export default ToastContainer;
