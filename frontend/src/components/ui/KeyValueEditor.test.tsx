import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import KeyValueEditor from './KeyValueEditor';
import type { KeyValuePair } from './KeyValueEditor';

describe('KeyValueEditor', () => {
  const defaultItems: KeyValuePair[] = [
    { key: 'API_KEY', value: 'abc123' },
    { key: 'DB_HOST', value: 'localhost' },
  ];

  it('renders items', () => {
    render(() => <KeyValueEditor items={defaultItems} onChange={() => {}} />);
    const inputs = screen.getAllByRole('textbox');
    // 2 items * 2 inputs (key + value) = 4
    expect(inputs.length).toBeGreaterThanOrEqual(4);
  });

  it('renders key/value headers when items exist', () => {
    render(() => <KeyValueEditor items={defaultItems} onChange={() => {}} />);
    expect(screen.getByText('Key')).toBeInTheDocument();
    expect(screen.getByText('Value')).toBeInTheDocument();
  });

  it('renders empty state when no items', () => {
    render(() => <KeyValueEditor items={[]} onChange={() => {}} />);
    expect(screen.getByText(/No.*environment variables/i)).toBeInTheDocument();
  });

  it('renders empty state for secrets when secretMode', () => {
    render(() => <KeyValueEditor items={[]} onChange={() => {}} secretMode />);
    expect(screen.getByText(/No.*secrets/i)).toBeInTheDocument();
  });

  it('shows "Add Variable" button', () => {
    render(() => <KeyValueEditor items={[]} onChange={() => {}} />);
    expect(screen.getByText(/Add Variable/)).toBeInTheDocument();
  });

  it('shows "Add Secret" button in secret mode', () => {
    render(() => <KeyValueEditor items={[]} onChange={() => {}} secretMode />);
    expect(screen.getByText(/Add Secret/)).toBeInTheDocument();
  });

  it('calls onChange when adding a row', () => {
    const onChange = vi.fn();
    render(() => <KeyValueEditor items={[]} onChange={onChange} />);
    fireEvent.click(screen.getByText(/Add Variable/));
    expect(onChange).toHaveBeenCalledOnce();
    const newItems = onChange.mock.calls[0][0] as KeyValuePair[];
    expect(newItems).toHaveLength(1);
    expect(newItems[0].key).toBe('');
    expect(newItems[0].value).toBe('');
    expect(newItems[0].isNew).toBe(true);
  });

  it('calls onChange when removing a row', () => {
    const onChange = vi.fn();
    render(() => <KeyValueEditor items={defaultItems} onChange={onChange} />);
    const removeButtons = screen.getAllByTitle('Remove');
    fireEvent.click(removeButtons[0]);
    expect(onChange).toHaveBeenCalledOnce();
    const newItems = onChange.mock.calls[0][0] as KeyValuePair[];
    expect(newItems).toHaveLength(1);
    expect(newItems[0].key).toBe('DB_HOST');
  });

  it('auto-uppercases and sanitizes keys', () => {
    const onChange = vi.fn();
    const items: KeyValuePair[] = [{ key: '', value: '', isNew: true }];
    render(() => <KeyValueEditor items={items} onChange={onChange} />);
    const inputs = screen.getAllByRole('textbox');
    // First input is the key
    fireEvent.input(inputs[0], { target: { value: 'my-key' } });
    expect(onChange).toHaveBeenCalled();
    const newItems = onChange.mock.calls[0][0] as KeyValuePair[];
    expect(newItems[0].key).toBe('MY_KEY');
  });

  it('renders custom placeholders', () => {
    render(() => (
      <KeyValueEditor
        items={[{ key: '', value: '', isNew: true }]}
        onChange={() => {}}
        keyPlaceholder="VARIABLE_NAME"
        valuePlaceholder="variable_value"
      />
    ));
    expect(screen.getByPlaceholderText('VARIABLE_NAME')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('variable_value')).toBeInTheDocument();
  });
});
