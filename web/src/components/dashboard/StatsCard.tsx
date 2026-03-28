import { Component, JSX, Show } from 'solid-js';

interface StatsCardProps {
  label: string;
  value: string | number;
  icon?: JSX.Element;
  change?: { value: number; direction: 'up' | 'down' | 'neutral' };
  variant?: 'default' | 'success' | 'danger' | 'warning' | 'info';
  onClick?: () => void;
}

const variantStyles: Record<string, { accent: string; iconBg: string }> = {
  default: { accent: 'var(--text-primary)', iconBg: 'var(--bg-tertiary)' },
  success: { accent: '#3fb950', iconBg: 'rgba(63, 185, 80, 0.1)' },
  danger: { accent: '#ff7b72', iconBg: 'rgba(255, 123, 114, 0.1)' },
  warning: { accent: '#d29922', iconBg: 'rgba(210, 153, 34, 0.1)' },
  info: { accent: '#58a6ff', iconBg: 'rgba(88, 166, 255, 0.1)' },
};

/**
 * StatsCard - Dashboard stat card with icon, value, and change indicator.
 */
export const StatsCard: Component<StatsCardProps> = (props) => {
  const variant = () => variantStyles[props.variant || 'default'];

  const changeColor = () => {
    if (!props.change) return '';
    switch (props.change.direction) {
      case 'up': return '#3fb950';
      case 'down': return '#ff7b72';
      default: return 'var(--text-tertiary)';
    }
  };

  const changeSymbol = () => {
    if (!props.change) return '';
    switch (props.change.direction) {
      case 'up': return '↑';
      case 'down': return '↓';
      default: return '→';
    }
  };

  return (
    <div
      class={`rounded-lg p-4 transition-all ${props.onClick ? 'cursor-pointer hover:brightness-110' : ''}`}
      style={`background: var(--bg-secondary); border: 1px solid var(--border-primary);`}
      onClick={props.onClick}
    >
      <div class="flex items-start justify-between">
        <div>
          <p class="text-xs font-medium uppercase tracking-wider" style="color: var(--text-tertiary);">
            {props.label}
          </p>
          <p class="text-2xl font-bold mt-1" style={`color: ${variant().accent};`}>
            {props.value}
          </p>
        </div>
        <Show when={props.icon}>
          <div class="p-2 rounded-lg" style={`background: ${variant().iconBg};`}>
            {props.icon}
          </div>
        </Show>
      </div>
      <Show when={props.change}>
        <div class="mt-2 flex items-center gap-1">
          <span class="text-xs font-medium" style={`color: ${changeColor()};`}>
            {changeSymbol()} {Math.abs(props.change!.value)}%
          </span>
          <span class="text-xs" style="color: var(--text-tertiary);">vs last period</span>
        </div>
      </Show>
    </div>
  );
};
