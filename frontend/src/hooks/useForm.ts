import { createSignal, createMemo, Accessor } from 'solid-js';
import { createStore, SetStoreFunction } from 'solid-js/store';

/**
 * useForm - Form state management hook.
 */
export function useForm<T extends Record<string, any>>(initialValues: T) {
  const [values, setValues] = createStore<T>({ ...initialValues });
  const [errors, setErrors] = createStore<Partial<Record<keyof T, string>>>({});
  const [touched, setTouched] = createStore<Partial<Record<keyof T, boolean>>>({});
  const [submitted, setSubmitted] = createSignal(false);

  const setField = (name: keyof T, value: any) => {
    (setValues as SetStoreFunction<any>)(name as string, value);
    (setTouched as SetStoreFunction<any>)(name as string, true);
  };

  const setError = (name: keyof T, error: string) => {
    (setErrors as SetStoreFunction<any>)(name as string, error);
  };

  const clearError = (name: keyof T) => {
    (setErrors as SetStoreFunction<any>)(name as string, undefined);
  };

  const reset = () => {
    Object.keys(initialValues).forEach((key) => {
      (setValues as SetStoreFunction<any>)(key, (initialValues as any)[key]);
      (setErrors as SetStoreFunction<any>)(key, undefined);
      (setTouched as SetStoreFunction<any>)(key, false);
    });
    setSubmitted(false);
  };

  const isValid = createMemo(() => {
    return Object.values(errors).every((e) => !e);
  });

  const isDirty = createMemo(() => {
    return Object.keys(values).some(
      (key) => (values as any)[key] !== (initialValues as any)[key]
    );
  });

  const handleSubmit = (onSubmit: (values: T) => void | Promise<void>) => {
    return async (e: Event) => {
      e.preventDefault();
      setSubmitted(true);
      if (isValid()) {
        await onSubmit({ ...values });
      }
    };
  };

  return {
    values,
    errors,
    touched,
    submitted,
    setField,
    setError,
    clearError,
    reset,
    isValid,
    isDirty,
    handleSubmit,
  };
}
