import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import AccountUnavailablePage from './page';

let mockStatus = 'suspended';

jest.mock('next/navigation', () => ({
  useSearchParams: () => ({
    get: () => mockStatus,
  }),
}));

describe('AccountUnavailablePage', () => {
  it('shows suspended tenant account copy', () => {
    mockStatus = 'suspended';

    render(<AccountUnavailablePage />);

    expect(screen.getByText('This tenant account is suspended. Please contact platform support or your administrator.')).toBeInTheDocument();
  });

  it('shows inactive tenant account copy', () => {
    mockStatus = 'inactive';

    render(<AccountUnavailablePage />);

    expect(screen.getByText('This tenant account is inactive. Please contact platform support or your administrator.')).toBeInTheDocument();
  });
});
