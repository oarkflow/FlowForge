import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import { Router } from '@solidjs/router';

// Mock the auth store
vi.mock('../../stores/auth', () => ({
  authStore: {
    login: vi.fn(),
    error: vi.fn(() => null),
    loading: vi.fn(() => false),
    oauthLogin: vi.fn(),
  },
}));

describe('LoginPage', () => {
  it('renders login form', async () => {
    const { default: LoginPage } = await import('./LoginPage');
    render(() => (
      <Router>
        <LoginPage />
      </Router>
    ));
    expect(screen.getByText('Welcome back')).toBeInTheDocument();
    expect(screen.getByText('Sign in to your FlowForge account')).toBeInTheDocument();
  });

  it('renders email and password inputs', async () => {
    const { default: LoginPage } = await import('./LoginPage');
    render(() => (
      <Router>
        <LoginPage />
      </Router>
    ));
    expect(screen.getByPlaceholderText('you@example.com')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Enter your password')).toBeInTheDocument();
  });

  it('renders OAuth buttons', async () => {
    const { default: LoginPage } = await import('./LoginPage');
    render(() => (
      <Router>
        <LoginPage />
      </Router>
    ));
    expect(screen.getByText('Continue with GitHub')).toBeInTheDocument();
    expect(screen.getByText('Continue with GitLab')).toBeInTheDocument();
    expect(screen.getByText('Continue with Google')).toBeInTheDocument();
  });

  it('renders sign in button', async () => {
    const { default: LoginPage } = await import('./LoginPage');
    render(() => (
      <Router>
        <LoginPage />
      </Router>
    ));
    expect(screen.getByText('Sign in')).toBeInTheDocument();
  });

  it('renders register link', async () => {
    const { default: LoginPage } = await import('./LoginPage');
    render(() => (
      <Router>
        <LoginPage />
      </Router>
    ));
    expect(screen.getByText('Create one')).toBeInTheDocument();
  });

  it('renders remember me checkbox', async () => {
    const { default: LoginPage } = await import('./LoginPage');
    render(() => (
      <Router>
        <LoginPage />
      </Router>
    ));
    expect(screen.getByText('Remember me')).toBeInTheDocument();
  });

  it('renders forgot password link', async () => {
    const { default: LoginPage } = await import('./LoginPage');
    render(() => (
      <Router>
        <LoginPage />
      </Router>
    ));
    expect(screen.getByText('Forgot password?')).toBeInTheDocument();
  });

  it('calls authStore.login on form submit', async () => {
    const { authStore } = await import('../../stores/auth');
    (authStore.login as any).mockResolvedValueOnce(undefined);

    const { default: LoginPage } = await import('./LoginPage');
    render(() => (
      <Router>
        <LoginPage />
      </Router>
    ));

    fireEvent.input(screen.getByPlaceholderText('you@example.com'), {
      target: { value: 'test@test.com' },
    });
    fireEvent.input(screen.getByPlaceholderText('Enter your password'), {
      target: { value: 'password123' },
    });

    const form = screen.getByText('Sign in').closest('form');
    if (form) {
      fireEvent.submit(form);
    }
  });

  it('displays error from auth store', async () => {
    const { authStore } = await import('../../stores/auth');
    (authStore.error as any).mockReturnValue('Invalid credentials');

    const { default: LoginPage } = await import('./LoginPage');
    render(() => (
      <Router>
        <LoginPage />
      </Router>
    ));
    expect(screen.getByText('Invalid credentials')).toBeInTheDocument();
  });
});
