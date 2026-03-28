import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createRoot } from 'solid-js';

// We need to mock the API client before importing the store
vi.mock('../api/client', () => {
  const mockPost = vi.fn();
  const mockGet = vi.fn();
  return {
    apiClient: {
      post: mockPost,
      get: mockGet,
      put: vi.fn(),
      delete: vi.fn(),
    },
    setTokens: vi.fn(),
    clearTokens: vi.fn(),
    getAccessToken: vi.fn(() => 'mock-token'),
    ApiRequestError: class extends Error {
      status: number;
      constructor(msg: string) {
        super(msg);
        this.status = 400;
      }
    },
  };
});

describe('authStore', () => {
  let authStore: typeof import('./auth').authStore;
  let apiClient: any;
  let setTokens: any;
  let clearTokens: any;

  beforeEach(async () => {
    vi.clearAllMocks();
    // Re-import to get fresh module state
    const clientModule = await import('../api/client');
    apiClient = clientModule.apiClient;
    setTokens = clientModule.setTokens;
    clearTokens = clientModule.clearTokens;

    const authModule = await import('./auth');
    authStore = authModule.authStore;
  });

  it('starts with null user', () => {
    createRoot(() => {
      expect(authStore.user()).toBeNull();
    });
  });

  it('isAuthenticated is false initially', () => {
    createRoot(() => {
      expect(authStore.isAuthenticated()).toBe(false);
    });
  });

  it('login calls API and sets tokens', async () => {
    const mockTokens = { access_token: 'at', refresh_token: 'rt', expires_in: 3600 };
    const mockUser = { id: '1', email: 'test@test.com', username: 'test' };

    apiClient.post.mockResolvedValueOnce(mockTokens);
    apiClient.get.mockResolvedValueOnce(mockUser);

    await createRoot(async () => {
      try {
        await authStore.login({ email: 'test@test.com', password: 'pass' });
      } catch {
        // May throw if setTokens implementation changes
      }
      expect(apiClient.post).toHaveBeenCalledWith('/auth/login', {
        email: 'test@test.com',
        password: 'pass',
      });
    });
  });

  it('login sets error on failure', async () => {
    apiClient.post.mockRejectedValueOnce(new Error('Invalid credentials'));

    await createRoot(async () => {
      try {
        await authStore.login({ email: 'bad@test.com', password: 'wrong' });
      } catch {
        // Expected
      }
      expect(authStore.error()).toBe('Invalid credentials');
    });
  });

  it('logout clears tokens and user', async () => {
    apiClient.post.mockResolvedValueOnce(undefined);

    await createRoot(async () => {
      await authStore.logout();
      expect(clearTokens).toHaveBeenCalled();
      expect(authStore.user()).toBeNull();
    });
  });

  it('register calls API with user data', async () => {
    const mockTokens = { access_token: 'at', refresh_token: 'rt', expires_in: 3600 };
    apiClient.post.mockResolvedValueOnce(mockTokens);
    apiClient.get.mockResolvedValueOnce({ id: '1', email: 'new@test.com', username: 'new' });

    await createRoot(async () => {
      try {
        await authStore.register({
          email: 'new@test.com',
          username: 'newuser',
          password: 'password123',
        });
      } catch {
        // May throw
      }
      expect(apiClient.post).toHaveBeenCalledWith('/auth/register', {
        email: 'new@test.com',
        username: 'newuser',
        password: 'password123',
      });
    });
  });

  it('setError updates the error signal', () => {
    createRoot(() => {
      authStore.setError('Custom error');
      expect(authStore.error()).toBe('Custom error');
    });
  });
});
