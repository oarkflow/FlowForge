import { describe, it, expect, beforeEach, vi } from 'vitest';
import { server } from '../test/mocks/server';
import { http, HttpResponse } from 'msw';

describe('API Client', () => {
  let apiClient: typeof import('./client').apiClient;
  let api: typeof import('./client').api;
  let setTokens: typeof import('./client').setTokens;
  let clearTokens: typeof import('./client').clearTokens;
  let ApiRequestError: typeof import('./client').ApiRequestError;

  beforeEach(async () => {
    // Clear localStorage before each test
    localStorage.clear();
    // Re-import to get fresh module state
    const module = await import('./client');
    apiClient = module.apiClient;
    api = module.api;
    setTokens = module.setTokens;
    clearTokens = module.clearTokens;
    ApiRequestError = module.ApiRequestError;
  });

  describe('apiClient', () => {
    it('makes GET requests', async () => {
      const health = await apiClient.get<{ status: string }>('/system/health');
      expect(health).toHaveProperty('status', 'healthy');
    });

    it('makes POST requests', async () => {
      const tokens = await apiClient.post<{ access_token: string }>('/auth/login', {
        email: 'test@flowforge.dev',
        password: 'password123',
      });
      expect(tokens).toHaveProperty('access_token');
    });

    it('throws ApiRequestError on failure', async () => {
      try {
        await apiClient.post('/auth/login', {
          email: 'bad@test.com',
          password: 'wrong',
        });
        expect.unreachable('Should have thrown');
      } catch (err) {
        expect(err).toBeInstanceOf(ApiRequestError);
        expect((err as any).status).toBe(401);
      }
    });

    it('handles 204 responses', async () => {
      const result = await apiClient.post('/auth/logout');
      expect(result).toBeUndefined();
    });
  });

  describe('token management', () => {
    it('setTokens stores tokens in localStorage', () => {
      setTokens({
        access_token: 'test-at',
        refresh_token: 'test-rt',
        expires_in: 3600,
      });
      expect(localStorage.getItem('ff_access_token')).toBe('test-at');
      expect(localStorage.getItem('ff_refresh_token')).toBe('test-rt');
    });

    it('clearTokens removes tokens from localStorage', () => {
      setTokens({
        access_token: 'test-at',
        refresh_token: 'test-rt',
        expires_in: 3600,
      });
      clearTokens();
      expect(localStorage.getItem('ff_access_token')).toBeNull();
      expect(localStorage.getItem('ff_refresh_token')).toBeNull();
    });
  });

  describe('api.projects', () => {
    it('list returns paginated projects', async () => {
      const result = await api.projects.list({ page: '1', per_page: '10' });
      expect(result).toHaveProperty('data');
      expect(result).toHaveProperty('total');
      expect(result).toHaveProperty('page');
    });

    it('get returns a project', async () => {
      const project = await api.projects.get('proj-1');
      expect(project).toHaveProperty('id', 'proj-1');
      expect(project).toHaveProperty('name', 'Frontend App');
    });

    it('create returns new project', async () => {
      const project = await api.projects.create({ name: 'My Project' });
      expect(project).toHaveProperty('id');
      expect(project).toHaveProperty('name', 'My Project');
    });

    it('get returns 404 for unknown project', async () => {
      try {
        await api.projects.get('unknown');
        expect.unreachable('Should have thrown');
      } catch (err) {
        expect(err).toBeInstanceOf(ApiRequestError);
        expect((err as any).status).toBe(404);
      }
    });
  });

  describe('api.agents', () => {
    it('list returns agents', async () => {
      const agents = await api.agents.list();
      expect(Array.isArray(agents)).toBe(true);
      expect(agents).toHaveLength(2);
    });

    it('create returns new agent with token', async () => {
      const result = await api.agents.create({ name: 'new-agent', executor: 'docker' });
      expect(result).toHaveProperty('token');
      expect(result).toHaveProperty('name', 'new-agent');
    });
  });

  describe('api.runs', () => {
    it('list returns runs', async () => {
      const result = await api.runs.list('proj-1', 'pipe-1');
      expect(result).toHaveProperty('data');
    });

    it('get returns run detail', async () => {
      const run = await api.runs.get('proj-1', 'pipe-1', 'run-1');
      expect(run).toHaveProperty('id', 'run-1');
      expect(run).toHaveProperty('stages');
      expect(run).toHaveProperty('jobs');
      expect(run).toHaveProperty('steps');
    });
  });

  describe('api.system', () => {
    it('health returns system health', async () => {
      const health = await api.system.health();
      expect(health).toHaveProperty('status', 'healthy');
      expect(health).toHaveProperty('version');
    });
  });

  describe('ApiRequestError', () => {
    it('has correct properties', () => {
      const err = new ApiRequestError({
        error: 'validation',
        message: 'Invalid data',
        status: 422,
        details: { email: ['required'] },
      });
      expect(err.message).toBe('Invalid data');
      expect(err.status).toBe(422);
      expect(err.details).toEqual({ email: ['required'] });
      expect(err.name).toBe('ApiRequestError');
    });
  });

  describe('refresh token flow', () => {
    it('retries request after 401 with refresh token', async () => {
      let callCount = 0;

      server.use(
        http.get('/api/v1/users/me', () => {
          callCount++;
          if (callCount === 1) {
            return HttpResponse.json(
              { error: 'auth', message: 'Token expired', status: 401 },
              { status: 401 },
            );
          }
          return HttpResponse.json({ id: '1', email: 'test@test.com', username: 'test' });
        }),
      );

      // Set tokens so refresh can happen
      setTokens({ access_token: 'old-token', refresh_token: 'valid-rt', expires_in: 3600 });

      const user = await api.users.me();
      expect(user).toHaveProperty('id', '1');
    });
  });
});
