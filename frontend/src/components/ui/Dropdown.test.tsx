import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import Dropdown, { DropdownItem, DropdownSeparator } from './Dropdown';

describe('Dropdown', () => {
  it('renders trigger', () => {
    render(() => (
      <Dropdown trigger={<button>Menu</button>}>
        <div>Content</div>
      </Dropdown>
    ));
    expect(screen.getByText('Menu')).toBeInTheDocument();
  });

  it('does not show content initially', () => {
    render(() => (
      <Dropdown trigger={<button>Menu</button>}>
        <div>Dropdown Content</div>
      </Dropdown>
    ));
    expect(screen.queryByText('Dropdown Content')).not.toBeInTheDocument();
  });

  it('shows content when trigger is clicked', () => {
    render(() => (
      <Dropdown trigger={<button>Menu</button>}>
        <div>Dropdown Content</div>
      </Dropdown>
    ));
    fireEvent.click(screen.getByText('Menu'));
    expect(screen.getByText('Dropdown Content')).toBeInTheDocument();
  });

  it('hides content when trigger is clicked again', () => {
    render(() => (
      <Dropdown trigger={<button>Menu</button>}>
        <div>Dropdown Content</div>
      </Dropdown>
    ));
    fireEvent.click(screen.getByText('Menu'));
    expect(screen.getByText('Dropdown Content')).toBeInTheDocument();
    fireEvent.click(screen.getByText('Menu'));
    expect(screen.queryByText('Dropdown Content')).not.toBeInTheDocument();
  });

  it('closes on Escape key', () => {
    render(() => (
      <Dropdown trigger={<button>Menu</button>}>
        <div>Dropdown Content</div>
      </Dropdown>
    ));
    fireEvent.click(screen.getByText('Menu'));
    expect(screen.getByText('Dropdown Content')).toBeInTheDocument();
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(screen.queryByText('Dropdown Content')).not.toBeInTheDocument();
  });

  it('closes when content is clicked', () => {
    render(() => (
      <Dropdown trigger={<button>Menu</button>}>
        <div>Click me</div>
      </Dropdown>
    ));
    fireEvent.click(screen.getByText('Menu'));
    fireEvent.click(screen.getByText('Click me'));
    expect(screen.queryByText('Click me')).not.toBeInTheDocument();
  });

  it('aligns left by default', () => {
    render(() => (
      <Dropdown trigger={<button>Menu</button>}>
        <div>Content</div>
      </Dropdown>
    ));
    fireEvent.click(screen.getByText('Menu'));
    const dropdown = screen.getByText('Content').closest('[class*="absolute"]');
    expect(dropdown?.className).toContain('left-0');
  });

  it('aligns right when specified', () => {
    render(() => (
      <Dropdown trigger={<button>Menu</button>} align="right">
        <div>Content</div>
      </Dropdown>
    ));
    fireEvent.click(screen.getByText('Menu'));
    const dropdown = screen.getByText('Content').closest('[class*="absolute"]');
    expect(dropdown?.className).toContain('right-0');
  });
});

describe('DropdownItem', () => {
  it('renders children', () => {
    render(() => <DropdownItem>Item Text</DropdownItem>);
    expect(screen.getByText('Item Text')).toBeInTheDocument();
  });

  it('handles click events', () => {
    const onClick = vi.fn();
    render(() => <DropdownItem onClick={onClick}>Click</DropdownItem>);
    fireEvent.click(screen.getByText('Click'));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it('renders icon', () => {
    render(() => (
      <DropdownItem icon={<span data-testid="icon">I</span>}>
        With Icon
      </DropdownItem>
    ));
    expect(screen.getByTestId('icon')).toBeInTheDocument();
  });

  it('applies danger styles', () => {
    render(() => <DropdownItem danger>Delete</DropdownItem>);
    const button = screen.getByText('Delete').closest('button');
    expect(button?.className).toContain('text-red-400');
  });

  it('is disabled when disabled prop is true', () => {
    render(() => <DropdownItem disabled>Disabled</DropdownItem>);
    expect(screen.getByRole('button')).toBeDisabled();
  });
});

describe('DropdownSeparator', () => {
  it('renders a separator', () => {
    const { container } = render(() => <DropdownSeparator />);
    const separator = container.querySelector('.border-t');
    expect(separator).not.toBeNull();
  });
});
