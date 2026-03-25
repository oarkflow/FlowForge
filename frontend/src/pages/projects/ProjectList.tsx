import type { Component } from 'solid-js';
import { createSignal, For, Show } from 'solid-js';
import { A } from '@solidjs/router';
import PageContainer from '../../components/layout/PageContainer';
import Card from '../../components/ui/Card';
import Button from '../../components/ui/Button';
import Badge from '../../components/ui/Badge';
import Input from '../../components/ui/Input';
import Modal from '../../components/ui/Modal';
import type { Project } from '../../types';

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------
const mockProjects: Project[] = [
  {
    id: 'p1',
    name: 'flowforge-api',
    slug: 'flowforge-api',
    description: 'FlowForge backend API server — Go + GoFiber',
    visibility: 'private',
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-03-23T18:30:00Z',
  },
  {
    id: 'p2',
    name: 'flowforge-ui',
    slug: 'flowforge-ui',
    description: 'FlowForge frontend dashboard — SolidJS + TailwindCSS',
    visibility: 'private',
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-03-24T09:12:00Z',
  },
  {
    id: 'p3',
    name: 'acme-microservice',
    slug: 'acme-microservice',
    description: 'Customer-facing API microservice with gRPC endpoints',
    visibility: 'internal',
    created_at: '2026-02-01T14:00:00Z',
    updated_at: '2026-03-22T11:45:00Z',
  },
  {
    id: 'p4',
    name: 'docs-site',
    slug: 'docs-site',
    description: 'Public documentation website built with Next.js',
    visibility: 'public',
    created_at: '2026-02-10T08:00:00Z',
    updated_at: '2026-03-20T16:20:00Z',
  },
];

// ---------------------------------------------------------------------------
// Project card component
// ---------------------------------------------------------------------------
const ProjectCard: Component<{ project: Project }> = (props) => {
  const p = props.project;

  const visibilityBadge = () => {
    switch (p.visibility) {
      case 'private': return { variant: 'default' as const, label: 'Private' };
      case 'internal': return { variant: 'info' as const, label: 'Internal' };
      case 'public': return { variant: 'success' as const, label: 'Public' };
    }
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
  };

  return (
    <A
      href={`/projects/${p.id}/pipelines/${p.id}/runs`}
      class="block group"
    >
      <div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-5 hover:border-[var(--color-border-secondary)] hover:bg-[var(--color-bg-tertiary)] transition-all duration-150">
        <div class="flex items-start justify-between mb-3">
          <div class="flex items-center gap-3 min-w-0">
            <div class="w-10 h-10 rounded-lg bg-[var(--color-accent-bg)] flex items-center justify-center shrink-0">
              <svg class="w-5 h-5 text-indigo-400" viewBox="0 0 20 20" fill="currentColor">
                <path d="M3 3.5A1.5 1.5 0 014.5 2h6.879a1.5 1.5 0 011.06.44l4.122 4.12A1.5 1.5 0 0117 7.622V16.5a1.5 1.5 0 01-1.5 1.5h-11A1.5 1.5 0 013 16.5v-13z" />
              </svg>
            </div>
            <div class="min-w-0">
              <h3 class="text-sm font-semibold text-[var(--color-text-primary)] group-hover:text-indigo-400 transition-colors truncate">
                {p.name}
              </h3>
              <p class="text-xs text-[var(--color-text-tertiary)] mt-0.5 truncate">
                {p.slug}
              </p>
            </div>
          </div>
          <Badge variant={visibilityBadge().variant} size="sm">
            {visibilityBadge().label}
          </Badge>
        </div>

        <Show when={p.description}>
          <p class="text-sm text-[var(--color-text-secondary)] mb-4 line-clamp-2">
            {p.description}
          </p>
        </Show>

        <div class="flex items-center justify-between pt-3 border-t border-[var(--color-border-primary)]">
          <div class="flex items-center gap-4 text-xs text-[var(--color-text-tertiary)]">
            <span class="flex items-center gap-1">
              <svg class="w-3.5 h-3.5" viewBox="0 0 16 16" fill="currentColor">
                <path fill-rule="evenodd" d="M11.75 2.5a.75.75 0 100 1.5.75.75 0 000-1.5zm-2.25.75a2.25 2.25 0 113 2.122V6A2.5 2.5 0 0110 8.5H6a1 1 0 00-1 1v1.128a2.251 2.251 0 11-1.5 0V5.372a2.25 2.25 0 111.5 0v1.836A2.492 2.492 0 016 7h4a1 1 0 001-1v-.628A2.25 2.25 0 019.5 3.25zM4.25 12a.75.75 0 100 1.5.75.75 0 000-1.5zM3.5 3.25a.75.75 0 111.5 0 .75.75 0 01-1.5 0z" />
              </svg>
              3 pipelines
            </span>
            <span>Updated {formatDate(p.updated_at)}</span>
          </div>
          <svg class="w-4 h-4 text-[var(--color-text-tertiary)] group-hover:text-[var(--color-text-secondary)] transition-colors" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
          </svg>
        </div>
      </div>
    </A>
  );
};

