import type { Component } from 'solid-js';
import { createSignal, createResource, onMount, onCleanup, Show, For } from 'solid-js';
import { A, useNavigate } from '@solidjs/router';
import PageContainer from '../../components/layout/PageContainer';
import Card from '../../components/ui/Card';
import Badge from '../../components/ui/Badge';
import Button from '../../components/ui/Button';
import Table, { type TableColumn } from '../../components/ui/Table';
import { toast } from '../../components/ui/Toast';
import { api } from '../../api/client';
import { getEventSocket } from '../../api/websocket';
import type { Project, Agent, PipelineRun, RunStatus, TriggerType, AgentStatus, SystemHealth } from '../../types';
import { formatDuration, formatRelativeTime, getStatusBgColor, getAgentStatusVariant } from '../../utils/helpers';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
interface StatItem { label: string; value: string; trend: number; icon: string; }
interface RunWithMeta extends PipelineRun { pipeline_name: string; project_id: string; project_name: string; }
interface ActivityEvent { id: string; type: string; message: string; actor: string; time: string; }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
const agentDotColor = (s: AgentStatus) => s === 'online' ? 'bg-emerald-400' : s === 'busy' ? 'bg-violet-400 animate-pulse' : s === 'draining' ? 'bg-amber-400' : 'bg-gray-500';
const activityColor = (t: string) => t === 'success' ? 'bg-emerald-400' : t === 'failure' ? 'bg-red-400' : t === 'deploy' ? 'bg-blue-400' : t === 'agent' ? 'bg-violet-400' : 'bg-amber-400';

const SkeletonCard: Component = () => (
  <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-5 animate-pulse">
    <div class="flex items-center justify-between mb-3"><div class="h-3 w-24 bg-[var(--color-bg-tertiary)] rounded" /><div class="h-5 w-5 bg-[var(--color-bg-tertiary)] rounded" /></div>
    <div class="h-7 w-16 bg-[var(--color-bg-tertiary)] rounded mb-2" /><div class="h-3 w-20 bg-[var(--color-bg-tertiary)] rounded" />
  </div>
);

