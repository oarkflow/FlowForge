import type { Component } from 'solid-js';
import { createSignal, createResource, For, Show, Switch, Match } from 'solid-js';
import PageContainer from '../../components/layout/PageContainer';
import Card from '../../components/ui/Card';
import Badge from '../../components/ui/Badge';
import Button from '../../components/ui/Button';
import Input from '../../components/ui/Input';
import Modal from '../../components/ui/Modal';
import { toast } from '../../components/ui/Toast';
import { api, ApiRequestError, apiClient } from '../../api/client';
import type { User, Organization, Agent, AuditLog, SystemHealth } from '../../types';
import { formatRelativeTime, getAgentStatusVariant, formatDuration } from '../../utils/helpers';

// ---------------------------------------------------------------------------
// Fetchers
// ---------------------------------------------------------------------------
async function fetchSystemHealth(): Promise<SystemHealth> {
  return api.system.health();
}

async function fetchSystemInfo(): Promise<Record<string, unknown>> {
  return api.system.info();
}

async function fetchUsers(): Promise<User[]> {
  return apiClient.get<User[]>('/admin/users');
}

async function fetchOrganizations(): Promise<Organization[]> {
  return api.orgs.list();
}

async function fetchAgents(): Promise<Agent[]> {
  return api.agents.list();
}

async function fetchAuditLogs(): Promise<AuditLog[]> {
  const res = await api.auditLogs.list({ page: '1', per_page: '50' });
  return res.data;
}

// ---------------------------------------------------------------------------
// Sidebar nav
// ---------------------------------------------------------------------------
const navItems = [
  { id: 'health', label: 'System Health', icon: 'M3.172 5.172a4 4 0 015.656 0L10 6.343l1.172-1.171a4 4 0 115.656 5.656L10 17.657l-6.828-6.829a4 4 0 010-5.656z' },
  { id: 'users', label: 'Users', icon: 'M7 8a3 3 0 100-6 3 3 0 000 6zM14.5 9a2.5 2.5 0 100-5 2.5 2.5 0 000 5zM1.615 16.428a1.224 1.224 0 01-.569-1.175 6.002 6.002 0 0111.908 0c.058.467-.172.92-.57 1.174A9.953 9.953 0 017 18a9.953 9.953 0 01-5.385-1.572zM14.5 16h-.106c.07-.297.088-.611.048-.933a7.47 7.47 0 00-1.588-3.755 4.502 4.502 0 015.874 2.636.818.818 0 01-.36.98A7.465 7.465 0 0114.5 16z' },
  { id: 'orgs', label: 'Organizations', icon: 'M4 16.5v-13h-.25a.75.75 0 010-1.5h12.5a.75.75 0 010 1.5H16v13h.25a.75.75 0 010 1.5h-3.5a.75.75 0 01-.75-.75v-2.5a.75.75 0 00-.75-.75h-2.5a.75.75 0 00-.75.75v2.5a.75.75 0 01-.75.75h-3.5a.75.75 0 010-1.5H4z' },
  { id: 'agents', label: 'Agents', icon: 'M14 6a2.5 2.5 0 00-4-2 2.5 2.5 0 00-4 2 2.5 2.5 0 00-.126.75A1.5 1.5 0 005.5 8h.75a.75.75 0 00.75-.75V6.5a1 1 0 112 0v.75c0 .414.336.75.75.75h.75a1.5 1.5 0 01-.376-1.25A2.5 2.5 0 0014 6z' },
  { id: 'audit', label: 'Audit Logs', icon: 'M3 3.5A1.5 1.5 0 014.5 2h6.879a1.5 1.5 0 011.06.44l3.122 3.12A1.5 1.5 0 0116 6.622V16.5a1.5 1.5 0 01-1.5 1.5h-11A1.5 1.5 0 012 16.5v-13z' },
  { id: 'flags', label: 'Feature Flags', icon: 'M3.5 2A1.5 1.5 0 002 3.5V5c0 .325.078.628.22.898l5.04 9.576A3.5 3.5 0 0010.52 17h3.98A1.5 1.5 0 0016 15.5V14a.75.75 0 00-1.5 0v1.5h-3.98a2 2 0 01-1.788-1.106L5.22 5.898A.498.498 0 015 5.5V3.5h11V5a.75.75 0 001.5 0V3.5A1.5 1.5 0 0016 2H3.5z' },
];

const actionColors: Record<string, string> = {
  'create': 'text-emerald-400',
  'update': 'text-amber-400',
  'delete': 'text-red-400',
  'trigger': 'text-violet-400',
  'cancel': 'text-gray-400',
  'login': 'text-blue-400',
};

const getActionColor = (action: string) => {
  const verb = action.split('.')[1] || action;
  return actionColors[verb] || 'text-[var(--color-text-secondary)]';
};

