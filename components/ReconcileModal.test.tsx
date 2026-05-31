import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import ReconcileModal from './ReconcileModal';

const baseProps = {
  levyAccounts: [],
  periodLabel: 'October 2026',
  onClose: vi.fn(),
  onConfirm: vi.fn(),
};

describe('ReconcileModal', () => {
  it('does not render sample statement load in production mode by default', () => {
    render(
      <ReconcileModal
        {...baseProps}
        onConfirm={vi.fn()}
      />
    );

    expect(screen.getByText(/Upload your bank statement CSV\./i)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /load sample statement/i })).not.toBeInTheDocument();
  });

  it('renders and processes sample statement when explicitly enabled', async () => {
    render(
      <ReconcileModal
        {...baseProps}
        allowSampleStatement
        onConfirm={vi.fn()}
      />
    );

    const button = screen.getByRole('button', { name: /load sample statement/i });
    fireEvent.click(button);

    expect(await screen.findByText(/sample-statement\.csv/i)).toBeInTheDocument();
    expect(screen.getByText(/7 transactions/i)).toBeInTheDocument();
  });
});
