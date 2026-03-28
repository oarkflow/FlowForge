import type { Component } from 'solid-js';
import { createSignal, createResource, For, Show, createEffect } from 'solid-js';
import PageContainer from '../../components/layout/PageContainer';
import Badge from '../../components/ui/Badge';
import Button from '../../components/ui/Button';
import Input from '../../components/ui/Input';
import Modal from '../../components/ui/Modal';
import Select from '../../components/ui/Select';
import { toast } from '../../components/ui/Toast';
import { api, ApiRequestError } from '../../api/client';
import type { Agent, AgentStatus, ScalingPolicy, ScalingEvent, ScalingMetrics } from '../../types';
import { formatRelativeTime, getAgentStatusVariant, copyToClipboard } from '../../utils/helpers';

// ---------------------------------------------------------------------------
// Fetchers
// ---------------------------------------------------------------------------
async function fetchAgents() {
	return api.agents.list();
}

async function fetchScalingPolicies() {
	return api.scaling.listPolicies();
}

async function fetchScalingMetrics() {
	return api.scaling.getMetrics();
}

async function fetchRecentEvents() {
	return api.scaling.listRecentEvents();
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const AgentsPage: Component = () => {
	// Tab state
	const [activeTab, setActiveTab] = createSignal<'agents' | 'scaling'>('agents');

	// --- Agents tab state ---
	const [agents, { refetch, mutate }] = createResource(fetchAgents);
	const [search, setSearch] = createSignal('');
	const [filterStatus, setFilterStatus] = createSignal('all');
	const [showRegister, setShowRegister] = createSignal(false);
	const [showDetail, setShowDetail] = createSignal<Agent | null>(null);
	const [registering, setRegistering] = createSignal(false);

	// Register form
	const [newAgentName, setNewAgentName] = createSignal('');
	const [newAgentExecutor, setNewAgentExecutor] = createSignal('docker');
	const [newAgentLabels, setNewAgentLabels] = createSignal('');
	const [newAgentToken, setNewAgentToken] = createSignal('');
	const [registerError, setRegisterError] = createSignal('');

	// --- Scaling tab state ---
	const [policies, { refetch: refetchPolicies }] = createResource(() => activeTab() === 'scaling', (shouldFetch) => shouldFetch ? fetchScalingPolicies() : Promise.resolve([]));
	const [metrics, { refetch: refetchMetrics }] = createResource(() => activeTab() === 'scaling', (shouldFetch) => shouldFetch ? fetchScalingMetrics() : Promise.resolve(undefined));
	const [recentEvents, { refetch: refetchEvents }] = createResource(() => activeTab() === 'scaling', (shouldFetch) => shouldFetch ? fetchRecentEvents() : Promise.resolve([]));
	const [showPolicyModal, setShowPolicyModal] = createSignal(false);
	const [editingPolicy, setEditingPolicy] = createSignal<ScalingPolicy | null>(null);
	const [showEventsFor, setShowEventsFor] = createSignal<string | null>(null);
	const [policyEvents, setPolicyEvents] = createSignal<ScalingEvent[]>([]);
	const [savingPolicy, setSavingPolicy] = createSignal(false);

	// Policy form fields
	const [policyName, setPolicyName] = createSignal('');
	const [policyDescription, setPolicyDescription] = createSignal('');
	const [policyExecutor, setPolicyExecutor] = createSignal('docker');
	const [policyLabels, setPolicyLabels] = createSignal('');
	const [policyMinAgents, setPolicyMinAgents] = createSignal(1);
	const [policyMaxAgents, setPolicyMaxAgents] = createSignal(10);
	const [policyScaleUpThreshold, setPolicyScaleUpThreshold] = createSignal(5);
	const [policyScaleDownThreshold, setPolicyScaleDownThreshold] = createSignal(0);
	const [policyScaleUpStep, setPolicyScaleUpStep] = createSignal(1);
	const [policyScaleDownStep, setPolicyScaleDownStep] = createSignal(1);
	const [policyCooldown, setPolicyCooldown] = createSignal(300);

	const agentList = () => agents() ?? [];

	const filteredAgents = () => {
		let result = agentList();
		const q = search().toLowerCase();
		if (q) {
			result = result.filter(a => a.name.toLowerCase().includes(q) || a.labels.some(l => l.includes(q)));
		}
		if (filterStatus() !== 'all') {
			result = result.filter(a => a.status === filterStatus());
		}
		return result;
	};

	const counts = () => {
		const c = { online: 0, busy: 0, draining: 0, offline: 0 };
		agentList().forEach(a => c[a.status]++);
		return c;
	};

	const handleRegister = async () => {
		if (!newAgentName().trim()) return;
		setRegistering(true);
		setRegisterError('');
		try {
			const labels = newAgentLabels().split(',').map(l => l.trim()).filter(Boolean);
			const result = await api.agents.create({
				name: newAgentName().trim(),
				executor: newAgentExecutor() as Agent['executor'],
				labels,
			});
			setNewAgentToken(result.token);
			toast.success(`Agent "${newAgentName()}" registered`);
			refetch();
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to register agent';
			setRegisterError(msg);
			toast.error(msg);
		} finally {
			setRegistering(false);
		}
	};

	const handleDrain = async (agent: Agent) => {
		try {
			await api.agents.drain(agent.id);
			mutate(prev => prev?.map(a => a.id === agent.id ? { ...a, status: 'draining' as AgentStatus } : a));
			toast.info(`Agent "${agent.name}" is draining`);
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to drain agent');
		}
	};

	const handleDelete = async (agent: Agent) => {
		try {
			await api.agents.delete(agent.id);
			mutate(prev => prev?.filter(a => a.id !== agent.id));
			toast.success(`Agent "${agent.name}" removed`);
			setShowDetail(null);
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to remove agent');
		}
	};

	// --- Scaling policy handlers ---
	const openCreatePolicy = () => {
		setEditingPolicy(null);
		setPolicyName('');
		setPolicyDescription('');
		setPolicyExecutor('docker');
		setPolicyLabels('');
		setPolicyMinAgents(1);
		setPolicyMaxAgents(10);
		setPolicyScaleUpThreshold(5);
		setPolicyScaleDownThreshold(0);
		setPolicyScaleUpStep(1);
		setPolicyScaleDownStep(1);
		setPolicyCooldown(300);
		setShowPolicyModal(true);
	};

	const openEditPolicy = (policy: ScalingPolicy) => {
		setEditingPolicy(policy);
		setPolicyName(policy.name);
		setPolicyDescription(policy.description);
		setPolicyExecutor(policy.executor_type);
		setPolicyLabels(policy.labels);
		setPolicyMinAgents(policy.min_agents);
		setPolicyMaxAgents(policy.max_agents);
		setPolicyScaleUpThreshold(policy.scale_up_threshold);
		setPolicyScaleDownThreshold(policy.scale_down_threshold);
		setPolicyScaleUpStep(policy.scale_up_step);
		setPolicyScaleDownStep(policy.scale_down_step);
		setPolicyCooldown(policy.cooldown_seconds);
		setShowPolicyModal(true);
	};

	const handleSavePolicy = async () => {
		if (!policyName().trim()) return;
		if (policyMinAgents() > policyMaxAgents()) {
			toast.error('Min agents cannot exceed max agents');
			return;
		}
		setSavingPolicy(true);
		try {
			const data: Partial<ScalingPolicy> = {
				name: policyName().trim(),
				description: policyDescription(),
				executor_type: policyExecutor(),
				labels: policyLabels(),
				min_agents: policyMinAgents(),
				max_agents: policyMaxAgents(),
				scale_up_threshold: policyScaleUpThreshold(),
				scale_down_threshold: policyScaleDownThreshold(),
				scale_up_step: policyScaleUpStep(),
				scale_down_step: policyScaleDownStep(),
				cooldown_seconds: policyCooldown(),
			};
			if (editingPolicy()) {
				await api.scaling.updatePolicy(editingPolicy()!.id, data);
				toast.success(`Policy "${policyName()}" updated`);
			} else {
				await api.scaling.createPolicy(data);
				toast.success(`Policy "${policyName()}" created`);
			}
			setShowPolicyModal(false);
			refetchPolicies();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to save policy');
		} finally {
			setSavingPolicy(false);
		}
	};

	const handleDeletePolicy = async (policy: ScalingPolicy) => {
		try {
			await api.scaling.deletePolicy(policy.id);
			toast.success(`Policy "${policy.name}" deleted`);
			refetchPolicies();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to delete policy');
		}
	};

	const handleTogglePolicy = async (policy: ScalingPolicy) => {
		try {
			if (policy.enabled) {
				await api.scaling.disablePolicy(policy.id);
				toast.info(`Policy "${policy.name}" disabled`);
			} else {
				await api.scaling.enablePolicy(policy.id);
				toast.success(`Policy "${policy.name}" enabled`);
			}
			refetchPolicies();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to toggle policy');
		}
	};

	const handleViewEvents = async (policyId: string) => {
		setShowEventsFor(policyId);
		try {
			const events = await api.scaling.listEvents(policyId);
			setPolicyEvents(events);
		} catch {
			setPolicyEvents([]);
		}
	};

	const policyList = () => policies() ?? [];
	const enabledPoliciesCount = () => policyList().filter(p => p.enabled).length;

	const getActionIcon = (action: string) => {
		if (action === 'scale_up') return '↑';
		if (action === 'scale_down') return '↓';
		return '—';
	};

	const getActionColor = (action: string) => {
		if (action === 'scale_up') return 'text-emerald-400';
		if (action === 'scale_down') return 'text-amber-400';
		return 'text-gray-500';
	};

	return (
		<PageContainer
			title="Agents"
			description="Manage build agents, workers, and auto-scaling"
			actions={
				<div class="flex items-center gap-2">
					<Show when={activeTab() === 'agents'}>
						<Button
							onClick={() => { setShowRegister(true); setNewAgentToken(''); setNewAgentName(''); setNewAgentLabels(''); setRegisterError(''); }}
							icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>}
						>
							Register Agent
						</Button>
					</Show>
					<Show when={activeTab() === 'scaling'}>
						<Button
							onClick={openCreatePolicy}
							icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>}
						>
							Create Policy
						</Button>
					</Show>
				</div>
			}
		>
			{/* Tab Navigation */}
			<div class="flex items-center gap-1 mb-6 border-b border-[var(--color-border-primary)]">
				<button
					class={`px-4 py-2.5 text-sm font-medium transition-colors relative ${activeTab() === 'agents'
							? 'text-[var(--color-accent)] after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:bg-[var(--color-accent)]'
							: 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'
						}`}
					onClick={() => setActiveTab('agents')}
				>
					Agents
				</button>
				<button
					class={`px-4 py-2.5 text-sm font-medium transition-colors relative ${activeTab() === 'scaling'
							? 'text-[var(--color-accent)] after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:bg-[var(--color-accent)]'
							: 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'
						}`}
					onClick={() => setActiveTab('scaling')}
				>
					Auto-Scaling
				</button>
			</div>

			{/* ===== AGENTS TAB ===== */}
			<Show when={activeTab() === 'agents'}>
				{/* Error state */}
				<Show when={agents.error}>
					<div class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30 flex items-center justify-between">
						<p class="text-sm text-red-400">Failed to load agents: {(agents.error as Error)?.message}</p>
						<Button size="sm" variant="outline" onClick={refetch}>Retry</Button>
					</div>
				</Show>

				{/* Summary cards */}
				<div class="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-6">
					<button onClick={() => setFilterStatus(filterStatus() === 'online' ? 'all' : 'online')} class={`bg-[var(--color-bg-secondary)] border rounded-xl p-4 text-center transition-colors ${filterStatus() === 'online' ? 'border-emerald-500/50' : 'border-[var(--color-border-primary)] hover:border-[var(--color-border-secondary)]'}`}>
						<p class="text-2xl font-bold text-emerald-400">{counts().online}</p>
						<p class="text-xs text-[var(--color-text-tertiary)] mt-1">Online</p>
					</button>
					<button onClick={() => setFilterStatus(filterStatus() === 'busy' ? 'all' : 'busy')} class={`bg-[var(--color-bg-secondary)] border rounded-xl p-4 text-center transition-colors ${filterStatus() === 'busy' ? 'border-violet-500/50' : 'border-[var(--color-border-primary)] hover:border-[var(--color-border-secondary)]'}`}>
						<p class="text-2xl font-bold text-violet-400">{counts().busy}</p>
						<p class="text-xs text-[var(--color-text-tertiary)] mt-1">Busy</p>
					</button>
					<button onClick={() => setFilterStatus(filterStatus() === 'draining' ? 'all' : 'draining')} class={`bg-[var(--color-bg-secondary)] border rounded-xl p-4 text-center transition-colors ${filterStatus() === 'draining' ? 'border-amber-500/50' : 'border-[var(--color-border-primary)] hover:border-[var(--color-border-secondary)]'}`}>
						<p class="text-2xl font-bold text-amber-400">{counts().draining}</p>
						<p class="text-xs text-[var(--color-text-tertiary)] mt-1">Draining</p>
					</button>
					<button onClick={() => setFilterStatus(filterStatus() === 'offline' ? 'all' : 'offline')} class={`bg-[var(--color-bg-secondary)] border rounded-xl p-4 text-center transition-colors ${filterStatus() === 'offline' ? 'border-gray-500/50' : 'border-[var(--color-border-primary)] hover:border-[var(--color-border-secondary)]'}`}>
						<p class="text-2xl font-bold text-gray-500">{counts().offline}</p>
						<p class="text-xs text-[var(--color-text-tertiary)] mt-1">Offline</p>
					</button>
				</div>

				{/* Search */}
				<div class="mb-6">
					<Input
						placeholder="Search agents by name or label..."
						value={search()}
						onInput={(e) => setSearch(e.currentTarget.value)}
						icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z" clip-rule="evenodd" /></svg>}
					/>
				</div>

				<Show when={!agents.loading} fallback={
					<div class="space-y-3">
						<For each={[1, 2, 3]}>{() => <div class="h-20 bg-[var(--color-bg-secondary)] rounded-xl animate-pulse" />}</For>
					</div>
				}>
					<Show when={filteredAgents().length > 0} fallback={
						<div class="text-center py-16">
							<svg class="w-12 h-12 mx-auto text-[var(--color-text-tertiary)] mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
								<path stroke-linecap="round" stroke-linejoin="round" d="M8.25 3v1.5M4.5 8.25H3m18 0h-1.5M4.5 12H3m18 0h-1.5m-15 3.75H3m18 0h-1.5M8.25 19.5V21M12 3v1.5m0 15V21m3.75-18v1.5m0 15V21m-9-1.5h10.5a2.25 2.25 0 002.25-2.25V6.75a2.25 2.25 0 00-2.25-2.25H6.75A2.25 2.25 0 004.5 6.75v10.5a2.25 2.25 0 002.25 2.25z" />
							</svg>
							<p class="text-[var(--color-text-secondary)]">No agents found</p>
							<Show when={search() || filterStatus() !== 'all'}>
								<Button variant="ghost" class="mt-2" onClick={() => { setSearch(''); setFilterStatus('all'); }}>Clear filters</Button>
							</Show>
						</div>
					}>
						<div class="space-y-3">
							<For each={filteredAgents()}>
								{(agent) => (
									<div
										class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-5 hover:border-[var(--color-border-secondary)] transition-all cursor-pointer"
										onClick={() => setShowDetail(agent)}
									>
										<div class="flex items-center justify-between">
											<div class="flex items-center gap-4">
												<div class={`w-10 h-10 rounded-lg flex items-center justify-center ${agent.status === 'online' ? 'bg-emerald-500/10' :
														agent.status === 'busy' ? 'bg-violet-500/10' :
															agent.status === 'draining' ? 'bg-amber-500/10' : 'bg-gray-500/10'
													}`}>
													<svg class={`w-5 h-5 ${agent.status === 'online' ? 'text-emerald-400' :
															agent.status === 'busy' ? 'text-violet-400' :
																agent.status === 'draining' ? 'text-amber-400' : 'text-gray-500'
														}`} viewBox="0 0 20 20" fill="currentColor">
														<path d="M14 6a2.5 2.5 0 00-4-2 2.5 2.5 0 00-4 2H4.5A1.5 1.5 0 003 7.5v8A1.5 1.5 0 004.5 17h11a1.5 1.5 0 001.5-1.5v-8A1.5 1.5 0 0015.5 6H14zM8 8.5a1 1 0 11-2 0 1 1 0 012 0zm5 0a1 1 0 11-2 0 1 1 0 012 0zM7 11a1 1 0 000 2h6a1 1 0 100-2H7z" />
													</svg>
												</div>
												<div>
													<div class="flex items-center gap-2">
														<p class="text-sm font-semibold text-[var(--color-text-primary)] font-mono">{agent.name}</p>
														<Badge variant={getAgentStatusVariant(agent.status)} dot size="sm">{agent.status}</Badge>
													</div>
													<div class="flex items-center gap-3 mt-1">
														<span class="text-xs text-[var(--color-text-tertiary)]">{agent.os}/{agent.arch}</span>
														<span class="text-xs text-[var(--color-text-tertiary)]">{agent.executor}</span>
														<Show when={agent.version}>
															<span class="text-xs text-[var(--color-text-tertiary)]">v{agent.version}</span>
														</Show>
														<Show when={agent.ip_address}>
															<span class="text-xs text-[var(--color-text-tertiary)]">{agent.ip_address}</span>
														</Show>
													</div>
												</div>
											</div>

											<div class="flex items-center gap-4">
												<div class="text-right hidden sm:block">
													<p class="text-xs text-[var(--color-text-secondary)]">{agent.cpu_cores} CPU · {Math.round((agent.memory_mb || 0) / 1024)}GB RAM</p>
													<p class="text-xs text-[var(--color-text-tertiary)] mt-0.5">
														Last seen: {agent.last_seen_at ? formatRelativeTime(agent.last_seen_at) : 'Never'}
													</p>
												</div>
												<div class="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
													<Show when={agent.status === 'online'}>
														<Button size="sm" variant="ghost" onClick={() => handleDrain(agent)}>Drain</Button>
													</Show>
													<Button size="sm" variant="danger" onClick={() => handleDelete(agent)}>Remove</Button>
												</div>
											</div>
										</div>

										<div class="flex flex-wrap gap-1.5 mt-3">
											<For each={agent.labels}>
												{(label) => (
													<span class="text-xs px-2 py-0.5 rounded-full bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] border border-[var(--color-border-primary)]">{label}</span>
												)}
											</For>
										</div>
									</div>
								)}
							</For>
						</div>
					</Show>
				</Show>
			</Show>

			{/* ===== AUTO-SCALING TAB ===== */}
			<Show when={activeTab() === 'scaling'}>
				{/* Metrics Overview */}
				<div class="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-6">
					<div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-4 text-center">
						<p class="text-2xl font-bold text-blue-400">{metrics()?.queue_depth ?? 0}</p>
						<p class="text-xs text-[var(--color-text-tertiary)] mt-1">Queue Depth</p>
					</div>
					<div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-4 text-center">
						<p class="text-2xl font-bold text-emerald-400">{metrics()?.online_agents ?? 0}<span class="text-sm text-[var(--color-text-tertiary)]">/{metrics()?.total_agents ?? 0}</span></p>
						<p class="text-xs text-[var(--color-text-tertiary)] mt-1">Online / Total</p>
					</div>
					<div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-4 text-center">
						<p class="text-2xl font-bold text-violet-400">{metrics()?.busy_agents ?? 0}</p>
						<p class="text-xs text-[var(--color-text-tertiary)] mt-1">Busy Agents</p>
					</div>
					<div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-4 text-center">
						<p class="text-2xl font-bold text-cyan-400">{enabledPoliciesCount()}</p>
						<p class="text-xs text-[var(--color-text-tertiary)] mt-1">Active Policies</p>
					</div>
				</div>

				{/* Agents by executor type breakdown */}
				<Show when={metrics()?.agents_by_executor && Object.keys(metrics()!.agents_by_executor).length > 0}>
					<div class="mb-6 bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-4">
						<p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider mb-3">Agents by Executor</p>
						<div class="flex flex-wrap gap-4">
							<For each={Object.entries(metrics()?.agents_by_executor || {})}>
								{([exec, count]) => (
									<div class="flex items-center gap-2">
										<span class="text-sm font-mono text-[var(--color-text-secondary)]">{exec}</span>
										<Badge variant="default" size="sm">{count as number}</Badge>
									</div>
								)}
							</For>
						</div>
					</div>
				</Show>

				{/* Scaling Policies */}
				<div class="mb-6">
					<div class="flex items-center justify-between mb-4">
						<h3 class="text-sm font-semibold text-[var(--color-text-primary)]">Scaling Policies</h3>
						<Button size="sm" variant="outline" onClick={() => { refetchPolicies(); refetchMetrics(); }}>
							Refresh
						</Button>
					</div>

					<Show when={!policies.loading} fallback={
						<div class="space-y-3">
							<For each={[1, 2]}>{() => <div class="h-32 bg-[var(--color-bg-secondary)] rounded-xl animate-pulse" />}</For>
						</div>
					}>
						<Show when={policyList().length > 0} fallback={
							<div class="text-center py-12 bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl">
								<svg class="w-12 h-12 mx-auto text-[var(--color-text-tertiary)] mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
									<path stroke-linecap="round" stroke-linejoin="round" d="M3.75 3v11.25A2.25 2.25 0 006 16.5h2.25M3.75 3h-1.5m1.5 0h16.5m0 0h1.5m-1.5 0v11.25A2.25 2.25 0 0118 16.5h-2.25m-7.5 0h7.5m-7.5 0l-1 3m8.5-3l1 3m0 0l.5 1.5m-.5-1.5h-9.5m0 0l-.5 1.5" />
								</svg>
								<p class="text-[var(--color-text-secondary)] mb-2">No scaling policies configured</p>
								<p class="text-xs text-[var(--color-text-tertiary)] mb-4">Create a policy to automatically scale agent capacity based on queue depth</p>
								<Button onClick={openCreatePolicy}>Create First Policy</Button>
							</div>
						}>
							<div class="space-y-3">
								<For each={policyList()}>
									{(policy) => (
										<div class={`bg-[var(--color-bg-secondary)] border rounded-xl p-5 transition-all ${policy.enabled ? 'border-[var(--color-border-primary)] hover:border-[var(--color-border-secondary)]' : 'border-[var(--color-border-primary)] opacity-60'}`}>
											<div class="flex items-start justify-between mb-3">
												<div>
													<div class="flex items-center gap-2">
														<p class="text-sm font-semibold text-[var(--color-text-primary)]">{policy.name}</p>
														<Badge variant={policy.enabled ? 'success' : 'default'} size="sm">{policy.enabled ? 'Enabled' : 'Disabled'}</Badge>
														<Badge variant="default" size="sm">{policy.executor_type}</Badge>
													</div>
													<Show when={policy.description}>
														<p class="text-xs text-[var(--color-text-tertiary)] mt-1">{policy.description}</p>
													</Show>
												</div>
												<div class="flex items-center gap-1">
													<Button size="sm" variant="ghost" onClick={() => handleViewEvents(policy.id)}>Events</Button>
													<Button size="sm" variant="ghost" onClick={() => openEditPolicy(policy)}>Edit</Button>
													<Button size="sm" variant="ghost" onClick={() => handleTogglePolicy(policy)}>
														{policy.enabled ? 'Disable' : 'Enable'}
													</Button>
													<Button size="sm" variant="danger" onClick={() => handleDeletePolicy(policy)}>Delete</Button>
												</div>
											</div>

											<div class="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-6 gap-4">
												<div>
													<p class="text-xs text-[var(--color-text-tertiary)]">Agents Range</p>
													<p class="text-sm text-[var(--color-text-primary)] font-mono">{policy.min_agents} – {policy.max_agents}</p>
												</div>
												<div>
													<p class="text-xs text-[var(--color-text-tertiary)]">Desired</p>
													<p class="text-sm text-[var(--color-text-primary)] font-mono">{policy.desired_agents}</p>
												</div>
												<div>
													<p class="text-xs text-[var(--color-text-tertiary)]">Scale Up When</p>
													<p class="text-sm text-[var(--color-text-primary)] font-mono">queue &gt; {policy.scale_up_threshold}</p>
												</div>
												<div>
													<p class="text-xs text-[var(--color-text-tertiary)]">Scale Down When</p>
													<p class="text-sm text-[var(--color-text-primary)] font-mono">queue ≤ {policy.scale_down_threshold}</p>
												</div>
												<div>
													<p class="text-xs text-[var(--color-text-tertiary)]">Queue / Active</p>
													<p class="text-sm text-[var(--color-text-primary)] font-mono">{policy.queue_depth} / {policy.active_agents}</p>
												</div>
												<div>
													<p class="text-xs text-[var(--color-text-tertiary)]">Cooldown</p>
													<p class="text-sm text-[var(--color-text-primary)] font-mono">{policy.cooldown_seconds}s</p>
												</div>
											</div>

											<Show when={policy.labels}>
												<div class="flex flex-wrap gap-1.5 mt-3">
													<For each={policy.labels.split(',').filter(l => l.trim())}>
														{(label) => (
															<span class="text-xs px-2 py-0.5 rounded-full bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] border border-[var(--color-border-primary)]">{label.trim()}</span>
														)}
													</For>
												</div>
											</Show>

											<Show when={policy.last_scale_action}>
												<div class="mt-3 pt-3 border-t border-[var(--color-border-primary)] flex items-center gap-2">
													<span class={`text-sm font-bold ${getActionColor(policy.last_scale_action)}`}>{getActionIcon(policy.last_scale_action)}</span>
													<span class="text-xs text-[var(--color-text-tertiary)]">
														Last action: <span class="text-[var(--color-text-secondary)]">{policy.last_scale_action.replace('_', ' ')}</span>
														{policy.last_scale_at && ` · ${formatRelativeTime(policy.last_scale_at)}`}
													</span>
												</div>
											</Show>
										</div>
									)}
								</For>
							</div>
						</Show>
					</Show>
				</div>

				{/* Recent Scaling Events */}
				<div>
					<h3 class="text-sm font-semibold text-[var(--color-text-primary)] mb-4">Recent Scaling Events</h3>
					<Show when={!recentEvents.loading} fallback={
						<div class="h-24 bg-[var(--color-bg-secondary)] rounded-xl animate-pulse" />
					}>
						<Show when={(recentEvents() ?? []).length > 0} fallback={
							<div class="text-center py-8 bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl">
								<p class="text-xs text-[var(--color-text-tertiary)]">No scaling events recorded yet</p>
							</div>
						}>
							<div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl overflow-hidden">
								<table class="w-full">
									<thead>
										<tr class="border-b border-[var(--color-border-primary)]">
											<th class="text-left text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider px-4 py-3">Action</th>
											<th class="text-left text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider px-4 py-3">From → To</th>
											<th class="text-left text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider px-4 py-3 hidden sm:table-cell">Reason</th>
											<th class="text-left text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider px-4 py-3 hidden md:table-cell">Queue</th>
											<th class="text-right text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider px-4 py-3">Time</th>
										</tr>
									</thead>
									<tbody>
										<For each={(recentEvents() ?? []).slice(0, 20)}>
											{(event) => (
												<tr class="border-b border-[var(--color-border-primary)] last:border-b-0">
													<td class="px-4 py-3">
														<div class="flex items-center gap-2">
															<span class={`text-lg font-bold ${getActionColor(event.action)}`}>{getActionIcon(event.action)}</span>
															<span class="text-xs text-[var(--color-text-secondary)]">{event.action.replace('_', ' ')}</span>
														</div>
													</td>
													<td class="px-4 py-3">
														<span class="text-sm font-mono text-[var(--color-text-primary)]">{event.from_count} → {event.to_count}</span>
													</td>
													<td class="px-4 py-3 hidden sm:table-cell">
														<span class="text-xs text-[var(--color-text-tertiary)] max-w-xs truncate block">{event.reason}</span>
													</td>
													<td class="px-4 py-3 hidden md:table-cell">
														<span class="text-xs text-[var(--color-text-tertiary)]">{event.queue_depth} jobs / {event.active_agents} agents</span>
													</td>
													<td class="px-4 py-3 text-right">
														<span class="text-xs text-[var(--color-text-tertiary)]">{formatRelativeTime(event.created_at)}</span>
													</td>
												</tr>
											)}
										</For>
									</tbody>
								</table>
							</div>
						</Show>
					</Show>
				</div>
			</Show>

			{/* ===== MODALS ===== */}

			{/* Register Agent Modal */}
			<Show when={showRegister()}>
				<Modal open={showRegister()} onClose={() => setShowRegister(false)} title="Register New Agent" description="Add a build agent to the pool" footer={
					<Show when={!newAgentToken()} fallback={
						<Button onClick={() => setShowRegister(false)}>Done</Button>
					}>
						<Button variant="ghost" onClick={() => setShowRegister(false)}>Cancel</Button>
						<Button onClick={handleRegister} disabled={!newAgentName().trim()} loading={registering()}>Register</Button>
					</Show>
				}>
					<Show when={!newAgentToken()} fallback={
						<div class="space-y-4">
							<div class="p-4 rounded-lg bg-emerald-500/10 border border-emerald-500/30">
								<p class="text-sm text-emerald-400 font-medium mb-2">Agent registered!</p>
								<p class="text-xs text-[var(--color-text-tertiary)] mb-3">Use this token to connect the agent. It won't be shown again.</p>
								<div class="flex items-center gap-2">
									<code class="flex-1 text-xs font-mono bg-[var(--color-bg-primary)] px-3 py-2 rounded border border-[var(--color-border-primary)] text-[var(--color-text-primary)] break-all">{newAgentToken()}</code>
									<Button size="sm" variant="outline" onClick={() => { copyToClipboard(newAgentToken()); toast.success('Copied!'); }}>Copy</Button>
								</div>
							</div>
							<div class="p-4 rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)]">
								<p class="text-xs font-medium text-[var(--color-text-secondary)] mb-2">Quick Start</p>
								<pre class="text-xs font-mono text-[var(--color-text-tertiary)] whitespace-pre-wrap">
									{`# Run the agent
flowforge-agent \\
  --server https://flowforge.example.com \\
  --token ${newAgentToken()} \\
  --name ${newAgentName()} \\
  --executor ${newAgentExecutor()}`}</pre>
							</div>
						</div>
					}>
						<div class="space-y-4">
							<Show when={registerError()}>
								<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-sm text-red-400">{registerError()}</div>
							</Show>
							<Input label="Agent Name" placeholder="e.g. agent-linux-04" value={newAgentName()} onInput={(e) => setNewAgentName(e.currentTarget.value)} />
							<Select label="Executor Type" value={newAgentExecutor()} onChange={(e) => setNewAgentExecutor(e.currentTarget.value)} options={[
								{ value: 'docker', label: 'Docker' },
								{ value: 'kubernetes', label: 'Kubernetes' },
								{ value: 'local', label: 'Local Process' },
							]} />
							<Input label="Labels" placeholder="docker, linux, amd64 (comma separated)" value={newAgentLabels()} onInput={(e) => setNewAgentLabels(e.currentTarget.value)} hint="Labels help match jobs to compatible agents" />
						</div>
					</Show>
				</Modal>
			</Show>

			{/* Agent Detail Modal */}
			<Show when={showDetail()}>
				<Modal open={!!showDetail()} onClose={() => setShowDetail(null)} title={showDetail()!.name} size="lg" footer={
					<div class="flex items-center gap-2">
						<Show when={showDetail()!.status === 'online'}>
							<Button variant="outline" onClick={() => { handleDrain(showDetail()!); setShowDetail(null); }}>Drain Agent</Button>
						</Show>
						<Button variant="danger" onClick={() => handleDelete(showDetail()!)}>Remove Agent</Button>
					</div>
				}>
					<div class="grid grid-cols-2 gap-6">
						<div class="space-y-4">
							<div>
								<p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider mb-1">Status</p>
								<Badge variant={getAgentStatusVariant(showDetail()!.status)} dot>{showDetail()!.status}</Badge>
							</div>
							<div>
								<p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider mb-1">Executor</p>
								<p class="text-sm text-[var(--color-text-primary)]">{showDetail()!.executor}</p>
							</div>
							<div>
								<p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider mb-1">Version</p>
								<p class="text-sm text-[var(--color-text-primary)]">{showDetail()!.version || '-'}</p>
							</div>
							<div>
								<p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider mb-1">IP Address</p>
								<p class="text-sm font-mono text-[var(--color-text-primary)]">{showDetail()!.ip_address || '-'}</p>
							</div>
						</div>
						<div class="space-y-4">
							<div>
								<p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider mb-1">Platform</p>
								<p class="text-sm text-[var(--color-text-primary)]">{showDetail()!.os} / {showDetail()!.arch}</p>
							</div>
							<div>
								<p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider mb-1">CPU Cores</p>
								<p class="text-sm text-[var(--color-text-primary)]">{showDetail()!.cpu_cores || '-'}</p>
							</div>
							<div>
								<p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider mb-1">Memory</p>
								<p class="text-sm text-[var(--color-text-primary)]">{showDetail()!.memory_mb ? `${Math.round(showDetail()!.memory_mb! / 1024)} GB` : '-'}</p>
							</div>
							<div>
								<p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider mb-1">Last Seen</p>
								<p class="text-sm text-[var(--color-text-primary)]">{showDetail()!.last_seen_at ? formatRelativeTime(showDetail()!.last_seen_at!) : 'Never'}</p>
							</div>
						</div>
					</div>
					<div class="mt-4">
						<p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider mb-2">Labels</p>
						<div class="flex flex-wrap gap-1.5">
							<For each={showDetail()!.labels}>
								{(label) => (
									<span class="text-xs px-2 py-1 rounded-full bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] border border-[var(--color-border-primary)]">{label}</span>
								)}
							</For>
						</div>
					</div>
				</Modal>
			</Show>

			{/* Create/Edit Policy Modal */}
			<Show when={showPolicyModal()}>
				<Modal
					open={showPolicyModal()}
					onClose={() => setShowPolicyModal(false)}
					title={editingPolicy() ? 'Edit Scaling Policy' : 'Create Scaling Policy'}
					description={editingPolicy() ? 'Modify auto-scaling behavior' : 'Define rules for automatic agent scaling'}
					size="lg"
					footer={
						<>
							<Button variant="ghost" onClick={() => setShowPolicyModal(false)}>Cancel</Button>
							<Button onClick={handleSavePolicy} disabled={!policyName().trim()} loading={savingPolicy()}>
								{editingPolicy() ? 'Update Policy' : 'Create Policy'}
							</Button>
						</>
					}
				>
					<div class="space-y-4">
						<Input
							label="Policy Name"
							placeholder="e.g. Docker Auto-Scale"
							value={policyName()}
							onInput={(e) => setPolicyName(e.currentTarget.value)}
						/>
						<Input
							label="Description"
							placeholder="Scale Docker agents based on queue depth"
							value={policyDescription()}
							onInput={(e) => setPolicyDescription(e.currentTarget.value)}
						/>

						<div class="grid grid-cols-2 gap-4">
							<Select
								label="Executor Type"
								value={policyExecutor()}
								onChange={(e) => setPolicyExecutor(e.currentTarget.value)}
								options={[
									{ value: 'docker', label: 'Docker' },
									{ value: 'kubernetes', label: 'Kubernetes' },
									{ value: 'local', label: 'Local' },
								]}
							/>
							<Input
								label="Labels"
								placeholder="linux, amd64 (comma separated)"
								value={policyLabels()}
								onInput={(e) => setPolicyLabels(e.currentTarget.value)}
								hint="Match agents with these labels"
							/>
						</div>

						<div class="grid grid-cols-2 gap-4">
							<Input
								label="Min Agents"
								type="number"
								value={String(policyMinAgents())}
								onInput={(e) => setPolicyMinAgents(Math.max(0, parseInt(e.currentTarget.value) || 0))}
								hint="Minimum agent count"
							/>
							<Input
								label="Max Agents"
								type="number"
								value={String(policyMaxAgents())}
								onInput={(e) => setPolicyMaxAgents(Math.max(1, parseInt(e.currentTarget.value) || 1))}
								hint="Maximum agent count"
							/>
						</div>

						<div class="grid grid-cols-2 gap-4">
							<Input
								label="Scale Up Threshold"
								type="number"
								value={String(policyScaleUpThreshold())}
								onInput={(e) => setPolicyScaleUpThreshold(Math.max(0, parseInt(e.currentTarget.value) || 0))}
								hint="Queue depth that triggers scale up"
							/>
							<Input
								label="Scale Down Threshold"
								type="number"
								value={String(policyScaleDownThreshold())}
								onInput={(e) => setPolicyScaleDownThreshold(Math.max(0, parseInt(e.currentTarget.value) || 0))}
								hint="Queue depth that triggers scale down"
							/>
						</div>

						<div class="grid grid-cols-3 gap-4">
							<Input
								label="Scale Up Step"
								type="number"
								value={String(policyScaleUpStep())}
								onInput={(e) => setPolicyScaleUpStep(Math.max(1, parseInt(e.currentTarget.value) || 1))}
								hint="Agents to add"
							/>
							<Input
								label="Scale Down Step"
								type="number"
								value={String(policyScaleDownStep())}
								onInput={(e) => setPolicyScaleDownStep(Math.max(1, parseInt(e.currentTarget.value) || 1))}
								hint="Agents to remove"
							/>
							<Input
								label="Cooldown (seconds)"
								type="number"
								value={String(policyCooldown())}
								onInput={(e) => setPolicyCooldown(Math.max(30, parseInt(e.currentTarget.value) || 30))}
								hint="Min time between actions"
							/>
						</div>

						{/* Visual explanation */}
						<div class="p-4 rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)]">
							<p class="text-xs font-medium text-[var(--color-text-secondary)] mb-2">Policy Behavior</p>
							<div class="space-y-1.5 text-xs text-[var(--color-text-tertiary)]">
								<p>• When queue depth exceeds <span class="text-[var(--color-text-primary)] font-mono">{policyScaleUpThreshold()}</span> jobs, add <span class="text-emerald-400 font-mono">{policyScaleUpStep()}</span> agent{policyScaleUpStep() > 1 ? 's' : ''} (up to max <span class="font-mono">{policyMaxAgents()}</span>)</p>
								<p>• When queue depth drops to <span class="text-[var(--color-text-primary)] font-mono">{policyScaleDownThreshold()}</span> or below, remove <span class="text-amber-400 font-mono">{policyScaleDownStep()}</span> agent{policyScaleDownStep() > 1 ? 's' : ''} (down to min <span class="font-mono">{policyMinAgents()}</span>)</p>
								<p>• Wait at least <span class="text-[var(--color-text-primary)] font-mono">{policyCooldown()}s</span> between scaling actions to prevent thrashing</p>
								<p>• Matches <span class="text-[var(--color-text-primary)] font-mono">{policyExecutor()}</span> agents{policyLabels() ? ` with labels: ${policyLabels()}` : ''}</p>
							</div>
						</div>
					</div>
				</Modal>
			</Show>

			{/* Policy Events Modal */}
			<Show when={showEventsFor()}>
				<Modal
					open={!!showEventsFor()}
					onClose={() => setShowEventsFor(null)}
					title="Scaling Events"
					description={`Event history for policy`}
					size="lg"
				>
					<Show when={policyEvents().length > 0} fallback={
						<div class="text-center py-8">
							<p class="text-xs text-[var(--color-text-tertiary)]">No events recorded for this policy</p>
						</div>
					}>
						<div class="space-y-2 max-h-96 overflow-y-auto">
							<For each={policyEvents()}>
								{(event) => (
									<div class="flex items-center gap-3 p-3 rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)]">
										<span class={`text-xl font-bold ${getActionColor(event.action)}`}>{getActionIcon(event.action)}</span>
										<div class="flex-1 min-w-0">
											<div class="flex items-center gap-2">
												<span class="text-xs font-medium text-[var(--color-text-secondary)]">{event.action.replace('_', ' ')}</span>
												<span class="text-xs font-mono text-[var(--color-text-primary)]">{event.from_count} → {event.to_count}</span>
											</div>
											<p class="text-xs text-[var(--color-text-tertiary)] truncate">{event.reason}</p>
										</div>
										<div class="text-right shrink-0">
											<p class="text-xs text-[var(--color-text-tertiary)]">{event.queue_depth} jobs / {event.active_agents} agents</p>
											<p class="text-xs text-[var(--color-text-tertiary)]">{formatRelativeTime(event.created_at)}</p>
										</div>
									</div>
								)}
							</For>
						</div>
					</Show>
				</Modal>
			</Show>
		</PageContainer>
	);
};

export default AgentsPage;
