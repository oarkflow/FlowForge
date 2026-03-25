import { createSignal } from 'solid-js';
import { api } from '../api/client';
import type { Project } from '../types';

const [projects, setProjects] = createSignal<Project[]>([]);
const [currentProject, setCurrentProject] = createSignal<Project | null>(null);
const [loading, setLoading] = createSignal(false);
const [error, setError] = createSignal<string | null>(null);

const fetchProjects = async (page = 1, perPage = 50) => {
  setLoading(true);
  setError(null);
  try {
    const data = await api.projects.list({ page: String(page), per_page: String(perPage) });
    setProjects((data as any).items || (Array.isArray(data) ? data : []));
    return data;
  } catch (e) {
    setError(e instanceof Error ? e.message : 'Failed to fetch projects');
    throw e;
  } finally {
    setLoading(false);
  }
};

const fetchProject = async (id: string) => {
  setLoading(true);
  setError(null);
  try {
    const project = await api.projects.get(id);
    setCurrentProject(project);
    return project;
  } catch (e) {
    setError(e instanceof Error ? e.message : 'Failed to fetch project');
    throw e;
  } finally {
    setLoading(false);
  }
};

const createProject = async (data: Partial<Project>) => {
  const project = await api.projects.create(data);
  setProjects((prev) => [project, ...prev]);
  return project;
};

const updateProject = async (id: string, data: Partial<Project>) => {
  const updated = await api.projects.update(id, data);
  setProjects((prev) => prev.map((p) => (p.id === id ? updated : p)));
  if (currentProject()?.id === id) setCurrentProject(updated);
  return updated;
};

const deleteProject = async (id: string) => {
  await api.projects.delete(id);
  setProjects((prev) => prev.filter((p) => p.id !== id));
  if (currentProject()?.id === id) setCurrentProject(null);
};

export const projectStore = {
  projects,
  currentProject,
  loading,
  error,
  fetchProjects,
  fetchProject,
  createProject,
  updateProject,
  deleteProject,
  setCurrentProject,
};
