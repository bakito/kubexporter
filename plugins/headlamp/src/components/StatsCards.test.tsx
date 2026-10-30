import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import StatsCards from './StatsCards';

vi.mock('@iconify/react', () => ({
  Icon: ({ icon }: { icon: string }) => <span data-testid="icon" data-icon={icon} />,
}));

describe('StatsCards', () => {
  it('renders all stats with formatted values', () => {
    render(
      <StatsCards
        stats={{ kinds: 42, resources: 1234567, size: 2048, namespaces: 12, errors: 3, duration: '1m30s' }}
      />,
    );

    const expected: Record<string, string> = {
      Resources: '1,234,567',
      Kinds: '42',
      Namespaces: '12',
      Size: '2.0 KB',
      Duration: '1m30s',
      Errors: '3',
    };
    for (const [label, value] of Object.entries(expected)) {
      expect(screen.getByText(label)).toBeTruthy();
      expect(screen.getByText(value)).toBeTruthy();
    }
    expect(screen.getAllByTestId('icon')).toHaveLength(6);
  });

  it('renders zero values', () => {
    render(<StatsCards stats={{ kinds: 0, resources: 0, size: 0, namespaces: 0, errors: 0, duration: '0s' }} />);

    expect(screen.getByText('0 B')).toBeTruthy();
    expect(screen.getByText('0s')).toBeTruthy();
    expect(screen.getAllByText('0')).toHaveLength(4);
  });
});
