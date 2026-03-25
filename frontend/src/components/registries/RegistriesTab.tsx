import { createSignal, createEffect, Show, For, onMount } from 'solid-js';
import { api } from '../../api/client';
import type { Registry, RegistryType, RegistryImage, RegistryTag } from '../../types';

// ─── Registry type metadata ─────────────────────────────────────────────────
const REGISTRY_TYPES: { value: RegistryType; label: string; icon: string }[] = [
	{ value: 'dockerhub', label: 'Docker Hub', icon: '🐳' },
	{ value: 'ghcr', label: 'GitHub Container Registry', icon: '🐙' },
	{ value: 'ecr', label: 'AWS ECR', icon: '☁️' },
	{ value: 'gcr', label: 'Google Container Registry', icon: '🌐' },
	{ value: 'acr', label: 'Azure Container Registry', icon: '🔷' },
	{ value: 'harbor', label: 'Harbor', icon: '⚓' },
	{ value: 'generic', label: 'Generic Registry', icon: '📦' },
];

function getRegistryMeta(type: RegistryType) {
	return REGISTRY_TYPES.find(t => t.value === type) ?? REGISTRY_TYPES[6];
}

// ─── Dynamic form field config ──────────────────────────────────────────────
interface FieldConfig {
	urlLabel?: string;
	urlPlaceholder?: string;
	urlRequired: boolean;
	userLabel: string;
	userPlaceholder: string;
	passLabel: string;
	passPlaceholder: string;
	note?: string;
}

function getFieldConfig(type: RegistryType): FieldConfig {
	switch (type) {
		case 'dockerhub':
			return {
				urlRequired: false,
				userLabel: 'Username',
				userPlaceholder: 'Docker Hub username',
				passLabel: 'Password / Access Token',
				passPlaceholder: 'Access token or password',
			};
		case 'ghcr':
			return {
				urlRequired: false,
				userLabel: 'GitHub User / Org',
				userPlaceholder: 'myuser or myorg',
				passLabel: 'Personal Access Token',
				passPlaceholder: 'ghp_xxxxxxxxxxxx',
			};
		case 'ecr':
			return {
				urlLabel: 'AWS Region',
				urlPlaceholder: 'us-east-1',
				urlRequired: true,
				userLabel: 'Access Key ID',
				userPlaceholder: 'AKIAXXXXXXXXXXXXXXXX',
				passLabel: 'Secret Access Key',
				passPlaceholder: 'Secret key',
				note: 'ECR integration requires AWS SDK. Full support coming in Cloud Integrations phase.',
			};
		case 'gcr':
			return {
				urlLabel: 'GCP Project ID',
				urlPlaceholder: 'my-project-id',
				urlRequired: true,
				userLabel: 'Service Account Email',
				userPlaceholder: 'sa@project.iam.gserviceaccount.com',
				passLabel: 'Service Account JSON Key',
				passPlaceholder: 'Paste JSON key content',
				note: 'GCR integration requires GCP SDK. Full support coming in Cloud Integrations phase.',
			};
		case 'acr':
			return {
				urlLabel: 'Registry URL',
				urlPlaceholder: 'myregistry.azurecr.io',
				urlRequired: true,
				userLabel: 'Username',
				userPlaceholder: 'ACR username',
				passLabel: 'Password',
				passPlaceholder: 'ACR password',
				note: 'ACR integration requires Azure SDK. Full support coming in Cloud Integrations phase.',
			};
		case 'harbor':
			return {
				urlLabel: 'Harbor URL',
				urlPlaceholder: 'https://harbor.example.com',
				urlRequired: true,
				userLabel: 'Username',
				userPlaceholder: 'admin',
				passLabel: 'Password',
				passPlaceholder: 'Harbor password',
			};
		default: // generic
			return {
				urlLabel: 'Registry URL',
				urlPlaceholder: 'https://registry.example.com',
				urlRequired: true,
				userLabel: 'Username',
				userPlaceholder: 'Username',
				passLabel: 'Password',
				passPlaceholder: 'Password or token',
			};
	}
}

