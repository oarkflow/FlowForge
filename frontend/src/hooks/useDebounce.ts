import { createSignal, createEffect, onCleanup, Accessor } from 'solid-js';

/**
 * useDebounce - Returns a debounced version of the given signal.
 */
export function useDebounce<T>(value: Accessor<T>, delay: number): Accessor<T> {
  const [debounced, setDebounced] = createSignal<T>(value());
  let timer: ReturnType<typeof setTimeout>;

  createEffect(() => {
    const v = value();
    clearTimeout(timer);
    timer = setTimeout(() => setDebounced(() => v), delay);
  });

  onCleanup(() => clearTimeout(timer));

  return debounced;
}

/**
 * useDebouncedFn - Returns a debounced version of the given function.
 */
export function useDebouncedFn<T extends (...args: any[]) => any>(fn: T, delay: number): T {
  let timer: ReturnType<typeof setTimeout>;

  const debounced = ((...args: any[]) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn(...args), delay);
  }) as T;

  onCleanup(() => clearTimeout(timer));

  return debounced;
}
