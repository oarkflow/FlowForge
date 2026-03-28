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

// ---------------------------------------------------------------------------
// Cron expression helpers
// ---------------------------------------------------------------------------

const DAY_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
const MONTH_NAMES = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'];

function ordinal(n: number): string {
	const s = ['th', 'st', 'nd', 'rd'];
	const v = n % 100;
	return n + (s[(v - 20) % 10] || s[v] || s[0]);
}

function formatTime(hour: number, minute: number): string {
	const h = hour % 12 || 12;
	const ampm = hour < 12 ? 'AM' : 'PM';
	const m = minute.toString().padStart(2, '0');
	return `${h}:${m} ${ampm}`;
}

/**
 * Returns a human-readable description of a 5-field cron expression.
 * Handles common patterns and special aliases.
 *
 * Examples:
 * - "0 3 * * *"      → "Every day at 3:00 AM"
 * - "* /15 * * * *"  → "Every 15 minutes"
 * - "0 9 * * 1"      → "Every Monday at 9:00 AM"
 * - "0 0 1 * *"      → "On the 1st of every month at 12:00 AM"
 */
export function describeCron(expression: string): string {
	if (!expression) return '';

	const trimmed = expression.trim();

	// Handle special aliases
	const aliases: Record<string, string> = {
		'@yearly': 'Once a year (January 1st at midnight)',
		'@annually': 'Once a year (January 1st at midnight)',
		'@monthly': 'Once a month (1st at midnight)',
		'@weekly': 'Once a week (Sunday at midnight)',
		'@daily': 'Once a day at midnight',
		'@midnight': 'Once a day at midnight',
		'@hourly': 'Every hour',
	};
	if (aliases[trimmed]) return aliases[trimmed];

	const parts = trimmed.split(/\s+/);
	if (parts.length !== 5) return trimmed;

	const [minute, hour, dom, month, dow] = parts;

	try {
		// Every minute
		if (minute === '*' && hour === '*' && dom === '*' && month === '*' && dow === '*') {
			return 'Every minute';
		}

		// Every N minutes: */N * * * *
		if (minute.startsWith('*/') && hour === '*' && dom === '*' && month === '*' && dow === '*') {
			const n = parseInt(minute.slice(2));
			if (n === 1) return 'Every minute';
			return `Every ${n} minutes`;
		}

		// Every N hours: 0 */N * * *
		if (hour.startsWith('*/') && dom === '*' && month === '*' && dow === '*') {
			const n = parseInt(hour.slice(2));
			const min = minute === '*' ? 0 : parseInt(minute);
			if (n === 1) return `Every hour at minute ${min}`;
			return `Every ${n} hours at minute ${min}`;
		}

		// Specific minute, every hour: N * * * *
		if (/^\d+$/.test(minute) && hour === '*' && dom === '*' && month === '*' && dow === '*') {
			const min = parseInt(minute);
			return `Every hour at minute ${min}`;
		}

		const hasSpecificMinute = /^\d+$/.test(minute);
		const hasSpecificHour = /^\d+$/.test(hour);
		const hasSpecificDom = /^\d+$/.test(dom);
		const hasSpecificMonth = /^\d+$/.test(month);
		const hasSpecificDow = /^\d+$/.test(dow);

		const min = hasSpecificMinute ? parseInt(minute) : 0;
		const hr = hasSpecificHour ? parseInt(hour) : 0;
		const timeStr = hasSpecificHour && hasSpecificMinute ? formatTime(hr, min) : null;

		// Daily at specific time: N N * * *
		if (hasSpecificMinute && hasSpecificHour && dom === '*' && month === '*' && dow === '*') {
			return `Every day at ${timeStr}`;
		}

		// Weekly: N N * * N
		if (hasSpecificMinute && hasSpecificHour && dom === '*' && month === '*' && hasSpecificDow) {
			const dayName = DAY_NAMES[parseInt(dow) % 7] || dow;
			return `Every ${dayName} at ${timeStr}`;
		}

		// Monthly: N N N * *
		if (hasSpecificMinute && hasSpecificHour && hasSpecificDom && month === '*' && dow === '*') {
			return `On the ${ordinal(parseInt(dom))} of every month at ${timeStr}`;
		}

		// Yearly: N N N N *
		if (hasSpecificMinute && hasSpecificHour && hasSpecificDom && hasSpecificMonth && dow === '*') {
			const monthName = MONTH_NAMES[parseInt(month) - 1] || month;
			return `On ${monthName} ${ordinal(parseInt(dom))} at ${timeStr}`;
		}

		// Multiple days of week: N N * * 1,3,5
		if (hasSpecificMinute && hasSpecificHour && dom === '*' && month === '*' && dow.includes(',')) {
			const days = dow.split(',').map(d => DAY_NAMES[parseInt(d.trim()) % 7] || d.trim());
			if (days.length === 2) {
				return `Every ${days[0]} and ${days[1]} at ${timeStr}`;
			}
			const lastDay = days.pop();
			return `Every ${days.join(', ')}, and ${lastDay} at ${timeStr}`;
		}

		// Range of days: N N * * 1-5
		if (hasSpecificMinute && hasSpecificHour && dom === '*' && month === '*' && dow.includes('-')) {
			const [start, end] = dow.split('-').map(d => parseInt(d.trim()));
			const startDay = DAY_NAMES[start % 7] || String(start);
			const endDay = DAY_NAMES[end % 7] || String(end);
			return `${startDay} through ${endDay} at ${timeStr}`;
		}

		// Fallback: show the expression
		return trimmed;
	} catch {
		return trimmed;
	}
}

/**
 * Formats a future date as a relative time string (e.g., "in 2 hours", "in 3 days").
 */
export function formatFutureRelativeTime(dateStr: string): string {
	const date = new Date(dateStr);
	const now = new Date();
	const diffMs = date.getTime() - now.getTime();

	if (diffMs < 0) return 'overdue';
	if (diffMs < 60_000) return 'in less than a minute';

	const diffMinutes = Math.floor(diffMs / 60_000);
	if (diffMinutes < 60) return `in ${diffMinutes}m`;
	const diffHours = Math.floor(diffMinutes / 60);
	if (diffHours < 24) return `in ${diffHours}h`;
	const diffDays = Math.floor(diffHours / 24);
	if (diffDays < 7) return `in ${diffDays}d`;
	return date.toLocaleDateString();
}

/**
 * Common timezones for the schedule timezone selector.
 */
export const COMMON_TIMEZONES = [
	'UTC',
	'America/New_York',
	'America/Chicago',
	'America/Denver',
	'America/Los_Angeles',
	'America/Anchorage',
	'Pacific/Honolulu',
	'America/Sao_Paulo',
	'America/Argentina/Buenos_Aires',
	'Europe/London',
	'Europe/Paris',
	'Europe/Berlin',
	'Europe/Moscow',
	'Africa/Cairo',
	'Asia/Dubai',
	'Asia/Kolkata',
	'Asia/Kathmandu',
	'Asia/Shanghai',
	'Asia/Tokyo',
	'Asia/Seoul',
	'Australia/Sydney',
	'Pacific/Auckland',
];
