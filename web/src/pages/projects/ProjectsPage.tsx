import type { Component } from 'solid-js';
import { createSignal, createResource, For, Show, onMount } from 'solid-js';
import { A, useNavigate } from '@solidjs/router';
import PageContainer from '../../components/layout/PageContainer';
import Button from '../../components/ui/Button';
import Badge from '../../components/ui/Badge';
import Input from '../../components/ui/Input';
import Modal from '../../components/ui/Modal';
import Select from '../../components/ui/Select';
import { toast } from '../../components/ui/Toast';
import { api, ApiRequestError } from '../../api/client';
import type { Project } from '../../types';
import { formatRelativeTime } from '../../utils/helpers';

// ---------------------------------------------------------------------------
// Fetcher
// ---------------------------------------------------------------------------
async function fetchProjects() {
  const res = await api.projects.list({ page: '1', per_page: '100' });
  return res.data;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const ProjectsPage: Component = () => {
  const [projects, { refetch, mutate }] = createResource(fetchProjects);
  const [search, setSearch] = createSignal('');
  const [showCreateModal, setShowCreateModal] = createSignal(false);
  const [creating, setCreating] = createSignal(false);
  const navigate = useNavigate();

  // Create form
  const [newName, setNewName] = createSignal('');
  const [newDescription, setNewDescription] = createSignal('');
  const [newVisibility, setNewVisibility] = createSignal('private');
  const [createError, setCreateError] = createSignal('');

  const filteredProjects = () => {
    const q = search().toLowerCase();
    const all = projects() ?? [];
    if (!q) return all;
    return all.filter(p =>
      p.name.toLowerCase().includes(q) ||
      p.slug.toLowerCase().includes(q) ||
      p.description?.toLowerCase().includes(q)
    );
  };

  const handleCreate = async () => {
    if (!newName().trim()) return;
    setCreating(true);
    setCreateError('');
    try {
      const created = await api.projects.create({
        name: newName().trim(),
        description: newDescription().trim() || undefined,
        visibility: newVisibility() as 'private' | 'internal' | 'public',
      });
      mutate(prev => prev ? [created, ...prev] : [created]);
      setShowCreateModal(false);
      setNewName('');
      setNewDescription('');
      setNewVisibility('private');
      toast.success(`Project "${created.name}" created`);
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to create project';
      setCreateError(msg);
      toast.error(msg);
    } finally {
      setCreating(false);
    }
  };

  return (
    <PageContainer
      title="Projects"
      description="Manage your CI/CD projects"
      actions={
        <div class="flex gap-2">
          <Button
            variant="outline"
            onClick={() => setShowCreateModal(true)}
            icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>}
          >Quick Create</Button>
          <Button
            onClick={() => navigate('/projects/import')}
            icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4 4a2 2 0 00-2 2v8a2 2 0 002 2h12a2 2 0 002-2V8a2 2 0 00-2-2h-5L9 4H4zm7 5a1 1 0 10-2 0v1.586l-.293-.293a1 1 0 10-1.414 1.414l2 2 .007.007a.997.997 0 001.4-.007l2-2a1 1 0 00-1.414-1.414L11 10.586V9z" clip-rule="evenodd" /></svg>}
          >Import Project</Button>
        </div>
      }
    >
      {/* Error state */}
      <Show when={projects.error}>
        <div class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30 flex items-center justify-between">
          <p class="text-sm text-red-400">Failed to load projects: {(projects.error as Error)?.message}</p>
          <Button size="sm" variant="outline" onClick={refetch}>Retry</Button>
        </div>
      </Show>

      {/* Search */}
      <div class="mb-6">
        <Input
          placeholder="Search projects by name, slug, or description..."
          value={search()}
          onInput={(e) => setSearch(e.currentTarget.value)}
          icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z" clip-rule="evenodd" /></svg>}
        />
      </div>

      {/* Loading skeleton */}
      <Show when={!projects.loading} fallback={
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <For each={[1, 2, 3, 4]}>{() => (
            <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-5 animate-pulse">
              <div class="h-5 w-40 bg-[var(--color-bg-tertiary)] rounded mb-2" />
              <div class="h-3 w-24 bg-[var(--color-bg-tertiary)] rounded mb-4" />
              <div class="h-4 w-full bg-[var(--color-bg-tertiary)] rounded mb-4" />
              <div class="flex justify-between"><div class="h-3 w-20 bg-[var(--color-bg-tertiary)] rounded" /><div class="h-3 w-24 bg-[var(--color-bg-tertiary)] rounded" /></div>
            </div>
          )}</For>
        </div>
      }>
        {/* Empty state */}
        <Show when={filteredProjects().length > 0} fallback={
          <div class="text-center py-16">
            <svg class="w-12 h-12 mx-auto text-[var(--color-text-tertiary)] mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" /></svg>
            <Show when={search()} fallback={
              <>
                <p class="text-[var(--color-text-secondary)] mb-2">No projects yet</p>
                <p class="text-sm text-[var(--color-text-tertiary)] mb-4">Create your first project to get started with CI/CD.</p>
                <div class="flex gap-2 justify-center">
                  <Button variant="outline" onClick={() => setShowCreateModal(true)}>Quick Create</Button>
                  <Button onClick={() => navigate('/projects/import')}>Import Project</Button>
                </div>
              </>
            }>
              <p class="text-[var(--color-text-secondary)]">No projects match "{search()}"</p>
            </Show>
          </div>
        }>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <For each={filteredProjects()}>
              {(project) => (
                <A href={`/projects/${project.id}`} class="block group">
                  <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-5 hover:border-[var(--color-border-secondary)] hover:bg-[var(--color-bg-tertiary)]/30 transition-all">
                    <div class="flex items-start justify-between mb-3">
                      <div class="min-w-0 flex-1">
                        <h3 class="text-base font-semibold text-[var(--color-text-primary)] group-hover:text-indigo-400 transition-colors truncate">{project.name}</h3>
                        <p class="text-xs text-[var(--color-text-tertiary)] font-mono mt-0.5">{project.slug}</p>
                      </div>
                      <Badge variant="default" size="sm">{project.visibility}</Badge>
                    </div>
                    <Show when={project.description}>
                      <p class="text-sm text-[var(--color-text-secondary)] mb-4 line-clamp-2">{project.description}</p>
                    </Show>
                    <div class="flex items-center justify-between text-xs text-[var(--color-text-tertiary)]">
                      <span class="capitalize">{project.visibility}</span>
                      <span>Created {formatRelativeTime(project.created_at)}</span>
                    </div>
                  </div>
                </A>
              )}
            </For>
          </div>
        </Show>
      </Show>

      {/* Create Project Modal */}
      <Show when={showCreateModal()}>
        <Modal
          open={showCreateModal()}
          onClose={() => { setShowCreateModal(false); setCreateError(''); }}
          title="Create Project"
          description="Set up a new CI/CD project"
          footer={
            <>
              <Button variant="ghost" onClick={() => { setShowCreateModal(false); setCreateError(''); }}>Cancel</Button>
              <Button onClick={handleCreate} loading={creating()} disabled={!newName().trim()}>Create Project</Button>
            </>
          }
        >
          <div class="space-y-4">
            <Show when={createError()}>
              <div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-sm text-red-400">{createError()}</div>
            </Show>
            <Input label="Project Name" placeholder="My Awesome Project" value={newName()} onInput={(e) => setNewName(e.currentTarget.value)} />
            <Show when={newName()}>
              <p class="text-xs text-[var(--color-text-tertiary)] -mt-2">
                Slug: <span class="font-mono">{newName().toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')}</span>
              </p>
            </Show>
            <Input label="Description" placeholder="Brief description of the project..." value={newDescription()} onInput={(e) => setNewDescription(e.currentTarget.value)} />
            <Select label="Visibility" value={newVisibility()} onChange={(e) => setNewVisibility(e.currentTarget.value)} options={[
              { value: 'private', label: 'Private — Only team members' },
              { value: 'internal', label: 'Internal — Organization members' },
              { value: 'public', label: 'Public — Everyone' },
            ]} />
          </div>
        </Modal>
      </Show>
    </PageContainer>
  );
};

export default ProjectsPage;
