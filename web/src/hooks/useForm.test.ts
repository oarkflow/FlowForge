import { describe, it, expect, vi } from 'vitest';
import { createRoot } from 'solid-js';
import { useForm } from './useForm';

describe('useForm', () => {
  it('initializes with provided values', () => {
    createRoot((dispose) => {
      const { values } = useForm({ name: 'John', email: 'john@test.com' });
      expect(values.name).toBe('John');
      expect(values.email).toBe('john@test.com');
      dispose();
    });
  });

  it('setField updates a value', () => {
    createRoot((dispose) => {
      const { values, setField } = useForm({ name: '' });
      setField('name', 'Jane');
      expect(values.name).toBe('Jane');
      dispose();
    });
  });

  it('setField marks field as touched', () => {
    createRoot((dispose) => {
      const { touched, setField } = useForm({ name: '' });
      expect(touched.name).toBeFalsy();
      setField('name', 'Jane');
      expect(touched.name).toBe(true);
      dispose();
    });
  });

  it('setError sets a field error', () => {
    createRoot((dispose) => {
      const { errors, setError } = useForm({ name: '' });
      setError('name', 'Name is required');
      expect(errors.name).toBe('Name is required');
      dispose();
    });
  });

  it('clearError removes a field error', () => {
    createRoot((dispose) => {
      const { errors, setError, clearError } = useForm({ name: '' });
      setError('name', 'Required');
      expect(errors.name).toBe('Required');
      clearError('name');
      expect(errors.name).toBeUndefined();
      dispose();
    });
  });

  it('isValid returns true when no errors', () => {
    createRoot((dispose) => {
      const { isValid } = useForm({ name: '' });
      expect(isValid()).toBe(true);
      dispose();
    });
  });

  it('isValid returns false when errors exist', () => {
    createRoot((dispose) => {
      const { isValid, setError } = useForm({ name: '' });
      setError('name', 'Required');
      expect(isValid()).toBe(false);
      dispose();
    });
  });

  it('isDirty returns false initially', () => {
    createRoot((dispose) => {
      const { isDirty } = useForm({ name: 'John' });
      expect(isDirty()).toBe(false);
      dispose();
    });
  });

  it('isDirty returns true after field change', () => {
    createRoot((dispose) => {
      const { isDirty, setField } = useForm({ name: 'John' });
      setField('name', 'Jane');
      expect(isDirty()).toBe(true);
      dispose();
    });
  });

  it('reset restores initial values', () => {
    createRoot((dispose) => {
      const { values, setField, reset, isDirty, submitted } = useForm({
        name: 'Original',
      });
      setField('name', 'Changed');
      expect(values.name).toBe('Changed');

      reset();
      expect(values.name).toBe('Original');
      expect(isDirty()).toBe(false);
      expect(submitted()).toBe(false);
      dispose();
    });
  });

  it('handleSubmit prevents default and calls callback when valid', async () => {
    await createRoot(async (dispose) => {
      const onSubmit = vi.fn();
      const { handleSubmit, submitted } = useForm({ name: 'John' });

      const handler = handleSubmit(onSubmit);
      const event = { preventDefault: vi.fn() } as unknown as Event;
      await handler(event);

      expect(event.preventDefault).toHaveBeenCalled();
      expect(onSubmit).toHaveBeenCalledWith({ name: 'John' });
      expect(submitted()).toBe(true);
      dispose();
    });
  });

  it('handleSubmit does not call callback when invalid', async () => {
    await createRoot(async (dispose) => {
      const onSubmit = vi.fn();
      const { handleSubmit, setError } = useForm({ name: '' });

      setError('name', 'Required');
      const handler = handleSubmit(onSubmit);
      const event = { preventDefault: vi.fn() } as unknown as Event;
      await handler(event);

      expect(event.preventDefault).toHaveBeenCalled();
      expect(onSubmit).not.toHaveBeenCalled();
      dispose();
    });
  });
});
