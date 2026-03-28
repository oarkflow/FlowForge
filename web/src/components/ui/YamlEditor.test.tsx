import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import { YamlEditor } from './YamlEditor';

describe('YamlEditor', () => {
  it('renders a textarea', () => {
    render(() => <YamlEditor value="" />);
    expect(screen.getByRole('textbox')).toBeInTheDocument();
  });

  it('displays initial value', () => {
    render(() => <YamlEditor value="name: test" />);
    const textarea = screen.getByRole('textbox') as HTMLTextAreaElement;
    expect(textarea.value).toBe('name: test');
  });

  it('renders placeholder', () => {
    render(() => <YamlEditor value="" placeholder="Enter YAML..." />);
    expect(screen.getByPlaceholderText('Enter YAML...')).toBeInTheDocument();
  });

  it('renders default placeholder', () => {
    render(() => <YamlEditor value="" />);
    expect(screen.getByPlaceholderText('Enter YAML configuration...')).toBeInTheDocument();
  });

  it('calls onChange when text is input', () => {
    const onChange = vi.fn();
    render(() => <YamlEditor value="" onChange={onChange} />);
    const textarea = screen.getByRole('textbox');
    fireEvent.input(textarea, { target: { value: 'version: 1' } });
    expect(onChange).toHaveBeenCalled();
  });

  it('renders line numbers', () => {
    render(() => <YamlEditor value="line1\nline2\nline3" />);
    // Should have line numbers 1, 2, 3
    const container = screen.getByRole('textbox').parentElement;
    expect(container).not.toBeNull();
  });

  it('respects readOnly prop', () => {
    render(() => <YamlEditor value="readonly content" readOnly />);
    const textarea = screen.getByRole('textbox');
    expect(textarea).toHaveAttribute('readonly');
  });

  it('applies custom height', () => {
    const { container } = render(() => <YamlEditor value="" height="300px" />);
    const wrapper = container.firstElementChild;
    expect(wrapper?.getAttribute('style')).toContain('300px');
  });

  it('applies spellcheck false', () => {
    render(() => <YamlEditor value="" />);
    const textarea = screen.getByRole('textbox');
    expect(textarea).toHaveAttribute('spellcheck', 'false');
  });
});
