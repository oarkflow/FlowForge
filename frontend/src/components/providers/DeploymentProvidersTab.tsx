import { createSignal, createEffect, Show, For, onMount } from 'solid-js';
import { api } from '../../api/client';
import type { ProjectDeploymentProvider, DeploymentProviderType } from '../../types';
import ConfirmDialog from '../ui/ConfirmDialog';

// ─── Provider type metadata ─────────────────────────────────────────────────
const PROVIDER_TYPES: { value: DeploymentProviderType; label: string; icon: string }[] = [
	{ value: 'aws', label: 'Amazon Web Services', icon: '☁️' },
	{ value: 'gcp', label: 'Google Cloud Platform', icon: '🌐' },
	{ value: 'azure', label: 'Microsoft Azure', icon: '🔷' },
	{ value: 'digitalocean', label: 'DigitalOcean', icon: '🌊' },
	{ value: 'custom', label: 'Custom Provider', icon: '⚙️' },
];

function getProviderMeta(type: string) {
	return PROVIDER_TYPES.find(t => t.value === type) ?? { value: 'custom' as const, label: type, icon: '⚙️' };
}

// ─── AWS Regions ─────────────────────────────────────────────────────────────
const AWS_REGIONS = [
	'us-east-1', 'us-east-2', 'us-west-1', 'us-west-2',
	'eu-west-1', 'eu-west-2', 'eu-west-3', 'eu-central-1', 'eu-north-1',
	'ap-southeast-1', 'ap-southeast-2', 'ap-northeast-1', 'ap-northeast-2', 'ap-south-1',
	'sa-east-1', 'ca-central-1', 'me-south-1', 'af-south-1',
];

type AuthMode = 'access_key' | 'assume_role' | 'default';

// ═══════════════════════════════════════════════════════════════════════════
// DeploymentProvidersTab Component
// ═══════════════════════════════════════════════════════════════════════════

interface DeploymentProvidersTabProps {
	projectId: string;
}

