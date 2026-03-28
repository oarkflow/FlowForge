import { createSignal, Accessor } from 'solid-js';

/**
 * useLocalStorage - Persists a signal value in localStorage.
 */
export function useLocalStorage<T>(
  key: string,
  defaultValue: T
): [Accessor<T>, (value: T) => void] {
  const stored = localStorage.getItem(key);
  let initial: T = defaultValue;

  if (stored !== null) {
    try {
      initial = JSON.parse(stored);
    } catch {
      initial = defaultValue;
    }
  }

  const [value, setValue] = createSignal<T>(initial);

  const setAndStore = (newValue: T) => {
    setValue(() => newValue);
    try {
      localStorage.setItem(key, JSON.stringify(newValue));
    } catch {
      // Ignore storage errors (quota exceeded, etc.)
    }
  };

  return [value, setAndStore];
}