// ---------------------------------------------------------------------------
// Project list page
// ---------------------------------------------------------------------------
const ProjectList: Component = () => {
  const [projects] = createSignal(mockProjects);
  const [search, setSearch] = createSignal('');
  const [showCreate, setShowCreate] = createSignal(false);
  const [newName, setNewName] = createSignal('');
  const [newDesc, setNewDesc] = createSignal('');

  const filteredProjects = () => {
    const q = search().toLowerCase();
    if (!q) return projects();
    return projects().filter(
      (p) =>
        p.name.toLowerCase().includes(q) ||
        p.description?.toLowerCase().includes(q) ||
        p.slug.toLowerCase().includes(q)
    );
  };

  const handleCreate = (e: Event) => {
    e.preventDefault();
    // TODO: API call
    setShowCreate(false);
    setNewName('');
    setNewDesc('');
  };

  return (
    <PageContainer
      title="Projects"
      description="Manage your CI/CD projects and their pipelines."
      actions={
        <Button
          variant="primary"
          size="md"
          onClick={() => setShowCreate(true)}
          icon={
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
              <path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" />
            </svg>
          }
        >
          New Project
        </Button>
      }
    >
      {/* Search bar */}
      <div class="mb-6">
        <Input
          placeholder="Search projects..."
          value={search()}
          onInput={(e) => setSearch(e.currentTarget.value)}
          icon={
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z" clip-rule="evenodd" />
            </svg>
          }
          class="max-w-md"
        />
      </div>

      {/* Project grid */}
      <Show
        when={filteredProjects().length > 0}
        fallback={
          <div class="flex items-center justify-center py-20 border border-dashed border-[var(--color-border-primary)] rounded-xl">
            <div class="text-center">
              <svg class="w-12 h-12 mx-auto text-[var(--color-text-tertiary)] mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
              <p class="text-sm text-[var(--color-text-tertiary)]">
                {search() ? 'No projects match your search.' : 'No projects yet.'}
              </p>
              <Show when={!search()}>
                <Button
                  variant="primary"
                  size="sm"
                  class="mt-4"
                  onClick={() => setShowCreate(true)}
                >
                  Create your first project
                </Button>
              </Show>
            </div>
          </div>
        }
      >
        <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          <For each={filteredProjects()}>
            {(project) => <ProjectCard project={project} />}
          </For>
        </div>
      </Show>

      {/* Create project modal */}
      <Modal
        open={showCreate()}
        onClose={() => setShowCreate(false)}
        title="Create New Project"
        description="Set up a new project to manage your CI/CD pipelines."
        footer={
          <>
            <Button variant="ghost" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button variant="primary" onClick={handleCreate}>
              Create Project
            </Button>
          </>
        }
      >
        <form onSubmit={handleCreate} class="space-y-4">
          <Input
            label="Project Name"
            placeholder="my-awesome-project"
            value={newName()}
            onInput={(e) => setNewName(e.currentTarget.value)}
            required
          />
          <Input
            label="Description"
            placeholder="A brief description of the project..."
            value={newDesc()}
            onInput={(e) => setNewDesc(e.currentTarget.value)}
          />
        </form>
      </Modal>
    </PageContainer>
  );
};

export default ProjectList;
