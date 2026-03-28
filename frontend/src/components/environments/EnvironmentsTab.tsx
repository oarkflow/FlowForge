import type { Component } from 'solid-js';
import { createSignal, createResource, For, Show, createMemo, onMount } from 'solid-js';
import Card from '../ui/Card';
import Badge from '../ui/Badge';
import type { BadgeVariant } from '../ui/Badge';
import Button from '../ui/Button';
import Input from '../ui/Input';
import Modal from '../ui/Modal';
import KeyValueEditor, { type KeyValuePair } from '../ui/KeyValueEditor';
import { toast } from '../ui/Toast';
import { api, ApiRequestError } from '../../api/client';
import type { Environment, Deployment, EnvOverride, DeployStrategy, RollingConfig, BlueGreenConfig, CanaryConfig, CanaryStep, HealthResult, Approval, ProjectEnvironmentChainEdge } from '../../types';
import { formatRelativeTime } from '../../utils/helpers';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function toSlug(name: string): string {
	return name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
}

function deploymentStatusVariant(status?: string): BadgeVariant {
	switch (status) {
		case 'live': return 'success';
		case 'deploying': return 'running';
		case 'pending': return 'queued';
		case 'failed': return 'error';
		case 'rolled_back': return 'warning';
		default: return 'default';
	}
}

function deploymentStatusLabel(status?: string): string {
	switch (status) {
		case 'live': return 'Live';
		case 'deploying': return 'Deploying';
		case 'pending': return 'Pending';
		case 'failed': return 'Failed';
		case 'rolled_back': return 'Rolled Back';
		default: return 'No Deploys';
	}
}

function strategyLabel(strategy?: string): string {
	switch (strategy) {
		case 'recreate': return 'Recreate';
		case 'rolling': return 'Rolling Update';
		case 'blue_green': return 'Blue-Green';
		case 'canary': return 'Canary';
		default: return 'Recreate';
	}
}

function strategyVariant(strategy?: string): BadgeVariant {
	switch (strategy) {
		case 'rolling': return 'info';
		case 'blue_green': return 'running';
		case 'canary': return 'warning';
		default: return 'default';
	}
}

function parseStrategyConfig<T>(configStr: string, fallback: T): T {
	try {
		if (!configStr || configStr === '{}') return fallback;
		return JSON.parse(configStr) as T;
	} catch {
		return fallback;
	}
}

function parseHealthResults(resultsStr: string): HealthResult[] {
	try {
		if (!resultsStr || resultsStr === '[]') return [];
		return JSON.parse(resultsStr) as HealthResult[];
	} catch {
		return [];
	}
}

const defaultRollingConfig: RollingConfig = { max_surge: 25, max_unavailable: 25, batch_size: 1 };
const defaultBlueGreenConfig: BlueGreenConfig = { validation_timeout: 300, auto_promote: false };
const defaultCanaryConfig: CanaryConfig = { steps: [{ weight: 10, duration: 300 }, { weight: 50, duration: 300 }], analysis_duration: 300, auto_promote: false };

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------
interface EnvironmentsTabProps {
	projectId: string;
	pipelines: { id: string; name: string }[];
}

