import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createRoot } from 'solid-js';
import { useLocalStorage } from './useLocalStorage';

describe('useLocalStorage', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
  });

  it('returns default value when nothing stored', () => {
    createRoot((dispose) => {
      const [value] = useLocalStorage('test-key', 'default');
      expect(value()).toBe('default');
      dispose();
    });
  });

  it('returns stored value when present', () => {
    localStorage.setItem('test-key', JSON.stringify('stored'));
    createRoot((dispose) => {
      const [value] = useLocalStorage('test-key', 'default');
      expect(value()).toBe('stored');
      dispose();
    });
  });

  it('sets value and persists to localStorage', () => {
    createRoot((dispose) => {
      const [value, setValue] = useLocalStorage('test-key', 'initial');
      setValue('updated');
      expect(value()).toBe('updated');
      expect(localStorage.setItem).toHaveBeenCalledWith('test-key', '"updated"');
      dispose();
    });
  });

  it('handles object values', () => {
    createRoot((dispose) => {
      const [value, setValue] = useLocalStorage('obj-key', { count: 0 });
      expect(value()).toEqual({ count: 0 });

      setValue({ count: 5 });
      expect(value()).toEqual({ count: 5 });
      expect(localStorage.setItem).toHaveBeenCalledWith('obj-key', '{"count":5}');
      dispose();
    });
  });

  it('handles array values', () => {
    createRoot((dispose) => {
      const [value, setValue] = useLocalStorage('arr-key', [1, 2, 3]);
      expect(value()).toEqual([1, 2, 3]);

      setValue([4, 5]);
      expect(value()).toEqual([4, 5]);
      dispose();
    });
  });

  it('handles boolean values', () => {
    createRoot((dispose) => {
      const [value, setValue] = useLocalStorage('bool-key', false);
      expect(value()).toBe(false);

      setValue(true);
      expect(value()).toBe(true);
      dispose();
    });
  });

  it('returns default on invalid JSON in storage', () => {
    (localStorage.getItem as any).mockReturnValueOnce('not-json');
    createRoot((dispose) => {
      const [value] = useLocalStorage('bad-key', 'fallback');
      expect(value()).toBe('fallback');
      dispose();
    });
  });

  it('handles number values', () => {
    createRoot((dispose) => {
      const [value, setValue] = useLocalStorage('num-key', 0);
      expect(value()).toBe(0);

      setValue(42);
      expect(value()).toBe(42);
      dispose();
    });
  });
});