export default function DeploymentProvidersTab(props: DeploymentProvidersTabProps) {
	const [providers, setProviders] = createSignal<ProjectDeploymentProvider[]>([]);
	const [loading, setLoading] = createSignal(true);
	const [error, setError] = createSignal('');

	// Modal states
	const [showCreateModal, setShowCreateModal] = createSignal(false);
	const [editingProvider, setEditingProvider] = createSignal<ProjectDeploymentProvider | null>(null);
	const [deletingProvider, setDeletingProvider] = createSignal<ProjectDeploymentProvider | null>(null);

	// Test result states
	const [testResults, setTestResults] = createSignal<Record<string, { success: boolean; message: string }>>({});
	const [testingId, setTestingId] = createSignal<string | null>(null);

	// ─── Load providers ──────────────────────────────────────────────────────
	async function loadProviders() {
		setLoading(true);
		setError('');
		try {
			const data = await api.deploymentProviders.list(props.projectId);
			setProviders(data ?? []);
		} catch (err: any) {
			setError(err.message || 'Failed to load deployment providers');
		} finally {
			setLoading(false);
		}
	}

	onMount(() => loadProviders());

	createEffect(() => {
		const _id = props.projectId;
		if (_id) loadProviders();
	});

	// ─── Test connection ──────────────────────────────────────────────────────
	async function testConnection(provider: ProjectDeploymentProvider) {
		setTestingId(provider.id);
		try {
			const result = await api.deploymentProviders.test(props.projectId, provider.id);
			setTestResults(prev => ({ ...prev, [provider.id]: result }));
		} catch (err: any) {
			setTestResults(prev => ({ ...prev, [provider.id]: { success: false, message: err.message } }));
		} finally {
			setTestingId(null);
		}
	}

	// ─── Delete provider ──────────────────────────────────────────────────────
	async function confirmDeleteProvider() {
		const provider = deletingProvider();
		if (!provider) return;
		try {
			await api.deploymentProviders.delete(props.projectId, provider.id);
			await loadProviders();
		} catch (err: any) {
			setError(err.message || 'Failed to delete provider');
		} finally {
			setDeletingProvider(null);
		}
	}

	// ─── Get masked config summary ───────────────────────────────────────────
	function getConfigSummary(provider: ProjectDeploymentProvider): string {
		const config = provider.config;
		if (!config || Object.keys(config).length === 0) return 'No configuration';
		const parts: string[] = [];
		if (config.region) parts.push(`Region: ${config.region}`);
		if (config.auth_mode) parts.push(`Auth: ${config.auth_mode}`);
		if (config.role_arn) parts.push(`Role: ${typeof config.role_arn === 'string' && config.role_arn !== '***' ? config.role_arn : '***'}`);
		return parts.length > 0 ? parts.join(' · ') : 'Configured';
	}

	// ─── Render ───────────────────────────────────────────────────────────────
	return (
		<div class="space-y-6">
			{/* Header */}
			<div class="flex items-center justify-between">
				<div>
					<h2 class="text-xl font-semibold text-white">Deployment Providers</h2>
					<p class="text-sm text-gray-400 mt-1">Manage cloud provider connections for deploying this project</p>
				</div>
				<button
					onClick={() => setShowCreateModal(true)}
					class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
				>
					+ Add Provider
				</button>
			</div>

			{/* Error banner */}
			<Show when={error()}>
				<div class="bg-red-500/10 border border-red-500/30 rounded-lg p-4 text-red-400 text-sm">
					{error()}
					<button onClick={() => setError('')} class="ml-2 underline">dismiss</button>
				</div>
			</Show>

			{/* Loading */}
			<Show when={loading()}>
				<div class="flex items-center justify-center py-12">
					<div class="animate-spin h-8 w-8 border-2 border-blue-500 border-t-transparent rounded-full" />
					<span class="ml-3 text-gray-400">Loading providers…</span>
				</div>
			</Show>

			{/* Empty state */}
			<Show when={!loading() && providers().length === 0}>
				<div class="text-center py-16 bg-gray-800/50 rounded-xl border border-gray-700/50">
					<div class="text-5xl mb-4">☁️</div>
					<h3 class="text-lg font-medium text-white mb-2">No deployment providers configured</h3>
					<p class="text-sm text-gray-400 mb-6 max-w-md mx-auto">
						Connect a cloud provider to enable automated deployments. Supports AWS, GCP, Azure, DigitalOcean, and custom providers.
					</p>
					<button
						onClick={() => setShowCreateModal(true)}
						class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
					>
						+ Add Your First Provider
					</button>
				</div>
			</Show>

			{/* Provider cards */}
			<Show when={!loading() && providers().length > 0}>
				<div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
					<For each={providers()}>
						{(provider) => {
							const meta = getProviderMeta(provider.provider_type);
							const testResult = () => testResults()[provider.id];
							const isTesting = () => testingId() === provider.id;

							return (
								<div class="bg-gray-800/60 border border-gray-700/50 rounded-xl p-5 hover:border-gray-600/50 transition-colors">
									{/* Header row */}
									<div class="flex items-start justify-between mb-3">
										<div class="flex items-center gap-3">
											<span class="text-2xl">{meta.icon}</span>
											<div>
												<div class="flex items-center gap-2">
													<h3 class="text-white font-medium">{provider.name}</h3>
													<Show when={provider.is_default}>
														<span class="text-xs bg-yellow-500/20 text-yellow-400 px-2 py-0.5 rounded-full">
															⭐ Default
														</span>
													</Show>
												</div>
												<p class="text-xs text-gray-400 mt-0.5">{meta.label}</p>
											</div>
										</div>
										<div class="flex items-center gap-2">
											<span class={`text-xs px-2 py-1 rounded-full ${provider.is_active ? 'bg-green-500/20 text-green-400' : 'bg-gray-600/30 text-gray-400'}`}>
												{provider.is_active ? '● Active' : '○ Inactive'}
											</span>
											<Show when={testResult()}>
												<span class={`text-xs px-2 py-1 rounded-full ${testResult()!.success ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'}`}>
													{testResult()!.success ? '✓ Connected' : '✗ Failed'}
												</span>
											</Show>
										</div>
									</div>

									{/* Config summary */}
									<div class="space-y-1 text-sm mb-4">
										<div class="flex items-center gap-2 text-gray-400">
											<span class="text-gray-500">Config:</span>
											<span class="truncate">{getConfigSummary(provider)}</span>
										</div>
										<div class="flex items-center gap-2 text-gray-400">
											<span class="text-gray-500">Updated:</span>
											<span>{new Date(provider.updated_at).toLocaleDateString()}</span>
										</div>
									</div>

									{/* Test error message */}
									<Show when={testResult() && !testResult()!.success}>
										<div class="text-xs text-red-400 bg-red-500/10 rounded p-2 mb-3 break-words">
											{testResult()!.message}
										</div>
									</Show>

									{/* Actions */}
									<div class="flex flex-wrap gap-2">
										<button
											onClick={() => testConnection(provider)}
											disabled={isTesting()}
											class="px-3 py-1.5 text-xs bg-gray-700 hover:bg-gray-600 text-gray-300 rounded-lg transition-colors disabled:opacity-50"
										>
											{isTesting() ? 'Testing…' : 'Test Connection'}
										</button>
										<button
											onClick={() => setEditingProvider(provider)}
											class="px-3 py-1.5 text-xs bg-gray-700 hover:bg-gray-600 text-gray-300 rounded-lg transition-colors"
										>
											Edit
										</button>
										<button
											onClick={() => setDeletingProvider(provider)}
											class="px-3 py-1.5 text-xs bg-red-900/30 hover:bg-red-900/50 text-red-400 rounded-lg transition-colors"
										>
											Delete
										</button>
									</div>
								</div>
							);
						}}
					</For>
				</div>
			</Show>

			{/* Create Modal */}
			<Show when={showCreateModal()}>
				<ProviderFormModal
					projectId={props.projectId}
					onClose={() => setShowCreateModal(false)}
					onSaved={() => { setShowCreateModal(false); loadProviders(); }}
				/>
			</Show>

			{/* Edit Modal */}
			<Show when={editingProvider()}>
				<ProviderFormModal
					projectId={props.projectId}
					provider={editingProvider()!}
					onClose={() => setEditingProvider(null)}
					onSaved={() => { setEditingProvider(null); loadProviders(); }}
				/>
			</Show>

			{/* Delete Confirm Dialog */}
			<ConfirmDialog
				open={!!deletingProvider()}
				title="Delete Provider"
				onConfirm={confirmDeleteProvider}
				onCancel={() => setDeletingProvider(null)}
				confirmLabel="Delete"
				variant="danger"
			>
				<p class="text-sm text-[var(--color-text-secondary)]">
					Delete provider "{deletingProvider()?.name}"? This cannot be undone.
				</p>
			</ConfirmDialog>
		</div>
	);
}