// ---------------------------------------------------------------------------
// Fetcher
// ---------------------------------------------------------------------------
async function fetchEnvironments(projectId: string) {
	const envs = await api.environments.list(projectId);
	// Fetch latest deployment for each environment
	const envsWithDeployments = await Promise.all(
		envs.map(async (env) => {
			try {
				const deployments = await api.deployments.list(projectId, env.id);
				return { env, latestDeployment: deployments.length > 0 ? deployments[0] : null };
			} catch {
				return { env, latestDeployment: null };
			}
		})
	);
	return envsWithDeployments;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const EnvironmentsTab: Component<EnvironmentsTabProps> = (props) => {
	const [environments, { refetch }] = createResource(() => props.projectId, fetchEnvironments);

	// Create modal
	const [showCreate, setShowCreate] = createSignal(false);
	const [createName, setCreateName] = createSignal('');
	const [createDesc, setCreateDesc] = createSignal('');
	const [createUrl, setCreateUrl] = createSignal('');
	const [createIsProduction, setCreateIsProduction] = createSignal(false);
	const [createAutoDeploy, setCreateAutoDeploy] = createSignal('');
	const [creating, setCreating] = createSignal(false);
	const createSlug = createMemo(() => toSlug(createName()));

	// Detail modal
	const [selectedEnv, setSelectedEnv] = createSignal<Environment | null>(null);
	const [deployments, setDeployments] = createSignal<Deployment[]>([]);
	const [overrides, setOverrides] = createSignal<KeyValuePair[]>([]);
	const [overridesDirty, setOverridesDirty] = createSignal(false);
	const [loadingDetail, setLoadingDetail] = createSignal(false);

	// Edit mode in detail
	const [editMode, setEditMode] = createSignal(false);
	const [editName, setEditName] = createSignal('');
	const [editDesc, setEditDesc] = createSignal('');
	const [editUrl, setEditUrl] = createSignal('');
	const [editIsProduction, setEditIsProduction] = createSignal(false);
	const [editAutoDeploy, setEditAutoDeploy] = createSignal('');
	const [savingEdit, setSavingEdit] = createSignal(false);

	// Deploy modal
	const [showDeploy, setShowDeploy] = createSignal(false);
	const [deployVersion, setDeployVersion] = createSignal('');
	const [deployCommitSha, setDeployCommitSha] = createSignal('');
	const [deployImageTag, setDeployImageTag] = createSignal('');
	const [deploying, setDeploying] = createSignal(false);

	// Rollback modal
	const [showRollback, setShowRollback] = createSignal(false);
	const [rollbackTargetId, setRollbackTargetId] = createSignal('');
	const [rollingBack, setRollingBack] = createSignal(false);

	// Promote modal
	const [showPromote, setShowPromote] = createSignal(false);
	const [promoteSourceEnvId, setPromoteSourceEnvId] = createSignal('');
	const [promoting, setPromoting] = createSignal(false);

	// Lock modal
	const [showLock, setShowLock] = createSignal(false);
	const [lockReason, setLockReason] = createSignal('');
	const [locking, setLocking] = createSignal(false);

	// Saving overrides
	const [savingOverrides, setSavingOverrides] = createSignal(false);

	// Delete confirm
	const [showDeleteEnv, setShowDeleteEnv] = createSignal(false);
	const [deletingEnv, setDeletingEnv] = createSignal(false);

	// Strategy configuration modal
	const [showStrategyConfig, setShowStrategyConfig] = createSignal(false);
	const [strategyType, setStrategyType] = createSignal<DeployStrategy>('recreate');
	const [rollingConfig, setRollingConfig] = createSignal<RollingConfig>({ ...defaultRollingConfig });
	const [blueGreenConfig, setBlueGreenConfig] = createSignal<BlueGreenConfig>({ ...defaultBlueGreenConfig });
	const [canaryConfig, setCanaryConfig] = createSignal<CanaryConfig>({ ...defaultCanaryConfig });
	const [healthCheckUrl, setHealthCheckUrl] = createSignal('');
	const [healthCheckPath, setHealthCheckPath] = createSignal('/health');
	const [healthCheckInterval, setHealthCheckInterval] = createSignal(30);
	const [healthCheckTimeout, setHealthCheckTimeout] = createSignal(10);
	const [healthCheckRetries, setHealthCheckRetries] = createSignal(3);
	const [healthCheckExpectedStatus, setHealthCheckExpectedStatus] = createSignal(200);
	const [savingStrategy, setSavingStrategy] = createSignal(false);

	// Canary advance
	const [advancingCanary, setAdvancingCanary] = createSignal(false);

	// Health check
	const [checkingHealth, setCheckingHealth] = createSignal(false);
	const [latestHealthResult, setLatestHealthResult] = createSignal<HealthResult | null>(null);

	// Approval / protection rules
	const [approvalBanner, setApprovalBanner] = createSignal<{ show: boolean; approval?: Approval }>({ show: false });
	const [showProtectionRules, setShowProtectionRules] = createSignal(false);
	const [protRequireApproval, setProtRequireApproval] = createSignal(false);
	const [protMinApprovals, setProtMinApprovals] = createSignal(1);
	const [protApprovers, setProtApprovers] = createSignal('');
	const [savingProtection, setSavingProtection] = createSignal(false);

	// Promotion chain state
	const [chainEdges, setChainEdges] = createSignal<ProjectEnvironmentChainEdge[]>([]);
	const [chainDraft, setChainDraft] = createSignal<{ source_environment_id: string; target_environment_id: string }[]>([]);
	const [chainDirty, setChainDirty] = createSignal(false);
	const [savingChain, setSavingChain] = createSignal(false);
	const [loadingChain, setLoadingChain] = createSignal(false);
	const [showChainEditor, setShowChainEditor] = createSignal(false);

	// Successful (live or rolled_back) deployments for rollback
	const rollbackCandidates = createMemo(() =>
		deployments().filter(d => d.status === 'live' || d.status === 'rolled_back')
	);

	// Other environments for promote (exclude current)
	const otherEnvs = createMemo(() =>
		(environments() ?? []).filter(e => e.env.id !== selectedEnv()?.id).map(e => e.env)
	);

	// Active canary deployment (deploying + canary strategy)
	const activeCanaryDeployment = createMemo(() =>
		deployments().find(d => d.strategy === 'canary' && (d.status === 'deploying' || d.status === 'pending') && d.canary_weight < 100)
	);

	// Active deployment (currently deploying)
	const activeDeployment = createMemo(() =>
		deployments().find(d => d.status === 'deploying' || d.status === 'pending')
	);

	// ---------------------------------------------------------------------------
	// Handlers
	// ---------------------------------------------------------------------------
	const handleCreate = async () => {
		if (!createName().trim()) return;
		setCreating(true);
		try {
			await api.environments.create(props.projectId, {
				name: createName().trim(),
				slug: createSlug(),
				description: createDesc().trim(),
				url: createUrl().trim(),
				is_production: createIsProduction(),
				auto_deploy_branch: createAutoDeploy().trim(),
			});
			toast.success(`Environment "${createName()}" created`);
			setShowCreate(false);
			setCreateName('');
			setCreateDesc('');
			setCreateUrl('');
			setCreateIsProduction(false);
			setCreateAutoDeploy('');
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to create environment');
		} finally {
			setCreating(false);
		}
	};

	const openDetail = async (env: Environment) => {
		setSelectedEnv(env);
		setEditMode(false);
		setLoadingDetail(true);
		setOverridesDirty(false);
		setLatestHealthResult(null);
		try {
			const [deps, ovr] = await Promise.all([
				api.deployments.list(props.projectId, env.id),
				api.envOverrides.list(props.projectId, env.id),
			]);
			setDeployments(deps);
			setOverrides(ovr.map(o => ({
				id: o.id,
				key: o.key,
				value: o.is_secret ? '' : (o.value_enc ?? ''),
			})));
		} catch (err) {
			toast.error('Failed to load environment details');
		} finally {
			setLoadingDetail(false);
		}
	};

	const handleEditSave = async () => {
		const env = selectedEnv();
		if (!env) return;
		setSavingEdit(true);
		try {
			const updated = await api.environments.update(props.projectId, env.id, {
				name: editName().trim(),
				description: editDesc().trim(),
				url: editUrl().trim(),
				is_production: editIsProduction(),
				auto_deploy_branch: editAutoDeploy().trim(),
			});
			setSelectedEnv(updated);
			setEditMode(false);
			toast.success('Environment updated');
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to update environment');
		} finally {
			setSavingEdit(false);
		}
	};

	const startEdit = () => {
		const env = selectedEnv();
		if (!env) return;
		setEditName(env.name);
		setEditDesc(env.description);
		setEditUrl(env.url);
		setEditIsProduction(env.is_production);
		setEditAutoDeploy(env.auto_deploy_branch);
		setEditMode(true);
	};

	const handleDeploy = async () => {
		const env = selectedEnv();
		if (!env) return;
		setDeploying(true);
		try {
			const result = await api.deployments.trigger(props.projectId, env.id, {
				version: deployVersion().trim() || undefined,
				commit_sha: deployCommitSha().trim() || undefined,
				image_tag: deployImageTag().trim() || undefined,
			});
			// Check if the response indicates approval is required
			if (result && typeof result === 'object' && 'approval_required' in result && (result as any).approval_required) {
				const approval = (result as any).approval as Approval | undefined;
				toast.info('Deployment requires approval before proceeding');
				setApprovalBanner({ show: true, approval });
			} else {
				toast.success('Deployment triggered');
			}
			setShowDeploy(false);
			setDeployVersion('');
			setDeployCommitSha('');
			setDeployImageTag('');
			// Refresh detail
			openDetail(env);
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to trigger deployment');
		} finally {
			setDeploying(false);
		}
	};

	const handleRollback = async () => {
		const env = selectedEnv();
		if (!env || !rollbackTargetId()) return;
		setRollingBack(true);
		try {
			await api.deployments.rollback(props.projectId, env.id, rollbackTargetId());
			toast.success('Rollback initiated');
			setShowRollback(false);
			setRollbackTargetId('');
			openDetail(env);
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to rollback');
		} finally {
			setRollingBack(false);
		}
	};

	const handlePromote = async () => {
		const env = selectedEnv();
		if (!env || !promoteSourceEnvId()) return;
		setPromoting(true);
		try {
			await api.deployments.promote(props.projectId, env.id, promoteSourceEnvId());
			toast.success('Promotion initiated');
			setShowPromote(false);
			setPromoteSourceEnvId('');
			openDetail(env);
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to promote');
		} finally {
			setPromoting(false);
		}
	};

	const handleLock = async () => {
		const env = selectedEnv();
		if (!env) return;
		setLocking(true);
		try {
			const updated = await api.environments.lock(props.projectId, env.id, lockReason().trim());
			setSelectedEnv(updated);
			toast.success('Environment locked');
			setShowLock(false);
			setLockReason('');
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to lock');
		} finally {
			setLocking(false);
		}
	};

	const handleUnlock = async () => {
		const env = selectedEnv();
		if (!env) return;
		try {
			const updated = await api.environments.unlock(props.projectId, env.id);
			setSelectedEnv(updated);
			toast.success('Environment unlocked');
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to unlock');
		}
	};

	const handleToggleFreeze = async () => {
		const env = selectedEnv();
		if (!env) return;
		try {
			const updated = await api.environments.update(props.projectId, env.id, {
				deploy_freeze: !env.deploy_freeze,
			});
			setSelectedEnv(updated);
			toast.success(updated.deploy_freeze ? 'Deploy freeze enabled' : 'Deploy freeze disabled');
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to toggle deploy freeze');
		}
	};

	const handleSaveOverrides = async () => {
		const env = selectedEnv();
		if (!env) return;
		setSavingOverrides(true);
		try {
			const items = overrides().filter(v => v.key.trim());
			await api.envOverrides.save(props.projectId, env.id, items.map(v => ({
				key: v.key,
				value: v.value,
				is_secret: false,
			})));
			toast.success('Environment overrides saved');
			setOverridesDirty(false);
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to save overrides');
		} finally {
			setSavingOverrides(false);
		}
	};

	const handleDeleteEnv = async () => {
		const env = selectedEnv();
		if (!env) return;
		setDeletingEnv(true);
		try {
			await api.environments.delete(props.projectId, env.id);
			toast.success(`Environment "${env.name}" deleted`);
			setShowDeleteEnv(false);
			setSelectedEnv(null);
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to delete environment');
		} finally {
			setDeletingEnv(false);
		}
	};

	const handleOverrideChange = (items: KeyValuePair[]) => {
		setOverrides(items);
		setOverridesDirty(true);
	};

	// Strategy handlers
	const openStrategyConfig = () => {
		const env = selectedEnv();
		if (!env) return;
		setStrategyType(env.strategy || 'recreate');
		setHealthCheckUrl(env.health_check_url || '');
		setHealthCheckPath(env.health_check_path || '/health');
		setHealthCheckInterval(env.health_check_interval || 30);
		setHealthCheckTimeout(env.health_check_timeout || 10);
		setHealthCheckRetries(env.health_check_retries || 3);
		setHealthCheckExpectedStatus(env.health_check_expected_status || 200);

		// Parse strategy-specific config
		const configStr = env.strategy_config || '{}';
		switch (env.strategy) {
			case 'rolling':
				setRollingConfig(parseStrategyConfig<RollingConfig>(configStr, { ...defaultRollingConfig }));
				break;
			case 'blue_green':
				setBlueGreenConfig(parseStrategyConfig<BlueGreenConfig>(configStr, { ...defaultBlueGreenConfig }));
				break;
			case 'canary':
				setCanaryConfig(parseStrategyConfig<CanaryConfig>(configStr, { ...defaultCanaryConfig }));
				break;
		}
		setShowStrategyConfig(true);
	};

	const handleSaveStrategy = async () => {
		const env = selectedEnv();
		if (!env) return;
		setSavingStrategy(true);
		try {
			let configStr = '{}';
			switch (strategyType()) {
				case 'rolling':
					configStr = JSON.stringify(rollingConfig());
					break;
				case 'blue_green':
					configStr = JSON.stringify(blueGreenConfig());
					break;
				case 'canary':
					configStr = JSON.stringify(canaryConfig());
					break;
			}
			const updated = await api.strategy.update(props.projectId, env.id, {
				strategy: strategyType(),
				strategy_config: configStr,
				health_check_url: healthCheckUrl().trim(),
				health_check_interval: healthCheckInterval(),
				health_check_timeout: healthCheckTimeout(),
				health_check_retries: healthCheckRetries(),
				health_check_path: healthCheckPath().trim(),
				health_check_expected_status: healthCheckExpectedStatus(),
			});
			setSelectedEnv(updated);
			setShowStrategyConfig(false);
			toast.success('Deployment strategy updated');
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to update strategy');
		} finally {
			setSavingStrategy(false);
		}
	};

	const handleAdvanceCanary = async (dep: Deployment, nextWeight: number) => {
		const env = selectedEnv();
		if (!env) return;
		setAdvancingCanary(true);
		try {
			await api.deployments.advanceCanary(props.projectId, env.id, dep.id, nextWeight);
			toast.success(`Canary advanced to ${nextWeight}%`);
			openDetail(env);
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to advance canary');
		} finally {
			setAdvancingCanary(false);
		}
	};

	const handleCheckHealth = async (dep: Deployment) => {
		const env = selectedEnv();
		if (!env) return;
		setCheckingHealth(true);
		try {
			const result = await api.deployments.checkHealth(props.projectId, env.id, dep.id);
			setLatestHealthResult(result);
			if (result.healthy) {
				toast.success('Health check passed');
			} else {
				toast.error(`Health check failed: ${result.error || 'unhealthy'}`);
			}
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to check health');
		} finally {
			setCheckingHealth(false);
		}
	};

	const addCanaryStep = () => {
		const cfg = canaryConfig();
		const lastWeight = cfg.steps.length > 0 ? cfg.steps[cfg.steps.length - 1].weight : 0;
		const newWeight = Math.min(lastWeight + 25, 100);
		setCanaryConfig({
			...cfg,
			steps: [...cfg.steps, { weight: newWeight, duration: 300 }],
		});
	};

	// Protection rules handlers
	const openProtectionRules = () => {
		const env = selectedEnv();
		if (!env) return;
		try {
			const rules = env.protection_rules ? JSON.parse(env.protection_rules) : {};
			setProtRequireApproval(!!rules.require_approval);
			setProtMinApprovals(rules.min_approvals || 1);
		} catch { /* ignore parse error */ }
		try {
			const approvers = env.required_approvers ? JSON.parse(env.required_approvers) : [];
			setProtApprovers(Array.isArray(approvers) ? approvers.join(', ') : '');
		} catch { /* ignore parse error */ }
		setShowProtectionRules(true);
	};

	const handleSaveProtectionRules = async () => {
		const env = selectedEnv();
		if (!env) return;
		setSavingProtection(true);
		try {
			const approversList = protApprovers().split(',').map(s => s.trim()).filter(Boolean);
			await api.protectionRules.update(props.projectId, env.id, {
				require_approval: protRequireApproval(),
				min_approvals: protMinApprovals(),
				required_approvers: approversList,
			});
			toast.success('Protection rules updated');
			setShowProtectionRules(false);
			openDetail(env);
			refetch();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to update protection rules');
		} finally {
			setSavingProtection(false);
		}
	};

	const removeCanaryStep = (index: number) => {
		const cfg = canaryConfig();
		setCanaryConfig({
			...cfg,
			steps: cfg.steps.filter((_, i) => i !== index),
		});
	};

	const updateCanaryStep = (index: number, field: keyof CanaryStep, value: number) => {
		const cfg = canaryConfig();
		const steps = [...cfg.steps];
		steps[index] = { ...steps[index], [field]: value };
		setCanaryConfig({ ...cfg, steps });
	};

	// Promotion chain handlers
	const loadChain = async () => {
		setLoadingChain(true);
		try {
			const edges = await api.environmentChain.get(props.projectId);
			setChainEdges(edges);
			setChainDraft(edges.map(e => ({ source_environment_id: e.source_environment_id, target_environment_id: e.target_environment_id })));
			setChainDirty(false);
		} catch {
			// Chain may not exist yet, that's fine
			setChainEdges([]);
			setChainDraft([]);
		} finally {
			setLoadingChain(false);
		}
	};

	const handleSaveChain = async () => {
		setSavingChain(true);
		try {
			const edges = chainDraft().filter(e => e.source_environment_id && e.target_environment_id).map((e, i) => ({
				source_environment_id: e.source_environment_id,
				target_environment_id: e.target_environment_id,
				position: i,
			}));
			await api.environmentChain.update(props.projectId, edges);
			toast.success('Promotion chain saved');
			setChainDirty(false);
			await loadChain();
		} catch (err) {
			toast.error(err instanceof ApiRequestError ? err.message : 'Failed to save promotion chain');
		} finally {
			setSavingChain(false);
		}
	};

	const addChainLink = () => {
		setChainDraft([...chainDraft(), { source_environment_id: '', target_environment_id: '' }]);
		setChainDirty(true);
	};

	const removeChainLink = (index: number) => {
		setChainDraft(chainDraft().filter((_, i) => i !== index));
		setChainDirty(true);
	};

	const updateChainLink = (index: number, field: 'source_environment_id' | 'target_environment_id', value: string) => {
		const draft = [...chainDraft()];
		draft[index] = { ...draft[index], [field]: value };
		setChainDraft(draft);
		setChainDirty(true);
	};

	const moveChainLink = (index: number, direction: 'up' | 'down') => {
		const draft = [...chainDraft()];
		const newIndex = direction === 'up' ? index - 1 : index + 1;
		if (newIndex < 0 || newIndex >= draft.length) return;
		[draft[index], draft[newIndex]] = [draft[newIndex], draft[index]];
		setChainDraft(draft);
		setChainDirty(true);
	};

	// Helper: get environment name by id
	const envNameById = (id: string): string => {
		const found = (environments() ?? []).find(e => e.env.id === id);
		return found ? found.env.name : id.substring(0, 8);
	};

	// Load chain on mount
	onMount(() => {
		loadChain();
	});

	// ---------------------------------------------------------------------------
	// Render
	// ---------------------------------------------------------------------------
	return (
		<div>
			{/* Header */}
			<div class="flex items-center justify-between mb-4">
				<p class="text-sm text-[var(--color-text-tertiary)]">
					{(environments() ?? []).length} environment{(environments() ?? []).length !== 1 ? 's' : ''}
				</p>
				<Button size="sm" onClick={() => setShowCreate(true)}
					icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>}
				>Add Environment</Button>
			</div>

			{/* Environment Cards Grid */}
			<Show when={!environments.loading} fallback={
				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
					<For each={[1, 2, 3]}>{() => (
						<div class="h-48 bg-[var(--color-bg-secondary)] rounded-xl animate-pulse" />
					)}</For>
				</div>
			}>
				<Show when={(environments() ?? []).length > 0} fallback={
					<div class="text-center py-12">
						<svg class="w-12 h-12 mx-auto text-[var(--color-text-tertiary)] mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
							<path stroke-linecap="round" stroke-linejoin="round" d="M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3m0 3h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008zm-3 6h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008z" />
						</svg>
						<p class="text-[var(--color-text-secondary)] mb-2">No environments yet</p>
						<p class="text-sm text-[var(--color-text-tertiary)] mb-4">Create your first environment (e.g. staging, production) to manage deployments.</p>
						<Button onClick={() => setShowCreate(true)}>Create Environment</Button>
					</div>
				}>
					{/* Promotion Flow Visualization */}
					<Show when={(environments() ?? []).length >= 2}>
						<div class="mb-6 p-4 rounded-xl bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
							<div class="flex items-center gap-2 mb-3">
								<svg class="w-4 h-4 text-[var(--color-text-tertiary)]" viewBox="0 0 20 20" fill="currentColor">
									<path fill-rule="evenodd" d="M3 10a.75.75 0 01.75-.75h10.638L10.23 5.29a.75.75 0 111.04-1.08l5.5 5.25a.75.75 0 010 1.08l-5.5 5.25a.75.75 0 11-1.04-1.08l4.158-3.96H3.75A.75.75 0 013 10z" clip-rule="evenodd" />
								</svg>
								<span class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Promotion Flow</span>
							</div>
							<div class="flex items-center gap-2 overflow-x-auto pb-1">
								<For each={(environments() ?? []).sort((a, b) => {
									// Sort: non-production first, production last
									if (a.env.is_production !== b.env.is_production) return a.env.is_production ? 1 : -1;
									return a.env.name.localeCompare(b.env.name);
								})}>
									{({ env, latestDeployment }, i) => (
										<>
											<Show when={i() > 0}>
												<div class="flex flex-col items-center gap-0.5 flex-shrink-0">
													<svg class="w-5 h-5 text-[var(--color-text-tertiary)]" viewBox="0 0 20 20" fill="currentColor">
														<path fill-rule="evenodd" d="M3 10a.75.75 0 01.75-.75h10.638L10.23 5.29a.75.75 0 111.04-1.08l5.5 5.25a.75.75 0 010 1.08l-5.5 5.25a.75.75 0 11-1.04-1.08l4.158-3.96H3.75A.75.75 0 013 10z" clip-rule="evenodd" />
													</svg>
													<span class="text-[9px] text-[var(--color-text-tertiary)]">promote</span>
												</div>
											</Show>
											<div
												class={`flex-shrink-0 px-4 py-2.5 rounded-lg border-2 cursor-pointer transition-all min-w-[140px] text-center ${env.is_production
													? 'border-amber-500/40 bg-amber-500/5 hover:border-amber-500/60'
													: latestDeployment?.status === 'live'
														? 'border-emerald-500/40 bg-emerald-500/5 hover:border-emerald-500/60'
														: 'border-[var(--color-border-primary)] bg-[var(--color-bg-tertiary)] hover:border-indigo-500/30'
													}`}
												onClick={() => openDetail(env)}
											>
												<div class="text-xs font-semibold text-[var(--color-text-primary)] truncate">{env.name}</div>
												<div class="flex items-center justify-center gap-1.5 mt-1">
													<div class={`w-1.5 h-1.5 rounded-full ${latestDeployment?.status === 'live' ? 'bg-emerald-400' :
														latestDeployment?.status === 'deploying' ? 'bg-violet-400 animate-pulse' :
															latestDeployment?.status === 'failed' ? 'bg-red-400' :
																'bg-gray-500'
														}`} />
													<span class="text-[10px] text-[var(--color-text-tertiary)]">
														{latestDeployment?.version ? `v${latestDeployment.version}` : 'No deploy'}
													</span>
												</div>
												<Show when={env.lock_owner_id}>
													<div class="mt-1">
														<span class="text-[9px] text-red-400 font-medium">LOCKED</span>
													</div>
												</Show>
											</div>
										</>
									)}
								</For>
							</div>
						</div>
					</Show>

					<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
						<For each={environments() ?? []}>
							{({ env, latestDeployment }) => (
								<div
									class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-5 hover:border-indigo-500/30 transition-all cursor-pointer group"
									onClick={() => openDetail(env)}
								>
									{/* Header row */}
									<div class="flex items-start justify-between mb-3">
										<div class="min-w-0 flex-1">
											<div class="flex items-center gap-2 mb-1">
												<h3 class="text-sm font-semibold text-[var(--color-text-primary)] truncate">{env.name}</h3>
												<Show when={env.is_production}>
													<Badge variant="warning" size="sm">Production</Badge>
												</Show>
											</div>
											<p class="text-xs text-[var(--color-text-tertiary)] font-mono">{env.slug}</p>
										</div>
										<Badge variant={deploymentStatusVariant(latestDeployment?.status)} dot size="sm">
											{deploymentStatusLabel(latestDeployment?.status)}
										</Badge>
									</div>

									{/* URL */}
									<Show when={env.url}>
										<a
											href={env.url}
											target="_blank"
											rel="noopener noreferrer"
											class="text-xs text-indigo-400 hover:text-indigo-300 truncate block mb-3"
											onClick={(e) => e.stopPropagation()}
										>
											{env.url}
										</a>
									</Show>

									{/* Indicators */}
									<div class="flex items-center gap-2 mb-3 flex-wrap">
										<Show when={env.strategy && env.strategy !== 'recreate'}>
											<Badge variant={strategyVariant(env.strategy)} size="sm">{strategyLabel(env.strategy)}</Badge>
										</Show>
										<Show when={env.lock_owner_id}>
											<Badge variant="error" size="sm">
												<svg class="w-3 h-3 mr-1" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 1a4.5 4.5 0 00-4.5 4.5V9H5a2 2 0 00-2 2v6a2 2 0 002 2h10a2 2 0 002-2v-6a2 2 0 00-2-2h-.5V5.5A4.5 4.5 0 0010 1zm3 8V5.5a3 3 0 10-6 0V9h6z" clip-rule="evenodd" /></svg>
												Locked
											</Badge>
										</Show>
										<Show when={env.deploy_freeze}>
											<Badge variant="info" size="sm">
												<svg class="w-3 h-3 mr-1" viewBox="0 0 20 20" fill="currentColor"><path d="M10 2a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 2zm0 13a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 15zm-8-5a.75.75 0 01.75-.75h1.5a.75.75 0 010 1.5h-1.5A.75.75 0 012 10zm13 0a.75.75 0 01.75-.75h1.5a.75.75 0 010 1.5h-1.5A.75.75 0 0115 10z" /></svg>
												Frozen
											</Badge>
										</Show>
										<Show when={env.auto_deploy_branch}>
											<Badge variant="info" size="sm">Auto: {env.auto_deploy_branch}</Badge>
										</Show>
										<Show when={(() => { try { const r = env.protection_rules ? JSON.parse(env.protection_rules) : {}; return !!r.require_approval; } catch { return false; } })()}>
											<Badge variant="warning" size="sm">
												<svg class="w-3 h-3 mr-1" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" /></svg>
												Approval Required
											</Badge>
										</Show>
									</div>

									{/* Canary progress indicator on card */}
									<Show when={latestDeployment?.strategy === 'canary' && latestDeployment?.status === 'deploying'}>
										<div class="mb-3">
											<div class="flex items-center justify-between text-xs text-[var(--color-text-tertiary)] mb-1">
												<span>Canary</span>
												<span>{latestDeployment!.canary_weight}%</span>
											</div>
											<div class="w-full h-1.5 bg-[var(--color-bg-tertiary)] rounded-full overflow-hidden">
												<div class="h-full bg-amber-500 rounded-full transition-all duration-500" style={{ width: `${latestDeployment!.canary_weight}%` }} />
											</div>
										</div>
									</Show>

									{/* Latest deployment info */}
									<Show when={latestDeployment}>
										<div class="border-t border-[var(--color-border-primary)] pt-3 mt-auto">
											<div class="flex items-center justify-between text-xs">
												<span class="text-[var(--color-text-tertiary)]">
													{latestDeployment!.version && <span class="font-mono mr-2">{latestDeployment!.version}</span>}
													{latestDeployment!.deployed_by && <span>by {latestDeployment!.deployed_by}</span>}
												</span>
												<span class="text-[var(--color-text-tertiary)]">{formatRelativeTime(latestDeployment!.created_at)}</span>
											</div>
										</div>
									</Show>

									{/* Quick actions */}
									<div class="flex items-center gap-2 mt-3 opacity-0 group-hover:opacity-100 transition-opacity">
										<Button size="sm" variant="outline" onClick={(e: Event) => { e.stopPropagation(); setSelectedEnv(env); setShowDeploy(true); }}>Deploy</Button>
										<Button size="sm" variant="ghost" onClick={(e: Event) => { e.stopPropagation(); openDetail(env); }}>Details</Button>
									</div>
								</div>
							)}
						</For>
					</div>
				</Show>
			</Show>

			{/* ===== Promotion Chain Section ===== */}
			<Show when={(environments() ?? []).length >= 2}>
				<div class="mt-6 p-4 rounded-xl bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
					<div class="flex items-center justify-between mb-3">
						<div class="flex items-center gap-2">
							<svg class="w-4 h-4 text-[var(--color-text-tertiary)]" viewBox="0 0 20 20" fill="currentColor">
								<path fill-rule="evenodd" d="M3 10a.75.75 0 01.75-.75h10.638L10.23 5.29a.75.75 0 111.04-1.08l5.5 5.25a.75.75 0 010 1.08l-5.5 5.25a.75.75 0 11-1.04-1.08l4.158-3.96H3.75A.75.75 0 013 10z" clip-rule="evenodd" />
							</svg>
							<span class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Promotion Chain</span>
						</div>
						<Button size="sm" variant="ghost" onClick={() => { setShowChainEditor(!showChainEditor()); if (!showChainEditor()) loadChain(); }}>
							{showChainEditor() ? 'Close Editor' : 'Configure'}
						</Button>
					</div>

					{/* Chain visualization */}
					<Show when={!loadingChain()} fallback={
						<div class="h-12 bg-[var(--color-bg-tertiary)] rounded-lg animate-pulse" />
					}>
						<Show when={chainEdges().length > 0} fallback={
							<Show when={!showChainEditor()}>
								<div class="text-center py-4">
									<p class="text-sm text-[var(--color-text-tertiary)] mb-2">No promotion chain configured</p>
									<p class="text-xs text-[var(--color-text-tertiary)] mb-3">Define the order deployments promote through environments (e.g., Beta → Stage → Prod)</p>
									<Button size="sm" variant="outline" onClick={() => setShowChainEditor(true)}>
										Configure Promotion Chain
									</Button>
								</div>
							</Show>
						}>
							<div class="flex items-center gap-2 overflow-x-auto pb-1">
								{(() => {
									// Build ordered chain from edges
									const edges = chainEdges();
									const envList = (environments() ?? []);
									// Find starting environments (sources that aren't targets)
									const targetIds = new Set(edges.map(e => e.target_environment_id));
									const startEdges = edges.filter(e => !targetIds.has(e.source_environment_id));
									const orderedIds: string[] = [];
									// Walk the chain
									const walk = (id: string) => {
										if (orderedIds.includes(id)) return;
										orderedIds.push(id);
										const next = edges.find(e => e.source_environment_id === id);
										if (next) walk(next.target_environment_id);
									};
									if (startEdges.length > 0) {
										walk(startEdges[0].source_environment_id);
									} else if (edges.length > 0) {
										walk(edges[0].source_environment_id);
									}
									// Ensure we have target of last edge
									if (edges.length > 0) {
										const lastEdge = edges[edges.length - 1];
										if (!orderedIds.includes(lastEdge.target_environment_id)) {
											orderedIds.push(lastEdge.target_environment_id);
										}
									}
									return (
										<For each={orderedIds}>
											{(envId, i) => {
												const envData = envList.find(e => e.env.id === envId);
												const env = envData?.env;
												const latestDeployment = envData?.latestDeployment;
												return (
													<>
														<Show when={i() > 0}>
															<div class="flex flex-col items-center gap-0.5 flex-shrink-0">
																<svg class="w-5 h-5 text-indigo-400" viewBox="0 0 20 20" fill="currentColor">
																	<path fill-rule="evenodd" d="M3 10a.75.75 0 01.75-.75h10.638L10.23 5.29a.75.75 0 111.04-1.08l5.5 5.25a.75.75 0 010 1.08l-5.5 5.25a.75.75 0 11-1.04-1.08l4.158-3.96H3.75A.75.75 0 013 10z" clip-rule="evenodd" />
																</svg>
																<span class="text-[9px] text-indigo-400/60">promote</span>
															</div>
														</Show>
														<div
															class={`flex-shrink-0 px-4 py-2.5 rounded-lg border-2 cursor-pointer transition-all min-w-[140px] text-center ${env?.is_production
																	? 'border-amber-500/40 bg-amber-500/5 hover:border-amber-500/60'
																	: latestDeployment?.status === 'live'
																		? 'border-emerald-500/40 bg-emerald-500/5 hover:border-emerald-500/60'
																		: 'border-[var(--color-border-primary)] bg-[var(--color-bg-tertiary)] hover:border-indigo-500/30'
																}`}
															onClick={() => env && openDetail(env)}
														>
															<div class="text-xs font-semibold text-[var(--color-text-primary)] truncate">{env?.name ?? envId.substring(0, 8)}</div>
															<div class="flex items-center justify-center gap-1.5 mt-1">
																<div class={`w-1.5 h-1.5 rounded-full ${latestDeployment?.status === 'live' ? 'bg-emerald-400' :
																		latestDeployment?.status === 'deploying' ? 'bg-violet-400 animate-pulse' :
																			latestDeployment?.status === 'failed' ? 'bg-red-400' :
																				'bg-gray-500'
																	}`} />
																<span class="text-[10px] text-[var(--color-text-tertiary)]">
																	{latestDeployment?.version ? `v${latestDeployment.version}` : 'No deploy'}
																</span>
															</div>
														</div>
													</>
												);
											}}
										</For>
									);
								})()}
							</div>
						</Show>
					</Show>

					{/* Chain editor */}
					<Show when={showChainEditor()}>
						<div class="mt-4 pt-4 border-t border-[var(--color-border-primary)]">
							<p class="text-xs text-[var(--color-text-tertiary)] mb-3">
								Define source → target pairs to create the promotion chain. Each link represents an allowed promotion path.
							</p>
							<div class="space-y-2 mb-3">
								<For each={chainDraft()}>
									{(link, i) => (
										<div class="flex items-center gap-2">
											<span class="text-xs text-[var(--color-text-tertiary)] w-6">#{i() + 1}</span>
											<select
												class="flex-1 px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
												value={link.source_environment_id}
												onChange={(e) => updateChainLink(i(), 'source_environment_id', e.currentTarget.value)}
											>
												<option value="">Select source...</option>
												<For each={(environments() ?? []).map(e => e.env)}>
													{(env) => <option value={env.id}>{env.name}</option>}
												</For>
											</select>
											<svg class="w-5 h-5 text-[var(--color-text-tertiary)] flex-shrink-0" viewBox="0 0 20 20" fill="currentColor">
												<path fill-rule="evenodd" d="M3 10a.75.75 0 01.75-.75h10.638L10.23 5.29a.75.75 0 111.04-1.08l5.5 5.25a.75.75 0 010 1.08l-5.5 5.25a.75.75 0 11-1.04-1.08l4.158-3.96H3.75A.75.75 0 013 10z" clip-rule="evenodd" />
											</svg>
											<select
												class="flex-1 px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
												value={link.target_environment_id}
												onChange={(e) => updateChainLink(i(), 'target_environment_id', e.currentTarget.value)}
											>
												<option value="">Select target...</option>
												<For each={(environments() ?? []).map(e => e.env)}>
													{(env) => <option value={env.id}>{env.name}</option>}
												</For>
											</select>
											{/* Move up/down */}
											<button
												type="button"
												onClick={() => moveChainLink(i(), 'up')}
												disabled={i() === 0}
												class="p-1 text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] disabled:opacity-30"
												title="Move up"
											>
												<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M14.77 12.79a.75.75 0 01-1.06-.02L10 8.832 6.29 12.77a.75.75 0 11-1.08-1.04l4.25-4.5a.75.75 0 011.08 0l4.25 4.5a.75.75 0 01-.02 1.06z" clip-rule="evenodd" /></svg>
											</button>
											<button
												type="button"
												onClick={() => moveChainLink(i(), 'down')}
												disabled={i() === chainDraft().length - 1}
												class="p-1 text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] disabled:opacity-30"
												title="Move down"
											>
												<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" /></svg>
											</button>
											{/* Remove */}
											<button
												type="button"
												onClick={() => removeChainLink(i())}
												class="p-1 text-red-400 hover:text-red-300"
												title="Remove link"
											>
												<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5z" clip-rule="evenodd" /></svg>
											</button>
										</div>
									)}
								</For>
							</div>
							<Show when={chainDraft().length === 0}>
								<p class="text-sm text-[var(--color-text-tertiary)] text-center py-3">No chain links defined. Add a link to create the promotion flow.</p>
							</Show>
							<div class="flex items-center justify-between mt-3">
								<button
									type="button"
									onClick={addChainLink}
									class="text-sm text-indigo-400 hover:text-indigo-300 flex items-center gap-1"
								>
									<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>
									Add Link
								</button>
								<div class="flex items-center gap-2">
									<Show when={chainDirty()}>
										<span class="text-xs text-amber-400">Unsaved changes</span>
									</Show>
									<Button size="sm" onClick={handleSaveChain} loading={savingChain()} disabled={!chainDirty()}>
										Save Chain
									</Button>
								</div>
							</div>
						</div>
					</Show>
				</div>
			</Show>

			{/* ===== Create Environment Modal ===== */}
			<Show when={showCreate()}>
				<Modal open={showCreate()} onClose={() => setShowCreate(false)} title="Create Environment" description="Add a new deployment target" footer={
					<>
						<Button variant="ghost" onClick={() => setShowCreate(false)}>Cancel</Button>
						<Button onClick={handleCreate} loading={creating()} disabled={!createName().trim()}>Create Environment</Button>
					</>
				}>
					<div class="space-y-4">
						<div>
							<Input label="Name" placeholder="e.g. Staging, Production" value={createName()} onInput={(e) => setCreateName(e.currentTarget.value)} />
							<Show when={createName().trim()}>
								<p class="text-xs text-[var(--color-text-tertiary)] mt-1">Slug: <span class="font-mono">{createSlug()}</span></p>
							</Show>
						</div>
						<Input label="Description" placeholder="Brief description of this environment..." value={createDesc()} onInput={(e) => setCreateDesc(e.currentTarget.value)} />
						<Input label="URL" placeholder="https://staging.example.com" value={createUrl()} onInput={(e) => setCreateUrl(e.currentTarget.value)} />
						<div class="flex items-center gap-3">
							<label class="relative inline-flex items-center cursor-pointer">
								<input type="checkbox" checked={createIsProduction()} onChange={(e) => setCreateIsProduction(e.currentTarget.checked)} class="sr-only peer" />
								<div class="w-9 h-5 bg-[var(--color-bg-tertiary)] peer-focus:ring-2 peer-focus:ring-indigo-500/40 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-500"></div>
							</label>
							<span class="text-sm text-[var(--color-text-secondary)]">Production environment</span>
						</div>
						<Input label="Auto-deploy Branch" placeholder="e.g. main (leave empty to disable)" value={createAutoDeploy()} onInput={(e) => setCreateAutoDeploy(e.currentTarget.value)} />
					</div>
				</Modal>
			</Show>

			{/* ===== Environment Detail Modal ===== */}
			<Show when={selectedEnv()}>
				<Modal open={!!selectedEnv()} onClose={() => { setSelectedEnv(null); setEditMode(false); }} title={selectedEnv()!.name} size="xl" footer={
					<div class="flex items-center gap-2 w-full">
						<div class="flex-1 flex items-center gap-2">
							<Button size="sm" variant="danger" onClick={() => setShowDeleteEnv(true)}>Delete</Button>
						</div>
						<Button variant="ghost" onClick={() => { setSelectedEnv(null); setEditMode(false); }}>Close</Button>
					</div>
				}>
					<Show when={!loadingDetail()} fallback={
						<div class="space-y-4">
							<div class="h-20 bg-[var(--color-bg-secondary)] rounded animate-pulse" />
							<div class="h-40 bg-[var(--color-bg-secondary)] rounded animate-pulse" />
						</div>
					}>
						{/* Info / Edit section */}
						<Show when={!editMode()} fallback={
							<div class="space-y-4 mb-6 p-4 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
								<Input label="Name" value={editName()} onInput={(e) => setEditName(e.currentTarget.value)} />
								<Input label="Description" value={editDesc()} onInput={(e) => setEditDesc(e.currentTarget.value)} />
								<Input label="URL" value={editUrl()} onInput={(e) => setEditUrl(e.currentTarget.value)} />
								<div class="flex items-center gap-3">
									<label class="relative inline-flex items-center cursor-pointer">
										<input type="checkbox" checked={editIsProduction()} onChange={(e) => setEditIsProduction(e.currentTarget.checked)} class="sr-only peer" />
										<div class="w-9 h-5 bg-[var(--color-bg-tertiary)] peer-focus:ring-2 peer-focus:ring-indigo-500/40 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-500"></div>
									</label>
									<span class="text-sm text-[var(--color-text-secondary)]">Production</span>
								</div>
								<Input label="Auto-deploy Branch" value={editAutoDeploy()} onInput={(e) => setEditAutoDeploy(e.currentTarget.value)} />
								<div class="flex items-center gap-2">
									<Button size="sm" onClick={handleEditSave} loading={savingEdit()}>Save</Button>
									<Button size="sm" variant="ghost" onClick={() => setEditMode(false)}>Cancel</Button>
								</div>
							</div>
						}>
							<div class="mb-6">
								<div class="flex items-center justify-between mb-3">
									<div class="flex items-center gap-3">
										<Show when={selectedEnv()!.is_production}>
											<Badge variant="warning" size="sm">Production</Badge>
										</Show>
										<Badge variant={strategyVariant(selectedEnv()!.strategy)} size="sm">{strategyLabel(selectedEnv()!.strategy)}</Badge>
										<Show when={selectedEnv()!.url}>
											<a href={selectedEnv()!.url} target="_blank" rel="noopener noreferrer" class="text-sm text-indigo-400 hover:text-indigo-300">{selectedEnv()!.url}</a>
										</Show>
									</div>
									<div class="flex items-center gap-2">
										<Button size="sm" variant="ghost" onClick={openStrategyConfig}>
											<svg class="w-4 h-4 mr-1" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z" clip-rule="evenodd" /></svg>
											Strategy
										</Button>
										<Button size="sm" variant="ghost" onClick={openProtectionRules}>
											<svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" /></svg>
											Protection
										</Button>
										<Button size="sm" variant="ghost" onClick={startEdit}>Edit</Button>
									</div>
								</div>
								<Show when={selectedEnv()!.description}>
									<p class="text-sm text-[var(--color-text-secondary)] mb-3">{selectedEnv()!.description}</p>
								</Show>
								<div class="text-xs text-[var(--color-text-tertiary)] space-y-1">
									<p>Slug: <span class="font-mono">{selectedEnv()!.slug}</span></p>
									<Show when={selectedEnv()!.auto_deploy_branch}>
										<p>Auto-deploy: <span class="font-mono">{selectedEnv()!.auto_deploy_branch}</span></p>
									</Show>
									<Show when={selectedEnv()!.health_check_path}>
										<p>Health check: <span class="font-mono">{selectedEnv()!.health_check_path}</span></p>
									</Show>
								</div>
							</div>
						</Show>

						{/* Action buttons row */}
						<div class="flex flex-wrap items-center gap-2 mb-6 p-3 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
							<Button size="sm" onClick={() => setShowDeploy(true)}
								icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 17a.75.75 0 01-.75-.75V5.612L5.29 9.77a.75.75 0 01-1.08-1.04l5.25-5.5a.75.75 0 011.08 0l5.25 5.5a.75.75 0 11-1.08 1.04l-3.96-4.158V16.25A.75.75 0 0110 17z" clip-rule="evenodd" /></svg>}
							>Deploy</Button>
							<Button size="sm" variant="outline" onClick={() => { setRollbackTargetId(''); setShowRollback(true); }}
								disabled={rollbackCandidates().length === 0}
							>Rollback</Button>
							<Button size="sm" variant="outline" onClick={() => { setPromoteSourceEnvId(''); setShowPromote(true); }}
								disabled={otherEnvs().length === 0}
							>Promote</Button>
							<div class="flex-1" />
							<Show when={selectedEnv()!.lock_owner_id} fallback={
								<Button size="sm" variant="ghost" onClick={() => { setLockReason(''); setShowLock(true); }}>
									<svg class="w-4 h-4 mr-1" viewBox="0 0 20 20" fill="currentColor"><path d="M10 2a5 5 0 00-5 5v2a2 2 0 00-2 2v5a2 2 0 002 2h10a2 2 0 002-2v-5a2 2 0 00-2-2V7a5 5 0 00-5-5zm3 7V7a3 3 0 10-6 0v2h6z" /></svg>
									Lock
								</Button>
							}>
								<Button size="sm" variant="ghost" onClick={handleUnlock}>
									<svg class="w-4 h-4 mr-1" viewBox="0 0 20 20" fill="currentColor"><path d="M10 2a5 5 0 00-5 5v2a2 2 0 00-2 2v5a2 2 0 002 2h10a2 2 0 002-2v-5a2 2 0 00-2-2V7a5 5 0 00-5-5zm3 7V7a3 3 0 10-6 0v2h6z" /></svg>
									Unlock
								</Button>
							</Show>
							<button
								class={`flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg transition-colors ${selectedEnv()!.deploy_freeze
									? 'bg-blue-500/10 text-blue-400 hover:bg-blue-500/20'
									: 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)]'
									}`}
								onClick={handleToggleFreeze}
								title={selectedEnv()!.deploy_freeze ? 'Disable deploy freeze' : 'Enable deploy freeze'}
							>
								<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M10 2a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 2zm0 13a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 15zm-8-5a.75.75 0 01.75-.75h1.5a.75.75 0 010 1.5h-1.5A.75.75 0 012 10zm13 0a.75.75 0 01.75-.75h1.5a.75.75 0 010 1.5h-1.5A.75.75 0 0115 10z" /></svg>
								{selectedEnv()!.deploy_freeze ? 'Unfreeze' : 'Freeze'}
							</button>
						</div>

						{/* Lock info */}
						<Show when={selectedEnv()!.lock_owner_id}>
							<div class="mb-4 p-3 rounded-lg bg-red-500/10 border border-red-500/20">
								<div class="flex items-center gap-2 text-sm text-red-400">
									<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 1a4.5 4.5 0 00-4.5 4.5V9H5a2 2 0 00-2 2v6a2 2 0 002 2h10a2 2 0 002-2v-6a2 2 0 00-2-2h-.5V5.5A4.5 4.5 0 0010 1zm3 8V5.5a3 3 0 10-6 0V9h6z" clip-rule="evenodd" /></svg>
									<span>Locked{selectedEnv()!.lock_reason ? `: ${selectedEnv()!.lock_reason}` : ''}</span>
									<Show when={selectedEnv()!.locked_at}>
										<span class="text-xs text-red-400/60">({formatRelativeTime(selectedEnv()!.locked_at!)})</span>
									</Show>
								</div>
							</div>
						</Show>

						{/* Approval Required Banner */}
						<Show when={approvalBanner().show && approvalBanner().approval}>
							<div class="mb-4 p-3 rounded-lg bg-amber-500/10 border border-amber-500/20">
								<div class="flex items-center justify-between">
									<div class="flex items-center gap-2 text-sm text-amber-400">
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" /></svg>
										<span>Deployment is pending approval ({approvalBanner().approval!.current_approvals}/{approvalBanner().approval!.min_approvals} approvals)</span>
									</div>
									<button class="text-xs text-amber-400 hover:text-amber-300 underline" onClick={() => setApprovalBanner({ show: false })}>Dismiss</button>
								</div>
							</div>
						</Show>

						{/* Approval Required indicator for pending deployments */}
						<Show when={deployments().some(d => d.status === 'pending')}>
							<div class="mb-4 p-3 rounded-lg bg-amber-500/5 border border-amber-500/15">
								<div class="flex items-center gap-2 text-sm text-amber-400">
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
									<span>One or more deployments are awaiting approval</span>
									<a href="/approvals" class="ml-auto text-xs text-indigo-400 hover:text-indigo-300">View Approvals →</a>
								</div>
							</div>
						</Show>

						{/* ===== Deployment Progress Section (strategy-specific) ===== */}
						<Show when={activeDeployment()}>
							{(dep) => (
								<div class="mb-6 p-4 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
									<h3 class="text-sm font-semibold text-[var(--color-text-primary)] mb-3 flex items-center gap-2">
										<svg class="w-4 h-4 animate-spin text-indigo-400" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
										Active Deployment
										<Badge variant={strategyVariant(dep().strategy)} size="sm">{strategyLabel(dep().strategy)}</Badge>
									</h3>

									{/* Canary Progress */}
									<Show when={dep().strategy === 'canary'}>
										<div class="space-y-3">
											<div class="flex items-center justify-between text-sm">
												<span class="text-[var(--color-text-secondary)]">Canary Traffic</span>
												<span class="font-mono text-amber-400">{dep().canary_weight}%</span>
											</div>
											<div class="w-full h-3 bg-[var(--color-bg-tertiary)] rounded-full overflow-hidden">
												<div class="h-full bg-gradient-to-r from-amber-500 to-amber-400 rounded-full transition-all duration-500" style={{ width: `${dep().canary_weight}%` }} />
											</div>
											<div class="flex items-center gap-2">
												<Show when={dep().canary_weight < 100}>
													{(() => {
														const cfg = parseStrategyConfig<CanaryConfig>(selectedEnv()!.strategy_config, defaultCanaryConfig);
														const currentStep = cfg.steps.findIndex(s => s.weight > dep().canary_weight);
														const nextWeight = currentStep >= 0 ? cfg.steps[currentStep].weight : 100;
														return (
															<Button size="sm" onClick={() => handleAdvanceCanary(dep(), nextWeight)} loading={advancingCanary()}>
																Advance to {nextWeight}%
															</Button>
														);
													})()}
												</Show>
												<Show when={dep().canary_weight >= 100}>
													<Badge variant="success" size="sm">Fully promoted</Badge>
												</Show>
												<Button size="sm" variant="ghost" onClick={() => handleCheckHealth(dep())} loading={checkingHealth()}>
													Check Health
												</Button>
											</div>
										</div>
									</Show>

									{/* Blue-Green Progress */}
									<Show when={dep().strategy === 'blue_green'}>
										<div class="space-y-3">
											<div class="flex items-center gap-4">
												<div class="flex-1 p-3 rounded-lg bg-blue-500/10 border border-blue-500/20 text-center">
													<p class="text-xs text-blue-400 font-semibold uppercase mb-1">Blue (Current)</p>
													<Badge variant="success" size="sm">Active</Badge>
												</div>
												<svg class="w-6 h-6 text-[var(--color-text-tertiary)]" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M3 10a.75.75 0 01.75-.75h10.638L10.23 5.29a.75.75 0 111.04-1.08l5.5 5.25a.75.75 0 010 1.08l-5.5 5.25a.75.75 0 11-1.04-1.08l4.158-3.96H3.75A.75.75 0 013 10z" clip-rule="evenodd" /></svg>
												<div class="flex-1 p-3 rounded-lg bg-green-500/10 border border-green-500/20 text-center">
													<p class="text-xs text-green-400 font-semibold uppercase mb-1">Green (New)</p>
													<Badge variant="running" size="sm">Deploying</Badge>
												</div>
											</div>
											<Button size="sm" variant="ghost" onClick={() => handleCheckHealth(dep())} loading={checkingHealth()}>
												Check Health
											</Button>
										</div>
									</Show>

									{/* Rolling Progress */}
									<Show when={dep().strategy === 'rolling'}>
										<div class="space-y-3">
											{(() => {
												const state = parseStrategyConfig<{ completed_batches?: number; total_batches?: number }>(dep().strategy_state, { completed_batches: 0, total_batches: 1 });
												const completed = state.completed_batches ?? 0;
												const total = state.total_batches ?? 1;
												const pct = total > 0 ? Math.round((completed / total) * 100) : 0;
												return (
													<>
														<div class="flex items-center justify-between text-sm">
															<span class="text-[var(--color-text-secondary)]">Rolling Update Progress</span>
															<span class="font-mono text-indigo-400">{completed}/{total} batches ({pct}%)</span>
														</div>
														<div class="w-full h-3 bg-[var(--color-bg-tertiary)] rounded-full overflow-hidden">
															<div class="h-full bg-gradient-to-r from-indigo-500 to-indigo-400 rounded-full transition-all duration-500" style={{ width: `${pct}%` }} />
														</div>
													</>
												);
											})()}
											<Button size="sm" variant="ghost" onClick={() => handleCheckHealth(dep())} loading={checkingHealth()}>
												Check Health
											</Button>
										</div>
									</Show>

									{/* Recreate progress (simple) */}
									<Show when={dep().strategy === 'recreate' || !dep().strategy}>
										<div class="space-y-3">
											<div class="flex items-center gap-2 text-sm text-[var(--color-text-secondary)]">
												<span>Replacing all instances...</span>
											</div>
											<div class="w-full h-3 bg-[var(--color-bg-tertiary)] rounded-full overflow-hidden">
												<div class="h-full bg-gradient-to-r from-indigo-500 to-indigo-400 rounded-full animate-pulse" style={{ width: '60%' }} />
											</div>
											<Button size="sm" variant="ghost" onClick={() => handleCheckHealth(dep())} loading={checkingHealth()}>
												Check Health
											</Button>
										</div>
									</Show>

									{/* Latest health check result */}
									<Show when={latestHealthResult()}>
										{(result) => (
											<div class={`mt-3 p-3 rounded-lg border ${result().healthy ? 'bg-green-500/10 border-green-500/20' : 'bg-red-500/10 border-red-500/20'}`}>
												<div class="flex items-center justify-between text-sm">
													<div class="flex items-center gap-2">
														<Show when={result().healthy} fallback={
															<svg class="w-4 h-4 text-red-400" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clip-rule="evenodd" /></svg>
														}>
															<svg class="w-4 h-4 text-green-400" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" /></svg>
														</Show>
														<span class={result().healthy ? 'text-green-400' : 'text-red-400'}>
															{result().healthy ? 'Healthy' : 'Unhealthy'}
														</span>
													</div>
													<div class="flex items-center gap-3 text-xs text-[var(--color-text-tertiary)]">
														<span>Status: {result().status_code}</span>
														<span>Latency: {result().latency_ms}ms</span>
													</div>
												</div>
												<Show when={result().error}>
													<p class="text-xs text-red-400 mt-1">{result().error}</p>
												</Show>
											</div>
										)}
									</Show>
								</div>
							)}
						</Show>

						{/* ===== Health Check Results ===== */}
						<Show when={deployments().length > 0 && deployments()[0].health_check_results && deployments()[0].health_check_results !== '[]'}>
							<div class="mb-6">
								<h3 class="text-sm font-semibold text-[var(--color-text-primary)] mb-3">Recent Health Checks</h3>
								<div class="space-y-2">
									<For each={parseHealthResults(deployments()[0].health_check_results).slice(0, 5)}>
										{(result) => (
											<div class={`flex items-center justify-between p-2 rounded-lg text-xs ${result.healthy ? 'bg-green-500/5 border border-green-500/10' : 'bg-red-500/5 border border-red-500/10'}`}>
												<div class="flex items-center gap-2">
													<div class={`w-2 h-2 rounded-full ${result.healthy ? 'bg-green-400' : 'bg-red-400'}`} />
													<span class={result.healthy ? 'text-green-400' : 'text-red-400'}>
														{result.healthy ? 'OK' : 'Failed'}
													</span>
													<span class="text-[var(--color-text-tertiary)]">Status {result.status_code}</span>
													<span class="text-[var(--color-text-tertiary)]">{result.latency_ms}ms</span>
												</div>
												<span class="text-[var(--color-text-tertiary)]">{result.checked_at ? formatRelativeTime(result.checked_at) : ''}</span>
											</div>
										)}
									</For>
								</div>
							</div>
						</Show>

						{/* Deployment History */}
						<div class="mb-6">
							<h3 class="text-sm font-semibold text-[var(--color-text-primary)] mb-3">Deployment History</h3>
							<Show when={deployments().length > 0} fallback={
								<p class="text-sm text-[var(--color-text-tertiary)] py-4 text-center">No deployments yet.</p>
							}>
								{/* Timeline view */}
								<div class="space-y-0 relative">
									<div class="absolute left-[15px] top-3 bottom-3 w-[2px] bg-[var(--color-border-primary)]" />
									<For each={deployments()}>
										{(dep, i) => (
											<div class="relative flex items-start gap-3 pl-0">
												{/* Timeline dot */}
												<div class={`relative z-10 flex-shrink-0 w-[32px] flex justify-center pt-2`}>
													<div class={`w-3 h-3 rounded-full border-2 ${dep.status === 'live' ? 'bg-emerald-400 border-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.4)]' :
														dep.status === 'deploying' ? 'bg-violet-400 border-violet-400 animate-pulse' :
															dep.status === 'failed' ? 'bg-red-400 border-red-400' :
																dep.status === 'rolled_back' ? 'bg-amber-400 border-amber-400' :
																	dep.status === 'pending' ? 'bg-gray-400 border-gray-400' :
																		'bg-gray-500 border-gray-500'
														}`} />
												</div>
												{/* Content */}
												<div class={`flex-1 p-3 rounded-lg mb-2 border transition-colors ${i() === 0 ? 'bg-[var(--color-bg-secondary)] border-[var(--color-border-primary)]' : 'bg-transparent border-transparent hover:bg-[var(--color-bg-secondary)] hover:border-[var(--color-border-primary)]'
													}`}>
													<div class="flex items-center justify-between gap-2">
														<div class="flex items-center gap-2 min-w-0">
															<Badge variant={deploymentStatusVariant(dep.status)} dot size="sm">{deploymentStatusLabel(dep.status)}</Badge>
															<Show when={dep.version}>
																<span class="text-sm font-mono text-[var(--color-text-primary)]">{dep.version}</span>
															</Show>
															<Badge variant={strategyVariant(dep.strategy)} size="sm">{strategyLabel(dep.strategy)}</Badge>
															<Show when={dep.strategy === 'canary' && dep.canary_weight > 0 && dep.canary_weight < 100}>
																<span class="text-xs text-amber-400">{dep.canary_weight}%</span>
															</Show>
														</div>
														<span class="text-xs text-[var(--color-text-tertiary)] flex-shrink-0">{formatRelativeTime(dep.created_at)}</span>
													</div>
													<div class="flex items-center gap-3 mt-1 text-xs text-[var(--color-text-tertiary)]">
														<Show when={dep.commit_sha}>
															<span class="font-mono">{dep.commit_sha!.substring(0, 7)}</span>
														</Show>
														<Show when={dep.deployed_by}>
															<span>by {dep.deployed_by}</span>
														</Show>
														<Show when={dep.image_tag}>
															<span class="font-mono">{dep.image_tag}</span>
														</Show>
													</div>
												</div>
											</div>
										)}
									</For>
								</div>
							</Show>
						</div>

						{/* Environment Overrides */}
						<div>
							<h3 class="text-sm font-semibold text-[var(--color-text-primary)] mb-3">Environment-Specific Variables</h3>
							<KeyValueEditor
								items={overrides()}
								onChange={handleOverrideChange}
								keyPlaceholder="VARIABLE_NAME"
								valuePlaceholder="value"
							/>
							<Show when={overrides().length > 0 || overridesDirty()}>
								<div class="flex items-center justify-between mt-4 pt-3 border-t border-[var(--color-border-primary)]">
									<p class="text-xs text-[var(--color-text-tertiary)]">
										{overridesDirty() ? 'You have unsaved changes.' : 'All changes saved.'}
									</p>
									<Button size="sm" onClick={handleSaveOverrides} loading={savingOverrides()} disabled={!overridesDirty()}>
										Save Overrides
									</Button>
								</div>
							</Show>
						</div>
					</Show>
				</Modal>
			</Show>

			{/* ===== Strategy Config Modal ===== */}
			<Show when={showStrategyConfig()}>
				<Modal open={showStrategyConfig()} onClose={() => setShowStrategyConfig(false)} title="Deployment Strategy" description={`Configure deployment strategy for ${selectedEnv()?.name ?? 'environment'}`} size="lg" footer={
					<>
						<Button variant="ghost" onClick={() => setShowStrategyConfig(false)}>Cancel</Button>
						<Button onClick={handleSaveStrategy} loading={savingStrategy()}>Save Strategy</Button>
					</>
				}>
					<div class="space-y-6">
						{/* Strategy Type Selector */}
						<div>
							<label class="block text-sm font-medium text-[var(--color-text-secondary)] mb-2">Strategy Type</label>
							<div class="grid grid-cols-2 gap-3">
								{([
									{ value: 'recreate' as DeployStrategy, label: 'Recreate', desc: 'Stop all, then deploy new version' },
									{ value: 'rolling' as DeployStrategy, label: 'Rolling Update', desc: 'Gradually replace instances' },
									{ value: 'blue_green' as DeployStrategy, label: 'Blue-Green', desc: 'Deploy alongside, then switch traffic' },
									{ value: 'canary' as DeployStrategy, label: 'Canary', desc: 'Gradually shift traffic to new version' },
								] as const).map((opt) => (
									<label
										class={`relative flex flex-col p-3 rounded-lg border-2 cursor-pointer transition-all ${strategyType() === opt.value
											? 'border-indigo-500 bg-indigo-500/10'
											: 'border-[var(--color-border-primary)] hover:border-[var(--color-border-secondary)]'
											}`}
									>
										<input
											type="radio"
											name="strategy-type"
											value={opt.value}
											checked={strategyType() === opt.value}
											onChange={() => setStrategyType(opt.value)}
											class="sr-only"
										/>
										<span class="text-sm font-medium text-[var(--color-text-primary)]">{opt.label}</span>
										<span class="text-xs text-[var(--color-text-tertiary)] mt-1">{opt.desc}</span>
									</label>
								))}
							</div>
						</div>

						{/* Rolling Config */}
						<Show when={strategyType() === 'rolling'}>
							<div class="p-4 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] space-y-4">
								<h4 class="text-sm font-semibold text-[var(--color-text-primary)]">Rolling Update Settings</h4>
								<div class="grid grid-cols-3 gap-4">
									<div>
										<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Batch Size</label>
										<input
											type="number"
											min="1"
											value={rollingConfig().batch_size}
											onInput={(e) => setRollingConfig({ ...rollingConfig(), batch_size: parseInt(e.currentTarget.value) || 1 })}
											class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
										/>
									</div>
									<div>
										<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Max Surge (%)</label>
										<input
											type="number"
											min="0"
											max="100"
											value={rollingConfig().max_surge}
											onInput={(e) => setRollingConfig({ ...rollingConfig(), max_surge: parseInt(e.currentTarget.value) || 0 })}
											class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
										/>
									</div>
									<div>
										<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Max Unavailable (%)</label>
										<input
											type="number"
											min="0"
											max="100"
											value={rollingConfig().max_unavailable}
											onInput={(e) => setRollingConfig({ ...rollingConfig(), max_unavailable: parseInt(e.currentTarget.value) || 0 })}
											class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
										/>
									</div>
								</div>
							</div>
						</Show>

						{/* Blue-Green Config */}
						<Show when={strategyType() === 'blue_green'}>
							<div class="p-4 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] space-y-4">
								<h4 class="text-sm font-semibold text-[var(--color-text-primary)]">Blue-Green Settings</h4>
								<div>
									<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Validation Timeout (seconds)</label>
									<input
										type="number"
										min="30"
										value={blueGreenConfig().validation_timeout}
										onInput={(e) => setBlueGreenConfig({ ...blueGreenConfig(), validation_timeout: parseInt(e.currentTarget.value) || 300 })}
										class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
									/>
								</div>
								<div class="flex items-center gap-3">
									<label class="relative inline-flex items-center cursor-pointer">
										<input type="checkbox" checked={blueGreenConfig().auto_promote} onChange={(e) => setBlueGreenConfig({ ...blueGreenConfig(), auto_promote: e.currentTarget.checked })} class="sr-only peer" />
										<div class="w-9 h-5 bg-[var(--color-bg-tertiary)] peer-focus:ring-2 peer-focus:ring-indigo-500/40 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-500"></div>
									</label>
									<span class="text-sm text-[var(--color-text-secondary)]">Auto-promote after health check passes</span>
								</div>
							</div>
						</Show>

						{/* Canary Config */}
						<Show when={strategyType() === 'canary'}>
							<div class="p-4 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] space-y-4">
								<h4 class="text-sm font-semibold text-[var(--color-text-primary)]">Canary Settings</h4>
								<div>
									<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Analysis Duration per Step (seconds)</label>
									<input
										type="number"
										min="30"
										value={canaryConfig().analysis_duration}
										onInput={(e) => setCanaryConfig({ ...canaryConfig(), analysis_duration: parseInt(e.currentTarget.value) || 300 })}
										class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
									/>
								</div>
								<div class="flex items-center gap-3">
									<label class="relative inline-flex items-center cursor-pointer">
										<input type="checkbox" checked={canaryConfig().auto_promote} onChange={(e) => setCanaryConfig({ ...canaryConfig(), auto_promote: e.currentTarget.checked })} class="sr-only peer" />
										<div class="w-9 h-5 bg-[var(--color-bg-tertiary)] peer-focus:ring-2 peer-focus:ring-indigo-500/40 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-500"></div>
									</label>
									<span class="text-sm text-[var(--color-text-secondary)]">Auto-promote through steps</span>
								</div>

								{/* Canary Steps Editor */}
								<div>
									<div class="flex items-center justify-between mb-2">
										<label class="text-xs text-[var(--color-text-tertiary)]">Traffic Weight Steps</label>
										<button
											type="button"
											onClick={addCanaryStep}
											class="text-xs text-indigo-400 hover:text-indigo-300"
										>+ Add Step</button>
									</div>
									<div class="space-y-2">
										<For each={canaryConfig().steps}>
											{(step, index) => (
												<div class="flex items-center gap-3 p-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)]">
													<span class="text-xs text-[var(--color-text-tertiary)] w-8">#{index() + 1}</span>
													<div class="flex-1">
														<label class="block text-xs text-[var(--color-text-tertiary)] mb-0.5">Weight %</label>
														<input
															type="number"
															min="1"
															max="100"
															value={step.weight}
															onInput={(e) => updateCanaryStep(index(), 'weight', parseInt(e.currentTarget.value) || 0)}
															class="w-full px-2 py-1 rounded bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] text-xs text-[var(--color-text-primary)] focus:outline-none focus:ring-1 focus:ring-indigo-500/40"
														/>
													</div>
													<div class="flex-1">
														<label class="block text-xs text-[var(--color-text-tertiary)] mb-0.5">Duration (s)</label>
														<input
															type="number"
															min="30"
															value={step.duration}
															onInput={(e) => updateCanaryStep(index(), 'duration', parseInt(e.currentTarget.value) || 60)}
															class="w-full px-2 py-1 rounded bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] text-xs text-[var(--color-text-primary)] focus:outline-none focus:ring-1 focus:ring-indigo-500/40"
														/>
													</div>
													<button
														type="button"
														onClick={() => removeCanaryStep(index())}
														class="text-red-400 hover:text-red-300 mt-3"
														title="Remove step"
													>
														<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5z" clip-rule="evenodd" /></svg>
													</button>
												</div>
											)}
										</For>
									</div>
									<Show when={canaryConfig().steps.length === 0}>
										<p class="text-xs text-[var(--color-text-tertiary)] py-3 text-center">No canary steps defined. Add at least one step.</p>
									</Show>
								</div>

								{/* Visual step preview */}
								<Show when={canaryConfig().steps.length > 0}>
									<div>
										<label class="block text-xs text-[var(--color-text-tertiary)] mb-2">Step Preview</label>
										<div class="flex items-end gap-1 h-16">
											<For each={canaryConfig().steps}>
												{(step) => (
													<div
														class="flex-1 bg-amber-500/20 border border-amber-500/30 rounded-t flex items-end justify-center transition-all"
														style={{ height: `${Math.max(step.weight, 5)}%` }}
													>
														<span class="text-[10px] text-amber-400 pb-1">{step.weight}%</span>
													</div>
												)}
											</For>
											<div
												class="flex-1 bg-green-500/20 border border-green-500/30 rounded-t flex items-end justify-center"
												style={{ height: '100%' }}
											>
												<span class="text-[10px] text-green-400 pb-1">100%</span>
											</div>
										</div>
									</div>
								</Show>
							</div>
						</Show>

						{/* Health Check Configuration */}
						<div class="p-4 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] space-y-4">
							<h4 class="text-sm font-semibold text-[var(--color-text-primary)]">Health Check Configuration</h4>
							<div class="grid grid-cols-2 gap-4">
								<div class="col-span-2">
									<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Health Check URL (base)</label>
									<input
										type="text"
										placeholder="https://staging.example.com"
										value={healthCheckUrl()}
										onInput={(e) => setHealthCheckUrl(e.currentTarget.value)}
										class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
									/>
								</div>
								<div>
									<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Health Check Path</label>
									<input
										type="text"
										placeholder="/health"
										value={healthCheckPath()}
										onInput={(e) => setHealthCheckPath(e.currentTarget.value)}
										class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
									/>
								</div>
								<div>
									<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Expected Status Code</label>
									<input
										type="number"
										min="100"
										max="599"
										value={healthCheckExpectedStatus()}
										onInput={(e) => setHealthCheckExpectedStatus(parseInt(e.currentTarget.value) || 200)}
										class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
									/>
								</div>
								<div>
									<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Interval (seconds)</label>
									<input
										type="number"
										min="5"
										value={healthCheckInterval()}
										onInput={(e) => setHealthCheckInterval(parseInt(e.currentTarget.value) || 30)}
										class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
									/>
								</div>
								<div>
									<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Timeout (seconds)</label>
									<input
										type="number"
										min="1"
										value={healthCheckTimeout()}
										onInput={(e) => setHealthCheckTimeout(parseInt(e.currentTarget.value) || 10)}
										class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
									/>
								</div>
								<div>
									<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Retries</label>
									<input
										type="number"
										min="0"
										max="10"
										value={healthCheckRetries()}
										onInput={(e) => setHealthCheckRetries(parseInt(e.currentTarget.value) || 3)}
										class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
									/>
								</div>
							</div>
						</div>
					</div>
				</Modal>
			</Show>

			{/* ===== Deploy Modal ===== */}
			<Show when={showDeploy()}>
				<Modal open={showDeploy()} onClose={() => setShowDeploy(false)} title="Trigger Deployment" description={`Deploy to ${selectedEnv()?.name ?? 'environment'} using ${strategyLabel(selectedEnv()?.strategy)} strategy`} footer={
					<>
						<Button variant="ghost" onClick={() => setShowDeploy(false)}>Cancel</Button>
						<Button onClick={handleDeploy} loading={deploying()}>Deploy</Button>
					</>
				}>
					<div class="space-y-4">
						<Input label="Version" placeholder="e.g. v1.2.3" value={deployVersion()} onInput={(e) => setDeployVersion(e.currentTarget.value)} />
						<Input label="Commit SHA" placeholder="e.g. abc1234" value={deployCommitSha()} onInput={(e) => setDeployCommitSha(e.currentTarget.value)} />
						<Input label="Image Tag" placeholder="e.g. myapp:latest" value={deployImageTag()} onInput={(e) => setDeployImageTag(e.currentTarget.value)} />
						<p class="text-xs text-[var(--color-text-tertiary)]">Fill in at least one field to identify the deployment.</p>
						<Show when={selectedEnv()?.strategy && selectedEnv()?.strategy !== 'recreate'}>
							<div class="p-3 rounded-lg bg-indigo-500/10 border border-indigo-500/20">
								<p class="text-xs text-indigo-400">
									<strong>Strategy:</strong> {strategyLabel(selectedEnv()?.strategy)}
									<Show when={selectedEnv()?.strategy === 'canary'}>
										{' — '}Traffic will be gradually shifted according to configured canary steps.
									</Show>
									<Show when={selectedEnv()?.strategy === 'blue_green'}>
										{' — '}New version will be deployed alongside current, then traffic will be switched.
									</Show>
									<Show when={selectedEnv()?.strategy === 'rolling'}>
										{' — '}Instances will be updated in batches to minimize downtime.
									</Show>
								</p>
							</div>
						</Show>
					</div>
				</Modal>
			</Show>

			{/* ===== Rollback Modal ===== */}
			<Show when={showRollback()}>
				<Modal open={showRollback()} onClose={() => setShowRollback(false)} title="Rollback Deployment" description="Select a previous deployment to rollback to" footer={
					<>
						<Button variant="ghost" onClick={() => setShowRollback(false)}>Cancel</Button>
						<Button variant="danger" onClick={handleRollback} loading={rollingBack()} disabled={!rollbackTargetId()}>Rollback</Button>
					</>
				}>
					<div class="space-y-3">
						<Show when={rollbackCandidates().length > 0} fallback={
							<p class="text-sm text-[var(--color-text-tertiary)]">No previous successful deployments to rollback to.</p>
						}>
							<For each={rollbackCandidates()}>
								{(dep) => (
									<label class={`flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors ${rollbackTargetId() === dep.id
										? 'border-indigo-500/50 bg-indigo-500/10'
										: 'border-[var(--color-border-primary)] hover:border-[var(--color-border-secondary)]'
										}`}>
										<input
											type="radio"
											name="rollback-target"
											value={dep.id}
											checked={rollbackTargetId() === dep.id}
											onChange={() => setRollbackTargetId(dep.id)}
											class="text-indigo-500"
										/>
										<div class="flex-1 min-w-0">
											<div class="flex items-center gap-2">
												<span class="text-sm font-mono text-[var(--color-text-primary)]">{dep.version || dep.commit_sha?.substring(0, 7) || dep.id.substring(0, 8)}</span>
												<Badge variant={deploymentStatusVariant(dep.status)} size="sm">{deploymentStatusLabel(dep.status)}</Badge>
											</div>
											<p class="text-xs text-[var(--color-text-tertiary)] mt-0.5">{formatRelativeTime(dep.created_at)} · {dep.deployed_by || 'unknown'}</p>
										</div>
									</label>
								)}
							</For>
						</Show>
					</div>
				</Modal>
			</Show>

			{/* ===== Promote Modal ===== */}
			<Show when={showPromote()}>
				<Modal open={showPromote()} onClose={() => setShowPromote(false)} title="Promote Deployment" description={`Promote a deployment from another environment to ${selectedEnv()?.name}`} footer={
					<>
						<Button variant="ghost" onClick={() => setShowPromote(false)}>Cancel</Button>
						<Button onClick={handlePromote} loading={promoting()} disabled={!promoteSourceEnvId()}>Promote</Button>
					</>
				}>
					<div class="space-y-3">
						<p class="text-sm text-[var(--color-text-secondary)] mb-2">Select source environment:</p>
						<For each={otherEnvs()}>
							{(env) => (
								<label class={`flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors ${promoteSourceEnvId() === env.id
									? 'border-indigo-500/50 bg-indigo-500/10'
									: 'border-[var(--color-border-primary)] hover:border-[var(--color-border-secondary)]'
									}`}>
									<input
										type="radio"
										name="promote-source"
										value={env.id}
										checked={promoteSourceEnvId() === env.id}
										onChange={() => setPromoteSourceEnvId(env.id)}
										class="text-indigo-500"
									/>
									<div class="flex-1 min-w-0">
										<div class="flex items-center gap-2">
											<span class="text-sm font-medium text-[var(--color-text-primary)]">{env.name}</span>
											<Show when={env.is_production}>
												<Badge variant="warning" size="sm">Production</Badge>
											</Show>
										</div>
										<p class="text-xs text-[var(--color-text-tertiary)] font-mono">{env.slug}</p>
									</div>
								</label>
							)}
						</For>
					</div>
				</Modal>
			</Show>

			{/* ===== Lock Modal ===== */}
			<Show when={showLock()}>
				<Modal open={showLock()} onClose={() => setShowLock(false)} title="Lock Environment" description="Prevent new deployments to this environment" size="sm" footer={
					<>
						<Button variant="ghost" onClick={() => setShowLock(false)}>Cancel</Button>
						<Button variant="danger" onClick={handleLock} loading={locking()}>Lock Environment</Button>
					</>
				}>
					<Input label="Reason (optional)" placeholder="e.g. Release in progress" value={lockReason()} onInput={(e) => setLockReason(e.currentTarget.value)} />
				</Modal>
			</Show>

			{/* ===== Delete Environment Confirm ===== */}
			<Show when={showDeleteEnv()}>
				<Modal open={showDeleteEnv()} onClose={() => setShowDeleteEnv(false)} title="Delete Environment" size="sm" footer={
					<>
						<Button variant="ghost" onClick={() => setShowDeleteEnv(false)}>Cancel</Button>
						<Button variant="danger" onClick={handleDeleteEnv} loading={deletingEnv()}>Delete</Button>
					</>
				}>
					<p class="text-sm text-[var(--color-text-secondary)]">
						Are you sure you want to delete <strong>{selectedEnv()?.name}</strong>? This will remove the environment and all its deployment history. This action cannot be undone.
					</p>
				</Modal>
			</Show>

			{/* ===== Protection Rules Modal ===== */}
			<Show when={showProtectionRules()}>
				<Modal open={showProtectionRules()} onClose={() => setShowProtectionRules(false)} title="Protection Rules" description={`Configure approval requirements for ${selectedEnv()?.name ?? 'environment'}`} size="sm" footer={
					<>
						<Button variant="ghost" onClick={() => setShowProtectionRules(false)}>Cancel</Button>
						<Button onClick={handleSaveProtectionRules} loading={savingProtection()}>Save Rules</Button>
					</>
				}>
					<div class="space-y-4">
						<div class="flex items-center gap-3">
							<label class="relative inline-flex items-center cursor-pointer">
								<input type="checkbox" checked={protRequireApproval()} onChange={(e) => setProtRequireApproval(e.currentTarget.checked)} class="sr-only peer" />
								<div class="w-9 h-5 bg-[var(--color-bg-tertiary)] peer-focus:ring-2 peer-focus:ring-indigo-500/40 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-500"></div>
							</label>
							<span class="text-sm text-[var(--color-text-secondary)]">Require approval before deployment</span>
						</div>
						<Show when={protRequireApproval()}>
							<div>
								<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Minimum Approvals Required</label>
								<input
									type="number"
									min="1"
									max="10"
									value={protMinApprovals()}
									onInput={(e) => setProtMinApprovals(parseInt(e.currentTarget.value) || 1)}
									class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
								/>
							</div>
							<div>
								<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Required Approvers (comma-separated user IDs or names)</label>
								<input
									type="text"
									placeholder="e.g. user1, user2, admin"
									value={protApprovers()}
									onInput={(e) => setProtApprovers(e.currentTarget.value)}
									class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40"
								/>
								<p class="text-xs text-[var(--color-text-tertiary)] mt-1">Only these users will be able to approve deployments to this environment.</p>
							</div>
						</Show>
					</div>
				</Modal>
			</Show>
		</div>
	);
};

export default EnvironmentsTab;
