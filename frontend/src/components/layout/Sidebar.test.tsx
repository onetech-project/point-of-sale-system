import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import Sidebar from './Sidebar';
import { useAuth } from '@/store/auth';

jest.mock('next/navigation', () => ({
  usePathname: () => '/dashboard',
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('@/store/auth', () => ({
  useAuth: jest.fn(),
}));

jest.mock('@/i18n/provider', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

jest.mock('./LanguageSwitcher', () => ({
  __esModule: true,
  default: () => <div>Language Switcher</div>,
}));

const mockUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;

it('shows the settings menu to managers', () => {
  mockUseAuth.mockReturnValue({
    user: {
      email: 'manager@example.com',
      firstName: 'Mina',
      role: 'manager',
    },
    logout: jest.fn(),
  } as any);

  render(<Sidebar isOpen onClose={jest.fn()} />);

  expect(screen.getByRole('link', { name: /settings/i })).toBeInTheDocument();
});
