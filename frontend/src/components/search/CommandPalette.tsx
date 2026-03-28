import type { Component } from 'solid-js';
import { Show, For, createSignal, createEffect, onMount, onCleanup } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { api } from '../../api/client';
import type { Project, Pipeline, PipelineRun } from '../../types';

interface CommandPaletteProps {
	isOpen: boolean;
	onClose: () => void;
}

const RECENT_SEARCHES_KEY = 'flowforge_recent_searches';
const MAX_RECENT = 5;

function getRecentSearches(): string[] {
	try {
		const stored = localStorage.getItem(RECENT_SEARCHES_KEY);
		return stored ? JSON.parse(stored) : [];
	} catch {
		return [];
	}
}

function saveRecentSearch(query: string) {
	const recent = getRecentSearches().filter(s => s !== query);
	recent.unshift(query);
	localStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(recent.slice(0, MAX_RECENT)));
}

const CommandPalette: Component<CommandPaletteProps> = (props) => {
	const navigate = useNavigate();
	const [query, setQuery] = createSignal('');
	const [projects, setProjects] = createSignal<Project[]>([]);
	const [pipelines, setPipelines] = createSignal<Pipeline[]>([]);
	const [runs, setRuns] = createSignal<PipelineRun[]>([]);
	const [loading, setLoading] = createSignal(false);
	const [selectedIndex, setSelectedIndex] = createSignal(0);
	const [recentSearches, setRecentSearches] = createSignal<string[]>(getRecentSearches());

	let inputRef: HTMLInputElement | undefined;
	let debounceTimer: ReturnType<typeof setTimeout> | undefined;

	// Quick actions
	const quickActions = [
		{ id: 'new-project', label: 'Create New Project', sublabel: 'Import a repository', icon: '➕', path: '/projects/import' },
		{ id: 'go-projects', label: 'Go to Projects', sublabel: 'View all projects', icon: '📁', path: '/projects' },
		{ id: 'go-runs', label: 'Go to Runs', sublabel: 'View all pipeline runs', icon: '▶️', path: '/runs' },
		{ id: 'go-agents', label: 'Go to Agents', sublabel: 'Manage build agents', icon: '🤖', path: '/agents' },
		{ id: 'go-templates', label: 'Browse Templates', sublabel: 'Pipeline template library', icon: '📋', path: '/templates' },
		{ id: 'go-approvals', label: 'Pending Approvals', sublabel: 'Review waiting runs', icon: '✅', path: '/approvals' },
		{ id: 'go-settings', label: 'Settings', sublabel: 'Account & app settings', icon: '⚙️', path: '/settings' },
		{ id: 'go-admin', label: 'Admin Panel', sublabel: 'System administration', icon: '🛡️', path: '/admin' },
	];

	const isCommandMode = () => query().startsWith('>');
	const commandQuery = () => query().startsWith('>') ? query().slice(1).trim().toLowerCase() : '';

	const filteredQuickActions = () => {
		if (!isCommandMode() && query().trim().length > 0) return [];
		const q = commandQuery();
		if (!q) return quickActions;
		return quickActions.filter(a =>
			a.label.toLowerCase().includes(q) || a.sublabel.toLowerCase().includes(q)
		);
	};

	// Focus input when opened
	createEffect(() => {
		if (props.isOpen) {
			setQuery('');
			setProjects([]);
			setPipelines([]);
			setRuns([]);
			setSelectedIndex(0);
			setRecentSearches(getRecentSearches());
			setTimeout(() => inputRef?.focus(), 50);
		}
	});

	const doSearch = async (q: string) => {
		if (!q.trim()) {
			setProjects([]);
			setPipelines([]);
			setRuns([]);
			return;
		}
		setLoading(true);
		try {
			const results = await api.search.query(q.trim());
			setProjects(results.projects || []);
			setPipelines(results.pipelines || []);
			setRuns(results.runs || []);
			setSelectedIndex(0);
		} catch {
			setProjects([]);
			setPipelines([]);
			setRuns([]);
		} finally {
			setLoading(false);
		}
	};

	const handleInput = (value: string) => {
		setQuery(value);
		if (debounceTimer) clearTimeout(debounceTimer);
		// In command mode (> prefix), don't search API — filter quick actions locally
		if (value.startsWith('>')) {
			setSelectedIndex(0);
			return;
		}
		debounceTimer = setTimeout(() => doSearch(value), 300);
	};

	// Build flat list of all results for keyboard navigation
	const allItems = () => {
		const items: { type: string; label: string; sublabel: string; path: string; icon?: string }[] = [];

		// If in command mode or no query, show quick actions
		const actions = filteredQuickActions();
		if (isCommandMode() || !hasQuery()) {
			for (const a of actions) {
				items.push({ type: 'action', label: a.label, sublabel: a.sublabel, path: a.path, icon: a.icon });
			}
		}

		if (!isCommandMode()) {
			for (const p of projects()) {
				items.push({ type: 'project', label: p.name, sublabel: p.description || '', path: `/projects/${p.id}` });
			}
			for (const p of pipelines()) {
				items.push({ type: 'pipeline', label: p.name, sublabel: p.project_id, path: `/projects/${p.project_id}/pipelines/${p.id}` });
			}
			for (const r of runs()) {
				items.push({ type: 'run', label: `Run #${r.number}`, sublabel: `${r.branch || ''} ${r.commit_sha?.slice(0, 7) || ''}`.trim(), path: `/pipelines/${r.pipeline_id}/runs/${r.id}` });
			}
		}

		return items;
	};

	const handleKeyDown = (e: KeyboardEvent) => {
		if (!props.isOpen) return;

		if (e.key === 'Escape') {
			e.preventDefault();
			props.onClose();
			return;
		}

		const items = allItems();
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			setSelectedIndex(prev => Math.min(prev + 1, items.length - 1));
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			setSelectedIndex(prev => Math.max(prev - 1, 0));
		} else if (e.key === 'Enter') {
			e.preventDefault();
			const item = items[selectedIndex()];
			if (item) {
				if (query().trim()) saveRecentSearch(query().trim());
				navigate(item.path);
				props.onClose();
			}
		}
	};

	const handleRecentClick = (search: string) => {
		setQuery(search);
		doSearch(search);
	};

	const handleItemClick = (path: string) => {
		if (query().trim()) saveRecentSearch(query().trim());
		navigate(path);
		props.onClose();
	};

	const typeIcon = (type: string, icon?: string) => {
		if (icon) return <span class="text-base">{icon}</span>;
		switch (type) {
			case 'project':
				return <span class="text-base">📁</span>;
			case 'pipeline':
				return <span class="text-base">🔧</span>;
			case 'run':
				return <span class="text-base">▶️</span>;
			case 'action':
				return <span class="text-base">⚡</span>;
			default:
				return null;
		}
	};

	const typeLabel = (type: string) => {
		switch (type) {
			case 'project': return 'Project';
			case 'pipeline': return 'Pipeline';
			case 'run': return 'Run';
			case 'action': return 'Action';
			default: return '';
		}
	};

	const hasResults = () => projects().length > 0 || pipelines().length > 0 || runs().length > 0 || (isCommandMode() && filteredQuickActions().length > 0);
	const hasQuery = () => query().trim().length > 0;

	return (
		<Show when={props.isOpen}>
			{/* Backdrop */}
			<div
				class="fixed inset-0 z-50 flex items-start justify-center pt-[15vh] bg-black/60 backdrop-blur-sm"
				onClick={(e) => {
					if (e.target === e.currentTarget) props.onClose();
				}}
				onKeyDown={handleKeyDown}
			>
				{/* Modal */}
				<div class="w-full max-w-lg rounded-xl bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] shadow-2xl overflow-hidden">
					{/* Search input */}
					<div class="flex items-center gap-3 px-4 py-3 border-b border-[var(--color-border-primary)]">
						<svg class="w-5 h-5 text-[var(--color-text-tertiary)] flex-shrink-0" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z" clip-rule="evenodd" />
						</svg>
						<input
							ref={inputRef}
							type="text"
							placeholder="Search or type > for commands..."
							class="flex-1 bg-transparent text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] focus:outline-none"
							value={query()}
							onInput={(e) => handleInput(e.currentTarget.value)}
							onKeyDown={handleKeyDown}
						/>
						<Show when={loading()}>
							<div class="w-4 h-4 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
						</Show>
						<kbd class="px-1.5 py-0.5 text-[10px] font-medium rounded bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-tertiary)]">ESC</kbd>
					</div>

					{/* Results area */}
					<div class="max-h-[360px] overflow-y-auto">
						{/* Quick actions when no query */}
						<Show when={!hasQuery()}>
							<div class="px-3 pt-2 pb-1">
								<p class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider px-1">Quick Actions</p>
							</div>
							{(() => {
								let actionIdx = 0;
								return (
									<For each={quickActions.slice(0, 5)}>
										{(action) => {
											const idx = actionIdx++;
											return (
												<button
													class={`w-full text-left px-4 py-2.5 flex items-center gap-3 transition-colors ${selectedIndex() === idx ? 'bg-indigo-500/20 text-[var(--color-text-primary)]' : 'hover:bg-[var(--color-bg-hover)] text-[var(--color-text-secondary)]'}`}
													onClick={() => handleItemClick(action.path)}
													onMouseEnter={() => setSelectedIndex(idx)}
												>
													<span class="text-base">{action.icon}</span>
													<div class="flex-1 min-w-0">
														<p class="text-sm font-medium truncate">{action.label}</p>
														<p class="text-xs text-[var(--color-text-tertiary)] truncate">{action.sublabel}</p>
													</div>
													<span class="text-[10px] text-[var(--color-text-tertiary)]">Action</span>
												</button>
											);
										}}
									</For>
								);
							})()}
							<div class="px-4 py-2 border-t border-[var(--color-border-primary)]">
								<p class="text-[10px] text-[var(--color-text-tertiary)]">
									Type <kbd class="px-1 py-0.5 rounded bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)]">&gt;</kbd> to see all commands
								</p>
							</div>
						</Show>

						{/* Command mode results */}
						<Show when={isCommandMode()}>
							<div class="px-3 pt-2 pb-1">
								<p class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider px-1">Commands</p>
							</div>
							<Show when={filteredQuickActions().length > 0} fallback={
								<div class="flex flex-col items-center justify-center py-8 text-[var(--color-text-tertiary)]">
									<p class="text-sm">No matching commands.</p>
								</div>
							}>
								{(() => {
									let actionIdx = 0;
									return (
										<For each={filteredQuickActions()}>
											{(action) => {
												const idx = actionIdx++;
												return (
													<button
														class={`w-full text-left px-4 py-2.5 flex items-center gap-3 transition-colors ${selectedIndex() === idx ? 'bg-indigo-500/20 text-[var(--color-text-primary)]' : 'hover:bg-[var(--color-bg-hover)] text-[var(--color-text-secondary)]'}`}
														onClick={() => handleItemClick(action.path)}
														onMouseEnter={() => setSelectedIndex(idx)}
													>
														<span class="text-base">{action.icon}</span>
														<div class="flex-1 min-w-0">
															<p class="text-sm font-medium truncate">{action.label}</p>
															<p class="text-xs text-[var(--color-text-tertiary)] truncate">{action.sublabel}</p>
														</div>
														<span class="text-[10px] text-[var(--color-text-tertiary)]">Action</span>
													</button>
												);
											}}
										</For>
									);
								})()}
							</Show>
						</Show>

						{/* Recent searches when no query */}
						<Show when={!hasQuery() && recentSearches().length > 0}>
							<div class="px-4 py-2">
								<p class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider mb-1">Recent Searches</p>
								<For each={recentSearches()}>
									{(search) => (
										<button
											class="w-full text-left px-3 py-2 rounded-lg text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] transition-colors flex items-center gap-2"
											onClick={() => handleRecentClick(search)}
										>
											<svg class="w-4 h-4 text-[var(--color-text-tertiary)]" viewBox="0 0 20 20" fill="currentColor">
												<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm.75-13a.75.75 0 00-1.5 0v5c0 .414.336.75.75.75h4a.75.75 0 000-1.5h-3.25V5z" clip-rule="evenodd" />
											</svg>
											{search}
										</button>
									)}
								</For>
							</div>
						</Show>

						{/* No results */}
						<Show when={hasQuery() && !isCommandMode() && !loading() && !hasResults()}>
							<div class="flex flex-col items-center justify-center py-8 text-[var(--color-text-tertiary)]">
								<svg class="w-8 h-8 mb-2 opacity-50" viewBox="0 0 20 20" fill="currentColor">
									<path fill-rule="evenodd" d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z" clip-rule="evenodd" />
								</svg>
								<p class="text-sm">No results for "{query()}"</p>
							</div>
						</Show>

						{/* Grouped results */}
						<Show when={!isCommandMode() && hasResults() && (projects().length > 0 || pipelines().length > 0 || runs().length > 0)}>
							{(() => {
								let flatIdx = 0;
								return (
									<>
										{/* Projects */}
										<Show when={projects().length > 0}>
											<div class="px-3 pt-2 pb-1">
												<p class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider px-1">Projects</p>
											</div>
											<For each={projects()}>
												{(p) => {
													const idx = flatIdx++;
													return (
														<button
															class={`w-full text-left px-4 py-2.5 flex items-center gap-3 transition-colors ${selectedIndex() === idx ? 'bg-indigo-500/20 text-[var(--color-text-primary)]' : 'hover:bg-[var(--color-bg-hover)] text-[var(--color-text-secondary)]'}`}
															onClick={() => handleItemClick(`/projects/${p.id}`)}
															onMouseEnter={() => setSelectedIndex(idx)}
														>
															{typeIcon('project')}
															<div class="flex-1 min-w-0">
																<p class="text-sm font-medium truncate">{p.name}</p>
																<Show when={p.description}>
																	<p class="text-xs text-[var(--color-text-tertiary)] truncate">{p.description}</p>
																</Show>
															</div>
															<span class="text-[10px] text-[var(--color-text-tertiary)]">{typeLabel('project')}</span>
														</button>
													);
												}}
											</For>
										</Show>

										{/* Pipelines */}
										<Show when={pipelines().length > 0}>
											<div class="px-3 pt-2 pb-1">
												<p class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider px-1">Pipelines</p>
											</div>
											<For each={pipelines()}>
												{(p) => {
													const idx = flatIdx++;
													return (
														<button
															class={`w-full text-left px-4 py-2.5 flex items-center gap-3 transition-colors ${selectedIndex() === idx ? 'bg-indigo-500/20 text-[var(--color-text-primary)]' : 'hover:bg-[var(--color-bg-hover)] text-[var(--color-text-secondary)]'}`}
															onClick={() => handleItemClick(`/projects/${p.project_id}/pipelines/${p.id}`)}
															onMouseEnter={() => setSelectedIndex(idx)}
														>
															{typeIcon('pipeline')}
															<div class="flex-1 min-w-0">
																<p class="text-sm font-medium truncate">{p.name}</p>
															</div>
															<span class="text-[10px] text-[var(--color-text-tertiary)]">{typeLabel('pipeline')}</span>
														</button>
													);
												}}
											</For>
										</Show>

										{/* Runs */}
										<Show when={runs().length > 0}>
											<div class="px-3 pt-2 pb-1">
												<p class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider px-1">Runs</p>
											</div>
											<For each={runs()}>
												{(r) => {
													const idx = flatIdx++;
													return (
														<button
															class={`w-full text-left px-4 py-2.5 flex items-center gap-3 transition-colors ${selectedIndex() === idx ? 'bg-indigo-500/20 text-[var(--color-text-primary)]' : 'hover:bg-[var(--color-bg-hover)] text-[var(--color-text-secondary)]'}`}
															onClick={() => handleItemClick(`/pipelines/${r.pipeline_id}/runs/${r.id}`)}
															onMouseEnter={() => setSelectedIndex(idx)}
														>
															{typeIcon('run')}
															<div class="flex-1 min-w-0">
																<p class="text-sm font-medium truncate">Run #{r.number}</p>
																<p class="text-xs text-[var(--color-text-tertiary)] truncate">
																	{r.branch || ''} {r.commit_sha?.slice(0, 7) || ''}
																</p>
															</div>
															<span class={`px-1.5 py-0.5 text-[10px] font-medium rounded ${r.status === 'success' ? 'bg-green-500/20 text-green-400' : r.status === 'failure' ? 'bg-red-500/20 text-red-400' : r.status === 'running' ? 'bg-yellow-500/20 text-yellow-400' : 'bg-gray-500/20 text-gray-400'}`}>
																{r.status}
															</span>
														</button>
													);
												}}
											</For>
										</Show>
									</>
								);
							})()}
						</Show>
					</div>

					{/* Footer */}
					<div class="flex items-center justify-between px-4 py-2 border-t border-[var(--color-border-primary)] text-[10px] text-[var(--color-text-tertiary)]">
						<div class="flex items-center gap-3">
							<span class="flex items-center gap-1">
								<kbd class="px-1 py-0.5 rounded bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)]">↑↓</kbd>
								navigate
							</span>
							<span class="flex items-center gap-1">
								<kbd class="px-1 py-0.5 rounded bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)]">↵</kbd>
								select
							</span>
							<span class="flex items-center gap-1">
								<kbd class="px-1 py-0.5 rounded bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)]">esc</kbd>
								close
							</span>
							<span class="flex items-center gap-1">
								<kbd class="px-1 py-0.5 rounded bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)]">&gt;</kbd>
								commands
							</span>
						</div>
					</div>
				</div>
			</div>
		</Show>
	);
};

export default CommandPalette;
