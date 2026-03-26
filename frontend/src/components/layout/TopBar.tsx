import type { Component } from 'solid-js';
import { Show, For, createSignal, createEffect, onCleanup } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { authStore } from '../../stores/auth';
import { api } from '../../api/client';
import type { InAppNotification } from '../../types';
import Dropdown, { DropdownItem, DropdownSeparator } from '../ui/Dropdown';

function timeAgo(dateStr: string): string {
	const now = Date.now();
	const then = new Date(dateStr).getTime();
	const seconds = Math.floor((now - then) / 1000);
	if (seconds < 60) return 'just now';
	const minutes = Math.floor(seconds / 60);
	if (minutes < 60) return `${minutes}m ago`;
	const hours = Math.floor(minutes / 60);
	if (hours < 24) return `${hours}h ago`;
	const days = Math.floor(hours / 24);
	return `${days}d ago`;
}

function notifTypeColor(type: string): string {
	switch (type) {
		case 'success': return 'text-green-400';
		case 'warning': return 'text-yellow-400';
		case 'error': return 'text-red-400';
		default: return 'text-blue-400';
	}
}

function notifTypeIcon(type: string) {
	switch (type) {
		case 'success':
			return <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" /></svg>;
		case 'warning':
			return <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 6a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 6zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" /></svg>;
		case 'error':
			return <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clip-rule="evenodd" /></svg>;
		default:
			return <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z" clip-rule="evenodd" /></svg>;
	}
}

