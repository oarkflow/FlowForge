import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import Button from './Button';

describe('Button', () => {
  it('renders children text', () => {
    render(() => <Button>Click me</Button>);
    expect(screen.getByText('Click me')).toBeInTheDocument();
  });

  it('renders as a button element', () => {
    render(() => <Button>Test</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('handles click events', async () => {
    const onClick = vi.fn();
    render(() => <Button onClick={onClick}>Click</Button>);
    fireEvent.click(screen.getByRole('button'));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it('is disabled when disabled prop is true', () => {
    render(() => <Button disabled>Disabled</Button>);
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('is disabled when loading is true', () => {
    render(() => <Button loading>Loading</Button>);
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('shows spinner when loading', () => {
    render(() => <Button loading>Loading</Button>);
    const button = screen.getByRole('button');
    const spinner = button.querySelector('.animate-spin');
    expect(spinner).not.toBeNull();
  });

  it('does not show spinner when not loading', () => {
    render(() => <Button>Normal</Button>);
    const button = screen.getByRole('button');
    const spinner = button.querySelector('.animate-spin');
    expect(spinner).toBeNull();
  });

  it('renders with icon', () => {
    render(() => (
      <Button icon={<span data-testid="icon">I</span>}>With Icon</Button>
    ));
    expect(screen.getByTestId('icon')).toBeInTheDocument();
    expect(screen.getByText('With Icon')).toBeInTheDocument();
  });

  it('hides icon when loading', () => {
    render(() => (
      <Button loading icon={<span data-testid="icon">I</span>}>Loading</Button>
    ));
    expect(screen.queryByTestId('icon')).not.toBeInTheDocument();
  });

  it('applies primary variant styles by default', () => {
    render(() => <Button>Primary</Button>);
    const button = screen.getByRole('button');
    expect(button.className).toContain('bg-indigo-600');
  });

  it('applies secondary variant styles', () => {
    render(() => <Button variant="secondary">Secondary</Button>);
    const button = screen.getByRole('button');
    expect(button.className).not.toContain('bg-indigo-600');
  });

  it('applies danger variant styles', () => {
    render(() => <Button variant="danger">Danger</Button>);
    const button = screen.getByRole('button');
    expect(button.className).toContain('bg-red-600');
  });

  it('applies ghost variant styles', () => {
    render(() => <Button variant="ghost">Ghost</Button>);
    const button = screen.getByRole('button');
    expect(button.className).toContain('bg-transparent');
  });

  it('applies small size styles', () => {
    render(() => <Button size="sm">Small</Button>);
    const button = screen.getByRole('button');
    expect(button.className).toContain('text-xs');
  });

  it('applies large size styles', () => {
    render(() => <Button size="lg">Large</Button>);
    const button = screen.getByRole('button');
    expect(button.className).toContain('text-base');
  });

  it('passes through extra HTML attributes', () => {
    render(() => <Button type="submit" data-testid="submit-btn">Submit</Button>);
    const button = screen.getByTestId('submit-btn');
    expect(button).toHaveAttribute('type', 'submit');
  });

  it('applies custom class', () => {
    render(() => <Button class="custom-class">Custom</Button>);
    const button = screen.getByRole('button');
    expect(button.className).toContain('custom-class');
  });
});
