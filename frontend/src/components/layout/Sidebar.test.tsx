import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import Sidebar from './Sidebar';
import { useAuth } from '@/store/auth';

let mockPathname = '/dashboard';

jest.mock('next/navigation', () => ({
  usePathname: () => mockPathname,
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

beforeEach(() => {
  mockPathname = '/dashboard';
  jest.clearAllMocks();
});

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

it('shows one Orders menu item and removes Offline Orders', () => {
  mockUseAuth.mockReturnValue({
    user: {
      email: 'cashier@example.com',
      firstName: 'Cici',
      role: 'cashier',
    },
    logout: jest.fn(),
  } as any);

  render(<Sidebar isOpen onClose={jest.fn()} />);

  expect(screen.getByRole('link', { name: /^orders$/i })).toBeInTheDocument();
  expect(screen.queryByRole('link', { name: /offline orders/i })).not.toBeInTheDocument();
});

it.each(['owner', 'manager', 'cashier'])('shows QR Generator to %s users', role => {
  mockUseAuth.mockReturnValue({
    user: {
      email: `${role}@example.com`,
      firstName: role,
      role,
    },
    logout: jest.fn(),
  } as any);

  render(<Sidebar isOpen onClose={jest.fn()} />);

  const qrLink = screen.getByRole('link', { name: /qr generator/i });
  expect(qrLink).toBeInTheDocument();
  expect(qrLink).toHaveAttribute('href', '/menu-qr');
});
