import { createSignal, createMemo } from 'solid-js';
import { apiClient, setTokens, clearTokens } from '../api/client';
import type { User, AuthTokens, LoginRequest } from '../types';

// ---------------------------------------------------------------------------
// Signals
// ---------------------------------------------------------------------------
const [user, setUser] = createSignal<User | null>(null);
const [loading, setLoading] = createSignal(true);
const [error, setError] = createSignal<string | null>(null);

const isAuthenticated = createMemo(() => user() !== null);

// ---------------------------------------------------------------------------
// Actions
// ---------------------------------------------------------------------------
async function login(credentials: LoginRequest): Promise<void> {
  setError(null);
  setLoading(true);
  try {
    const tokens = await apiClient.post<AuthTokens>('/auth/login', credentials);
    setTokens(tokens);
    await fetchUser();
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Login failed';
    setError(message);
    throw err;
  } finally {
    setLoading(false);
  }
}

async function register(data: { email: string; username: string; password: string; display_name?: string }): Promise<void> {
  setError(null);
  setLoading(true);
  try {
    const tokens = await apiClient.post<AuthTokens>('/auth/register', data);
    setTokens(tokens);
    await fetchUser();
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Registration failed';
    setError(message);
    throw err;
  } finally {
    setLoading(false);
  }
}

async function fetchUser(): Promise<void> {
  try {
    const me = await apiClient.get<User>('/users/me');
    setUser(me);
  } catch {
    setUser(null);
  }
}

async function logout(): Promise<void> {
  try {
    await apiClient.post('/auth/logout');
  } catch {
    // Ignore errors on logout
  } finally {
    clearTokens();
    setUser(null);
  }
}

async function initialize(): Promise<void> {
  const token = localStorage.getItem('ff_access_token');
  if (token) {
    await fetchUser();
  }
  setLoading(false);
}

function oauthLogin(provider: 'github' | 'gitlab' | 'google'): void {
  window.location.href = `/api/v1/auth/oauth/${provider}`;
}

// ---------------------------------------------------------------------------
// Export store
// ---------------------------------------------------------------------------
export const authStore = {
  user,
  loading,
  error,
  isAuthenticated,
  login,
  register,
  logout,
  initialize,
  fetchUser,
  oauthLogin,
  setError,
};
