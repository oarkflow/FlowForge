import { render } from '@solidjs/testing-library';
import { Router } from '@solidjs/router';
import type { Component, JSX } from 'solid-js';

/**
 * Render a component wrapped with Router provider.
 * Use this for components that depend on @solidjs/router.
 */
export function renderWithRouter(ui: () => JSX.Element) {
  return render(() => (
    <Router>
      {ui()}
    </Router>
  ));
}

/**
 * Render a component without any providers.
 * Use this for simple UI components.
 */
export { render } from '@solidjs/testing-library';

/**
 * Create a mock for useNavigate.
 */
export function createMockNavigate() {
  return vi.fn();
}

/**
 * Wait for async operations to settle in SolidJS.
 */
export async function waitForAsync(ms = 50): Promise<void> {
  await new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Create a mock project for testing.
 */
export function createMockProject(overrides?: Record<string, unknown>) {
  return {
    id: 'proj-test',
    name: 'Test Project',
    slug: 'test-project',
    description: 'A test project',
    visibility: 'private' as const,
    created_by: 'user-1',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Create a mock pipeline for testing.
 */
export function createMockPipeline(overrides?: Record<string, unknown>) {
  return {
    id: 'pipe-test',
    project_id: 'proj-test',
    name: 'Test Pipeline',
    description: 'A test pipeline',
    config_source: 'db' as const,
    config_content: 'version: "1"',
    config_version: 1,
    triggers: {},
    is_active: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}
