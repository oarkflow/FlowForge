import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import Tabs from './Tabs';

describe('Tabs', () => {
  const tabs = [
    { id: 'tab1', label: 'Tab 1' },
    { id: 'tab2', label: 'Tab 2' },
    { id: 'tab3', label: 'Tab 3' },
  ];

  it('renders all tabs', () => {
    render(() => <Tabs tabs={tabs} activeTab="tab1" onTabChange={() => {}} />);
    expect(screen.getByText('Tab 1')).toBeInTheDocument();
    expect(screen.getByText('Tab 2')).toBeInTheDocument();
    expect(screen.getByText('Tab 3')).toBeInTheDocument();
  });

  it('renders with tablist role', () => {
    render(() => <Tabs tabs={tabs} activeTab="tab1" onTabChange={() => {}} />);
    expect(screen.getByRole('tablist')).toBeInTheDocument();
  });

  it('renders tab buttons with tab role', () => {
    render(() => <Tabs tabs={tabs} activeTab="tab1" onTabChange={() => {}} />);
    const tabButtons = screen.getAllByRole('tab');
    expect(tabButtons).toHaveLength(3);
  });

  it('marks active tab with aria-selected', () => {
    render(() => <Tabs tabs={tabs} activeTab="tab2" onTabChange={() => {}} />);
    const tabButtons = screen.getAllByRole('tab');
    expect(tabButtons[0]).toHaveAttribute('aria-selected', 'false');
    expect(tabButtons[1]).toHaveAttribute('aria-selected', 'true');
    expect(tabButtons[2]).toHaveAttribute('aria-selected', 'false');
  });

  it('calls onTabChange when tab is clicked', () => {
    const onTabChange = vi.fn();
    render(() => <Tabs tabs={tabs} activeTab="tab1" onTabChange={onTabChange} />);
    fireEvent.click(screen.getByText('Tab 2'));
    expect(onTabChange).toHaveBeenCalledWith('tab2');
  });

  it('renders disabled tabs', () => {
    const tabsWithDisabled = [
      { id: 'tab1', label: 'Tab 1' },
      { id: 'tab2', label: 'Tab 2', disabled: true },
    ];
    render(() => <Tabs tabs={tabsWithDisabled} activeTab="tab1" onTabChange={() => {}} />);
    const tabButtons = screen.getAllByRole('tab');
    expect(tabButtons[1]).toBeDisabled();
  });

  it('renders tabs with icons', () => {
    const tabsWithIcons = [
      { id: 'tab1', label: 'Tab 1', icon: <span data-testid="icon-1">I</span> },
    ];
    render(() => <Tabs tabs={tabsWithIcons} activeTab="tab1" onTabChange={() => {}} />);
    expect(screen.getByTestId('icon-1')).toBeInTheDocument();
  });

  it('applies active styles to active tab', () => {
    render(() => <Tabs tabs={tabs} activeTab="tab1" onTabChange={() => {}} />);
    const activeTab = screen.getByText('Tab 1').closest('button');
    expect(activeTab?.className).toContain('text-[var(--color-text-primary)]');
  });

  it('applies inactive styles to non-active tabs', () => {
    render(() => <Tabs tabs={tabs} activeTab="tab1" onTabChange={() => {}} />);
    const inactiveTab = screen.getByText('Tab 2').closest('button');
    expect(inactiveTab?.className).toContain('text-[var(--color-text-tertiary)]');
  });

  it('applies custom class', () => {
    render(() => <Tabs tabs={tabs} activeTab="tab1" onTabChange={() => {}} class="my-tabs" />);
    expect(screen.getByRole('tablist').className).toContain('my-tabs');
  });
});