function formatBytes(bytes: number): string {
	if (bytes === 0) return '0 B';
	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function truncateDigest(digest: string): string {
	if (!digest) return '';
	if (digest.startsWith('sha256:')) {
		return 'sha256:' + digest.slice(7, 19) + '…';
	}
	return digest.length > 20 ? digest.slice(0, 20) + '…' : digest;
}

// ═══════════════════════════════════════════════════════════════════════════
// RegistriesTab Component
// ═══════════════════════════════════════════════════════════════════════════

interface RegistriesTabProps {
	projectId: string;
}

export default function RegistriesTab(props: RegistriesTabProps) {
	const [registries, setRegistries] = createSignal<Registry[]>([]);
	const [loading, setLoading] = createSignal(true);
	const [error, setError] = createSignal('');

	// Modal states
	const [showCreateModal, setShowCreateModal] = createSignal(false);
	const [editingRegistry, setEditingRegistry] = createSignal<Registry | null>(null);
	const [browsingRegistry, setBrowsingRegistry] = createSignal<Registry | null>(null);

	// Test result states
	const [testResults, setTestResults] = createSignal<Record<string, { success: boolean; message: string }>>({});
	const [testingId, setTestingId] = createSignal<string | null>(null);

	// ─── Load registries ──────────────────────────────────────────────────────
	async function loadRegistries() {
		setLoading(true);
		setError('');
		try {
			const data = await api.registries.list(props.projectId);
			setRegistries(data ?? []);
		} catch (err: any) {
			setError(err.message || 'Failed to load registries');
		} finally {
			setLoading(false);
		}
	}

	onMount(() => loadRegistries());

	createEffect(() => {
		// Re-load when projectId changes
		const _id = props.projectId;
		if (_id) loadRegistries();
	});

	// ─── Test connection ──────────────────────────────────────────────────────
	async function testConnection(reg: Registry) {
		setTestingId(reg.id);
		try {
			const result = await api.registries.test(props.projectId, reg.id);
			setTestResults(prev => ({ ...prev, [reg.id]: result }));
		} catch (err: any) {
			setTestResults(prev => ({ ...prev, [reg.id]: { success: false, message: err.message } }));
		} finally {
			setTestingId(null);
		}
	}

	// ─── Delete registry ──────────────────────────────────────────────────────
	async function deleteRegistry(reg: Registry) {
		if (!confirm(`Delete registry "${reg.name}"? This cannot be undone.`)) return;
		try {
			await api.registries.delete(props.projectId, reg.id);
			await loadRegistries();
		} catch (err: any) {
			setError(err.message || 'Failed to delete registry');
		}
	}

	// ─── Set default ──────────────────────────────────────────────────────────
	async function setDefault(reg: Registry) {
		try {
			await api.registries.setDefault(props.projectId, reg.id);
			await loadRegistries();
		} catch (err: any) {
			setError(err.message || 'Failed to set default registry');
		}
	}

	// ─── Render ───────────────────────────────────────────────────────────────
	return (
		<div class="space-y-6">
			{/* Header */}
			<div class="flex items-center justify-between">
				<div>
					<h2 class="text-xl font-semibold text-white">Container Registries</h2>
					<p class="text-sm text-gray-400 mt-1">Manage container image registries for this project</p>
				</div>
				<button
					onClick={() => setShowCreateModal(true)}
					class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
				>
					+ Add Registry
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
					<span class="ml-3 text-gray-400">Loading registries…</span>
				</div>
			</Show>

			{/* Empty state */}
			<Show when={!loading() && registries().length === 0}>
				<div class="text-center py-16 bg-gray-800/50 rounded-xl border border-gray-700/50">
					<div class="text-5xl mb-4">📦</div>
					<h3 class="text-lg font-medium text-white mb-2">No registries configured</h3>
					<p class="text-sm text-gray-400 mb-6 max-w-md mx-auto">
						Add a container registry to store and manage your Docker images. Supports Docker Hub, GHCR, Harbor, and more.
					</p>
					<button
						onClick={() => setShowCreateModal(true)}
						class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
					>
						+ Add Your First Registry
					</button>
				</div>
			</Show>

			{/* Registry cards */}
			<Show when={!loading() && registries().length > 0}>
				<div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
					<For each={registries()}>
						{(reg) => {
							const meta = getRegistryMeta(reg.type);
							const testResult = () => testResults()[reg.id];
							const isTesting = () => testingId() === reg.id;

							return (
								<div class="bg-gray-800/60 border border-gray-700/50 rounded-xl p-5 hover:border-gray-600/50 transition-colors">
									{/* Header row */}
									<div class="flex items-start justify-between mb-3">
										<div class="flex items-center gap-3">
											<span class="text-2xl">{meta.icon}</span>
											<div>
												<div class="flex items-center gap-2">
													<h3 class="text-white font-medium">{reg.name}</h3>
													<Show when={reg.is_default}>
														<span class="text-xs bg-yellow-500/20 text-yellow-400 px-2 py-0.5 rounded-full flex items-center gap-1">
															⭐ Default
														</span>
													</Show>
												</div>
												<p class="text-xs text-gray-400 mt-0.5">{meta.label}</p>
											</div>
										</div>
										{/* Test status indicator */}
										<Show when={testResult()}>
											<span class={`text-xs px-2 py-1 rounded-full ${testResult()!.success ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'}`}>
												{testResult()!.success ? '✓ Connected' : '✗ Failed'}
											</span>
										</Show>
									</div>

									{/* Details */}
									<div class="space-y-1 text-sm mb-4">
										<Show when={reg.url}>
											<div class="flex items-center gap-2 text-gray-400">
												<span class="text-gray-500">URL:</span>
												<span class="truncate">{reg.url}</span>
											</div>
										</Show>
										<Show when={reg.username}>
											<div class="flex items-center gap-2 text-gray-400">
												<span class="text-gray-500">User:</span>
												<span>{reg.username}</span>
											</div>
										</Show>
										<div class="flex items-center gap-2 text-gray-400">
											<span class="text-gray-500">Added:</span>
											<span>{new Date(reg.created_at).toLocaleDateString()}</span>
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
											onClick={() => testConnection(reg)}
											disabled={isTesting()}
											class="px-3 py-1.5 text-xs bg-gray-700 hover:bg-gray-600 text-gray-300 rounded-lg transition-colors disabled:opacity-50"
										>
											{isTesting() ? 'Testing…' : 'Test Connection'}
										</button>
										<button
											onClick={() => setBrowsingRegistry(reg)}
											class="px-3 py-1.5 text-xs bg-gray-700 hover:bg-gray-600 text-gray-300 rounded-lg transition-colors"
										>
											Browse Images
										</button>
										<button
											onClick={() => setEditingRegistry(reg)}
											class="px-3 py-1.5 text-xs bg-gray-700 hover:bg-gray-600 text-gray-300 rounded-lg transition-colors"
										>
											Edit
										</button>
										<Show when={!reg.is_default}>
											<button
												onClick={() => setDefault(reg)}
												class="px-3 py-1.5 text-xs bg-gray-700 hover:bg-gray-600 text-yellow-400 rounded-lg transition-colors"
											>
												Set Default
											</button>
										</Show>
										<button
											onClick={() => deleteRegistry(reg)}
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
				<RegistryFormModal
					projectId={props.projectId}
					onClose={() => setShowCreateModal(false)}
					onSaved={() => { setShowCreateModal(false); loadRegistries(); }}
				/>
			</Show>

			{/* Edit Modal */}
			<Show when={editingRegistry()}>
				<RegistryFormModal
					projectId={props.projectId}
					registry={editingRegistry()!}
					onClose={() => setEditingRegistry(null)}
					onSaved={() => { setEditingRegistry(null); loadRegistries(); }}
				/>
			</Show>

			{/* Image Browser Modal */}
			<Show when={browsingRegistry()}>
				<ImageBrowserModal
					projectId={props.projectId}
					registry={browsingRegistry()!}
					onClose={() => setBrowsingRegistry(null)}
				/>
			</Show>
		</div>
	);
}

// ═══════════════════════════════════════════════════════════════════════════
// Registry Form Modal (Create / Edit)
// ═══════════════════════════════════════════════════════════════════════════

interface RegistryFormModalProps {
	projectId: string;
	registry?: Registry;
	onClose: () => void;
	onSaved: () => void;
}

function RegistryFormModal(props: RegistryFormModalProps) {
	const isEdit = () => !!props.registry;

	const [name, setName] = createSignal(props.registry?.name ?? '');
	const [type, setType] = createSignal<RegistryType>(props.registry?.type ?? 'dockerhub');
	const [url, setUrl] = createSignal(props.registry?.url ?? '');
	const [username, setUsername] = createSignal(props.registry?.username ?? '');
	const [password, setPassword] = createSignal('');
	const [isDefault, setIsDefault] = createSignal(props.registry?.is_default ?? false);
	const [saving, setSaving] = createSignal(false);
	const [formError, setFormError] = createSignal('');
	const [testResult, setTestResult] = createSignal<{ success: boolean; message: string } | null>(null);
	const [testing, setTesting] = createSignal(false);

	const fieldConfig = () => getFieldConfig(type());

	async function handleSubmit(e: Event) {
		e.preventDefault();
		setFormError('');
		setSaving(true);

		try {
			if (isEdit()) {
				const data: any = {
					name: name(),
					type: type(),
					url: url(),
					username: username(),
					is_default: isDefault(),
				};
				if (password()) data.password = password();
				await api.registries.update(props.projectId, props.registry!.id, data);
			} else {
				await api.registries.create(props.projectId, {
					name: name(),
					type: type(),
					url: url(),
					username: username(),
					password: password(),
					is_default: isDefault(),
				});
			}
			props.onSaved();
		} catch (err: any) {
			setFormError(err.message || 'Failed to save registry');
		} finally {
			setSaving(false);
		}
	}

	async function handleTestBeforeSave() {
		setTesting(true);
		setTestResult(null);
		// We can only test after the registry is created. For new registries, just show info.
		if (!isEdit()) {
			setTestResult({ success: false, message: 'Save the registry first, then test the connection.' });
			setTesting(false);
			return;
		}
		try {
			const result = await api.registries.test(props.projectId, props.registry!.id);
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
						{isEdit() ? 'Edit Registry' : 'Add Registry'}
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
						<label class="block text-sm font-medium text-gray-300 mb-1">Name</label>
						<input
							type="text"
							value={name()}
							onInput={(e) => setName(e.currentTarget.value)}
							placeholder="My Docker Registry"
							required
							class="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-500 text-sm focus:outline-none focus:border-blue-500"
						/>
					</div>

					{/* Type */}
					<div>
						<label class="block text-sm font-medium text-gray-300 mb-1">Registry Type</label>
						<select
							value={type()}
							onChange={(e) => setType(e.currentTarget.value as RegistryType)}
							class="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white text-sm focus:outline-none focus:border-blue-500"
						>
							<For each={REGISTRY_TYPES}>
								{(rt) => <option value={rt.value}>{rt.icon} {rt.label}</option>}
							</For>
						</select>
					</div>

					{/* Note for stub types */}
					<Show when={fieldConfig().note}>
						<div class="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-3 text-yellow-400 text-xs">
							⚠️ {fieldConfig().note}
						</div>
					</Show>

					{/* URL (conditional) */}
					<Show when={fieldConfig().urlRequired}>
						<div>
							<label class="block text-sm font-medium text-gray-300 mb-1">{fieldConfig().urlLabel ?? 'Registry URL'}</label>
							<input
								type="text"
								value={url()}
								onInput={(e) => setUrl(e.currentTarget.value)}
								placeholder={fieldConfig().urlPlaceholder}
								class="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-500 text-sm focus:outline-none focus:border-blue-500"
							/>
						</div>
					</Show>

					{/* Username */}
					<div>
						<label class="block text-sm font-medium text-gray-300 mb-1">{fieldConfig().userLabel}</label>
						<input
							type="text"
							value={username()}
							onInput={(e) => setUsername(e.currentTarget.value)}
							placeholder={fieldConfig().userPlaceholder}
							class="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-500 text-sm focus:outline-none focus:border-blue-500"
						/>
					</div>

					{/* Password / Token */}
					<div>
						<label class="block text-sm font-medium text-gray-300 mb-1">
							{fieldConfig().passLabel}
							<Show when={isEdit()}>
								<span class="text-gray-500 font-normal ml-1">(leave empty to keep current)</span>
							</Show>
						</label>
						<input
							type="password"
							value={password()}
							onInput={(e) => setPassword(e.currentTarget.value)}
							placeholder={fieldConfig().passPlaceholder}
							class="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-500 text-sm focus:outline-none focus:border-blue-500"
						/>
					</div>

					{/* Default toggle */}
					<div class="flex items-center gap-3">
						<label class="relative inline-flex items-center cursor-pointer">
							<input
								type="checkbox"
								checked={isDefault()}
								onChange={(e) => setIsDefault(e.currentTarget.checked)}
								class="sr-only peer"
							/>
							<div class="w-9 h-5 bg-gray-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600" />
						</label>
						<span class="text-sm text-gray-300">Set as default registry</span>
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
								disabled={saving() || !name()}
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

// ═══════════════════════════════════════════════════════════════════════════
// Image Browser Modal
// ═══════════════════════════════════════════════════════════════════════════

interface ImageBrowserModalProps {
	projectId: string;
	registry: Registry;
	onClose: () => void;
}

function ImageBrowserModal(props: ImageBrowserModalProps) {
	const [images, setImages] = createSignal<RegistryImage[]>([]);
	const [loading, setLoading] = createSignal(true);
	const [error, setError] = createSignal('');
	const [selectedImage, setSelectedImage] = createSignal<string | null>(null);
	const [tags, setTags] = createSignal<RegistryTag[]>([]);
	const [tagsLoading, setTagsLoading] = createSignal(false);
	const [tagsError, setTagsError] = createSignal('');

	onMount(async () => {
		try {
			const data = await api.registries.listImages(props.projectId, props.registry.id);
			setImages(data ?? []);
		} catch (err: any) {
			setError(err.message || 'Failed to load images');
		} finally {
			setLoading(false);
		}
	});

	async function loadTags(imageName: string) {
		setSelectedImage(imageName);
		setTagsLoading(true);
		setTagsError('');
		try {
			const data = await api.registries.listTags(props.projectId, props.registry.id, imageName);
			setTags(data ?? []);
		} catch (err: any) {
			setTagsError(err.message || 'Failed to load tags');
		} finally {
			setTagsLoading(false);
		}
	}

	async function deleteTag(imageName: string, tagName: string) {
		if (!confirm(`Delete tag "${tagName}" from "${imageName}"? This cannot be undone.`)) return;
		try {
			await api.registries.deleteTag(props.projectId, props.registry.id, imageName, tagName);
			// Reload tags
			await loadTags(imageName);
		} catch (err: any) {
			setTagsError(err.message || 'Failed to delete tag');
		}
	}

	const meta = getRegistryMeta(props.registry.type);

	return (
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={(e) => { if (e.target === e.currentTarget) props.onClose(); }}>
			<div class="bg-gray-900 border border-gray-700 rounded-2xl shadow-2xl w-full max-w-3xl max-h-[85vh] flex flex-col">
				{/* Header */}
				<div class="flex items-center justify-between p-6 border-b border-gray-700/50 shrink-0">
					<div class="flex items-center gap-3">
						<span class="text-2xl">{meta.icon}</span>
						<div>
							<h3 class="text-lg font-semibold text-white">Browse Images</h3>
							<p class="text-sm text-gray-400">{props.registry.name}</p>
						</div>
					</div>
					<button onClick={props.onClose} class="text-gray-400 hover:text-white text-xl">&times;</button>
				</div>

				<div class="flex-1 overflow-y-auto p-6">
					{/* Error */}
					<Show when={error()}>
						<div class="bg-red-500/10 border border-red-500/30 rounded-lg p-3 text-red-400 text-sm mb-4">{error()}</div>
					</Show>

					{/* Loading */}
					<Show when={loading()}>
						<div class="flex items-center justify-center py-12">
							<div class="animate-spin h-6 w-6 border-2 border-blue-500 border-t-transparent rounded-full" />
							<span class="ml-3 text-gray-400">Loading images…</span>
						</div>
					</Show>

					{/* No images */}
					<Show when={!loading() && images().length === 0 && !error()}>
						<div class="text-center py-12 text-gray-400">
							<div class="text-3xl mb-3">📭</div>
							<p>No images found in this registry</p>
						</div>
					</Show>

					{/* Image list (when no image selected) */}
					<Show when={!loading() && !selectedImage() && images().length > 0}>
						<div class="space-y-2">
							<For each={images()}>
								{(img) => (
									<button
										onClick={() => loadTags(img.name)}
										class="w-full text-left bg-gray-800/60 border border-gray-700/50 rounded-lg p-4 hover:border-blue-500/50 transition-colors"
									>
										<div class="flex items-center justify-between">
											<div>
												<h4 class="text-white font-medium">{img.name}</h4>
												<div class="flex items-center gap-4 mt-1 text-xs text-gray-400">
													<Show when={img.tags && img.tags.length > 0}>
														<span>{img.tags.length} tags</span>
													</Show>
													<Show when={img.size > 0}>
														<span>{formatBytes(img.size)}</span>
													</Show>
													<Show when={img.pushed_at}>
														<span>Pushed: {new Date(img.pushed_at).toLocaleDateString()}</span>
													</Show>
													<Show when={img.pull_count > 0}>
														<span>{img.pull_count.toLocaleString()} pulls</span>
													</Show>
												</div>
											</div>
											<span class="text-gray-500">→</span>
										</div>
									</button>
								)}
							</For>
						</div>
					</Show>

					{/* Tag list (when an image is selected) */}
					<Show when={selectedImage()}>
						<div>
							<button
								onClick={() => { setSelectedImage(null); setTags([]); setTagsError(''); }}
								class="flex items-center gap-2 text-sm text-blue-400 hover:text-blue-300 mb-4"
							>
								← Back to images
							</button>
							<h4 class="text-white font-medium mb-3">{selectedImage()}</h4>

							<Show when={tagsError()}>
								<div class="bg-red-500/10 border border-red-500/30 rounded-lg p-3 text-red-400 text-sm mb-4">{tagsError()}</div>
							</Show>

							<Show when={tagsLoading()}>
								<div class="flex items-center justify-center py-8">
									<div class="animate-spin h-6 w-6 border-2 border-blue-500 border-t-transparent rounded-full" />
									<span class="ml-3 text-gray-400">Loading tags…</span>
								</div>
							</Show>

							<Show when={!tagsLoading() && tags().length === 0 && !tagsError()}>
								<div class="text-center py-8 text-gray-400">No tags found</div>
							</Show>

							<Show when={!tagsLoading() && tags().length > 0}>
								<div class="overflow-x-auto">
									<table class="w-full text-sm">
										<thead>
											<tr class="text-left text-gray-400 border-b border-gray-700">
												<th class="pb-2 pr-4">Tag</th>
												<th class="pb-2 pr-4">Digest</th>
												<th class="pb-2 pr-4">Size</th>
												<th class="pb-2 pr-4">Created</th>
												<th class="pb-2"></th>
											</tr>
										</thead>
										<tbody>
											<For each={tags()}>
												{(tag) => (
													<tr class="border-b border-gray-800 hover:bg-gray-800/50">
														<td class="py-2.5 pr-4 text-white font-mono text-xs">{tag.name}</td>
														<td class="py-2.5 pr-4 text-gray-400 font-mono text-xs" title={tag.digest}>
															{truncateDigest(tag.digest)}
														</td>
														<td class="py-2.5 pr-4 text-gray-400">{tag.size > 0 ? formatBytes(tag.size) : '-'}</td>
														<td class="py-2.5 pr-4 text-gray-400">
															{tag.created_at ? new Date(tag.created_at).toLocaleDateString() : '-'}
														</td>
														<td class="py-2.5">
															<button
																onClick={() => deleteTag(selectedImage()!, tag.name)}
																class="text-xs text-red-400 hover:text-red-300"
															>
																Delete
															</button>
														</td>
													</tr>
												)}
											</For>
										</tbody>
									</table>
								</div>
							</Show>
						</div>
					</Show>
				</div>
			</div>
		</div>
	);
}
