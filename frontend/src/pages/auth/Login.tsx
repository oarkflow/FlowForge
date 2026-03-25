import type { Component } from 'solid-js';
import { createSignal, Show } from 'solid-js';
import { A } from '@solidjs/router';
import { authStore } from '../../stores/auth';
import Input from '../../components/ui/Input';
import Button from '../../components/ui/Button';

const Login: Component = () => {
  const [email, setEmail] = createSignal('');
  const [password, setPassword] = createSignal('');
  const [submitting, setSubmitting] = createSignal(false);

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    if (submitting()) return;

    setSubmitting(true);
    authStore.setError(null);

    try {
      await authStore.login({ email: email(), password: password() });
    } catch {
      // error is set in the store
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div class="animate-fade-in">
      <div class="mb-8">
        <h2 class="text-2xl font-bold text-[var(--color-text-primary)] tracking-tight">
          Sign in to FlowForge
        </h2>
        <p class="text-sm text-[var(--color-text-secondary)] mt-2">
          Enter your credentials to access your CI/CD dashboard.
        </p>
      </div>

      {/* OAuth buttons */}
      <div class="space-y-2.5 mb-6">
        <button
          type="button"
          onClick={() => authStore.oauthLogin('github')}
          class="w-full flex items-center justify-center gap-2.5 px-4 py-2.5 text-sm font-medium rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors cursor-pointer"
        >
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" />
          </svg>
          Continue with GitHub
        </button>

        <button
          type="button"
          onClick={() => authStore.oauthLogin('gitlab')}
          class="w-full flex items-center justify-center gap-2.5 px-4 py-2.5 text-sm font-medium rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors cursor-pointer"
        >
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
            <path d="M22.65 14.39L12 22.13 1.35 14.39a.84.84 0 01-.3-.94l1.22-3.78 2.44-7.51A.42.42 0 014.82 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.49h8.1l2.44-7.51A.42.42 0 0118.6 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.51L23 13.45a.84.84 0 01-.35.94z" />
          </svg>
          Continue with GitLab
        </button>

        <button
          type="button"
          onClick={() => authStore.oauthLogin('google')}
          class="w-full flex items-center justify-center gap-2.5 px-4 py-2.5 text-sm font-medium rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors cursor-pointer"
        >
          <svg class="w-5 h-5" viewBox="0 0 24 24">
            <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" />
            <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
            <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" />
            <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
          </svg>
          Continue with Google
        </button>
      </div>

      {/* Divider */}
      <div class="relative my-6">
        <div class="absolute inset-0 flex items-center">
          <div class="w-full border-t border-[var(--color-border-primary)]" />
        </div>
        <div class="relative flex justify-center text-xs">
          <span class="bg-[var(--color-bg-primary)] px-3 text-[var(--color-text-tertiary)]">
            or continue with email
          </span>
        </div>
      </div>

      {/* Error message */}
      <Show when={authStore.error()}>
        <div class="mb-4 px-4 py-3 rounded-lg bg-[var(--color-error-bg)] border border-red-500/20">
          <p class="text-sm text-red-400">{authStore.error()}</p>
        </div>
      </Show>

      {/* Login form */}
      <form onSubmit={handleSubmit} class="space-y-4">
        <Input
          label="Email"
          type="email"
          placeholder="you@example.com"
          value={email()}
          onInput={(e) => setEmail(e.currentTarget.value)}
          required
          autocomplete="email"
          icon={
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
              <path d="M3 4a2 2 0 00-2 2v1.161l8.441 4.221a1.25 1.25 0 001.118 0L19 7.162V6a2 2 0 00-2-2H3z" />
              <path d="M19 8.839l-7.77 3.885a2.75 2.75 0 01-2.46 0L1 8.839V14a2 2 0 002 2h14a2 2 0 002-2V8.839z" />
            </svg>
          }
        />

        <Input
          label="Password"
          type="password"
          placeholder="Enter your password"
          value={password()}
          onInput={(e) => setPassword(e.currentTarget.value)}
          required
          autocomplete="current-password"
          icon={
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M10 1a4.5 4.5 0 00-4.5 4.5V9H5a2 2 0 00-2 2v6a2 2 0 002 2h10a2 2 0 002-2v-6a2 2 0 00-2-2h-.5V5.5A4.5 4.5 0 0010 1zm3 8V5.5a3 3 0 10-6 0V9h6z" clip-rule="evenodd" />
            </svg>
          }
        />

        <div class="flex items-center justify-between">
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              class="w-4 h-4 rounded border-[var(--color-border-primary)] bg-[var(--color-bg-secondary)] text-indigo-600 focus:ring-indigo-500/40 focus:ring-offset-0"
            />
            <span class="text-sm text-[var(--color-text-secondary)]">Remember me</span>
          </label>
          <a href="#" class="text-sm text-indigo-400 hover:text-indigo-300 transition-colors">
            Forgot password?
          </a>
        </div>

        <Button
          type="submit"
          variant="primary"
          loading={submitting()}
          class="w-full"
        >
          Sign in
        </Button>
      </form>

      <p class="mt-6 text-center text-sm text-[var(--color-text-tertiary)]">
        Don't have an account?{' '}
        <a href="#" class="text-indigo-400 hover:text-indigo-300 transition-colors font-medium">
          Create one
        </a>
      </p>
    </div>
  );
};

export default Login;
