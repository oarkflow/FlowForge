import type { JSX, Component } from 'solid-js';
import { Show, onMount, onCleanup, createSignal } from 'solid-js';
import { Portal } from 'solid-js/web';

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
  children: JSX.Element;
  footer?: JSX.Element;
  size?: 'sm' | 'md' | 'lg' | 'xl';
}

const sizeMap: Record<string, string> = {
  sm: 'max-w-sm',
  md: 'max-w-lg',
  lg: 'max-w-2xl',
  xl: 'max-w-4xl',
};

const Modal: Component<ModalProps> = (props) => {
  const [visible, setVisible] = createSignal(false);
  let backdropRef: HTMLDivElement | undefined;

  const size = () => props.size ?? 'md';

  // Animate in after mount
  onMount(() => {
    requestAnimationFrame(() => setVisible(true));
  });

  // Close on Escape
  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'Escape') props.onClose();
  };

  onMount(() => {
    document.addEventListener('keydown', handleKeyDown);
    // Prevent body scroll
    document.body.style.overflow = 'hidden';
  });

  onCleanup(() => {
    document.removeEventListener('keydown', handleKeyDown);
    document.body.style.overflow = '';
  });

  const handleBackdropClick = (e: MouseEvent) => {
    if (e.target === backdropRef) props.onClose();
  };

  return (
    <Show when={props.open}>
      <Portal>
        <div
          ref={backdropRef}
          class={`fixed inset-0 z-50 flex items-center justify-center p-4 transition-opacity duration-200 ${
            visible() ? 'opacity-100' : 'opacity-0'
          }`}
          style={{ "background-color": "rgba(0, 0, 0, 0.6)", "backdrop-filter": "blur(4px)" }}
          onClick={handleBackdropClick}
        >
          <div
            class={`${sizeMap[size()]} w-full bg-[var(--color-bg-elevated)] border border-[var(--color-border-primary)] rounded-xl shadow-2xl transition-transform duration-200 ${
              visible() ? 'scale-100' : 'scale-95'
            }`}
          >
            {/* Header */}
            <div class="flex items-center justify-between px-6 py-4 border-b border-[var(--color-border-primary)]">
              <div>
                <Show when={props.title}>
                  <h2 class="text-lg font-semibold text-[var(--color-text-primary)]">
                    {props.title}
                  </h2>
                </Show>
                <Show when={props.description}>
                  <p class="text-sm text-[var(--color-text-secondary)] mt-1">
                    {props.description}
                  </p>
                </Show>
              </div>
              <button
                onClick={props.onClose}
                class="p-1.5 rounded-lg text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
              >
                <svg class="w-5 h-5" viewBox="0 0 20 20" fill="currentColor">
                  <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
                </svg>
              </button>
            </div>

            {/* Body */}
            <div class="px-6 py-4">{props.children}</div>

            {/* Footer */}
            <Show when={props.footer}>
              <div class="flex items-center justify-end gap-3 px-6 py-4 border-t border-[var(--color-border-primary)]">
                {props.footer}
              </div>
            </Show>
          </div>
        </div>
      </Portal>
    </Show>
  );
};

export default Modal;
