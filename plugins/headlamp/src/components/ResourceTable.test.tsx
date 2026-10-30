import React from 'react';
import { fireEvent, render, screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ResourceKindStats } from '../types';
import ResourceTable from './ResourceTable';

vi.mock('@iconify/react', () => ({
  Icon: ({ icon }: { icon: string }) => <span data-testid="icon" data-icon={icon} />,
}));

function makeResource(groupKind: string, overrides: Partial<ResourceKindStats> = {}): ResourceKindStats {
  return {
    groupKind,
    instances: 1,
    exportedInstances: 1,
    exportedSizeBytes: 1024,
    queryDuration: '1ms',
    exportDuration: '2ms',
    ...overrides,
  };
}

// Returns the first column (kind) of every body row in display order.
function kindColumn(): string[] {
  const [, ...bodyRows] = screen.getAllByRole('row');
  return bodyRows.map((row) => within(row).getAllByRole('cell')[0].textContent ?? '');
}

describe('ResourceTable', () => {
  const resources = [
    makeResource('batch.Job', { instances: 5 }),
    makeResource('extensions.Deployment', { instances: 1 }),
    makeResource('ConfigMap', { instances: 30 }),
    makeResource('apps.Deployment', { instances: 10 }),
  ];

  it('renders a row per resource with formatted values', () => {
    render(<ResourceTable resources={[makeResource('apps.Deployment', { instances: 1234, exportedSizeBytes: 1536 })]} />);

    expect(screen.getByText('Resource Kinds (1)')).toBeTruthy();
    const [, row] = screen.getAllByRole('row');
    const cells = within(row).getAllByRole('cell').map((cell) => cell.textContent);
    expect(cells).toEqual(['apps.Deployment', '1,234', '1', '1.5 KB', '1ms', '2ms']);
  });

  it('sorts by kind and then group by default', () => {
    render(<ResourceTable resources={resources} />);

    expect(kindColumn()).toEqual(['ConfigMap', 'apps.Deployment', 'extensions.Deployment', 'batch.Job']);
  });

  it('reverses the kind order when the kind header is clicked', () => {
    render(<ResourceTable resources={resources} />);

    fireEvent.click(screen.getByText('Kind'));

    expect(kindColumn()).toEqual(['batch.Job', 'extensions.Deployment', 'apps.Deployment', 'ConfigMap']);
  });

  it('sorts numeric columns ascending and toggles to descending', () => {
    render(<ResourceTable resources={resources} />);

    fireEvent.click(screen.getByText('Instances'));
    expect(kindColumn()).toEqual(['extensions.Deployment', 'batch.Job', 'apps.Deployment', 'ConfigMap']);

    fireEvent.click(screen.getByText('Instances'));
    expect(kindColumn()).toEqual(['ConfigMap', 'apps.Deployment', 'batch.Job', 'extensions.Deployment']);
  });

  it('sorts string columns', () => {
    render(
      <ResourceTable
        resources={[
          makeResource('A', { queryDuration: '3ms' }),
          makeResource('B', { queryDuration: '1ms' }),
          makeResource('C', { queryDuration: '2ms' }),
        ]}
      />,
    );

    fireEvent.click(screen.getByText('Query Time'));

    expect(kindColumn()).toEqual(['B', 'C', 'A']);
  });

  it('filters case-insensitively and shows the filtered count', () => {
    render(<ResourceTable resources={resources} />);

    fireEvent.change(screen.getByPlaceholderText('Filter by kind...'), { target: { value: 'DEPLOY' } });

    expect(screen.getByText('Resource Kinds (2 / 4)')).toBeTruthy();
    expect(kindColumn()).toEqual(['apps.Deployment', 'extensions.Deployment']);
  });

  it('shows no rows when the filter matches nothing', () => {
    render(<ResourceTable resources={resources} />);

    fireEvent.change(screen.getByPlaceholderText('Filter by kind...'), { target: { value: 'secret' } });

    expect(screen.getByText('Resource Kinds (0 / 4)')).toBeTruthy();
    expect(kindColumn()).toEqual([]);
  });

  it('paginates rows and resets to the first page when filtering', () => {
    const many = Array.from({ length: 30 }, (_, i) => makeResource(`Kind${String(i).padStart(2, '0')}`));
    render(<ResourceTable resources={many} />);

    expect(kindColumn()).toHaveLength(25);

    fireEvent.click(screen.getByRole('button', { name: /next page/i }));
    expect(kindColumn()).toEqual(['Kind25', 'Kind26', 'Kind27', 'Kind28', 'Kind29']);

    fireEvent.change(screen.getByPlaceholderText('Filter by kind...'), { target: { value: 'Kind0' } });
    expect(kindColumn()).toHaveLength(10);
    expect(kindColumn()[0]).toBe('Kind00');
  });

  it('renders trend icons with the previous value', () => {
    render(
      <ResourceTable
        resources={[
          makeResource('apps.Deployment', {
            instances: 2000,
            prevInstances: 1000,
            exportedInstances: 5,
            prevExportedInstances: 8,
            exportedSizeBytes: 1024,
            prevExportedSizeBytes: 1024,
          }),
        ]}
      />,
    );

    const icons = screen.getAllByTestId('icon').map((icon) => icon.getAttribute('data-icon'));
    expect(icons).toEqual(['mdi:table', 'mdi:arrow-up', 'mdi:arrow-down', 'mdi:minus']);
    expect(screen.getByLabelText('Previous: 1,000')).toBeTruthy();
    expect(screen.getByLabelText('Previous: 8')).toBeTruthy();
    expect(screen.getByLabelText('Previous: 1,024')).toBeTruthy();
  });

  it('renders no trend icons without previous values', () => {
    render(<ResourceTable resources={[makeResource('ConfigMap')]} />);

    const icons = screen.getAllByTestId('icon').map((icon) => icon.getAttribute('data-icon'));
    expect(icons).toEqual(['mdi:table']);
  });
});
