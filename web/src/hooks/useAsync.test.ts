import { describe, it, expect, vi } from 'vitest';
import { createRoot } from 'solid-js';
import { useAsync } from './useAsync';

describe('useAsync', () => {
  it('starts with undefined data', () => {
    createRoot((dispose) => {
      const { data } = useAsync(async () => 'test', { immediate: false });
      expect(data()).toBeUndefined();
      dispose();
    });
  });

  it('starts with loading false when immediate is false', () => {
    createRoot((dispose) => {
      const { loading } = useAsync(async () => 'test', { immediate: false });
      expect(loading()).toBe(false);
      dispose();
    });
  });

  it('starts with null error', () => {
    createRoot((dispose) => {
      const { error } = useAsync(async () => 'test', { immediate: false });
      expect(error()).toBeNull();
      dispose();
    });
  });

  it('execute fetches data', async () => {
    await createRoot(async (dispose) => {
      const { data, execute, loading } = useAsync(
        async () => 'fetched data',
        { immediate: false },
      );
      await execute();
      expect(data()).toBe('fetched data');
      expect(loading()).toBe(false);
      dispose();
    });
  });

  it('sets error on failure', async () => {
    await createRoot(async (dispose) => {
      const { error, execute } = useAsync(
        async () => { throw new Error('Failed'); },
        { immediate: false },
      );
      await execute();
      expect(error()).toBeInstanceOf(Error);
      expect(error()?.message).toBe('Failed');
      dispose();
    });
  });

  it('converts non-Error throws to Error', async () => {
    await createRoot(async (dispose) => {
      const { error, execute } = useAsync(
        async () => { throw 'string error'; },
        { immediate: false },
      );
      await execute();
      expect(error()).toBeInstanceOf(Error);
      expect(error()?.message).toBe('string error');
      dispose();
    });
  });

  it('refetch is the same as execute', () => {
    createRoot((dispose) => {
      const { execute, refetch } = useAsync(async () => 'test', { immediate: false });
      expect(refetch).toBe(execute);
      dispose();
    });
  });

  it('loading is true during execution', async () => {
    await createRoot(async (dispose) => {
      let resolvePromise: (value: string) => void;
      const asyncFn = () => new Promise<string>((r) => { resolvePromise = r; });

      const { loading, execute } = useAsync(asyncFn, { immediate: false });
      expect(loading()).toBe(false);

      const promise = execute();
      expect(loading()).toBe(true);

      resolvePromise!('done');
      await promise;
      expect(loading()).toBe(false);
      dispose();
    });
  });

  it('executes immediately by default', async () => {
    await createRoot(async (dispose) => {
      const fn = vi.fn().mockResolvedValue('result');
      useAsync(fn);
      // Wait for the async function to be called
      await new Promise(r => setTimeout(r, 10));
      expect(fn).toHaveBeenCalledOnce();
      dispose();
    });
  });

  it('clears error on re-execute', async () => {
    await createRoot(async (dispose) => {
      let shouldFail = true;
      const { error, execute } = useAsync(
        async () => {
          if (shouldFail) throw new Error('Failed');
          return 'ok';
        },
        { immediate: false },
      );

      await execute();
      expect(error()).not.toBeNull();

      shouldFail = false;
      await execute();
      expect(error()).toBeNull();
      dispose();
    });
  });
});