const roleVariant: Record<string, 'success' | 'error' | 'warning' | 'info' | 'default'> = {
  owner: 'error', admin: 'warning', developer: 'info', viewer: 'default',
};

// ---------------------------------------------------------------------------
// Feature Flags type
// ---------------------------------------------------------------------------
interface FeatureFlag {
  id: string;
  key: string;
  name: string;
  enabled: boolean;
  description: string;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const AdminPage: Component = () => {
  const [activeSection, setActiveSection] = createSignal('health');

  // Resources
  const [health, { refetch: refetchHealth }] = createResource(fetchSystemHealth);
  const [systemInfo] = createResource(fetchSystemInfo);
  const [users, { refetch: refetchUsers }] = createResource(fetchUsers);
  const [orgs, { refetch: refetchOrgs, mutate: mutateOrgs }] = createResource(fetchOrganizations);
  const [agents, { refetch: refetchAgents, mutate: mutateAgents }] = createResource(fetchAgents);
  const [auditLogs, { refetch: refetchAuditLogs }] = createResource(fetchAuditLogs);
  const [featureFlags, { mutate: mutateFlags }] = createResource(async () => {
    try {
      return await apiClient.get<FeatureFlag[]>('/admin/feature-flags');
    } catch {
      return [] as FeatureFlag[];
    }
  });

  // Create org modal
  const [showCreateOrg, setShowCreateOrg] = createSignal(false);
  const [newOrgName, setNewOrgName] = createSignal('');
  const [newOrgSlug, setNewOrgSlug] = createSignal('');
  const [creatingOrg, setCreatingOrg] = createSignal(false);

  // Delete org confirmation
  const [orgToDelete, setOrgToDelete] = createSignal<Organization | null>(null);
  const [deletingOrg, setDeletingOrg] = createSignal(false);

  // Agent actions
  const [agentToDrain, setAgentToDrain] = createSignal<Agent | null>(null);
  const [drainingAgent, setDrainingAgent] = createSignal(false);
  const [agentToRemove, setAgentToRemove] = createSignal<Agent | null>(null);
  const [removingAgent, setRemovingAgent] = createSignal(false);

  // Computed: agent counts from real data
  const agentCounts = () => {
    const counts = { online: 0, busy: 0, draining: 0, offline: 0 };
    (agents() ?? []).forEach(a => counts[a.status]++);
    return counts;
  };

  // Computed: user lookup
  const getUserById = (id?: string) => (users() ?? []).find(u => u.id === id);

  // Format uptime
  const formatUptime = (seconds?: number) => {
    if (!seconds) return '-';
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    return `${days}d ${hours}h ${mins}m`;
  };

  // ---------------------------------------------------------------------------
  // Handlers
  // ---------------------------------------------------------------------------
  const handleCreateOrg = async () => {
    if (!newOrgName().trim()) return;
    setCreatingOrg(true);
    try {
      const created = await api.orgs.create({
        name: newOrgName().trim(),
        slug: newOrgSlug().trim() || newOrgName().toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, ''),
      });
      mutateOrgs(prev => prev ? [created, ...prev] : [created]);
      setShowCreateOrg(false);
      setNewOrgName('');
      setNewOrgSlug('');
      toast.success(`Organization "${created.name}" created`);
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to create organization';
      toast.error(msg);
    } finally {
      setCreatingOrg(false);
    }
  };

  const handleDeleteOrg = async () => {
    const org = orgToDelete();
    if (!org) return;
    setDeletingOrg(true);
    try {
      await api.orgs.delete(org.id);
      mutateOrgs(prev => prev?.filter(o => o.id !== org.id));
      setOrgToDelete(null);
      toast.success(`Organization "${org.name}" deleted`);
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to delete organization';
      toast.error(msg);
    } finally {
      setDeletingOrg(false);
    }
  };

  const handleDrainAgent = async () => {
    const agent = agentToDrain();
    if (!agent) return;
    setDrainingAgent(true);
    try {
      await api.agents.drain(agent.id);
      mutateAgents(prev => prev?.map(a => a.id === agent.id ? { ...a, status: 'draining' as const } : a));
      setAgentToDrain(null);
      toast.success(`Agent "${agent.name}" set to draining`);
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to drain agent';
      toast.error(msg);
    } finally {
      setDrainingAgent(false);
    }
  };

  const handleRemoveAgent = async () => {
    const agent = agentToRemove();
    if (!agent) return;
    setRemovingAgent(true);
    try {
      await api.agents.delete(agent.id);
      mutateAgents(prev => prev?.filter(a => a.id !== agent.id));
      setAgentToRemove(null);
      toast.success(`Agent "${agent.name}" removed`);
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to remove agent';
      toast.error(msg);
    } finally {
      setRemovingAgent(false);
    }
  };

