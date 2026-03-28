import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createRoot, createSignal } from 'solid-js';
import { useDebouncedFn } from './useDebounce';

describe('useDebouncedFn', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('delays function execution', () => {
    createRoot((dispose) => {
      const fn = vi.fn();
      const debounced = useDebouncedFn(fn, 300);

      debounced();
      expect(fn).not.toHaveBeenCalled();

      vi.advanceTimersByTime(300);
      expect(fn).toHaveBeenCalledOnce();
      dispose();
    });
  });

  it('resets timer on subsequent calls', () => {
    createRoot((dispose) => {
      const fn = vi.fn();
      const debounced = useDebouncedFn(fn, 300);

      debounced();
      vi.advanceTimersByTime(200);
      debounced(); // Reset
      vi.advanceTimersByTime(200);
      expect(fn).not.toHaveBeenCalled();

      vi.advanceTimersByTime(100);
      expect(fn).toHaveBeenCalledOnce();
      dispose();
    });
  });

  it('passes arguments to the debounced function', () => {
    createRoot((dispose) => {
      const fn = vi.fn();
      const debounced = useDebouncedFn(fn, 100);

      debounced('hello', 42);
      vi.advanceTimersByTime(100);
      expect(fn).toHaveBeenCalledWith('hello', 42);
      dispose();
    });
  });

  it('only calls with last arguments when called multiple times', () => {
    createRoot((dispose) => {
      const fn = vi.fn();
      const debounced = useDebouncedFn(fn, 200);

      debounced('first');
      debounced('second');
      debounced('third');

      vi.advanceTimersByTime(200);
      expect(fn).toHaveBeenCalledOnce();
      expect(fn).toHaveBeenCalledWith('third');
      dispose();
    });
  });
});

// useDebounce signal test requires reactive context
describe('useDebounce (signal)', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('debounces signal value changes', async () => {
    const { useDebounce } = await import('./useDebounce');

    createRoot((dispose) => {
      const [value, setValue] = createSignal('initial');
      const debounced = useDebounce(value, 300);

      expect(debounced()).toBe('initial');

      setValue('updated');
      // Should still be initial until timer fires
      expect(debounced()).toBe('initial');

      vi.advanceTimersByTime(300);
      expect(debounced()).toBe('updated');
      dispose();
    });
  });
});
