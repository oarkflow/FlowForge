import type { Component } from 'solid-js';
import { onMount } from 'solid-js';
import { useNavigate, useSearchParams } from '@solidjs/router';
import { setTokens } from '../../api/client';
import { authStore } from '../../stores/auth';

const OAuthCallback: Component = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  onMount(async () => {
    const accessToken = searchParams.access_token as string | undefined;
    const refreshTokenParam = searchParams.refresh_token as string | undefined;
    const error = searchParams.error as string | undefined;

    if (error) {
      authStore.setError(error);
      navigate('/auth/login', { replace: true });
      return;
    }

    if (accessToken && refreshTokenParam) {
      setTokens({
        access_token: accessToken,
        refresh_token: refreshTokenParam,
        expires_in: 3600,
      });
      await authStore.fetchUser();
      navigate('/', { replace: true });
    } else {
      authStore.setError('OAuth callback missing tokens');
      navigate('/auth/login', { replace: true });
    }
  });

  return (
    <div class="flex items-center justify-center min-h-[300px]">
      <div class="flex flex-col items-center gap-3">
        <svg class="animate-spin h-8 w-8 text-indigo-500" viewBox="0 0 24 24" fill="none">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
        <p class="text-sm text-[var(--color-text-secondary)]">Completing sign in...</p>
      </div>
    </div>
  );
};

export default OAuthCallback;
