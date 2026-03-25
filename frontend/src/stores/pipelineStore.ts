import { createSignal } from 'solid-js';
import { api } from '../api/client';
import type { Pipeline } from '../types';

const [pipelines, setPipelines] = createSignal<Pipeline[]>([]);
const [currentPipeline, setCurrentPipeline] = createSignal<Pipeline | null>(null);
const [loading, setLoading] = createSignal(false);
const [error, setError] = createSignal<string | null>(null);

const fetchPipelines = async (projectId: string) => {
  setLoading(true);
  setError(null);
  try {
    const data = await api.pipelines.list(projectId);
    setPipelines(Array.isArray(data) ? data : []);
    return data;
  } catch (e) {
    setError(e instanceof Error ? e.message : 'Failed to fetch pipelines');
    throw e;
  } finally {
    setLoading(false);
  }
};

const fetchPipeline = async (projectId: string, pipelineId: string) => {
  setLoading(true);
  setError(null);
  try {
    const pipeline = await api.pipelines.get(projectId, pipelineId);
    setCurrentPipeline(pipeline);
    return pipeline;
  } catch (e) {
    setError(e instanceof Error ? e.message : 'Failed to fetch pipeline');
    throw e;
  } finally {
    setLoading(false);
  }
};

const triggerPipeline = async (projectId: string, pipelineId: string, params?: Record<string, unknown>) => {
  return api.pipelines.trigger(projectId, pipelineId, params || {});
};

export const pipelineStore = {
  pipelines,
  currentPipeline,
  loading,
  error,
  fetchPipelines,
  fetchPipeline,
  triggerPipeline,
  setCurrentPipeline,
};
