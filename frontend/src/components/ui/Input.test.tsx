import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import Input from './Input';

describe('Input', () => {
  it('renders an input element', () => {
    render(() => <Input />);
    expect(screen.getByRole('textbox')).toBeInTheDocument();
  });

  it('renders with label', () => {
    render(() => <Input label="Email" />);
    expect(screen.getByText('Email')).toBeInTheDocument();
    expect(screen.getByLabelText('Email')).toBeInTheDocument();
  });

  it('renders placeholder text', () => {
    render(() => <Input placeholder="Enter email" />);
    expect(screen.getByPlaceholderText('Enter email')).toBeInTheDocument();
  });

  it('displays error message', () => {
    render(() => <Input error="This field is required" />);
    expect(screen.getByText('This field is required')).toBeInTheDocument();
  });

  it('adds error styling when error is present', () => {
    render(() => <Input error="Required" />);
    const input = screen.getByRole('textbox');
    expect(input.className).toContain('border-red-500');
  });

  it('displays hint text', () => {
    render(() => <Input hint="Enter your email address" />);
    expect(screen.getByText('Enter your email address')).toBeInTheDocument();
  });

  it('hides hint when error is present', () => {
    render(() => <Input hint="A hint" error="An error" />);
    expect(screen.queryByText('A hint')).not.toBeInTheDocument();
    expect(screen.getByText('An error')).toBeInTheDocument();
  });

  it('renders with icon', () => {
    render(() => <Input icon={<span data-testid="input-icon">@</span>} />);
    expect(screen.getByTestId('input-icon')).toBeInTheDocument();
  });

  it('renders with right icon', () => {
    render(() => <Input rightIcon={<span data-testid="right-icon">X</span>} />);
    expect(screen.getByTestId('right-icon')).toBeInTheDocument();
  });

  it('handles input events', () => {
    const onInput = vi.fn();
    render(() => <Input onInput={onInput} />);
    fireEvent.input(screen.getByRole('textbox'), { target: { value: 'test' } });
    expect(onInput).toHaveBeenCalled();
  });

  it('generates ID from label', () => {
    render(() => <Input label="Full Name" />);
    const input = screen.getByRole('textbox');
    expect(input.id).toBe('full-name');
  });

  it('uses custom ID when provided', () => {
    render(() => <Input id="custom-id" label="Test" />);
    const input = screen.getByRole('textbox');
    expect(input.id).toBe('custom-id');
  });

  it('passes through HTML attributes', () => {
    render(() => <Input type="email" required />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveAttribute('type', 'email');
    expect(input).toBeRequired();
  });

  it('applies custom class to wrapper', () => {
    render(() => <Input class="my-class" />);
    const wrapper = screen.getByRole('textbox').closest('.my-class');
    expect(wrapper).not.toBeNull();
  });
});
