import type { Component } from 'solid-js';
import { createSignal, createResource, For, Show, Switch, Match } from 'solid-js';
import PageContainer from '../../components/layout/PageContainer';
import Card from '../../components/ui/Card';
import Button from '../../components/ui/Button';
import Input from '../../components/ui/Input';
import Badge from '../../components/ui/Badge';
import Select from '../../components/ui/Select';
import Modal from '../../components/ui/Modal';
import { toast } from '../../components/ui/Toast';
import { api, ApiRequestError, apiClient } from '../../api/client';
import type { User, NotificationPreference } from '../../types';
import { copyToClipboard, formatRelativeTime } from '../../utils/helpers';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
interface ApiKey {
	id: string;
	name: string;
	prefix: string;
	scopes: string[];
	expires_at?: string;
	last_used_at?: string;
	created_at: string;
}

// ---------------------------------------------------------------------------
// Fetchers
// ---------------------------------------------------------------------------
async function fetchProfile(): Promise<User> {
	return api.users.me();
}

async function fetchApiKeys(): Promise<ApiKey[]> {
	return apiClient.get<ApiKey[]>('/users/me/api-keys');
}

// ---------------------------------------------------------------------------
// Sidebar nav items
// ---------------------------------------------------------------------------
const navItems = [
	{ id: 'profile', label: 'Profile', icon: 'M10 8a3 3 0 100-6 3 3 0 000 6zM3.465 14.493a1.23 1.23 0 00.41 1.412A9.957 9.957 0 0010 18c2.31 0 4.438-.784 6.131-2.1.43-.333.604-.903.408-1.41a7.002 7.002 0 00-13.074.003z' },
	{ id: 'api-keys', label: 'API Keys', icon: 'M8 16.25a.75.75 0 01.75-.75h2.5a.75.75 0 010 1.5h-2.5a.75.75 0 01-.75-.75zM3 13.25a.75.75 0 01.75-.75h12.5a.75.75 0 010 1.5H3.75a.75.75 0 01-.75-.75zM0 10.25a.75.75 0 01.75-.75h18.5a.75.75 0 010 1.5H.75a.75.75 0 01-.75-.75z' },
	{ id: 'notifications', label: 'Notifications', icon: 'M10 2a6 6 0 00-6 6c0 1.887-.454 3.665-1.257 5.234a.75.75 0 00.515 1.076 32.91 32.91 0 003.256.508 3.5 3.5 0 006.972 0 32.903 32.903 0 003.256-.508.75.75 0 00.515-1.076A11.448 11.448 0 0116 8a6 6 0 00-6-6z' },
	{ id: 'appearance', label: 'Appearance', icon: 'M1 4.25a3.733 3.733 0 012.25-.75h13.5c.844 0 1.623.279 2.25.75A2.25 2.25 0 0016.75 2H3.25A2.25 2.25 0 001 4.25zM1 7.25a3.733 3.733 0 012.25-.75h13.5c.844 0 1.623.279 2.25.75A2.25 2.25 0 0016.75 5H3.25A2.25 2.25 0 001 7.25zM7 8a1 1 0 011-1h13.5a2.25 2.25 0 012.25 2.25v7.5A2.25 2.25 0 0116.75 19H3.25A2.25 2.25 0 011 16.75v-7.5A2.25 2.25 0 013.25 7H7z' },
	{ id: 'security', label: 'Security', icon: 'M10 1a4.5 4.5 0 00-4.5 4.5V9H5a2 2 0 00-2 2v6a2 2 0 002 2h10a2 2 0 002-2v-6a2 2 0 00-2-2h-.5V5.5A4.5 4.5 0 0010 1zm3 8V5.5a3 3 0 10-6 0V9h6z' },
];

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const SettingsPage: Component = () => {
	const [activeSection, setActiveSection] = createSignal('profile');

	// Profile data from API
	const [profile, { refetch: refetchProfile }] = createResource(fetchProfile);
	const [apiKeys, { refetch: refetchApiKeys, mutate: mutateApiKeys }] = createResource(fetchApiKeys);

	// Profile form (populated from fetched data)
	const [displayName, setDisplayName] = createSignal('');
	const [email, setEmail] = createSignal('');
	const [username, setUsername] = createSignal('');
	const [profileLoaded, setProfileLoaded] = createSignal(false);
	const [savingProfile, setSavingProfile] = createSignal(false);

	// Password form
	const [currentPassword, setCurrentPassword] = createSignal('');
	const [newPassword, setNewPassword] = createSignal('');
	const [confirmPassword, setConfirmPassword] = createSignal('');
	const [savingPassword, setSavingPassword] = createSignal(false);

	// API key create
	const [showCreateKey, setShowCreateKey] = createSignal(false);
	const [newKeyName, setNewKeyName] = createSignal('');
	const [newKeyScopes, setNewKeyScopes] = createSignal<string[]>(['read']);
	const [newKeyExpiration, setNewKeyExpiration] = createSignal('90d');
	const [newKeyCreated, setNewKeyCreated] = createSignal('');
	const [creatingKey, setCreatingKey] = createSignal(false);

	// API key revoke confirmation
	const [keyToRevoke, setKeyToRevoke] = createSignal<ApiKey | null>(null);
	const [revokingKey, setRevokingKey] = createSignal(false);

	// Notification prefs
	const [notifPrefs, setNotifPrefs] = createSignal<NotificationPreference | null>(null);
	const [loadingNotifPrefs, setLoadingNotifPrefs] = createSignal(false);
	const [savingNotifs, setSavingNotifs] = createSignal(false);

	const fetchNotifPrefs = async () => {
		setLoadingNotifPrefs(true);
		try {
			const prefs = await api.notificationPrefs.get();
			setNotifPrefs(prefs);
		} catch {
			// Use defaults if fetch fails
			setNotifPrefs({
				id: '', user_id: '', email_enabled: true, in_app_enabled: true,
				pipeline_success: true, pipeline_failure: true,
				deployment_success: true, deployment_failure: true,
				approval_requested: true, approval_resolved: true,
				agent_offline: true, security_alerts: true,
				created_at: '', updated_at: '',
			});
		} finally {
			setLoadingNotifPrefs(false);
		}
	};

	const toggleNotifPref = (key: keyof NotificationPreference) => {
		const prefs = notifPrefs();
		if (!prefs) return;
		setNotifPrefs({ ...prefs, [key]: !prefs[key] } as NotificationPreference);
	};

	// Appearance
	const [theme, setTheme] = createSignal(localStorage.getItem('ff_theme') || 'dark');

	// Security
	const [showTotpSetup, setShowTotpSetup] = createSignal(false);
	const [totpSecret, setTotpSecret] = createSignal('');
	const [totpQrUrl, setTotpQrUrl] = createSignal('');
	const [totpCode, setTotpCode] = createSignal('');
	const [settingUpTotp, setSettingUpTotp] = createSignal(false);
	const [verifyingTotp, setVerifyingTotp] = createSignal(false);

	// Delete account confirm
	const [showDeleteConfirm, setShowDeleteConfirm] = createSignal(false);
	const [deleteConfirmText, setDeleteConfirmText] = createSignal('');

	// Sync profile form when data loads
	const syncProfileForm = () => {
		const p = profile();
		if (p && !profileLoaded()) {
			setDisplayName(p.display_name || '');
			setEmail(p.email);
			setUsername(p.username);
			setProfileLoaded(true);
		}
	};

	// ---------------------------------------------------------------------------
	// Handlers
	// ---------------------------------------------------------------------------
	const handleSaveProfile = async () => {
		setSavingProfile(true);
		try {
			await api.users.updateMe({
				display_name: displayName().trim() || undefined,
				email: email().trim(),
				username: username().trim(),
			});
			toast.success('Profile updated');
			refetchProfile();
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to update profile';
			toast.error(msg);
		} finally {
			setSavingProfile(false);
		}
	};

	const handleChangePassword = async () => {
		if (newPassword() !== confirmPassword()) {
			toast.error('Passwords do not match');
			return;
		}
		if (newPassword().length < 8) {
			toast.error('Password must be at least 8 characters');
			return;
		}
		setSavingPassword(true);
		try {
			await apiClient.put('/users/me/password', {
				current_password: currentPassword(),
				new_password: newPassword(),
			});
			toast.success('Password updated');
			setCurrentPassword('');
			setNewPassword('');
			setConfirmPassword('');
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to update password';
			toast.error(msg);
		} finally {
			setSavingPassword(false);
		}
	};

	const handleCreateKey = async () => {
		if (!newKeyName().trim()) return;
		setCreatingKey(true);
		try {
			const result = await apiClient.post<{ token: string; key: ApiKey }>('/users/me/api-keys', {
				name: newKeyName().trim(),
				scopes: newKeyScopes(),
				expiration: newKeyExpiration(),
			});
			setNewKeyCreated(result.token);
			refetchApiKeys();
			toast.success('API key created');
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to create API key';
			toast.error(msg);
		} finally {
			setCreatingKey(false);
		}
	};

	const handleRevokeKey = async () => {
		const key = keyToRevoke();
		if (!key) return;
		setRevokingKey(true);
		try {
			await apiClient.delete(`/users/me/api-keys/${key.id}`);
			mutateApiKeys(prev => prev?.filter(k => k.id !== key.id));
			setKeyToRevoke(null);
			toast.success(`API key "${key.name}" revoked`);
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to revoke API key';
			toast.error(msg);
		} finally {
			setRevokingKey(false);
		}
	};

	const handleSaveNotifications = async () => {
		const prefs = notifPrefs();
		if (!prefs) return;
		setSavingNotifs(true);
		try {
			await api.notificationPrefs.update(prefs);
			toast.success('Notification preferences saved');
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to save notification preferences';
			toast.error(msg);
		} finally {
			setSavingNotifs(false);
		}
	};

	const handleSaveAppearance = () => {
		localStorage.setItem('ff_theme', theme());
		document.documentElement.setAttribute('data-theme', theme());
		toast.success('Appearance settings saved');
	};

	const handleTotpSetup = async () => {
		setSettingUpTotp(true);
		try {
			const result = await api.auth.totpSetup();
			setTotpSecret(result.secret);
			setTotpQrUrl(result.qr_url);
			setShowTotpSetup(true);
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Failed to set up 2FA';
			toast.error(msg);
		} finally {
			setSettingUpTotp(false);
		}
	};

	const handleTotpVerify = async () => {
		if (totpCode().length !== 6) {
			toast.error('Enter a 6-digit verification code');
			return;
		}
		setVerifyingTotp(true);
		try {
			await api.auth.totpVerify(totpCode());
			toast.success('Two-factor authentication enabled');
			setShowTotpSetup(false);
			setTotpCode('');
			refetchProfile();
		} catch (err) {
			const msg = err instanceof ApiRequestError ? err.message : 'Invalid verification code';
			toast.error(msg);
		} finally {
			setVerifyingTotp(false);
		}
	};

	const getInitials = (user?: User) => {
		if (!user) return '??';
		const name = user.display_name || user.username;
		return name.split(' ').map(w => w[0]).join('').toUpperCase().slice(0, 2);
	};

	return (
		<PageContainer title="Settings" description="Manage your account and application settings">
			{/* Sync profile form data on load */}
			{syncProfileForm()}

			<div class="flex gap-6">
				{/* Sidebar nav */}
				<nav class="w-56 flex-shrink-0">
					<div class="space-y-1">
						<For each={navItems}>
							{(item) => (
								<button
									class={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors text-left ${activeSection() === item.id
										? 'bg-indigo-500/10 text-indigo-400 font-medium'
										: 'text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text-primary)]'
										}`}
									onClick={() => setActiveSection(item.id)}
								>
									<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
										<path fill-rule="evenodd" d={item.icon} clip-rule="evenodd" />
									</svg>
									{item.label}
								</button>
							)}
						</For>
					</div>
				</nav>

				{/* Content */}
				<div class="flex-1 min-w-0">
					<Switch>
						{/* ---- Profile ---- */}
						<Match when={activeSection() === 'profile'}>
							<Show when={!profile.loading} fallback={
								<Card title="Profile Information" description="Update your personal details and avatar">
									<div class="space-y-6 max-w-lg animate-pulse">
										<div class="flex items-center gap-4"><div class="w-16 h-16 rounded-full bg-[var(--color-bg-tertiary)]" /><div class="h-8 w-32 bg-[var(--color-bg-tertiary)] rounded" /></div>
										<div class="h-10 bg-[var(--color-bg-tertiary)] rounded" />
										<div class="h-10 bg-[var(--color-bg-tertiary)] rounded" />
										<div class="h-10 bg-[var(--color-bg-tertiary)] rounded" />
									</div>
								</Card>
							}>
								<Show when={profile.error}>
									<div class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30 flex items-center justify-between">
										<p class="text-sm text-red-400">Failed to load profile: {(profile.error as Error)?.message}</p>
										<Button size="sm" variant="outline" onClick={refetchProfile}>Retry</Button>
									</div>
								</Show>

								<Card title="Profile Information" description="Update your personal details and avatar">
									<div class="space-y-6 max-w-lg">
										{/* Avatar */}
										<div class="flex items-center gap-4">
											<Show when={profile()?.avatar_url} fallback={
												<div class="w-16 h-16 rounded-full bg-indigo-600 flex items-center justify-center text-white text-xl font-bold">
													{getInitials(profile())}
												</div>
											}>
												<img src={profile()!.avatar_url!} alt="Avatar" class="w-16 h-16 rounded-full object-cover" />
											</Show>
											<div>
												<Button size="sm" variant="outline">Change Avatar</Button>
												<p class="text-xs text-[var(--color-text-tertiary)] mt-1">JPG, PNG, or GIF. Max 2MB.</p>
											</div>
										</div>

										<Input label="Display Name" value={displayName()} onInput={(e) => setDisplayName(e.currentTarget.value)} />
										<Input label="Username" value={username()} onInput={(e) => setUsername(e.currentTarget.value)} hint="Used for mentions and commit attribution" />
										<Input label="Email" type="email" value={email()} onInput={(e) => setEmail(e.currentTarget.value)} />

										<div class="pt-2">
											<Button onClick={handleSaveProfile} loading={savingProfile()}>Save Changes</Button>
										</div>
									</div>
								</Card>

								<div class="mt-6">
									<Card title="Change Password">
										<div class="space-y-4 max-w-lg">
											<Input label="Current Password" type="password" placeholder="Enter current password" value={currentPassword()} onInput={(e) => setCurrentPassword(e.currentTarget.value)} />
											<Input label="New Password" type="password" placeholder="Enter new password" value={newPassword()} onInput={(e) => setNewPassword(e.currentTarget.value)} />
											<Input label="Confirm Password" type="password" placeholder="Confirm new password" value={confirmPassword()} onInput={(e) => setConfirmPassword(e.currentTarget.value)} />
											<Button onClick={handleChangePassword} loading={savingPassword()} disabled={!currentPassword() || !newPassword() || !confirmPassword()}>Update Password</Button>
										</div>
									</Card>
								</div>
							</Show>
						</Match>

						{/* ---- API Keys ---- */}
						<Match when={activeSection() === 'api-keys'}>
							<Show when={apiKeys.error}>
								<div class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30 flex items-center justify-between">
									<p class="text-sm text-red-400">Failed to load API keys: {(apiKeys.error as Error)?.message}</p>
									<Button size="sm" variant="outline" onClick={refetchApiKeys}>Retry</Button>
								</div>
							</Show>

							<Card title="API Keys" description="Manage API keys for programmatic access" actions={
								<Button size="sm" onClick={() => { setShowCreateKey(true); setNewKeyCreated(''); setNewKeyName(''); }}>Create Key</Button>
							} padding={false}>
								<Show when={!apiKeys.loading} fallback={
									<div class="p-5 space-y-3 animate-pulse">
										<For each={[1, 2, 3]}>{() => <div class="h-12 bg-[var(--color-bg-tertiary)] rounded" />}</For>
									</div>
								}>
									<Show when={(apiKeys() ?? []).length > 0} fallback={
										<div class="p-8 text-center">
											<p class="text-[var(--color-text-secondary)] mb-2">No API keys yet</p>
											<p class="text-sm text-[var(--color-text-tertiary)] mb-4">Create an API key for programmatic access to FlowForge.</p>
											<Button size="sm" onClick={() => { setShowCreateKey(true); setNewKeyCreated(''); setNewKeyName(''); }}>Create Key</Button>
										</div>
									}>
										<table class="w-full">
											<thead>
												<tr class="border-b border-[var(--color-border-primary)]">
													<th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Name</th>
													<th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Key Prefix</th>
													<th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Scopes</th>
													<th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Last Used</th>
													<th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Expires</th>
													<th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-right">Actions</th>
												</tr>
											</thead>
											<tbody>
												<For each={apiKeys() ?? []}>
													{(key) => (
														<tr class="border-b border-[var(--color-border-primary)] last:border-b-0 hover:bg-[var(--color-bg-hover)]">
															<td class="px-5 py-3 text-sm font-medium text-[var(--color-text-primary)]">{key.name}</td>
															<td class="px-5 py-3">
																<span class="text-xs font-mono text-[var(--color-text-tertiary)]">{key.prefix}...****</span>
															</td>
															<td class="px-5 py-3">
																<div class="flex gap-1">
																	<For each={key.scopes}>
																		{(scope) => <Badge size="sm" variant={scope === 'admin' ? 'warning' : scope === 'write' ? 'info' : 'default'}>{scope}</Badge>}
																	</For>
																</div>
															</td>
															<td class="px-5 py-3 text-xs text-[var(--color-text-tertiary)]">
																{key.last_used_at ? formatRelativeTime(key.last_used_at) : 'Never'}
															</td>
															<td class="px-5 py-3 text-xs text-[var(--color-text-tertiary)]">
																{key.expires_at ? new Date(key.expires_at).toLocaleDateString() : 'Never'}
															</td>
															<td class="px-5 py-3 text-right">
																<Button size="sm" variant="danger" onClick={() => setKeyToRevoke(key)}>Revoke</Button>
															</td>
														</tr>
													)}
												</For>
											</tbody>
										</table>
									</Show>
								</Show>
							</Card>

							{/* Create Key Modal */}
							<Show when={showCreateKey()}>
								<Modal open={showCreateKey()} onClose={() => setShowCreateKey(false)} title="Create API Key"
									footer={
										<Show when={!newKeyCreated()} fallback={
											<Button onClick={() => setShowCreateKey(false)}>Done</Button>
										}>
											<Button variant="ghost" onClick={() => setShowCreateKey(false)}>Cancel</Button>
											<Button onClick={handleCreateKey} loading={creatingKey()} disabled={!newKeyName().trim()}>Create Key</Button>
										</Show>
									}
								>
									<Show when={!newKeyCreated()} fallback={
										<div class="space-y-4">
											<div class="p-4 rounded-lg bg-emerald-500/10 border border-emerald-500/30">
												<p class="text-sm text-emerald-400 font-medium mb-2">API key created successfully!</p>
												<p class="text-xs text-[var(--color-text-tertiary)] mb-3">Copy this key now. You won't be able to see it again.</p>
												<div class="flex items-center gap-2">
													<code class="flex-1 text-sm font-mono bg-[var(--color-bg-primary)] px-3 py-2 rounded border border-[var(--color-border-primary)] text-[var(--color-text-primary)] break-all">{newKeyCreated()}</code>
													<Button size="sm" variant="outline" onClick={() => { copyToClipboard(newKeyCreated()); toast.success('Copied!'); }}>Copy</Button>
												</div>
											</div>
										</div>
									}>
										<div class="space-y-4">
											<Input label="Key Name" placeholder="e.g. CI Pipeline Key" value={newKeyName()} onInput={(e) => setNewKeyName(e.currentTarget.value)} />
											<Select label="Expiration" value={newKeyExpiration()} onChange={(e) => setNewKeyExpiration(e.currentTarget.value)} options={[
												{ value: '30d', label: '30 days' },
												{ value: '90d', label: '90 days' },
												{ value: '1y', label: '1 year' },
												{ value: 'never', label: 'No expiration' },
											]} />
											<div>
												<p class="text-sm font-medium text-[var(--color-text-primary)] mb-2">Scopes</p>
												<div class="space-y-2">
													<For each={[
														{ value: 'read', label: 'Read', desc: 'Read access to all resources' },
														{ value: 'write', label: 'Write', desc: 'Create and update resources' },
														{ value: 'admin', label: 'Admin', desc: 'Full administrative access' },
													]}>
														{(scope) => (
															<label class="flex items-center gap-3 cursor-pointer">
																<input
																	type="checkbox"
																	checked={newKeyScopes().includes(scope.value)}
																	onChange={(e) => {
																		if (e.currentTarget.checked) {
																			setNewKeyScopes(prev => [...prev, scope.value]);
																		} else {
																			setNewKeyScopes(prev => prev.filter(s => s !== scope.value));
																		}
																	}}
																	class="w-4 h-4 rounded border-[var(--color-border-primary)] bg-[var(--color-bg-secondary)] text-indigo-500 focus:ring-indigo-500/40"
																/>
																<div>
																	<span class="text-sm text-[var(--color-text-primary)]">{scope.label}</span>
																	<span class="text-xs text-[var(--color-text-tertiary)] ml-2">{scope.desc}</span>
																</div>
															</label>
														)}
													</For>
												</div>
											</div>
										</div>
									</Show>
								</Modal>
							</Show>

							{/* Revoke Confirmation Modal */}
							<Show when={keyToRevoke()}>
								<Modal
									open={!!keyToRevoke()}
									onClose={() => setKeyToRevoke(null)}
									title="Revoke API Key"
									footer={
										<>
											<Button variant="ghost" onClick={() => setKeyToRevoke(null)}>Cancel</Button>
											<Button variant="danger" onClick={handleRevokeKey} loading={revokingKey()}>Revoke Key</Button>
										</>
									}
								>
									<div class="space-y-3">
										<p class="text-sm text-[var(--color-text-secondary)]">
											Are you sure you want to revoke the API key <strong class="text-[var(--color-text-primary)]">"{keyToRevoke()!.name}"</strong>?
										</p>
										<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30">
											<p class="text-sm text-red-400">This action cannot be undone. Any applications using this key will lose access immediately.</p>
										</div>
									</div>
								</Modal>
							</Show>
						</Match>

						{/* ---- Notifications ---- */}
						<Match when={activeSection() === 'notifications'}>
							{(() => { if (!notifPrefs()) fetchNotifPrefs(); return null; })()}
							<Card title="Notification Preferences" description="Choose how you want to be notified">
								<Show when={!loadingNotifPrefs() && notifPrefs()} fallback={
									<div class="space-y-3 max-w-lg animate-pulse">
										<For each={[1, 2, 3, 4]}>{() => <div class="h-14 bg-[var(--color-bg-tertiary)] rounded" />}</For>
									</div>
								}>
									<div class="space-y-4 max-w-lg">
										{/* Delivery channels */}
										<h4 class="text-sm font-medium text-[var(--color-text-primary)]">Delivery Channels</h4>
										<For each={[
											{ key: 'email_enabled' as const, label: 'Email Notifications', desc: 'Receive notifications via email' },
											{ key: 'in_app_enabled' as const, label: 'In-App Notifications', desc: 'Show notifications in the notification center' },
										]}>
											{(item) => (
												<div class="flex items-center justify-between p-4 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
													<div>
														<p class="text-sm font-medium text-[var(--color-text-primary)]">{item.label}</p>
														<p class="text-xs text-[var(--color-text-tertiary)] mt-0.5">{item.desc}</p>
													</div>
													<button
														class={`w-10 h-6 rounded-full relative cursor-pointer transition-colors ${notifPrefs()?.[item.key] ? 'bg-indigo-500' : 'bg-gray-600'}`}
														onClick={() => toggleNotifPref(item.key)}
													>
														<div class={`absolute top-1 w-4 h-4 rounded-full bg-white transition-transform ${notifPrefs()?.[item.key] ? 'left-5' : 'left-1'}`} />
													</button>
												</div>
											)}
										</For>

										{/* Event categories */}
										<div class="pt-2">
											<h4 class="text-sm font-medium text-[var(--color-text-primary)] mb-3">Notify me about:</h4>
											<div class="space-y-2">
												<For each={[
													{ key: 'pipeline_success' as const, label: 'Pipeline run succeeds' },
													{ key: 'pipeline_failure' as const, label: 'Pipeline run fails' },
													{ key: 'deployment_success' as const, label: 'Deployment succeeds' },
													{ key: 'deployment_failure' as const, label: 'Deployment fails' },
													{ key: 'approval_requested' as const, label: 'Approval requested' },
													{ key: 'approval_resolved' as const, label: 'Approval resolved' },
													{ key: 'agent_offline' as const, label: 'Agent goes offline' },
													{ key: 'security_alerts' as const, label: 'Security alerts' },
												]}>
													{(item) => (
														<div class="flex items-center justify-between p-3 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
															<span class="text-sm text-[var(--color-text-secondary)]">{item.label}</span>
															<button
																class={`w-10 h-6 rounded-full relative cursor-pointer transition-colors ${notifPrefs()?.[item.key] ? 'bg-indigo-500' : 'bg-gray-600'}`}
																onClick={() => toggleNotifPref(item.key)}
															>
																<div class={`absolute top-1 w-4 h-4 rounded-full bg-white transition-transform ${notifPrefs()?.[item.key] ? 'left-5' : 'left-1'}`} />
															</button>
														</div>
													)}
												</For>
											</div>
										</div>

										<div class="pt-2">
											<Button onClick={handleSaveNotifications} loading={savingNotifs()}>Save Preferences</Button>
										</div>
									</div>
								</Show>
							</Card>
						</Match>

						{/* ---- Appearance ---- */}
						<Match when={activeSection() === 'appearance'}>
							<Card title="Appearance" description="Customize the look and feel of FlowForge">
								<div class="space-y-6 max-w-lg">
									<div>
										<h4 class="text-sm font-medium text-[var(--color-text-primary)] mb-3">Theme</h4>
										<div class="grid grid-cols-3 gap-3">
											<For each={[
												{ id: 'dark', label: 'Dark', colors: ['#0f1117', '#161b22', '#1c2128'] },
												{ id: 'light', label: 'Light', colors: ['#ffffff', '#f6f8fa', '#eaecef'] },
												{ id: 'system', label: 'System', colors: ['#0f1117', '#ffffff', '#0f1117'] },
											]}>
												{(t) => (
													<button
														class={`p-3 rounded-lg border-2 transition-colors ${theme() === t.id ? 'border-indigo-500' : 'border-[var(--color-border-primary)] hover:border-[var(--color-border-secondary)]'
															}`}
														onClick={() => setTheme(t.id)}
													>
														<div class="flex gap-1 mb-2">
															<For each={t.colors}>
																{(color) => <div class="w-6 h-6 rounded" style={{ "background-color": color }} />}
															</For>
														</div>
														<p class="text-xs text-[var(--color-text-secondary)]">{t.label}</p>
													</button>
												)}
											</For>
										</div>
									</div>

									<Select label="Log Font Size" options={[
										{ value: '12', label: '12px' },
										{ value: '13', label: '13px (Default)' },
										{ value: '14', label: '14px' },
										{ value: '16', label: '16px' },
									]} />

									<Select label="Date Format" options={[
										{ value: 'relative', label: 'Relative (e.g., "5 min ago")' },
										{ value: 'absolute', label: 'Absolute (e.g., "Mar 25, 2026")' },
										{ value: 'both', label: 'Both' },
									]} />

									<Button onClick={handleSaveAppearance}>Save</Button>
								</div>
							</Card>
						</Match>

						{/* ---- Security ---- */}
						<Match when={activeSection() === 'security'}>
							<div class="space-y-6">
								<Card title="Two-Factor Authentication" description="Add an extra layer of security to your account">
									<div class="flex items-center justify-between p-4 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] max-w-lg">
										<div class="flex items-center gap-3">
											<div class={`w-10 h-10 rounded-full flex items-center justify-center ${profile()?.totp_enabled ? 'bg-emerald-500/10' : 'bg-gray-500/10'}`}>
												<svg class={`w-5 h-5 ${profile()?.totp_enabled ? 'text-emerald-400' : 'text-gray-400'}`} viewBox="0 0 20 20" fill="currentColor">
													<path fill-rule="evenodd" d="M10 1a4.5 4.5 0 00-4.5 4.5V9H5a2 2 0 00-2 2v6a2 2 0 002 2h10a2 2 0 002-2v-6a2 2 0 00-2-2h-.5V5.5A4.5 4.5 0 0010 1zm3 8V5.5a3 3 0 10-6 0V9h6z" clip-rule="evenodd" />
												</svg>
											</div>
											<div>
												<p class="text-sm font-medium text-[var(--color-text-primary)]">TOTP Authentication</p>
												<p class="text-xs text-[var(--color-text-tertiary)]">
													{profile()?.totp_enabled ? 'Two-factor authentication is enabled' : 'Protect your account with an authenticator app'}
												</p>
											</div>
										</div>
										<Show when={profile()?.totp_enabled} fallback={
											<Button size="sm" onClick={handleTotpSetup} loading={settingUpTotp()}>Enable</Button>
										}>
											<Badge variant="success" dot>Enabled</Badge>
										</Show>
									</div>
								</Card>

								<Card title="Danger Zone">
									<div class="flex items-center justify-between p-4 rounded-lg border border-red-500/30 bg-red-500/5 max-w-lg">
										<div>
											<p class="text-sm font-medium text-red-400">Delete Account</p>
											<p class="text-xs text-[var(--color-text-tertiary)] mt-1">Permanently delete your account and all associated data.</p>
										</div>
										<Button variant="danger" size="sm" onClick={() => setShowDeleteConfirm(true)}>Delete Account</Button>
									</div>
								</Card>
							</div>

							{/* TOTP Setup Modal */}
							<Show when={showTotpSetup()}>
								<Modal open={showTotpSetup()} onClose={() => { setShowTotpSetup(false); setTotpCode(''); }} title="Set Up Two-Factor Authentication" footer={
									<>
										<Button variant="ghost" onClick={() => { setShowTotpSetup(false); setTotpCode(''); }}>Cancel</Button>
										<Button onClick={handleTotpVerify} loading={verifyingTotp()} disabled={totpCode().length !== 6}>Verify & Enable</Button>
									</>
								}>
									<div class="space-y-4">
										<p class="text-sm text-[var(--color-text-secondary)]">Scan this QR code with your authenticator app (Google Authenticator, Authy, etc.)</p>
										<Show when={totpQrUrl()} fallback={
											<div class="flex justify-center p-6 bg-white rounded-lg">
												<div class="w-40 h-40 bg-gray-200 rounded flex items-center justify-center text-gray-500 text-xs animate-pulse">Loading...</div>
											</div>
										}>
											<div class="flex justify-center p-6 bg-white rounded-lg">
												<img src={totpQrUrl()} alt="TOTP QR Code" class="w-40 h-40" />
											</div>
										</Show>
										<Show when={totpSecret()}>
											<p class="text-xs text-[var(--color-text-tertiary)] text-center">
												Or enter manually: <code class="font-mono bg-[var(--color-bg-tertiary)] px-2 py-0.5 rounded select-all">{totpSecret()}</code>
											</p>
										</Show>
										<Input label="Verification Code" placeholder="Enter 6-digit code from your app" value={totpCode()} onInput={(e) => setTotpCode(e.currentTarget.value.replace(/\D/g, '').slice(0, 6))} />
									</div>
								</Modal>
							</Show>

							{/* Delete Account Confirmation Modal */}
							<Show when={showDeleteConfirm()}>
								<Modal
									open={showDeleteConfirm()}
									onClose={() => { setShowDeleteConfirm(false); setDeleteConfirmText(''); }}
									title="Delete Account"
									footer={
										<>
											<Button variant="ghost" onClick={() => { setShowDeleteConfirm(false); setDeleteConfirmText(''); }}>Cancel</Button>
											<Button
												variant="danger"
												disabled={deleteConfirmText() !== 'DELETE'}
												onClick={async () => {
													try {
														await apiClient.delete('/users/me');
														toast.success('Account deleted');
														window.location.href = '/auth/login';
													} catch (err) {
														const msg = err instanceof ApiRequestError ? err.message : 'Failed to delete account';
														toast.error(msg);
													}
												}}
											>Delete Account</Button>
										</>
									}
								>
									<div class="space-y-4">
										<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30">
											<p class="text-sm text-red-400">This action is permanent and cannot be undone. All your data, projects, and settings will be permanently deleted.</p>
										</div>
										<Input
											label={<>Type <code class="font-mono text-red-400">DELETE</code> to confirm</>}
											placeholder="DELETE"
											value={deleteConfirmText()}
											onInput={(e) => setDeleteConfirmText(e.currentTarget.value)}
										/>
									</div>
								</Modal>
							</Show>
						</Match>
					</Switch>
				</div>
			</div>
		</PageContainer>
	);
};

export default SettingsPage;
