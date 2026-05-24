import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import SettingsPage from './page';
import { useAuth } from '@/store/auth';

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('@/store/auth', () => ({
  useAuth: jest.fn(),
}));

jest.mock('@/components/auth/ProtectedRoute', () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('@/components/layout/DashboardLayout', () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

const mockUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;

it('shows tenant settings but not payment settings to managers', () => {
  mockUseAuth.mockReturnValue({
    user: { role: 'manager' },
  } as any);

  render(<SettingsPage />);

  expect(screen.getByText('Tenant Settings')).toBeInTheDocument();
  expect(screen.queryByText('Payment Settings')).not.toBeInTheDocument();
  expect(screen.getByText('Subscription & Billing')).toBeInTheDocument();
});

it('shows payment settings to owners', () => {
  mockUseAuth.mockReturnValue({
    user: { role: 'owner' },
  } as any);

  render(<SettingsPage />);

  expect(screen.getByText('Payment Settings')).toBeInTheDocument();
  expect(screen.getByText('Tenant Settings')).toBeInTheDocument();
});
