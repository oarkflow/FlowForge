import type { Component, JSX } from 'solid-js';
import { splitProps } from 'solid-js';

export type BadgeVariant = 'default' | 'success' | 'warning' | 'error' | 'info' | 'running' | 'queued';
export type BadgeSize = 'sm' | 'md';

interface BadgeProps {
  variant?: BadgeVariant;
  size?: BadgeSize;
  dot?: boolean;
  children: JSX.Element;
  class?: string;
}

const variantStyles: Record<BadgeVariant, string> = {
  default:
    'bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] border-[var(--color-border-primary)]',
  success:
    'bg-[var(--color-success-bg)] text-emerald-400 border-emerald-500/20',
  warning:
    'bg-[var(--color-warning-bg)] text-amber-400 border-amber-500/20',
  error:
    'bg-[var(--color-error-bg)] text-red-400 border-red-500/20',
  info:
    'bg-[var(--color-info-bg)] text-blue-400 border-blue-500/20',
  running:
    'bg-[var(--color-running-bg)] text-violet-400 border-violet-500/20',
  queued:
    'bg-[var(--color-queued-bg)] text-gray-400 border-gray-500/20',
};

const dotColors: Record<BadgeVariant, string> = {
  default: 'bg-gray-400',
  success: 'bg-emerald-400',
  warning: 'bg-amber-400',
  error: 'bg-red-400',
  info: 'bg-blue-400',
  running: 'bg-violet-400 animate-pulse-dot',
  queued: 'bg-gray-400',
};

const Badge: Component<BadgeProps> = (props) => {
  const [local, _rest] = splitProps(props, ['variant', 'size', 'dot', 'children', 'class']);

  const variant = () => local.variant ?? 'default';
  const size = () => local.size ?? 'sm';

  return (
    <span
      class={`inline-flex items-center gap-1.5 rounded-full border font-medium ${variantStyles[variant()]} ${
        size() === 'sm' ? 'px-2 py-0.5 text-xs' : 'px-3 py-1 text-sm'
      } ${local.class ?? ''}`}
    >
      {local.dot && (
        <span class={`w-1.5 h-1.5 rounded-full ${dotColors[variant()]}`} />
      )}
      {local.children}
    </span>
  );
};

export default Badge;
