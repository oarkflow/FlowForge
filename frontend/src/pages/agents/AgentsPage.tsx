import type { Component } from 'solid-js';
import { createSignal, createResource, For, Show } from 'solid-js';
import PageContainer from '../../components/layout/PageContainer';
import Badge from '../../components/ui/Badge';
import Button from '../../components/ui/Button';
import Input from '../../components/ui/Input';
import Modal from '../../components/ui/Modal';
import Select from '../../components/ui/Select';
import { toast } from '../../components/ui/Toast';
import { api, ApiRequestError } from '../../api/client';
import type { Agent, AgentStatus } from '../../types';
import { formatRelativeTime, getAgentStatusVariant, copyToClipboard } from '../../utils/helpers';

// ---------------------------------------------------------------------------
// Fetcher
// ---------------------------------------------------------------------------
async function fetchAgents() {
  return api.agents.list();
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const AgentsPage: Component = () => {
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

  return (
    <PageContainer
      title="Agents"
      description="Manage build agents and workers"
      actions={
        <Button
          onClick={() => { setShowRegister(true); setNewAgentToken(''); setNewAgentName(''); setNewAgentLabels(''); setRegisterError(''); }}
          icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>}
        >
          Register Agent
        </Button>
      }
    >
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
                      <div class={`w-10 h-10 rounded-lg flex items-center justify-center ${
                        agent.status === 'online' ? 'bg-emerald-500/10' :
                        agent.status === 'busy' ? 'bg-violet-500/10' :
                        agent.status === 'draining' ? 'bg-amber-500/10' : 'bg-gray-500/10'
                      }`}>
                        <svg class={`w-5 h-5 ${
                          agent.status === 'online' ? 'text-emerald-400' :
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
    </PageContainer>
  );
};

export default AgentsPage;
