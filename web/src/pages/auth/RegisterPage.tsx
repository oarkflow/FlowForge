import type { Component } from 'solid-js';
import { createSignal, Show } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { authStore } from '../../stores/auth';
import Button from '../../components/ui/Button';
import Input from '../../components/ui/Input';

const RegisterPage: Component = () => {
  const navigate = useNavigate();
  const [email, setEmail] = createSignal('');
  const [username, setUsername] = createSignal('');
  const [password, setPassword] = createSignal('');
  const [confirmPassword, setConfirmPassword] = createSignal('');
  const [displayName, setDisplayName] = createSignal('');
  const [submitting, setSubmitting] = createSignal(false);
  const [validationError, setValidationError] = createSignal('');

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setValidationError('');
    authStore.setError(null);

    if (password() !== confirmPassword()) {
      setValidationError('Passwords do not match');
      return;
    }
    if (password().length < 8) {
      setValidationError('Password must be at least 8 characters');
      return;
    }

    setSubmitting(true);
    try {
      await authStore.register({
        email: email(),
        username: username(),
        password: password(),
        display_name: displayName() || undefined,
      });
      navigate('/', { replace: true });
    } catch {
      // Error is handled by authStore
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div>
      <div class="mb-8">
        <h2 class="text-2xl font-bold text-[var(--color-text-primary)] tracking-tight">
          Create an account
        </h2>
        <p class="text-sm text-[var(--color-text-secondary)] mt-1">
          Get started with FlowForge
        </p>
      </div>

      <form onSubmit={handleSubmit} class="flex flex-col gap-4">
        <Show when={authStore.error() || validationError()}>
          <div class="p-3 rounded-lg bg-[var(--color-error-bg)] border border-red-500/20 text-sm text-red-400">
            {authStore.error() || validationError()}
          </div>
        </Show>

        <Input
          label="Display Name"
          type="text"
          placeholder="John Doe"
          value={displayName()}
          onInput={(e) => setDisplayName(e.currentTarget.value)}
          autocomplete="name"
        />

        <Input
          label="Username"
          type="text"
          placeholder="johndoe"
          value={username()}
          onInput={(e) => setUsername(e.currentTarget.value)}
          required
          autocomplete="username"
        />

        <Input
          label="Email"
          type="email"
          placeholder="you@example.com"
          value={email()}
          onInput={(e) => setEmail(e.currentTarget.value)}
          required
          autocomplete="email"
        />

        <Input
          label="Password"
          type="password"
          placeholder="Minimum 8 characters"
          value={password()}
          onInput={(e) => setPassword(e.currentTarget.value)}
          required
          autocomplete="new-password"
        />

        <Input
          label="Confirm Password"
          type="password"
          placeholder="Re-enter your password"
          value={confirmPassword()}
          onInput={(e) => setConfirmPassword(e.currentTarget.value)}
          required
          autocomplete="new-password"
        />

        <Button type="submit" loading={submitting()} class="w-full mt-2">
          Create account
        </Button>
      </form>

      <p class="text-center text-sm text-[var(--color-text-tertiary)] mt-6">
        Already have an account?{' '}
        <a
          href="/auth/login"
          class="text-indigo-400 hover:text-indigo-300 transition-colors font-medium"
        >
          Sign in
        </a>
      </p>
    </div>
  );
};

export default RegisterPage;
