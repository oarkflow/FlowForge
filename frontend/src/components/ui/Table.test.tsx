import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@solidjs/testing-library';
import Table from './Table';
import type { TableColumn } from './Table';

interface TestRow {
  id: string;
  name: string;
  status: string;
}

describe('Table', () => {
  const columns: TableColumn<TestRow>[] = [
    { key: 'name', header: 'Name', render: (row) => <span>{row.name}</span> },
    { key: 'status', header: 'Status', render: (row) => <span>{row.status}</span> },
  ];

  const data: TestRow[] = [
    { id: '1', name: 'Item 1', status: 'active' },
    { id: '2', name: 'Item 2', status: 'inactive' },
    { id: '3', name: 'Item 3', status: 'active' },
  ];

  it('renders table element', () => {
    render(() => <Table columns={columns} data={data} />);
    expect(screen.getByRole('table')).toBeInTheDocument();
  });

  it('renders column headers', () => {
    render(() => <Table columns={columns} data={data} />);
    expect(screen.getByText('Name')).toBeInTheDocument();
    expect(screen.getByText('Status')).toBeInTheDocument();
  });

  it('renders all rows', () => {
    render(() => <Table columns={columns} data={data} />);
    expect(screen.getByText('Item 1')).toBeInTheDocument();
    expect(screen.getByText('Item 2')).toBeInTheDocument();
    expect(screen.getByText('Item 3')).toBeInTheDocument();
  });

  it('renders cell data', () => {
    render(() => <Table columns={columns} data={data} />);
    expect(screen.getByText('active')).toBeInTheDocument();
    expect(screen.getByText('inactive')).toBeInTheDocument();
  });

  it('shows empty message when no data', () => {
    render(() => <Table columns={columns} data={[]} emptyMessage="No items found" />);
    expect(screen.getByText('No items found')).toBeInTheDocument();
  });

  it('shows default empty message when no data and no custom message', () => {
    render(() => <Table columns={columns} data={[]} />);
    expect(screen.getByText('No data')).toBeInTheDocument();
  });

  it('shows loading state', () => {
    render(() => <Table columns={columns} data={[]} loading />);
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('calls onRowClick when row is clicked', () => {
    const onRowClick = vi.fn();
    render(() => <Table columns={columns} data={data} onRowClick={onRowClick} />);
    const row = screen.getByText('Item 1').closest('tr');
    if (row) {
      row.click();
      expect(onRowClick).toHaveBeenCalledWith(data[0]);
    }
  });

  it('applies cursor-pointer class when onRowClick is provided', () => {
    const onRowClick = vi.fn();
    render(() => <Table columns={columns} data={data} onRowClick={onRowClick} />);
    const row = screen.getByText('Item 1').closest('tr');
    expect(row?.className).toContain('cursor-pointer');
  });

  it('does not apply cursor-pointer class when onRowClick is not provided', () => {
    render(() => <Table columns={columns} data={data} />);
    const row = screen.getByText('Item 1').closest('tr');
    expect(row?.className).not.toContain('cursor-pointer');
  });

  it('applies custom class', () => {
    render(() => <Table columns={columns} data={data} class="my-table" />);
    const wrapper = screen.getByRole('table').closest('.my-table');
    expect(wrapper).not.toBeNull();
  });

  it('renders JSX headers', () => {
    const columnsWithJsx: TableColumn<TestRow>[] = [
      { key: 'name', header: <span data-testid="custom-header">Custom</span>, render: (row) => <span>{row.name}</span> },
    ];
    render(() => <Table columns={columnsWithJsx} data={data} />);
    expect(screen.getByTestId('custom-header')).toBeInTheDocument();
  });
});
