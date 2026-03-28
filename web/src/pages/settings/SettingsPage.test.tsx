import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import { Router } from '@solidjs/router';

// Mock the API client
vi.mock('../../api/client', () => ({
  api: {
    users: {
      me: vi.fn().mockResolvedValue({
        id: 'user-1',
        email: 'test@example.com',
        username: 'testuser',
        display_name: 'Test User',
        avatar_url: null,
        role: 'admin',
        totp_enabled: false,
        is_active: true,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-03-01T00:00:00Z',
      }),
      updateMe: vi.fn().mockResolvedValue({}),
    },
    notificationPrefs: {
      get: vi.fn().mockResolvedValue({
        id: 'np-1',
        user_id: 'user-1',
        email_enabled: true,
        in_app_enabled: true,
        pipeline_success: true,
        pipeline_failure: true,
        deployment_success: true,
        deployment_failure: true,
        approval_requested: true,
        approval_resolved: true,
        agent_offline: true,
        security_alerts: true,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      }),
      update: vi.fn().mockResolvedValue({}),
    },
    auth: {
      totpSetup: vi.fn().mockResolvedValue({ secret: 'TOTP_SECRET', qr_url: 'https://example.com/qr.png' }),
      totpVerify: vi.fn().mockResolvedValue({}),
    },
  },
  apiClient: {
    get: vi.fn().mockResolvedValue([]),
    post: vi.fn().mockResolvedValue({ token: 'ff_test_token', key: { id: 'key-1', name: 'Test Key', prefix: 'ff_' } }),
    put: vi.fn().mockResolvedValue({}),
    delete: vi.fn().mockResolvedValue({}),
  },
  ApiRequestError: class ApiRequestError extends Error {
    constructor(message: string, public status: number) {
      super(message);
    }
  },
}));

describe('SettingsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the settings page', async () => {
    const { default: SettingsPage } = await import('./SettingsPage');
    render(() => (
      <Router>
        <SettingsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('Settings')).toBeInTheDocument();
  });

  it('renders sidebar navigation items', async () => {
    const { default: SettingsPage } = await import('./SettingsPage');
    render(() => (
      <Router>
        <SettingsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('Profile')).toBeInTheDocument();
    expect(screen.getByText('API Keys')).toBeInTheDocument();
    expect(screen.getByText('Notifications')).toBeInTheDocument();
    expect(screen.getByText('Appearance')).toBeInTheDocument();
    expect(screen.getByText('Security')).toBeInTheDocument();
  });

  it('renders profile section by default', async () => {
    const { default: SettingsPage } = await import('./SettingsPage');
    render(() => (
      <Router>
        <SettingsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('Profile Information')).toBeInTheDocument();
  });

  it('renders profile form fields after loading', async () => {
    const { default: SettingsPage } = await import('./SettingsPage');
    render(() => (
      <Router>
        <SettingsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('Display Name')).toBeInTheDocument();
    expect(screen.getByText('Username')).toBeInTheDocument();
    expect(screen.getByText('Email')).toBeInTheDocument();
  });

  it('renders Save Changes button in profile', async () => {
    const { default: SettingsPage } = await import('./SettingsPage');
    render(() => (
      <Router>
        <SettingsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('Save Changes')).toBeInTheDocument();
  });

  it('renders Change Password section', async () => {
    const { default: SettingsPage } = await import('./SettingsPage');
    render(() => (
      <Router>
        <SettingsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    expect(screen.getByText('Change Password')).toBeInTheDocument();
  });

  it('switches to API Keys section', async () => {
    const { default: SettingsPage } = await import('./SettingsPage');
    render(() => (
      <Router>
        <SettingsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));

    const apiKeysNav = screen.getByText('API Keys');
    fireEvent.click(apiKeysNav);

    await new Promise(r => setTimeout(r, 100));
    // Should show the API keys section header or create button
    const createButtons = screen.getAllByText('Create Key');
    expect(createButtons.length).toBeGreaterThan(0);
  });

  it('switches to Appearance section', async () => {
    const { default: SettingsPage } = await import('./SettingsPage');
    render(() => (
      <Router>
        <SettingsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));

    const appearanceNav = screen.getByText('Appearance');
    fireEvent.click(appearanceNav);

    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('Theme')).toBeInTheDocument();
    expect(screen.getByText('Dark')).toBeInTheDocument();
    expect(screen.getByText('Light')).toBeInTheDocument();
    expect(screen.getByText('System')).toBeInTheDocument();
  });

  it('switches to Security section', async () => {
    const { default: SettingsPage } = await import('./SettingsPage');
    render(() => (
      <Router>
        <SettingsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));

    const securityNav = screen.getByText('Security');
    fireEvent.click(securityNav);

    await new Promise(r => setTimeout(r, 100));
    expect(screen.getByText('Two-Factor Authentication')).toBeInTheDocument();
    expect(screen.getByText('Danger Zone')).toBeInTheDocument();
    expect(screen.getByText('Delete Account')).toBeInTheDocument();
  });

  it('renders user initials when no avatar', async () => {
    const { default: SettingsPage } = await import('./SettingsPage');
    render(() => (
      <Router>
        <SettingsPage />
      </Router>
    ));
    await new Promise(r => setTimeout(r, 200));
    // With display_name "Test User", initials should be "TU"
    expect(screen.getByText('TU')).toBeInTheDocument();
  });
});
