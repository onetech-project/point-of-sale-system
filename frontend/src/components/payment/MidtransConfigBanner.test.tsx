import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import MidtransConfigBanner from './MidtransConfigBanner';
import { useAuth } from '@/store/auth';
import { tenantService } from '@/services/tenant';

jest.mock('next/link', () => ({
  __esModule: true,
  default: ({ href, children, ...props }: any) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

jest.mock('@/store/auth', () => ({
  useAuth: jest.fn(),
}));

jest.mock('@/services/tenant', () => ({
  tenantService: {
    getTenantInfo: jest.fn(),
  },
}));

const mockUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;
const mockTenantService = tenantService as jest.Mocked<typeof tenantService>;

beforeEach(() => {
  jest.clearAllMocks();
  mockTenantService.getTenantInfo.mockResolvedValue({
    id: 'tenant-1',
    businessName: 'Bistro One',
    slug: 'bistro-one',
    status: 'active',
    createdAt: '2026-05-21T00:00:00Z',
    midtrans_configured: false,
    midtrans_environment: 'sandbox',
  });
});

it('shows a settings CTA for owners when Midtrans is not configured', async () => {
  mockUseAuth.mockReturnValue({
    user: { role: 'owner' },
    isAuthenticated: true,
    isLoading: false,
  } as any);

  render(<MidtransConfigBanner />);

  expect(
    await screen.findByText('Payment gateway is not configured. Guest ordering is currently unavailable.')
  ).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Configure Payment Settings' })).toHaveAttribute(
    'href',
    '/settings/payment'
  );
});

it('shows contact-owner copy for managers when Midtrans is not configured', async () => {
  mockUseAuth.mockReturnValue({
    user: { role: 'manager' },
    isAuthenticated: true,
    isLoading: false,
  } as any);

  render(<MidtransConfigBanner />);

  expect(await screen.findByText('Contact an owner.')).toBeInTheDocument();
  expect(
    screen.queryByRole('link', { name: 'Configure Payment Settings' })
  ).not.toBeInTheDocument();
});

it('uses payment_configured when the default gateway is Manjo', async () => {
  mockTenantService.getTenantInfo.mockResolvedValue({
    id: 'tenant-1',
    businessName: 'Bistro One',
    slug: 'bistro-one',
    status: 'active',
    createdAt: '2026-05-21T00:00:00Z',
    midtrans_configured: false,
    midtrans_environment: 'sandbox',
    payment_gateway: 'manjo',
    payment_configured: true,
  });
  mockUseAuth.mockReturnValue({
    user: { role: 'owner' },
    isAuthenticated: true,
    isLoading: false,
  } as any);

  const { container } = render(<MidtransConfigBanner />);

  await waitFor(() => expect(mockTenantService.getTenantInfo).toHaveBeenCalled());
  await waitFor(() => expect(container).toBeEmptyDOMElement());
});

it('shows contact-owner copy for cashiers when Midtrans is not configured', async () => {
  mockUseAuth.mockReturnValue({
    user: { role: 'cashier' },
    isAuthenticated: true,
    isLoading: false,
  } as any);

  render(<MidtransConfigBanner />);

  expect(await screen.findByText('Contact an owner.')).toBeInTheDocument();
  expect(
    screen.queryByRole('link', { name: 'Configure Payment Settings' })
  ).not.toBeInTheDocument();
});
