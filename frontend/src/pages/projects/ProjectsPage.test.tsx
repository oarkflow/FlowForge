import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import { Router } from '@solidjs/router';

// Mock the API client
vi.mock('../../api/client', () => ({
  api: {
    projects: {
      list: vi.fn().mockResolvedValue({
        data: [
          { id: 'proj-1', name: 'Alpha Project', slug: 'alpha-project', description: 'A test project', visibility: 'private', created_at: '2026-01-15T10:00:00Z' },
          { id: 'proj-2', name: 'Beta Project', slug: 'beta-project', description: 'Another project', visibility: 'public', created_at: '2026-02-20T12:00:00Z' },
        ],
        total: 2,
        page: 1,
        per_page: 100,
      }),
      create: vi.fn().mockResolvedValue({
        id: 'proj-new',
        name: 'New Project',
        slug: 'new-project',
        visibility: 'private',
        created_at: '2026-03-01T00:00:00Z',
      }),
    },
  },
  ApiRequestError: class ApiRequestError extends Error {
    constructor(message: string, public status: number) {
      super(message);
    }
  },
}));

describe('ProjectsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the projects page', async () => {
    const { default: ProjectsPage } = await import('./ProjectsPage');
    render(() => (
      <Router>
        <ProjectsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('Projects')).toBeInTheDocument();
  });

  it('renders project cards after loading', async () => {
    const { default: ProjectsPage } = await import('./ProjectsPage');
    render(() => (
      <Router>
        <ProjectsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('Alpha Project')).toBeInTheDocument();
    expect(screen.getByText('Beta Project')).toBeInTheDocument();
  });

  it('renders search input', async () => {
    const { default: ProjectsPage } = await import('./ProjectsPage');
    render(() => (
      <Router>
        <ProjectsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByPlaceholderText(/search projects/i)).toBeInTheDocument();
  });

  it('renders Quick Create button', async () => {
    const { default: ProjectsPage } = await import('./ProjectsPage');
    render(() => (
      <Router>
        <ProjectsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('Quick Create')).toBeInTheDocument();
  });

  it('renders Import Project button', async () => {
    const { default: ProjectsPage } = await import('./ProjectsPage');
    render(() => (
      <Router>
        <ProjectsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('Import Project')).toBeInTheDocument();
  });

  it('renders project descriptions', async () => {
    const { default: ProjectsPage } = await import('./ProjectsPage');
    render(() => (
      <Router>
        <ProjectsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('A test project')).toBeInTheDocument();
    expect(screen.getByText('Another project')).toBeInTheDocument();
  });

  it('renders project slugs', async () => {
    const { default: ProjectsPage } = await import('./ProjectsPage');
    render(() => (
      <Router>
        <ProjectsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('alpha-project')).toBeInTheDocument();
    expect(screen.getByText('beta-project')).toBeInTheDocument();
  });

  it('filters projects by search', async () => {
    const { default: ProjectsPage } = await import('./ProjectsPage');
    render(() => (
      <Router>
        <ProjectsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));

    const searchInput = screen.getByPlaceholderText(/search projects/i);
    fireEvent.input(searchInput, { target: { value: 'Alpha' } });

    expect(screen.getByText('Alpha Project')).toBeInTheDocument();
    expect(screen.queryByText('Beta Project')).not.toBeInTheDocument();
  });
});