// ---------------------------------------------------------------------------
// Data fetcher
// ---------------------------------------------------------------------------
async function fetchDashboardData() {
  const [projectsRes, agents, health] = await Promise.all([
    api.projects.list({ page: '1', per_page: '100' }),
    api.agents.list(),
    api.system.health(),
  ]);

  const projects = projectsRes.data;

  // Fetch recent runs across all projects and their first pipeline
  const recentRuns: RunWithMeta[] = [];
  const pipelinesByProject = new Map<string, { pipeline_id: string; pipeline_name: string; project_name: string }[]>();

  for (const project of projects.slice(0, 5)) {
    try {
      const pipelines = await api.pipelines.list(project.id);
      const entries = pipelines.map(p => ({ pipeline_id: p.id, pipeline_name: p.name, project_name: project.name }));
      pipelinesByProject.set(project.id, entries);

      for (const pipeline of pipelines.slice(0, 3)) {
        try {
          const runsRes = await api.runs.list(project.id, pipeline.id, { page: '1', per_page: '3' });
          for (const run of runsRes.data) {
            recentRuns.push({ ...run, pipeline_name: pipeline.name, project_id: project.id, project_name: project.name });
          }
        } catch { /* pipeline may have no runs */ }
      }
    } catch { /* project may have no pipelines */ }
  }

  recentRuns.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());

  // Compute stats
  const totalPipelines = Array.from(pipelinesByProject.values()).reduce((sum, arr) => sum + arr.length, 0);
  const onlineAgents = agents.filter(a => a.status === 'online' || a.status === 'busy').length;
  const queuedRuns = recentRuns.filter(r => r.status === 'queued').length;
  const completedRuns = recentRuns.filter(r => r.status === 'success' || r.status === 'failure');
  const successRate = completedRuns.length > 0
    ? Math.round(completedRuns.filter(r => r.status === 'success').length / completedRuns.length * 100)
    : 0;
  const avgDuration = completedRuns.length > 0
    ? Math.round(completedRuns.reduce((sum, r) => sum + (r.duration_ms || 0), 0) / completedRuns.length)
    : 0;

  const stats: StatItem[] = [
    { label: 'Total Pipelines', value: String(totalPipelines), trend: 0, icon: 'M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z' },
    { label: 'Recent Runs', value: String(recentRuns.length), trend: 0, icon: 'M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z' },
    { label: 'Success Rate', value: `${successRate}%`, trend: 0, icon: 'M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z' },
    { label: 'Avg Duration', value: formatDuration(avgDuration), trend: 0, icon: 'M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z' },
    { label: 'Active Agents', value: `${onlineAgents} / ${agents.length}`, trend: 0, icon: 'M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2' },
    { label: 'Queued Jobs', value: String(queuedRuns), trend: 0, icon: 'M3.75 12h16.5m-16.5 3.75h16.5M3.75 19.5h16.5M5.625 4.5h12.75a1.875 1.875 0 010 3.75H5.625a1.875 1.875 0 010-3.75z' },
  ];

  return { stats, runs: recentRuns.slice(0, 10), agents, health, projects };
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const DashboardPage: Component = () => {
  const navigate = useNavigate();
  const [data, { refetch }] = createResource(fetchDashboardData);
  const [activity, setActivity] = createSignal<ActivityEvent[]>([]);

  // Subscribe to real-time events
  onMount(() => {
    const ws = getEventSocket();
    if (!ws.isConnected) ws.connect();

    const unsub1 = ws.on('run.completed', (payload: any) => {
      const status = payload?.status || 'success';
      const type = status === 'success' ? 'success' : 'failure';
      setActivity(prev => [{
        id: `evt-${Date.now()}`, type,
        message: `${payload?.pipeline_name || 'Pipeline'} #${payload?.number || '?'} ${status === 'success' ? 'completed successfully' : 'failed'}`,
        actor: payload?.author || 'system',
        time: new Date().toISOString(),
      }, ...prev].slice(0, 20));
      refetch();
    });

    const unsub2 = ws.on('agent.status', (payload: any) => {
      setActivity(prev => [{
        id: `evt-${Date.now()}`, type: 'agent',
        message: `${payload?.name || 'Agent'} is now ${payload?.status || 'unknown'}`,
        actor: 'system',
        time: new Date().toISOString(),
      }, ...prev].slice(0, 20));
      refetch();
    });

    onCleanup(() => { unsub1(); unsub2(); });
  });

  const handleRefresh = () => {
    refetch();
    toast.info('Refreshing dashboard...');
  };

  const runColumns: TableColumn<RunWithMeta>[] = [
    { key: 'status', header: '', width: '40px', align: 'center', render: (row) => <span class={`w-2.5 h-2.5 rounded-full inline-block ${getStatusBgColor(row.status)} ${row.status === 'running' ? 'animate-pulse' : ''}`} /> },
    { key: 'pipeline', header: 'Pipeline', render: (row) => (<div><div class="flex items-center gap-2"><span class="font-medium text-[var(--color-text-primary)]">{row.pipeline_name}</span><span class="text-[var(--color-text-tertiary)]">#{row.number}</span></div><div class="text-xs text-[var(--color-text-tertiary)] mt-0.5 truncate max-w-xs">{row.commit_message}</div></div>) },
    { key: 'branch', header: 'Branch', render: (row) => row.branch ? <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-[var(--color-bg-tertiary)] text-xs text-[var(--color-text-secondary)] font-mono">{row.branch}</span> : <span class="text-[var(--color-text-tertiary)]">-</span> },
    { key: 'trigger', header: 'Trigger', render: (row) => <span class="text-xs text-[var(--color-text-secondary)] capitalize">{row.trigger_type.replace('_', ' ')}</span> },
    { key: 'duration', header: 'Duration', align: 'right', render: (row) => <span class="text-xs font-mono text-[var(--color-text-secondary)]">{formatDuration(row.duration_ms)}</span> },
    { key: 'time', header: 'Started', align: 'right', render: (row) => <span class="text-xs text-[var(--color-text-tertiary)]">{formatRelativeTime(row.started_at ?? row.created_at)}</span> },
  ];

  return (
    <PageContainer title="Dashboard" description="Overview of your CI/CD platform" actions={<div class="flex gap-2"><Button variant="outline" size="sm" onClick={handleRefresh} loading={data.loading}>Refresh</Button><Button size="sm" onClick={() => navigate('/projects')}>New Project</Button></div>}>
      {/* Error state */}
      <Show when={data.error}>
        <div class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30">
          <div class="flex items-center justify-between">
            <p class="text-sm text-red-400">Failed to load dashboard data: {(data.error as Error)?.message || 'Unknown error'}</p>
            <Button size="sm" variant="outline" onClick={handleRefresh}>Retry</Button>
          </div>
        </div>
      </Show>

      {/* Stats */}
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4 mb-6">
        <Show when={!data.loading} fallback={<><SkeletonCard /><SkeletonCard /><SkeletonCard /><SkeletonCard /><SkeletonCard /><SkeletonCard /></>}>
          <For each={data()?.stats ?? []}>{(stat) => (
            <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-5 hover:border-[var(--color-border-secondary)] transition-colors">
              <div class="flex items-center justify-between mb-3">
                <span class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">{stat.label}</span>
                <div class="p-1.5 rounded-lg bg-[var(--color-bg-tertiary)]"><svg class="w-4 h-4 text-[var(--color-text-tertiary)]" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d={stat.icon} /></svg></div>
              </div>
              <div class="text-2xl font-bold text-[var(--color-text-primary)] tabular-nums">{stat.value}</div>
            </div>
          )}</For>
        </Show>
      </div>

      {/* Main grid */}
      <div class="grid grid-cols-1 xl:grid-cols-3 gap-6">
        <div class="xl:col-span-2">
          <Card title="Recent Pipeline Runs" description="Latest pipeline executions" padding={false} actions={<A href="/runs" class="text-xs text-indigo-400 hover:text-indigo-300">View all</A>}>
            <Show when={!data.loading} fallback={<div class="p-6 space-y-3">{Array.from({ length: 5 }).map(() => <div class="h-12 bg-[var(--color-bg-tertiary)] rounded animate-pulse" />)}</div>}>
              <Table
                columns={runColumns}
                data={data()?.runs ?? []}
                emptyMessage="No pipeline runs yet. Create a project and trigger a pipeline to get started."
                onRowClick={(row) => {
                  navigate(`/projects/${row.project_id}/pipelines/${row.pipeline_id}/runs/${row.id}`);
                }}
              />
            </Show>
          </Card>
        </div>

        <div class="flex flex-col gap-6">
          <Card title="Agent Status" padding={false} actions={<A href="/agents" class="text-xs text-indigo-400 hover:text-indigo-300">Manage</A>}>
            <Show when={!data.loading} fallback={<div class="p-4 space-y-3">{Array.from({ length: 3 }).map(() => <div class="h-10 bg-[var(--color-bg-tertiary)] rounded animate-pulse" />)}</div>}>
              <Show when={(data()?.agents ?? []).length > 0} fallback={
                <div class="p-6 text-center">
                  <p class="text-sm text-[var(--color-text-tertiary)]">No agents registered yet.</p>
                  <A href="/agents" class="text-xs text-indigo-400 hover:text-indigo-300 mt-1 inline-block">Register an agent</A>
                </div>
              }>
                <div class="divide-y divide-[var(--color-border-primary)]">
                  <For each={data()?.agents ?? []}>{(agent) => (
                    <div class="flex items-center justify-between px-5 py-3 hover:bg-[var(--color-bg-hover)] transition-colors">
                      <div class="flex items-center gap-3">
                        <span class={`w-2 h-2 rounded-full shrink-0 ${agentDotColor(agent.status)}`} />
                        <div><div class="text-sm font-medium text-[var(--color-text-primary)]">{agent.name}</div><div class="text-xs text-[var(--color-text-tertiary)]">{agent.executor} · {agent.os}/{agent.arch}</div></div>
                      </div>
                      <Badge variant={getAgentStatusVariant(agent.status)} size="sm">{agent.status}</Badge>
                    </div>
                  )}</For>
                </div>
              </Show>
            </Show>
          </Card>

          <Card title="Quick Actions">
            <div class="grid grid-cols-2 gap-2">
              {[
                { label: 'New Project', color: 'text-indigo-400', href: '/projects', icon: 'M12 4.5v15m7.5-7.5h-15' },
                { label: 'Add Agent', color: 'text-emerald-400', href: '/agents', icon: 'M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2' },
                { label: 'Add Secret', color: 'text-amber-400', href: '/settings', icon: 'M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z' },
                { label: 'View Logs', color: 'text-blue-400', href: '/runs', icon: 'M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z' },
              ].map(({ label, color, href, icon }) => (
                <button onClick={() => navigate(href)} class="flex flex-col items-center gap-2 p-3 rounded-lg bg-[var(--color-bg-tertiary)] hover:bg-[var(--color-bg-hover)] border border-[var(--color-border-primary)] transition-colors cursor-pointer">
                  <svg class={`w-5 h-5 ${color}`} fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d={icon} /></svg>
                  <span class="text-xs font-medium text-[var(--color-text-secondary)]">{label}</span>
                </button>
              ))}
            </div>
          </Card>
        </div>
      </div>

      {/* Activity Feed — populated via WebSocket events */}
      <Show when={activity().length > 0}>
        <div class="mt-6">
          <Card title="Live Activity" padding={false}>
            <div class="divide-y divide-[var(--color-border-primary)]">
              <For each={activity()}>{(event) => (
                <div class="flex items-start gap-3 px-5 py-3.5 hover:bg-[var(--color-bg-hover)] transition-colors">
                  <span class={`w-2 h-2 rounded-full mt-1.5 shrink-0 ${activityColor(event.type)}`} />
                  <div class="flex-1 min-w-0">
                    <p class="text-sm text-[var(--color-text-primary)]">{event.message}</p>
                    <div class="flex items-center gap-2 mt-0.5"><span class="text-xs text-[var(--color-text-tertiary)]">{event.actor}</span><span class="text-xs text-[var(--color-text-tertiary)]">·</span><span class="text-xs text-[var(--color-text-tertiary)]">{formatRelativeTime(event.time)}</span></div>
                  </div>
                </div>
              )}</For>
            </div>
          </Card>
        </div>
      </Show>
    </PageContainer>
  );
};

export default DashboardPage;