// ═══════════════════════════════════════════════════════════════════════════
// Provider Form Modal (Create / Edit)
// ═══════════════════════════════════════════════════════════════════════════

interface ProviderFormModalProps {
	projectId: string;
	provider?: ProjectDeploymentProvider;
	onClose: () => void;
	onSaved: () => void;
}

function ProviderFormModal(props: ProviderFormModalProps) {
	const isEdit = () => !!props.provider;

	const [name, setName] = createSignal(props.provider?.name ?? '');
	const [providerType, setProviderType] = createSignal<DeploymentProviderType>(
		(props.provider?.provider_type as DeploymentProviderType) ?? 'aws'
	);
	const [isActive, setIsActive] = createSignal(props.provider?.is_active ?? true);
	const [saving, setSaving] = createSignal(false);
	const [formError, setFormError] = createSignal('');
	const [testResult, setTestResult] = createSignal<{ success: boolean; message: string } | null>(null);
	const [testing, setTesting] = createSignal(false);

	// AWS config fields
	const [awsRegion, setAwsRegion] = createSignal(props.provider?.config?.region ?? 'us-east-1');
	const [awsAuthMode, setAwsAuthMode] = createSignal<AuthMode>(props.provider?.config?.auth_mode ?? 'access_key');
	const [awsAccessKeyId, setAwsAccessKeyId] = createSignal(
		props.provider?.config?.access_key_id && props.provider.config.access_key_id !== '***'
			? props.provider.config.access_key_id : ''
	);
	const [awsSecretAccessKey, setAwsSecretAccessKey] = createSignal('');
	const [awsRoleArn, setAwsRoleArn] = createSignal(
		props.provider?.config?.role_arn && props.provider.config.role_arn !== '***'
			? props.provider.config.role_arn : ''
	);
	const [awsExternalId, setAwsExternalId] = createSignal(props.provider?.config?.external_id ?? '');
	const [awsSessionName, setAwsSessionName] = createSignal(props.provider?.config?.session_name ?? '');

	function buildConfig(): Record<string, any> {
		if (providerType() === 'aws') {
			const config: Record<string, any> = {
				region: awsRegion(),
				auth_mode: awsAuthMode(),
			};
			if (awsAuthMode() === 'access_key') {
				if (awsAccessKeyId()) config.access_key_id = awsAccessKeyId();
				if (awsSecretAccessKey()) config.secret_access_key = awsSecretAccessKey();
			} else if (awsAuthMode() === 'assume_role') {
				if (awsRoleArn()) config.role_arn = awsRoleArn();
				if (awsExternalId()) config.external_id = awsExternalId();
				if (awsSessionName()) config.session_name = awsSessionName();
			}
			return config;
		}
		return {};
	}

	function isFormValid(): boolean {
		if (!name().trim()) return false;
		if (providerType() === 'aws') {
			if (!awsRegion()) return false;
			if (awsAuthMode() === 'access_key' && !isEdit()) {
				if (!awsAccessKeyId() || !awsSecretAccessKey()) return false;
			}
			if (awsAuthMode() === 'assume_role' && !awsRoleArn()) return false;
		}
		return true;
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		setFormError('');
		setSaving(true);

		try {
			const config = buildConfig();
			if (isEdit()) {
				await api.deploymentProviders.update(props.projectId, props.provider!.id, {
					name: name(),
					provider_type: providerType(),
					config,
					is_active: isActive(),
				});
			} else {
				await api.deploymentProviders.create(props.projectId, {
					name: name(),
					provider_type: providerType(),
					config,
					is_active: isActive(),
				});
			}
			props.onSaved();
		} catch (err: any) {
			setFormError(err.message || 'Failed to save provider');
		} finally {
			setSaving(false);
		}
	}

	async function handleTestBeforeSave() {
		setTesting(true);
		setTestResult(null);
		if (!isEdit()) {
			setTestResult({ success: false, message: 'Save the provider first, then test the connection.' });
			setTesting(false);
			return;
		}
		try {
			const result = await api.deploymentProviders.test(props.projectId, props.provider!.id);
			setTestResult(result);
		} catch (err: any) {
			setTestResult({ success: false, message: err.message });
		} finally {
			setTesting(false);
		}
	}

	return (
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={(e) => { if (e.target === e.currentTarget) props.onClose(); }}>
			<div class="bg-gray-900 border border-gray-700 rounded-2xl shadow-2xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
				{/* Header */}
				<div class="flex items-center justify-between p-6 border-b border-gray-700/50">
					<h3 class="text-lg font-semibold text-white">
						{isEdit() ? 'Edit Provider' : 'Add Deployment Provider'}
					</h3>
					<button onClick={props.onClose} class="text-gray-400 hover:text-white text-xl">&times;</button>
				</div>

				{/* Form */}
				<form onSubmit={handleSubmit} class="p-6 space-y-4">
					{/* Error */}
					<Show when={formError()}>
						<div class="bg-red-500/10 border border-red-500/30 rounded-lg p-3 text-red-400 text-sm">{formError()}</div>
					</Show>

					{/* Name */}
					<div>
						<label class="block text-sm font-medium text-gray-300 mb-1">Provider Name</label>
						<input
							type="text"
							value={name()}
							onInput={(e) => setName(e.currentTarget.value)}
							placeholder="e.g. AWS Production"
							required
							class="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-500 text-sm focus:outline-none focus:border-blue-500"
						/>
					</div>

					{/* Provider Type */}
					<div>
						<label class="block text-sm font-medium text-gray-300 mb-1">Provider Type</label>
						<select
							value={providerType()}
							onChange={(e) => setProviderType(e.currentTarget.value as DeploymentProviderType)}
							disabled={isEdit()}
							class="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white text-sm focus:outline-none focus:border-blue-500 disabled:opacity-60"
						>
							<For each={PROVIDER_TYPES}>
								{(pt) => <option value={pt.value}>{pt.icon} {pt.label}</option>}
							</For>
						</select>
					</div>

					{/* AWS-specific fields */}
					<Show when={providerType() === 'aws'}>
						<div class="space-y-4 p-4 bg-gray-800/50 rounded-lg border border-gray-700/50">
							<h4 class="text-sm font-medium text-gray-300 flex items-center gap-2">
								<span>☁️</span> AWS Configuration
							</h4>

							{/* Region */}
							<div>
								<label class="block text-sm font-medium text-gray-300 mb-1">Region</label>
								<select
									value={awsRegion()}
									onChange={(e) => setAwsRegion(e.currentTarget.value)}
									class="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white text-sm focus:outline-none focus:border-blue-500"
								>
									<For each={AWS_REGIONS}>
										{(region) => <option value={region}>{region}</option>}
									</For>
								</select>
							</div>

							{/* Auth Mode */}
							<div>
								<label class="block text-sm font-medium text-gray-300 mb-2">Authentication Mode</label>
								<div class="space-y-2">
									<label class="flex items-center gap-3 p-3 rounded-lg border border-gray-600 cursor-pointer hover:border-gray-500 transition-colors"
										classList={{ '!border-blue-500 bg-blue-500/10': awsAuthMode() === 'access_key' }}>
										<input type="radio" name="auth_mode" value="access_key"
											checked={awsAuthMode() === 'access_key'}
											onChange={() => setAwsAuthMode('access_key')}
											class="text-blue-600" />
										<div>
											<p class="text-sm text-white font-medium">Access Key</p>
											<p class="text-xs text-gray-400">Use IAM access key and secret</p>
										</div>
									</label>
									<label class="flex items-center gap-3 p-3 rounded-lg border border-gray-600 cursor-pointer hover:border-gray-500 transition-colors"
										classList={{ '!border-blue-500 bg-blue-500/10': awsAuthMode() === 'assume_role' }}>
										<input type="radio" name="auth_mode" value="assume_role"
											checked={awsAuthMode() === 'assume_role'}
											onChange={() => setAwsAuthMode('assume_role')}
											class="text-blue-600" />
										<div>
											<p class="text-sm text-white font-medium">Assume Role</p>
											<p class="text-xs text-gray-400">Use STS AssumeRole with an IAM role ARN</p>
										</div>
									</label>
									<label class="flex items-center gap-3 p-3 rounded-lg border border-gray-600 cursor-pointer hover:border-gray-500 transition-colors"
										classList={{ '!border-blue-500 bg-blue-500/10': awsAuthMode() === 'default' }}>
										<input type="radio" name="auth_mode" value="default"
											checked={awsAuthMode() === 'default'}
											onChange={() => setAwsAuthMode('default')}
											class="text-blue-600" />
										<div>
											<p class="text-sm text-white font-medium">Default Credentials</p>
											<p class="text-xs text-gray-400">Use environment variables, instance profile, or shared credentials</p>
										</div>
									</label>
								</div>
							</div>

							{/* Access Key fields */}
							<Show when={awsAuthMode() === 'access_key'}>
								<div>
									<label class="block text-sm font-medium text-gray-300 mb-1">
										Access Key ID
										<Show when={isEdit()}>
											<span class="text-gray-500 font-normal ml-1">(leave empty to keep current)</span>
										</Show>
									</label>
									<input
										type="text"
										value={awsAccessKeyId()}
										onInput={(e) => setAwsAccessKeyId(e.currentTarget.value)}
										placeholder="AKIAXXXXXXXXXXXXXXXX"
										class="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-500 text-sm focus:outline-none focus:border-blue-500 font-mono"
									/>
								</div>
								<div>
									<label class="block text-sm font-medium text-gray-300 mb-1">
										Secret Access Key
										<Show when={isEdit()}>
											<span class="text-gray-500 font-normal ml-1">(leave empty to keep current)</span>
										</Show>
									</label>
									<input
										type="password"
										value={awsSecretAccessKey()}
										onInput={(e) => setAwsSecretAccessKey(e.currentTarget.value)}
										placeholder="Enter secret access key"
										class="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-500 text-sm focus:outline-none focus:border-blue-500"
									/>
								</div>
							</Show>

							{/* Assume Role fields */}
							<Show when={awsAuthMode() === 'assume_role'}>
								<div>
									<label class="block text-sm font-medium text-gray-300 mb-1">Role ARN</label>
									<input
										type="text"
										value={awsRoleArn()}
										onInput={(e) => setAwsRoleArn(e.currentTarget.value)}
										placeholder="arn:aws:iam::123456789012:role/MyRole"
										required
										class="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-500 text-sm focus:outline-none focus:border-blue-500 font-mono"
									/>
								</div>
								<div>
									<label class="block text-sm font-medium text-gray-300 mb-1">External ID <span class="text-gray-500 font-normal">(optional)</span></label>
									<input
										type="text"
										value={awsExternalId()}
										onInput={(e) => setAwsExternalId(e.currentTarget.value)}
										placeholder="Optional external ID for cross-account access"
										class="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-500 text-sm focus:outline-none focus:border-blue-500"
									/>
								</div>
								<div>
									<label class="block text-sm font-medium text-gray-300 mb-1">Session Name <span class="text-gray-500 font-normal">(optional)</span></label>
									<input
										type="text"
										value={awsSessionName()}
										onInput={(e) => setAwsSessionName(e.currentTarget.value)}
										placeholder="flowforge-deploy"
										class="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-500 text-sm focus:outline-none focus:border-blue-500"
									/>
								</div>
							</Show>

							{/* Default credentials note */}
							<Show when={awsAuthMode() === 'default'}>
								<div class="bg-blue-500/10 border border-blue-500/30 rounded-lg p-3 text-blue-400 text-xs">
									ℹ️ The agent will use the default AWS credential chain: environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY),
									shared credentials file (~/.aws/credentials), or instance metadata (EC2/ECS).
								</div>
							</Show>
						</div>
					</Show>

					{/* Non-AWS placeholder */}
					<Show when={providerType() !== 'aws'}>
						<div class="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-3 text-yellow-400 text-xs">
							⚠️ {getProviderMeta(providerType()).label} configuration support is coming soon.
							You can create the provider now and configure it later.
						</div>
					</Show>

					{/* Active toggle */}
					<div class="flex items-center gap-3">
						<label class="relative inline-flex items-center cursor-pointer">
							<input
								type="checkbox"
								checked={isActive()}
								onChange={(e) => setIsActive(e.currentTarget.checked)}
								class="sr-only peer"
							/>
							<div class="w-9 h-5 bg-gray-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600" />
						</label>
						<span class="text-sm text-gray-300">Active</span>
					</div>

					{/* Test result */}
					<Show when={testResult()}>
						<div class={`text-sm rounded-lg p-3 ${testResult()!.success ? 'bg-green-500/10 border border-green-500/30 text-green-400' : 'bg-red-500/10 border border-red-500/30 text-red-400'}`}>
							{testResult()!.success ? '✓ ' : '✗ '}{testResult()!.message}
						</div>
					</Show>

					{/* Actions */}
					<div class="flex items-center justify-between pt-4 border-t border-gray-700/50">
						<button
							type="button"
							onClick={handleTestBeforeSave}
							disabled={testing()}
							class="px-3 py-2 text-sm bg-gray-700 hover:bg-gray-600 text-gray-300 rounded-lg transition-colors disabled:opacity-50"
						>
							{testing() ? 'Testing…' : 'Test Connection'}
						</button>
						<div class="flex gap-2">
							<button
								type="button"
								onClick={props.onClose}
								class="px-4 py-2 text-sm bg-gray-700 hover:bg-gray-600 text-gray-300 rounded-lg transition-colors"
							>
								Cancel
							</button>
							<button
								type="submit"
								disabled={saving() || !isFormValid()}
								class="px-4 py-2 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors disabled:opacity-50"
							>
								{saving() ? 'Saving…' : isEdit() ? 'Update' : 'Create'}
							</button>
						</div>
					</div>
				</form>
			</div>
		</div>
	);
}
