import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createRoot } from 'solid-js';

vi.mock('../api/client', () => ({
  api: {
    pipelines: {
      list: vi.fn(),
      get: vi.fn(),
      trigger: vi.fn(),
    },
  },
}));

describe('pipelineStore', () => {
  let pipelineStore: typeof import('./pipelineStore').pipelineStore;
  let api: any;

  beforeEach(async () => {
    vi.clearAllMocks();
    const clientModule = await import('../api/client');
    api = (clientModule as any).api;
    const storeModule = await import('./pipelineStore');
    pipelineStore = storeModule.pipelineStore;
  });

  it('starts with empty pipelines', () => {
    createRoot(() => {
      expect(pipelineStore.pipelines()).toEqual([]);
    });
  });

  it('starts with null currentPipeline', () => {
    createRoot(() => {
      expect(pipelineStore.currentPipeline()).toBeNull();
    });
  });

  it('fetchPipelines loads pipelines from API', async () => {
    const mockPipelines = [{ id: '1', name: 'Pipeline 1' }];
    api.pipelines.list.mockResolvedValueOnce(mockPipelines);

    await createRoot(async () => {
      await pipelineStore.fetchPipelines('proj-1');
      expect(api.pipelines.list).toHaveBeenCalledWith('proj-1');
    });
  });

  it('fetchPipelines sets error on failure', async () => {
    api.pipelines.list.mockRejectedValueOnce(new Error('API error'));

    await createRoot(async () => {
      try {
        await pipelineStore.fetchPipelines('proj-1');
      } catch {
        // Expected
      }
      expect(pipelineStore.error()).toBe('API error');
    });
  });

  it('fetchPipeline loads a single pipeline', async () => {
    const mockPipeline = { id: '1', name: 'Pipeline 1', project_id: 'proj-1' };
    api.pipelines.get.mockResolvedValueOnce(mockPipeline);

    await createRoot(async () => {
      const result = await pipelineStore.fetchPipeline('proj-1', '1');
      expect(api.pipelines.get).toHaveBeenCalledWith('proj-1', '1');
      expect(result).toEqual(mockPipeline);
    });
  });

  it('triggerPipeline calls API', async () => {
    const mockRun = { id: 'run-1', status: 'queued' };
    api.pipelines.trigger.mockResolvedValueOnce(mockRun);

    await createRoot(async () => {
      const result = await pipelineStore.triggerPipeline('proj-1', 'pipe-1', { branch: 'main' });
      expect(api.pipelines.trigger).toHaveBeenCalledWith('proj-1', 'pipe-1', { branch: 'main' });
      expect(result).toEqual(mockRun);
    });
  });

  it('triggerPipeline defaults to empty params', async () => {
    api.pipelines.trigger.mockResolvedValueOnce({ id: 'run-1' });

    await createRoot(async () => {
      await pipelineStore.triggerPipeline('proj-1', 'pipe-1');
      expect(api.pipelines.trigger).toHaveBeenCalledWith('proj-1', 'pipe-1', {});
    });
  });

  it('loading is true during fetch', async () => {
    let resolvePromise: any;
    api.pipelines.list.mockReturnValueOnce(
      new Promise(r => { resolvePromise = r; }),
    );

    await createRoot(async () => {
      const fetchPromise = pipelineStore.fetchPipelines('proj-1');
      expect(pipelineStore.loading()).toBe(true);
      resolvePromise([]);
      await fetchPromise;
      expect(pipelineStore.loading()).toBe(false);
    });
  });
});
