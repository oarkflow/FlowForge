import type { RunStatus, AgentStatus } from '../types';

export function formatDuration(ms?: number): string {
  if (!ms) return '-';
  if (ms < 1000) return `${ms}ms`;
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  if (minutes < 60) return `${minutes}m ${remainingSeconds}s`;
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${hours}h ${remainingMinutes}m`;
}

export function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSeconds = Math.floor(diffMs / 1000);
  if (diffSeconds < 10) return 'just now';
  if (diffSeconds < 60) return `${diffSeconds}s ago`;
  const diffMinutes = Math.floor(diffSeconds / 60);
  if (diffMinutes < 60) return `${diffMinutes}m ago`;
  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}

export function formatBytes(bytes?: number): string {
  if (bytes === undefined || bytes === null) return '-';
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const value = bytes / Math.pow(1024, i);
  return `${value.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

export function getStatusColor(status: RunStatus): string {
  const map: Record<RunStatus, string> = {
    success: 'text-emerald-400',
    failure: 'text-red-400',
    running: 'text-violet-400',
    queued: 'text-gray-400',
    pending: 'text-gray-400',
    cancelled: 'text-gray-500',
    skipped: 'text-gray-500',
    waiting_approval: 'text-amber-400',
  };
  return map[status] ?? 'text-gray-400';
}

export function getStatusBgColor(status: RunStatus): string {
  const map: Record<RunStatus, string> = {
    success: 'bg-emerald-400',
    failure: 'bg-red-400',
    running: 'bg-violet-400',
    queued: 'bg-gray-400',
    pending: 'bg-gray-400',
    cancelled: 'bg-gray-500',
    skipped: 'bg-gray-500',
    waiting_approval: 'bg-amber-400',
  };
  return map[status] ?? 'bg-gray-400';
}

export function getStatusBadgeVariant(status: RunStatus): 'success' | 'error' | 'warning' | 'info' | 'default' {
  const map: Record<RunStatus, 'success' | 'error' | 'warning' | 'info' | 'default'> = {
    success: 'success',
    failure: 'error',
    running: 'info',
    queued: 'default',
    pending: 'default',
    cancelled: 'warning',
    skipped: 'default',
    waiting_approval: 'warning',
  };
  return map[status] ?? 'default';
}

export function getAgentStatusVariant(status: AgentStatus): 'success' | 'error' | 'warning' | 'info' | 'default' {
  const map: Record<AgentStatus, 'success' | 'error' | 'warning' | 'info' | 'default'> = {
    online: 'success',
    busy: 'info',
    draining: 'warning',
    offline: 'default',
  };
  return map[status] ?? 'default';
}

export function truncateCommitSha(sha?: string): string {
  return sha ? sha.slice(0, 7) : '-';
}

export function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '');
}

export async function copyToClipboard(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
  }
}

export function cn(...args: (string | false | null | undefined)[]): string {
  return args.filter(Boolean).join(' ');
}
