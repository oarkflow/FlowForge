import type { Component } from 'solid-js';
import { createSignal, Show } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { authStore } from '../../stores/auth';
import Button from '../../components/ui/Button';
import Input from '../../components/ui/Input';

const LoginPage: Component = () => {
  const navigate = useNavigate();
  const [email, setEmail] = createSignal('');
  const [password, setPassword] = createSignal('');
  const [submitting, setSubmitting] = createSignal(false);

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      await authStore.login({ email: email(), password: password() });
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
          Welcome back
        </h2>
        <p class="text-sm text-[var(--color-text-secondary)] mt-1">
          Sign in to your FlowForge account
        </p>
      </div>

      {/* OAuth buttons */}
      <div class="flex flex-col gap-3 mb-6">
        <button
          onClick={() => authStore.oauthLogin('github')}
          class="flex items-center justify-center gap-3 w-full px-4 py-2.5 text-sm font-medium rounded-lg border border-[var(--color-border-primary)] bg-[var(--color-bg-secondary)] text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
        >
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
          </svg>
          Continue with GitHub
        </button>

        <button
          onClick={() => authStore.oauthLogin('gitlab')}
          class="flex items-center justify-center gap-3 w-full px-4 py-2.5 text-sm font-medium rounded-lg border border-[var(--color-border-primary)] bg-[var(--color-bg-secondary)] text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
        >
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
            <path d="M22.65 14.39L12 22.13 1.35 14.39a.84.84 0 01-.3-.94l1.22-3.78 2.44-7.51A.42.42 0 014.82 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.49h8.1l2.44-7.51A.42.42 0 0118.6 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.51L23 13.45a.84.84 0 01-.35.94z" />
          </svg>
          Continue with GitLab
        </button>

        <button
          onClick={() => authStore.oauthLogin('google')}
          class="flex items-center justify-center gap-3 w-full px-4 py-2.5 text-sm font-medium rounded-lg border border-[var(--color-border-primary)] bg-[var(--color-bg-secondary)] text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
        >
          <svg class="w-5 h-5" viewBox="0 0 24 24">
            <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4" />
            <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853" />
            <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05" />
            <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335" />
          </svg>
          Continue with Google
        </button>
      </div>

      {/* Divider */}
      <div class="relative mb-6">
        <div class="absolute inset-0 flex items-center">
          <div class="w-full border-t border-[var(--color-border-primary)]" />
        </div>
        <div class="relative flex justify-center text-xs uppercase">
          <span class="bg-[var(--color-bg-primary)] px-3 text-[var(--color-text-tertiary)]">
            or continue with email
          </span>
        </div>
      </div>

      {/* Email/password form */}
      <form onSubmit={handleSubmit} class="flex flex-col gap-4">
        <Show when={authStore.error()}>
          <div class="p-3 rounded-lg bg-[var(--color-error-bg)] border border-red-500/20 text-sm text-red-400">
            {authStore.error()}
          </div>
        </Show>

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
          placeholder="Enter your password"
          value={password()}
          onInput={(e) => setPassword(e.currentTarget.value)}
          required
          autocomplete="current-password"
        />

        <div class="flex items-center justify-between text-sm">
          <label class="flex items-center gap-2 text-[var(--color-text-secondary)] cursor-pointer">
            <input
              type="checkbox"
              class="rounded border-[var(--color-border-primary)] bg-[var(--color-bg-secondary)] text-indigo-600 focus:ring-indigo-500/40"
            />
            Remember me
          </label>
          <a
            href="/auth/forgot-password"
            class="text-indigo-400 hover:text-indigo-300 transition-colors"
          >
            Forgot password?
          </a>
        </div>

        <Button type="submit" loading={submitting()} class="w-full mt-2">
          Sign in
        </Button>
      </form>

      <p class="text-center text-sm text-[var(--color-text-tertiary)] mt-6">
        Don't have an account?{' '}
        <a
          href="/auth/register"
          class="text-indigo-400 hover:text-indigo-300 transition-colors font-medium"
        >
          Create one
        </a>
      </p>
    </div>
  );
};

export default LoginPage;
