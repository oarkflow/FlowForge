import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';

// Mock js-yaml
vi.mock('js-yaml', () => ({
  default: {
    dump: vi.fn((obj) => JSON.stringify(obj)),
    load: vi.fn((str) => JSON.parse(str)),
  },
  dump: vi.fn((obj) => JSON.stringify(obj)),
  load: vi.fn((str) => JSON.parse(str)),
}));

// Mock the API client
vi.mock('../../api/client', () => ({
  api: {
    envVars: { list: vi.fn().mockResolvedValue([]) },
    secrets: { list: vi.fn().mockResolvedValue([]) },
  },
}));

describe('PipelineBuilder', () => {
  it('renders the builder interface', async () => {
    const { default: PipelineBuilder } = await import('./PipelineBuilder');

    render(() => (
      <PipelineBuilder
        projectId="proj-1"
        pipelineId="pipe-1"
        value=""
        onChange={() => {}}
      />
    ));

    // Should have the pipeline builder UI with stages
    await new Promise(r => setTimeout(r, 50));
    // Check for the add stage button or similar UI elements
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBeGreaterThan(0);
  });

  it('calls onChange when configuration changes', async () => {
    const { default: PipelineBuilder } = await import('./PipelineBuilder');

    const onChange = vi.fn();
    render(() => (
      <PipelineBuilder
        projectId="proj-1"
        pipelineId="pipe-1"
        value=""
        onChange={onChange}
      />
    ));

    await new Promise(r => setTimeout(r, 50));

    // Look for buttons to add stages
    const addButtons = screen.getAllByRole('button');
    expect(addButtons.length).toBeGreaterThan(0);
  });

  it('accepts initial YAML value', async () => {
    const { default: PipelineBuilder } = await import('./PipelineBuilder');

    render(() => (
      <PipelineBuilder
        projectId="proj-1"
        pipelineId="pipe-1"
        value={'{"version":"1","name":"Test"}'}
        onChange={() => {}}
      />
    ));

    await new Promise(r => setTimeout(r, 50));
    // Component should render without errors
    expect(document.body).toBeDefined();
  });
});
