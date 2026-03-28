import { createSignal, onCleanup, Accessor } from 'solid-js';

/**
 * useAsync - Async data fetching hook with loading/error state.
 */
export function useAsync<T>(
  fn: () => Promise<T>,
  options?: { immediate?: boolean }
): {
  data: Accessor<T | undefined>;
  loading: Accessor<boolean>;
  error: Accessor<Error | null>;
  execute: () => Promise<void>;
  refetch: () => Promise<void>;
} {
  const [data, setData] = createSignal<T | undefined>(undefined);
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal<Error | null>(null);

  const execute = async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await fn();
      setData(() => result);
    } catch (e) {
      setError(e instanceof Error ? e : new Error(String(e)));
    } finally {
      setLoading(false);
    }
  };

  if (options?.immediate !== false) {
    execute();
  }

  return { data, loading, error, execute, refetch: execute };
}
