import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import Select from './Select';

describe('Select', () => {
  const defaultOptions = [
    { value: 'a', label: 'Option A' },
    { value: 'b', label: 'Option B' },
    { value: 'c', label: 'Option C' },
  ];

  it('renders a select element', () => {
    render(() => <Select options={defaultOptions} />);
    expect(screen.getByRole('combobox')).toBeInTheDocument();
  });

  it('renders all options', () => {
    render(() => <Select options={defaultOptions} />);
    const options = screen.getAllByRole('option');
    expect(options).toHaveLength(3);
    expect(options[0]).toHaveTextContent('Option A');
    expect(options[1]).toHaveTextContent('Option B');
    expect(options[2]).toHaveTextContent('Option C');
  });

  it('renders label', () => {
    render(() => <Select label="Choose" options={defaultOptions} />);
    expect(screen.getByText('Choose')).toBeInTheDocument();
  });

  it('renders placeholder option', () => {
    render(() => <Select placeholder="Select one..." options={defaultOptions} />);
    const options = screen.getAllByRole('option');
    expect(options).toHaveLength(4);
    expect(options[0]).toHaveTextContent('Select one...');
    expect(options[0]).toBeDisabled();
  });

  it('renders error message', () => {
    render(() => <Select error="Required" options={defaultOptions} />);
    expect(screen.getByText('Required')).toBeInTheDocument();
  });

  it('applies error styling', () => {
    render(() => <Select error="Required" options={defaultOptions} />);
    const select = screen.getByRole('combobox');
    expect(select.className).toContain('border-red-500');
  });

  it('handles change events', () => {
    const onChange = vi.fn();
    render(() => <Select options={defaultOptions} onChange={onChange} />);
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'b' } });
    expect(onChange).toHaveBeenCalled();
  });

  it('renders disabled options', () => {
    const options = [
      { value: 'a', label: 'Option A' },
      { value: 'b', label: 'Option B', disabled: true },
    ];
    render(() => <Select options={options} />);
    const allOptions = screen.getAllByRole('option');
    expect(allOptions[1]).toBeDisabled();
  });

  it('generates ID from label', () => {
    render(() => <Select label="My Select" options={defaultOptions} />);
    const select = screen.getByRole('combobox');
    expect(select.id).toBe('my-select');
  });

  it('uses custom ID when provided', () => {
    render(() => <Select id="custom" label="My Select" options={defaultOptions} />);
    const select = screen.getByRole('combobox');
    expect(select.id).toBe('custom');
  });
});
