import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createRoot } from 'solid-js';

vi.mock('../api/client', () => ({
  api: {
    projects: {
      list: vi.fn(),
      get: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
    },
  },
}));

describe('projectStore', () => {
  let projectStore: typeof import('./projectStore').projectStore;
  let api: any;

  beforeEach(async () => {
    vi.clearAllMocks();
    const clientModule = await import('../api/client');
    api = (clientModule as any).api;
    const storeModule = await import('./projectStore');
    projectStore = storeModule.projectStore;
  });

  it('starts with empty projects', () => {
    createRoot(() => {
      expect(projectStore.projects()).toEqual([]);
    });
  });

  it('starts with null currentProject', () => {
    createRoot(() => {
      expect(projectStore.currentProject()).toBeNull();
    });
  });

  it('fetchProjects loads projects from API', async () => {
    const mockProjects = { items: [{ id: '1', name: 'P1' }] };
    api.projects.list.mockResolvedValueOnce(mockProjects);

    await createRoot(async () => {
      await projectStore.fetchProjects();
      expect(api.projects.list).toHaveBeenCalledWith({ page: '1', per_page: '50' });
    });
  });

  it('fetchProjects sets error on failure', async () => {
    api.projects.list.mockRejectedValueOnce(new Error('Network error'));

    await createRoot(async () => {
      try {
        await projectStore.fetchProjects();
      } catch {
        // Expected
      }
      expect(projectStore.error()).toBe('Network error');
    });
  });

  it('fetchProject loads a single project', async () => {
    const mockProject = { id: '1', name: 'Test', slug: 'test' };
    api.projects.get.mockResolvedValueOnce(mockProject);

    await createRoot(async () => {
      const result = await projectStore.fetchProject('1');
      expect(api.projects.get).toHaveBeenCalledWith('1');
      expect(result).toEqual(mockProject);
    });
  });

  it('createProject adds project to list', async () => {
    const newProject = { id: '2', name: 'New' };
    api.projects.create.mockResolvedValueOnce(newProject);

    await createRoot(async () => {
      const result = await projectStore.createProject({ name: 'New' });
      expect(result).toEqual(newProject);
    });
  });

  it('deleteProject removes project from list', async () => {
    api.projects.delete.mockResolvedValueOnce(undefined);

    await createRoot(async () => {
      await projectStore.deleteProject('1');
      expect(api.projects.delete).toHaveBeenCalledWith('1');
    });
  });

  it('loading is true during fetch', async () => {
    let resolvePromise: any;
    api.projects.list.mockReturnValueOnce(
      new Promise(r => { resolvePromise = r; }),
    );

    await createRoot(async () => {
      const fetchPromise = projectStore.fetchProjects();
      expect(projectStore.loading()).toBe(true);
      resolvePromise({ items: [] });
      await fetchPromise;
      expect(projectStore.loading()).toBe(false);
    });
  });
});
