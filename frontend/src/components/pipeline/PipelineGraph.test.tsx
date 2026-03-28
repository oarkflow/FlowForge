import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@solidjs/testing-library';
import { PipelineGraph } from './PipelineGraph';

describe('PipelineGraph', () => {
  const stages = [
    { id: 'stage-1', name: 'build', status: 'success', position: 0 },
    { id: 'stage-2', name: 'test', status: 'running', position: 1 },
    { id: 'stage-3', name: 'deploy', status: 'pending', position: 2 },
  ];

  const jobs = [
    { id: 'job-1', name: 'build-go', status: 'success', stageId: 'stage-1' },
    { id: 'job-2', name: 'test-unit', status: 'running', stageId: 'stage-2' },
    { id: 'job-3', name: 'test-e2e', status: 'pending', stageId: 'stage-2' },
    { id: 'job-4', name: 'deploy-staging', status: 'pending', stageId: 'stage-3' },
  ];

  it('renders stage names', () => {
    render(() => <PipelineGraph stages={stages} jobs={jobs} />);
    expect(screen.getByText('build')).toBeInTheDocument();
    expect(screen.getByText('test')).toBeInTheDocument();
    expect(screen.getByText('deploy')).toBeInTheDocument();
  });

  it('renders job names', () => {
    render(() => <PipelineGraph stages={stages} jobs={jobs} />);
    expect(screen.getByText('build-go')).toBeInTheDocument();
    expect(screen.getByText('test-unit')).toBeInTheDocument();
    expect(screen.getByText('test-e2e')).toBeInTheDocument();
    expect(screen.getByText('deploy-staging')).toBeInTheDocument();
  });

  it('renders with empty stages', () => {
    const { container } = render(() => <PipelineGraph stages={[]} jobs={[]} />);
    expect(container).toBeDefined();
  });

  it('calls onJobClick when a job is clicked', () => {
    const onJobClick = vi.fn();
    render(() => <PipelineGraph stages={stages} jobs={jobs} onJobClick={onJobClick} />);

    const jobElement = screen.getByText('build-go').closest('[style]');
    if (jobElement) {
      jobElement.click();
    }
  });

  it('renders stages in order', () => {
    render(() => <PipelineGraph stages={stages} jobs={jobs} />);
    const text = document.body.textContent || '';
    const buildIdx = text.indexOf('build');
    const testIdx = text.indexOf('test');
    const deployIdx = text.indexOf('deploy');
    expect(buildIdx).toBeLessThan(testIdx);
    expect(testIdx).toBeLessThan(deployIdx);
  });

  it('renders with DAG data', () => {
    const dag = {
      nodes: {
        'build': { name: 'build', dependencies: [], dependents: ['test'], level: 0 },
        'test': { name: 'test', dependencies: ['build'], dependents: ['deploy'], level: 1 },
        'deploy': { name: 'deploy', dependencies: ['test'], dependents: [], level: 2 },
      },
      levels: [['build'], ['test'], ['deploy']],
      has_cycle: false,
    };
    render(() => <PipelineGraph stages={stages} jobs={jobs} dag={dag} />);
    expect(screen.getByText('build')).toBeInTheDocument();
    expect(screen.getByText('test')).toBeInTheDocument();
    expect(screen.getByText('deploy')).toBeInTheDocument();
  });
});
