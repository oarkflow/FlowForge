import { describe, it, expect } from 'vitest';
import { render, screen } from '@solidjs/testing-library';
import Badge from './Badge';

describe('Badge', () => {
  it('renders children text', () => {
    render(() => <Badge>Active</Badge>);
    expect(screen.getByText('Active')).toBeInTheDocument();
  });

  it('renders as a span', () => {
    render(() => <Badge>Test</Badge>);
    const badge = screen.getByText('Test');
    expect(badge.tagName).toBe('SPAN');
  });

  it('applies default variant styles', () => {
    render(() => <Badge>Default</Badge>);
    const badge = screen.getByText('Default');
    expect(badge.className).toContain('rounded-full');
  });

  it('applies success variant styles', () => {
    render(() => <Badge variant="success">Success</Badge>);
    const badge = screen.getByText('Success');
    expect(badge.className).toContain('text-emerald-400');
  });

  it('applies error variant styles', () => {
    render(() => <Badge variant="error">Error</Badge>);
    const badge = screen.getByText('Error');
    expect(badge.className).toContain('text-red-400');
  });

  it('applies warning variant styles', () => {
    render(() => <Badge variant="warning">Warning</Badge>);
    const badge = screen.getByText('Warning');
    expect(badge.className).toContain('text-amber-400');
  });

  it('applies info variant styles', () => {
    render(() => <Badge variant="info">Info</Badge>);
    const badge = screen.getByText('Info');
    expect(badge.className).toContain('text-blue-400');
  });

  it('applies running variant styles', () => {
    render(() => <Badge variant="running">Running</Badge>);
    const badge = screen.getByText('Running');
    expect(badge.className).toContain('text-violet-400');
  });

  it('renders dot when dot prop is true', () => {
    render(() => <Badge dot>With Dot</Badge>);
    const badge = screen.getByText('With Dot');
    const dot = badge.querySelector('.rounded-full');
    expect(dot).not.toBeNull();
  });

  it('does not render dot by default', () => {
    render(() => <Badge>No Dot</Badge>);
    const badge = screen.getByText('No Dot');
    // The badge itself is rounded-full, but there should be no dot child
    const children = badge.querySelectorAll('.w-1\\.5');
    expect(children).toHaveLength(0);
  });

  it('applies sm size by default', () => {
    render(() => <Badge>Small</Badge>);
    const badge = screen.getByText('Small');
    expect(badge.className).toContain('text-xs');
  });

  it('applies md size', () => {
    render(() => <Badge size="md">Medium</Badge>);
    const badge = screen.getByText('Medium');
    expect(badge.className).toContain('text-sm');
  });

  it('applies custom class', () => {
    render(() => <Badge class="my-badge">Custom</Badge>);
    const badge = screen.getByText('Custom');
    expect(badge.className).toContain('my-badge');
  });
});
