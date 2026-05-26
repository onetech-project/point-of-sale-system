import React from 'react';
import { render, waitFor } from '@testing-library/react';
import { usePathname } from 'next/navigation';
import { useOnborda } from 'onborda';
import { useAuth } from '@/store/auth';
import { OnboardingController } from './OnboardingProvider';

jest.mock('next/navigation', () => ({
  usePathname: jest.fn(),
}));

jest.mock('onborda', () => ({
  useOnborda: jest.fn(),
}));

jest.mock('@/store/auth', () => ({
  useAuth: jest.fn(),
}));

const mockUsePathname = usePathname as jest.MockedFunction<typeof usePathname>;
const mockUseOnborda = useOnborda as jest.MockedFunction<typeof useOnborda>;
const mockUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;

beforeEach(() => {
  jest.clearAllMocks();
  mockUseOnborda.mockReturnValue({
    startOnborda: jest.fn(),
  } as any);
});

it('does not start on public routes', async () => {
  const getProgress = jest.fn();
  mockUsePathname.mockReturnValue('/login');
  mockUseAuth.mockReturnValue({
    isAuthenticated: true,
    isLoading: false,
    user: { role: 'owner' },
  } as any);

  render(<OnboardingController service={{ getProgress }} />);

  await waitFor(() => expect(getProgress).not.toHaveBeenCalled());
  expect(mockUseOnborda().startOnborda).not.toHaveBeenCalled();
});

it('starts the incomplete role tour on dashboard', async () => {
  const startOnborda = jest.fn();
  const getProgress = jest.fn().mockResolvedValue({ completed: false });
  mockUseOnborda.mockReturnValue({ startOnborda } as any);
  mockUsePathname.mockReturnValue('/dashboard');
  mockUseAuth.mockReturnValue({
    isAuthenticated: true,
    isLoading: false,
    user: { role: 'manager' },
  } as any);

  render(<OnboardingController service={{ getProgress }} />);

  await waitFor(() => {
    expect(getProgress).toHaveBeenCalledWith('manager-onboarding', 1);
    expect(startOnborda).toHaveBeenCalledWith('manager-onboarding');
  });
});

it('does not start after completed progress', async () => {
  const startOnborda = jest.fn();
  const getProgress = jest.fn().mockResolvedValue({ completed: true });
  mockUseOnborda.mockReturnValue({ startOnborda } as any);
  mockUsePathname.mockReturnValue('/dashboard');
  mockUseAuth.mockReturnValue({
    isAuthenticated: true,
    isLoading: false,
    user: { role: 'cashier' },
  } as any);

  render(<OnboardingController service={{ getProgress }} />);

  await waitFor(() => expect(getProgress).toHaveBeenCalledWith('cashier-onboarding', 1));
  expect(startOnborda).not.toHaveBeenCalled();
});