  const handleToggleFlag = async (flag: FeatureFlag) => {
    const newState = !flag.enabled;
    try {
      await apiClient.put(`/admin/feature-flags/${flag.id}`, { enabled: newState });
      mutateFlags(prev => prev?.map(f => f.id === flag.id ? { ...f, enabled: newState } : f));
      toast.success(`Feature "${flag.name}" ${newState ? 'enabled' : 'disabled'}`);
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to toggle feature flag';
      toast.error(msg);
    }
  };

  return (
    <PageContainer title="Administration" description="System-wide administration and monitoring">
      <div class="flex gap-6">
        {/* Sidebar nav */}
        <nav class="w-56 flex-shrink-0">
          <div class="space-y-1">
            <For each={navItems}>
              {(item) => (
                <button
                  class={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors text-left ${
                    activeSection() === item.id
                      ? 'bg-indigo-500/10 text-indigo-400 font-medium'
                      : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text-primary)]'
                  }`}
                  onClick={() => setActiveSection(item.id)}
                >
                  <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d={item.icon} /></svg>
                  {item.label}
                </button>
              )}
            </For>
          </div>
        </nav>

        {/* Content */}
        <div class="flex-1 min-w-0">
          <Switch>
            {/* ---- System Health ---- */}
            <Match when={activeSection() === 'health'}>
              <div class="space-y-6">
                {/* Error */}
                <Show when={health.error}>
                  <div class="p-4 rounded-xl bg-red-500/10 border border-red-500/30 flex items-center justify-between">
                    <p class="text-sm text-red-400">Failed to load system health: {(health.error as Error)?.message}</p>
                    <Button size="sm" variant="outline" onClick={refetchHealth}>Retry</Button>
                  </div>
                </Show>

                {/* Status overview */}
                <Show when={!health.loading} fallback={
                  <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
                    <For each={[1, 2, 3]}>{() => (
                      <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-5 animate-pulse">
                        <div class="flex items-center gap-3"><div class="w-12 h-12 rounded-xl bg-[var(--color-bg-tertiary)]" /><div><div class="h-3 w-20 bg-[var(--color-bg-tertiary)] rounded mb-2" /><div class="h-5 w-24 bg-[var(--color-bg-tertiary)] rounded" /></div></div>
                      </div>
                    )}</For>
                  </div>
                }>
                  <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
                    <Card>
                      <div class="flex items-center gap-3">
                        <div class={`w-12 h-12 rounded-xl flex items-center justify-center ${
                          health()?.status === 'healthy' ? 'bg-emerald-500/10' : health()?.status === 'degraded' ? 'bg-amber-500/10' : 'bg-red-500/10'
                        }`}>
                          <svg class={`w-6 h-6 ${
                            health()?.status === 'healthy' ? 'text-emerald-400' : health()?.status === 'degraded' ? 'text-amber-400' : 'text-red-400'
                          }`} viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" /></svg>
                        </div>
                        <div>
                          <p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider">System Status</p>
                          <p class={`text-lg font-bold capitalize ${
                            health()?.status === 'healthy' ? 'text-emerald-400' : health()?.status === 'degraded' ? 'text-amber-400' : 'text-red-400'
                          }`}>{health()?.status || 'Unknown'}</p>
                        </div>
                      </div>
                    </Card>
                    <Card>
                      <div class="flex items-center gap-3">
                        <div class="w-12 h-12 rounded-xl bg-indigo-500/10 flex items-center justify-center">
                          <svg class="w-6 h-6 text-indigo-400" viewBox="0 0 20 20" fill="currentColor"><path d="M10 2a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 2zM10 15a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 15zM10 7a3 3 0 100 6 3 3 0 000-6z" /></svg>
                        </div>
                        <div>
                          <p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider">Version</p>
                          <p class="text-lg font-bold text-[var(--color-text-primary)]">{health()?.version || '-'}</p>
                        </div>
                      </div>
                    </Card>
                    <Card>
                      <div class="flex items-center gap-3">
                        <div class="w-12 h-12 rounded-xl bg-violet-500/10 flex items-center justify-center">
                          <svg class="w-6 h-6 text-violet-400" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm.75-13a.75.75 0 00-1.5 0v5c0 .414.336.75.75.75h4a.75.75 0 000-1.5h-3.25V5z" clip-rule="evenodd" /></svg>
                        </div>
                        <div>
                          <p class="text-xs text-[var(--color-text-tertiary)] uppercase tracking-wider">Uptime</p>
                          <p class="text-lg font-bold text-[var(--color-text-primary)]">{formatUptime(health()?.uptime_seconds)}</p>
                        </div>
                      </div>
                    </Card>
                  </div>
                </Show>

                {/* Services */}
                <Card title="Services" actions={<Button size="sm" variant="outline" onClick={refetchHealth}>Refresh</Button>}>
                  <Show when={health()} fallback={
                    <div class="space-y-3 animate-pulse">
                      <For each={[1, 2, 3, 4]}>{() => <div class="h-14 bg-[var(--color-bg-tertiary)] rounded-lg" />}</For>
                    </div>
                  }>
                    <div class="space-y-3">
                      <div class="flex items-center justify-between p-3 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
                        <div class="flex items-center gap-3">
                          <div class={`w-2.5 h-2.5 rounded-full ${health()?.status === 'healthy' ? 'bg-emerald-400' : 'bg-amber-400'}`} />
                          <div>
                            <p class="text-sm font-medium text-[var(--color-text-primary)]">API Server</p>
                            <p class="text-xs text-[var(--color-text-tertiary)]">GoFiber v3.1.0</p>
                          </div>
                        </div>
                        <Badge variant={health()?.status === 'healthy' ? 'success' : 'warning'} size="sm">{health()?.status}</Badge>
                      </div>

                      <div class="flex items-center justify-between p-3 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
                        <div class="flex items-center gap-3">
                          <div class={`w-2.5 h-2.5 rounded-full ${health()?.database?.status === 'healthy' || health()?.database?.status === 'ok' ? 'bg-emerald-400' : 'bg-red-400'}`} />
                          <div>
                            <p class="text-sm font-medium text-[var(--color-text-primary)]">Database (SQLite)</p>
                            <p class="text-xs text-[var(--color-text-tertiary)]">WAL mode</p>
                          </div>
                        </div>
                        <Badge variant={health()?.database?.status === 'healthy' || health()?.database?.status === 'ok' ? 'success' : 'error'} size="sm">{health()?.database?.status || 'unknown'}</Badge>
                      </div>

                      <div class="flex items-center justify-between p-3 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
                        <div class="flex items-center gap-3">
                          <div class={`w-2.5 h-2.5 rounded-full ${(health()?.agents?.online ?? 0) > 0 ? 'bg-emerald-400' : 'bg-gray-400'}`} />
                          <div>
                            <p class="text-sm font-medium text-[var(--color-text-primary)]">Agent Pool</p>
                            <p class="text-xs text-[var(--color-text-tertiary)]">{health()?.agents?.online ?? 0} of {health()?.agents?.total ?? 0} agents connected</p>
                          </div>
                        </div>
                        <Badge variant={(health()?.agents?.online ?? 0) > 0 ? 'success' : 'default'} size="sm">
                          {(health()?.agents?.online ?? 0) > 0 ? 'active' : 'no agents'}
                        </Badge>
                      </div>
                    </div>
                  </Show>
                </Card>

                {/* Quick stats */}
                <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
                  <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-lg px-4 py-3 text-center">
                    <p class="text-2xl font-bold text-[var(--color-text-primary)]">{users.loading ? '-' : (users() ?? []).length}</p>
                    <p class="text-xs text-[var(--color-text-tertiary)] mt-1">Total Users</p>
                  </div>
                  <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-lg px-4 py-3 text-center">
                    <p class="text-2xl font-bold text-[var(--color-text-primary)]">{orgs.loading ? '-' : (orgs() ?? []).length}</p>
                    <p class="text-xs text-[var(--color-text-tertiary)] mt-1">Organizations</p>
                  </div>
                  <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-lg px-4 py-3 text-center">
                    <p class="text-2xl font-bold text-[var(--color-text-primary)]">{agents.loading ? '-' : (agents() ?? []).length}</p>
                    <p class="text-xs text-[var(--color-text-tertiary)] mt-1">Total Agents</p>
                  </div>
                  <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-lg px-4 py-3 text-center">
                    <p class="text-2xl font-bold text-emerald-400">{agents.loading ? '-' : agentCounts().online + agentCounts().busy}</p>
                    <p class="text-xs text-[var(--color-text-tertiary)] mt-1">Active Agents</p>
                  </div>
                </div>
              </div>
            </Match>

            {/* ---- Users ---- */}
            <Match when={activeSection() === 'users'}>
              <Show when={users.error}>
                <div class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30 flex items-center justify-between">
                  <p class="text-sm text-red-400">Failed to load users: {(users.error as Error)?.message}</p>
                  <Button size="sm" variant="outline" onClick={refetchUsers}>Retry</Button>
                </div>
              </Show>

              <Card title="User Management" description={`${(users() ?? []).length} registered users`} actions={
                <Button size="sm">Invite User</Button>
              } padding={false}>
                <Show when={!users.loading} fallback={
                  <div class="p-5 space-y-3 animate-pulse">
                    <For each={[1, 2, 3]}>{() => <div class="h-14 bg-[var(--color-bg-tertiary)] rounded" />}</For>
                  </div>
                }>
                  <table class="w-full">
                    <thead>
                      <tr class="border-b border-[var(--color-border-primary)]">
                        <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">User</th>
                        <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Role</th>
                        <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">2FA</th>
                        <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Status</th>
                        <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Created</th>
                        <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      <For each={users() ?? []}>
                        {(user) => (
                          <tr class="border-b border-[var(--color-border-primary)] last:border-b-0 hover:bg-[var(--color-bg-hover)]">
                            <td class="px-5 py-3">
                              <div class="flex items-center gap-3">
                                <Show when={user.avatar_url} fallback={
                                  <div class="w-8 h-8 rounded-full bg-indigo-600 flex items-center justify-center text-xs font-bold text-white">
                                    {(user.display_name || user.username).split(' ').map(w => w[0]).join('').toUpperCase().slice(0, 2)}
                                  </div>
                                }>
                                  <img src={user.avatar_url!} alt="" class="w-8 h-8 rounded-full object-cover" />
                                </Show>
                                <div>
                                  <p class="text-sm font-medium text-[var(--color-text-primary)]">{user.display_name || user.username}</p>
                                  <p class="text-xs text-[var(--color-text-tertiary)]">{user.email}</p>
                                </div>
                              </div>
                            </td>
                            <td class="px-5 py-3"><Badge variant={roleVariant[user.role] || 'default'} size="sm">{user.role}</Badge></td>
                            <td class="px-5 py-3">
                              <Show when={user.totp_enabled} fallback={<span class="text-xs text-[var(--color-text-tertiary)]">Off</span>}>
                                <Badge variant="success" size="sm">Enabled</Badge>
                              </Show>
                            </td>
                            <td class="px-5 py-3">
                              <Badge variant={user.is_active ? 'success' : 'default'} dot size="sm">
                                {user.is_active ? 'Active' : 'Disabled'}
                              </Badge>
                            </td>
                            <td class="px-5 py-3 text-xs text-[var(--color-text-tertiary)]">
                              {formatRelativeTime(user.created_at)}
                            </td>
                            <td class="px-5 py-3 text-right">
                              <div class="flex items-center gap-1 justify-end">
                                <Button size="sm" variant="ghost">Edit</Button>
                                <Show when={user.role !== 'owner'}>
                                  <Button size="sm" variant="danger" onClick={async () => {
                                    try {
                                      await apiClient.put(`/admin/users/${user.id}`, { is_active: !user.is_active });
                                      refetchUsers();
                                      toast.success(`User ${user.is_active ? 'disabled' : 'enabled'}`);
                                    } catch (err) {
                                      toast.error(err instanceof ApiRequestError ? err.message : 'Failed to update user');
                                    }
                                  }}>{user.is_active ? 'Disable' : 'Enable'}</Button>
                                </Show>
                              </div>
                            </td>
                          </tr>
                        )}
                      </For>
                    </tbody>
                  </table>
                </Show>
              </Card>
            </Match>

            {/* ---- Organizations ---- */}
            <Match when={activeSection() === 'orgs'}>
              <Show when={orgs.error}>
                <div class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30 flex items-center justify-between">
                  <p class="text-sm text-red-400">Failed to load organizations: {(orgs.error as Error)?.message}</p>
                  <Button size="sm" variant="outline" onClick={refetchOrgs}>Retry</Button>
                </div>
              </Show>

              <Card title="Organizations" actions={<Button size="sm" onClick={() => setShowCreateOrg(true)}>Create Organization</Button>}>
                <Show when={!orgs.loading} fallback={
                  <div class="space-y-3 animate-pulse">
                    <For each={[1, 2]}>{() => <div class="h-20 bg-[var(--color-bg-tertiary)] rounded-lg" />}</For>
                  </div>
                }>
                  <Show when={(orgs() ?? []).length > 0} fallback={
                    <div class="text-center py-8">
                      <p class="text-[var(--color-text-secondary)] mb-2">No organizations yet</p>
                      <p class="text-sm text-[var(--color-text-tertiary)] mb-4">Create an organization to group projects and manage team access.</p>
                      <Button size="sm" onClick={() => setShowCreateOrg(true)}>Create Organization</Button>
                    </div>
                  }>
                    <div class="space-y-3">
                      <For each={orgs() ?? []}>
                        {(org) => (
                          <div class="flex items-center justify-between p-4 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
                            <div class="flex items-center gap-4">
                              <Show when={org.logo_url} fallback={
                                <div class="w-10 h-10 rounded-lg bg-indigo-500/10 flex items-center justify-center text-indigo-400 font-bold text-sm">
                                  {org.name.split(' ').map(w => w[0]).join('').toUpperCase().slice(0, 2)}
                                </div>
                              }>
                                <img src={org.logo_url!} alt="" class="w-10 h-10 rounded-lg object-cover" />
                              </Show>
                              <div>
                                <p class="text-sm font-semibold text-[var(--color-text-primary)]">{org.name}</p>
                                <p class="text-xs text-[var(--color-text-tertiary)]">@{org.slug} · Created {formatRelativeTime(org.created_at)}</p>
                              </div>
                            </div>
                            <div class="flex items-center gap-2">
                              <Button size="sm" variant="ghost">Members</Button>
                              <Button size="sm" variant="outline">Settings</Button>
                              <Button size="sm" variant="danger" onClick={() => setOrgToDelete(org)}>Delete</Button>
                            </div>
                          </div>
                        )}
                      </For>
                    </div>
                  </Show>
                </Show>
              </Card>

              {/* Create Org Modal */}
              <Show when={showCreateOrg()}>
                <Modal
                  open={showCreateOrg()}
                  onClose={() => { setShowCreateOrg(false); setNewOrgName(''); setNewOrgSlug(''); }}
                  title="Create Organization"
                  footer={
                    <>
                      <Button variant="ghost" onClick={() => setShowCreateOrg(false)}>Cancel</Button>
                      <Button onClick={handleCreateOrg} loading={creatingOrg()} disabled={!newOrgName().trim()}>Create</Button>
                    </>
                  }
                >
                  <div class="space-y-4">
                    <Input label="Organization Name" placeholder="My Team" value={newOrgName()} onInput={(e) => setNewOrgName(e.currentTarget.value)} />
                    <Input label="Slug" placeholder="my-team" value={newOrgSlug()} onInput={(e) => setNewOrgSlug(e.currentTarget.value)} hint="URL-friendly identifier. Auto-generated if left blank." />
                  </div>
                </Modal>
              </Show>

              {/* Delete Org Confirmation */}
              <Show when={orgToDelete()}>
                <Modal
                  open={!!orgToDelete()}
                  onClose={() => setOrgToDelete(null)}
                  title="Delete Organization"
                  footer={
                    <>
                      <Button variant="ghost" onClick={() => setOrgToDelete(null)}>Cancel</Button>
                      <Button variant="danger" onClick={handleDeleteOrg} loading={deletingOrg()}>Delete Organization</Button>
                    </>
                  }
                >
                  <div class="space-y-3">
                    <p class="text-sm text-[var(--color-text-secondary)]">
                      Are you sure you want to delete <strong class="text-[var(--color-text-primary)]">"{orgToDelete()!.name}"</strong>?
                    </p>
                    <div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30">
                      <p class="text-sm text-red-400">This will permanently delete the organization and remove all member associations. Projects within this organization will become unassociated.</p>
                    </div>
                  </div>
                </Modal>
              </Show>
            </Match>

            {/* ---- Agents ---- */}
            <Match when={activeSection() === 'agents'}>
              <div class="space-y-6">
                <Show when={agents.error}>
                  <div class="p-4 rounded-xl bg-red-500/10 border border-red-500/30 flex items-center justify-between">
                    <p class="text-sm text-red-400">Failed to load agents: {(agents.error as Error)?.message}</p>
                    <Button size="sm" variant="outline" onClick={refetchAgents}>Retry</Button>
                  </div>
                </Show>

                <Show when={!agents.loading} fallback={
                  <div class="grid grid-cols-2 sm:grid-cols-4 gap-4 animate-pulse">
                    <For each={[1, 2, 3, 4]}>{() => <div class="h-20 bg-[var(--color-bg-tertiary)] rounded-xl" />}</For>
                  </div>
                }>
                  <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
                    <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-4 text-center">
                      <p class="text-2xl font-bold text-emerald-400">{agentCounts().online}</p>
                      <p class="text-xs text-[var(--color-text-tertiary)] mt-1">Online</p>
                    </div>
                    <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-4 text-center">
                      <p class="text-2xl font-bold text-violet-400">{agentCounts().busy}</p>
                      <p class="text-xs text-[var(--color-text-tertiary)] mt-1">Busy</p>
                    </div>
                    <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-4 text-center">
                      <p class="text-2xl font-bold text-amber-400">{agentCounts().draining}</p>
                      <p class="text-xs text-[var(--color-text-tertiary)] mt-1">Draining</p>
                    </div>
                    <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-4 text-center">
                      <p class="text-2xl font-bold text-gray-500">{agentCounts().offline}</p>
                      <p class="text-xs text-[var(--color-text-tertiary)] mt-1">Offline</p>
                    </div>
                  </div>
                </Show>

                <Card title="All Agents" actions={<Button size="sm" variant="outline" onClick={refetchAgents}>Refresh</Button>} padding={false}>
                  <Show when={!agents.loading} fallback={
                    <div class="p-5 space-y-3 animate-pulse">
                      <For each={[1, 2, 3]}>{() => <div class="h-14 bg-[var(--color-bg-tertiary)] rounded" />}</For>
                    </div>
                  }>
                    <Show when={(agents() ?? []).length > 0} fallback={
                      <div class="p-8 text-center">
                        <p class="text-[var(--color-text-secondary)] mb-2">No agents registered</p>
                        <p class="text-sm text-[var(--color-text-tertiary)]">Register an agent to start running pipeline jobs.</p>
                      </div>
                    }>
                      <table class="w-full">
                        <thead>
                          <tr class="border-b border-[var(--color-border-primary)]">
                            <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Name</th>
                            <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Status</th>
                            <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Executor</th>
                            <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Labels</th>
                            <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-left">Resources</th>
                            <th class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] text-right">Actions</th>
                          </tr>
                        </thead>
                        <tbody>
                          <For each={agents() ?? []}>
                            {(agent) => (
                              <tr class="border-b border-[var(--color-border-primary)] last:border-b-0 hover:bg-[var(--color-bg-hover)]">
                                <td class="px-5 py-3">
                                  <p class="text-sm font-medium text-[var(--color-text-primary)] font-mono">{agent.name}</p>
                                  <p class="text-xs text-[var(--color-text-tertiary)]">{agent.ip_address || '-'} · v{agent.version || '?'}</p>
                                </td>
                                <td class="px-5 py-3"><Badge variant={getAgentStatusVariant(agent.status)} dot size="sm">{agent.status}</Badge></td>
                                <td class="px-5 py-3 text-sm text-[var(--color-text-secondary)]">{agent.executor}</td>
                                <td class="px-5 py-3">
                                  <div class="flex flex-wrap gap-1">
                                    <For each={agent.labels.slice(0, 3)}>
                                      {(label) => <span class="text-xs px-1.5 py-0.5 rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] border border-[var(--color-border-primary)]">{label}</span>}
                                    </For>
                                    <Show when={agent.labels.length > 3}>
                                      <span class="text-xs text-[var(--color-text-tertiary)]">+{agent.labels.length - 3}</span>
                                    </Show>
                                  </div>
                                </td>
                                <td class="px-5 py-3 text-xs text-[var(--color-text-secondary)]">{agent.cpu_cores || '?'} CPU · {agent.memory_mb ? Math.round(agent.memory_mb / 1024) + 'GB' : '?'}</td>
                                <td class="px-5 py-3 text-right">
                                  <div class="flex items-center gap-1 justify-end">
                                    <Show when={agent.status === 'online'}>
                                      <Button size="sm" variant="ghost" onClick={() => setAgentToDrain(agent)}>Drain</Button>
                                    </Show>
                                    <Button size="sm" variant="danger" onClick={() => setAgentToRemove(agent)}>Remove</Button>
                                  </div>
                                </td>
                              </tr>
                            )}
                          </For>
                        </tbody>
                      </table>
                    </Show>
                  </Show>
                </Card>
              </div>

              {/* Drain Agent Confirmation */}
              <Show when={agentToDrain()}>
                <Modal
                  open={!!agentToDrain()}
                  onClose={() => setAgentToDrain(null)}
                  title="Drain Agent"
                  footer={
                    <>
                      <Button variant="ghost" onClick={() => setAgentToDrain(null)}>Cancel</Button>
                      <Button variant="warning" onClick={handleDrainAgent} loading={drainingAgent()}>Drain Agent</Button>
                    </>
                  }
                >
                  <p class="text-sm text-[var(--color-text-secondary)]">
                    Draining agent <strong class="text-[var(--color-text-primary)] font-mono">"{agentToDrain()!.name}"</strong> will stop it from accepting new jobs. Currently running jobs will complete before the agent goes offline.
                  </p>
                </Modal>
              </Show>

              {/* Remove Agent Confirmation */}
              <Show when={agentToRemove()}>
                <Modal
                  open={!!agentToRemove()}
                  onClose={() => setAgentToRemove(null)}
                  title="Remove Agent"
                  footer={
                    <>
                      <Button variant="ghost" onClick={() => setAgentToRemove(null)}>Cancel</Button>
                      <Button variant="danger" onClick={handleRemoveAgent} loading={removingAgent()}>Remove Agent</Button>
                    </>
                  }
                >
                  <div class="space-y-3">
                    <p class="text-sm text-[var(--color-text-secondary)]">
                      Are you sure you want to remove agent <strong class="text-[var(--color-text-primary)] font-mono">"{agentToRemove()!.name}"</strong>?
                    </p>
                    <div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30">
                      <p class="text-sm text-red-400">This agent will be permanently deregistered. You will need to re-register it to use it again.</p>
                    </div>
                  </div>
                </Modal>
              </Show>
            </Match>

            {/* ---- Audit Logs ---- */}
            <Match when={activeSection() === 'audit'}>
              <Show when={auditLogs.error}>
                <div class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30 flex items-center justify-between">
                  <p class="text-sm text-red-400">Failed to load audit logs: {(auditLogs.error as Error)?.message}</p>
                  <Button size="sm" variant="outline" onClick={refetchAuditLogs}>Retry</Button>
                </div>
              </Show>

              <Card title="Audit Log" description="Track all system-wide actions" actions={
                <Button size="sm" variant="outline" onClick={refetchAuditLogs}>Refresh</Button>
              } padding={false}>
                <Show when={!auditLogs.loading} fallback={
                  <div class="p-5 space-y-3 animate-pulse">
                    <For each={[1, 2, 3, 4, 5]}>{() => <div class="h-14 bg-[var(--color-bg-tertiary)] rounded" />}</For>
                  </div>
                }>
                  <Show when={(auditLogs() ?? []).length > 0} fallback={
                    <div class="p-8 text-center">
                      <p class="text-[var(--color-text-secondary)]">No audit logs yet</p>
                    </div>
                  }>
                    <div class="divide-y divide-[var(--color-border-primary)]">
                      <For each={auditLogs() ?? []}>
                        {(log) => {
                          const actor = getUserById(log.actor_id);
                          return (
                            <div class="flex items-start gap-4 px-5 py-4 hover:bg-[var(--color-bg-hover)]">
                              <div class="w-8 h-8 rounded-full bg-[var(--color-bg-tertiary)] flex items-center justify-center flex-shrink-0 text-xs font-bold text-[var(--color-text-tertiary)]">
                                {actor ? (actor.display_name || actor.username).slice(0, 2).toUpperCase() : '??'}
                              </div>
                              <div class="flex-1 min-w-0">
                                <p class="text-sm text-[var(--color-text-primary)]">
                                  <span class="font-medium">{actor?.display_name || actor?.username || log.actor_id || 'System'}</span>
                                  {' '}
                                  <span class={getActionColor(log.action)}>{log.action}</span>
                                  {' '}
                                  <span class="text-[var(--color-text-tertiary)]">{log.resource}</span>
                                  <Show when={log.resource_id}>
                                    <span class="font-mono text-xs text-[var(--color-text-tertiary)]"> ({log.resource_id})</span>
                                  </Show>
                                </p>
                                <p class="text-xs text-[var(--color-text-tertiary)] mt-0.5">
                                  {formatRelativeTime(log.created_at)}{log.actor_ip ? ` · IP: ${log.actor_ip}` : ''}
                                </p>
                              </div>
                            </div>
                          );
                        }}
                      </For>
                    </div>
                  </Show>
                </Show>
              </Card>
            </Match>

            {/* ---- Feature Flags ---- */}
            <Match when={activeSection() === 'flags'}>
              <Card title="Feature Flags" description="Toggle experimental features">
                <Show when={!featureFlags.loading} fallback={
                  <div class="space-y-3 animate-pulse">
                    <For each={[1, 2, 3, 4]}>{() => <div class="h-16 bg-[var(--color-bg-tertiary)] rounded-lg" />}</For>
                  </div>
                }>
                  <Show when={(featureFlags() ?? []).length > 0} fallback={
                    <div class="text-center py-8">
                      <p class="text-[var(--color-text-secondary)]">No feature flags configured</p>
                    </div>
                  }>
                    <div class="space-y-3">
                      <For each={featureFlags() ?? []}>
                        {(flag) => (
                          <div class="flex items-center justify-between p-4 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
                            <div>
                              <div class="flex items-center gap-2">
                                <p class="text-sm font-medium text-[var(--color-text-primary)]">{flag.name}</p>
                                <code class="text-xs font-mono text-[var(--color-text-tertiary)] bg-[var(--color-bg-tertiary)] px-1.5 py-0.5 rounded">{flag.key}</code>
                              </div>
                              <p class="text-xs text-[var(--color-text-tertiary)] mt-1">{flag.description}</p>
                            </div>
                            <button
                              class={`w-10 h-6 rounded-full relative cursor-pointer transition-colors ${flag.enabled ? 'bg-indigo-500' : 'bg-gray-600'}`}
                              onClick={() => handleToggleFlag(flag)}
                            >
                              <div class={`absolute top-1 w-4 h-4 rounded-full bg-white transition-transform ${flag.enabled ? 'left-5' : 'left-1'}`} />
                            </button>
                          </div>
                        )}
                      </For>
                    </div>
                  </Show>
                </Show>
              </Card>
            </Match>
          </Switch>
        </div>
      </div>
    </PageContainer>
  );
};

export default AdminPage;