const TopBar: Component<{ onOpenSearch?: () => void }> = (props) => {
	const user = authStore.user;
	const navigate = useNavigate();
	const [unreadCount, setUnreadCount] = createSignal(0);
	const [notifications, setNotifications] = createSignal<InAppNotification[]>([]);
	const [showNotifPanel, setShowNotifPanel] = createSignal(false);
	const [loadingNotifs, setLoadingNotifs] = createSignal(false);

	// Poll for unread count every 30 seconds
	const fetchUnreadCount = async () => {
		try {
			const result = await api.inbox.unreadCount();
			setUnreadCount(result.count);
		} catch {
			// Silently fail — don't block UI
		}
	};

	const fetchNotifications = async () => {
		setLoadingNotifs(true);
		try {
			const items = await api.inbox.list({ limit: '10', offset: '0' });
			setNotifications(items);
		} catch {
			// Silently fail
		} finally {
			setLoadingNotifs(false);
		}
	};

	// Initial fetch + polling
	fetchUnreadCount();
	const pollInterval = setInterval(fetchUnreadCount, 30000);
	onCleanup(() => clearInterval(pollInterval));

	const handleBellClick = () => {
		const isOpen = !showNotifPanel();
		setShowNotifPanel(isOpen);
		if (isOpen) {
			fetchNotifications();
		}
	};

	const handleMarkAllRead = async () => {
		try {
			await api.inbox.markAllRead();
			setUnreadCount(0);
			setNotifications(prev => prev.map(n => ({ ...n, is_read: true })));
		} catch {
			// Silently fail
		}
	};

	const handleNotifClick = async (notif: InAppNotification) => {
		if (!notif.is_read) {
			try {
				await api.inbox.markRead(notif.id);
				setUnreadCount(prev => Math.max(0, prev - 1));
				setNotifications(prev => prev.map(n => n.id === notif.id ? { ...n, is_read: true } : n));
			} catch {
				// Silently fail
			}
		}
		if (notif.link) {
			navigate(notif.link);
			setShowNotifPanel(false);
		}
	};

	// Close panel on outside click
	let panelRef: HTMLDivElement | undefined;
	const handleClickOutside = (e: MouseEvent) => {
		if (panelRef && !panelRef.contains(e.target as Node)) {
			setShowNotifPanel(false);
		}
	};

	createEffect(() => {
		if (showNotifPanel()) {
			document.addEventListener('mousedown', handleClickOutside);
		} else {
			document.removeEventListener('mousedown', handleClickOutside);
		}
	});

	onCleanup(() => document.removeEventListener('mousedown', handleClickOutside));

	return (
		<header
			class="fixed top-0 right-0 z-20 flex items-center justify-between px-6 bg-[var(--color-bg-secondary)]/80 backdrop-blur-md border-b border-[var(--color-border-primary)]"
			style={{
				left: 'var(--sidebar-width)',
				height: 'var(--topbar-height)',
			}}
		>
			{/* Left: search bar that opens command palette */}
			<div class="flex items-center gap-3">
				<button
					class="relative flex items-center gap-2 w-100 pl-9 pr-3 py-1.5 text-sm rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-[var(--color-text-tertiary)] hover:border-[var(--color-border-focus)] focus:outline-none transition-colors cursor-pointer"
					onClick={() => props.onOpenSearch?.()}
				>
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
					<span>Search pipelines, projects...</span>
					<kbd class="ml-auto px-1.5 py-0.5 text-[10px] font-medium rounded bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-tertiary)]">⌘K</kbd>
				</button>
			</div>

			{/* Right: notifications + user */}
			<div class="flex items-center gap-3">
				{/* Notification bell with dropdown */}
				<div class="relative" ref={panelRef}>
					<button
						class="relative p-2 rounded-lg text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
						onClick={handleBellClick}
					>
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
							<path d="M14.857 17.082a23.848 23.848 0 005.454-1.31A8.967 8.967 0 0118 9.75V9A6 6 0 006 9v.75a8.967 8.967 0 01-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 01-5.714 0m5.714 0a3 3 0 11-5.714 0" stroke-linecap="round" stroke-linejoin="round" />
						</svg>
						{/* Unread count badge */}
						<Show when={unreadCount() > 0}>
							<span class="absolute -top-0.5 -right-0.5 min-w-[18px] h-[18px] px-1 flex items-center justify-center text-[10px] font-bold text-white bg-red-500 rounded-full leading-none">
								{unreadCount() > 99 ? '99+' : unreadCount()}
							</span>
						</Show>
					</button>

					{/* Notification dropdown panel */}
					<Show when={showNotifPanel()}>
						<div class="absolute right-0 top-full mt-2 w-96 max-h-[480px] rounded-xl bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] shadow-2xl overflow-hidden z-50">
							{/* Header */}
							<div class="flex items-center justify-between px-4 py-3 border-b border-[var(--color-border-primary)]">
								<h3 class="text-sm font-semibold text-[var(--color-text-primary)]">Notifications</h3>
								<Show when={unreadCount() > 0}>
									<button
										class="text-xs text-indigo-400 hover:text-indigo-300 transition-colors"
										onClick={handleMarkAllRead}
									>
										Mark all read
									</button>
								</Show>
							</div>

							{/* Notification list */}
							<div class="overflow-y-auto max-h-[380px]">
								<Show when={loadingNotifs()}>
									<div class="flex items-center justify-center py-8">
										<div class="w-5 h-5 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
									</div>
								</Show>

								<Show when={!loadingNotifs() && notifications().length === 0}>
									<div class="flex flex-col items-center justify-center py-8 text-[var(--color-text-tertiary)]">
										<svg class="w-8 h-8 mb-2 opacity-50" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
											<path d="M14.857 17.082a23.848 23.848 0 005.454-1.31A8.967 8.967 0 0118 9.75V9A6 6 0 006 9v.75a8.967 8.967 0 01-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 01-5.714 0m5.714 0a3 3 0 11-5.714 0" stroke-linecap="round" stroke-linejoin="round" />
										</svg>
										<p class="text-sm">No notifications</p>
									</div>
								</Show>

								<Show when={!loadingNotifs() && notifications().length > 0}>
									<For each={notifications()}>
										{(notif) => (
											<button
												class={`w-full text-left px-4 py-3 flex items-start gap-3 hover:bg-[var(--color-bg-hover)] transition-colors border-b border-[var(--color-border-primary)] last:border-b-0 ${!notif.is_read ? 'bg-[var(--color-bg-primary)]/50' : ''}`}
												onClick={() => handleNotifClick(notif)}
											>
												{/* Type icon */}
												<div class={`mt-0.5 flex-shrink-0 ${notifTypeColor(notif.type)}`}>
													{notifTypeIcon(notif.type)}
												</div>

												{/* Content */}
												<div class="flex-1 min-w-0">
													<div class="flex items-center gap-2">
														<p class={`text-sm truncate ${!notif.is_read ? 'font-semibold text-[var(--color-text-primary)]' : 'text-[var(--color-text-secondary)]'}`}>
															{notif.title}
														</p>
														<Show when={!notif.is_read}>
															<span class="flex-shrink-0 w-2 h-2 rounded-full bg-indigo-500" />
														</Show>
													</div>
													<Show when={notif.message}>
														<p class="text-xs text-[var(--color-text-tertiary)] mt-0.5 line-clamp-2">{notif.message}</p>
													</Show>
													<div class="flex items-center gap-2 mt-1">
														<span class="px-1.5 py-0.5 text-[10px] font-medium rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)]">
															{notif.category}
														</span>
														<span class="text-[10px] text-[var(--color-text-tertiary)]">{timeAgo(notif.created_at)}</span>
													</div>
												</div>
											</button>
										)}
									</For>
								</Show>
							</div>

							{/* Footer */}
							<div class="border-t border-[var(--color-border-primary)] px-4 py-2">
								<button
									class="w-full text-center text-xs text-indigo-400 hover:text-indigo-300 transition-colors py-1"
									onClick={() => {
										setShowNotifPanel(false);
										navigate('/settings');
									}}
								>
									Notification settings
								</button>
							</div>
						</div>
					</Show>
				</div>

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
								onClick={() => navigate('/settings')}
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
