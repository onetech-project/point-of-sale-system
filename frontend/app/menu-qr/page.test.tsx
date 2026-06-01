import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import MenuQrPage from './page';
import { tenantService } from '@/services/tenant';

jest.mock('@/components/auth/ProtectedRoute', () => ({
  __esModule: true,
  default: ({ children, allowedRoles }: { children: React.ReactNode; allowedRoles: string[] }) => (
    <div data-testid="protected-route" data-allowed-roles={allowedRoles.join(',')}>
      {children}
    </div>
  ),
}));

jest.mock('@/components/layout/DashboardLayout', () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="dashboard-layout">{children}</div>
  ),
}));

jest.mock('@/services/tenant', () => ({
  tenantService: {
    getTenantInfo: jest.fn(),
  },
}));

jest.mock('qrcode.react', () => ({
  QRCodeCanvas: ({ value, ...props }: { value: string; [key: string]: unknown }) => (
    <canvas data-testid="qr-code" data-value={value} aria-label={props['aria-label'] as string} />
  ),
}));

const mockTenantService = tenantService as jest.Mocked<typeof tenantService>;

beforeEach(() => {
  jest.clearAllMocks();
  process.env.NEXT_PUBLIC_SITE_URL = 'https://shop.example.com';
  jest.spyOn(window, 'open').mockImplementation(jest.fn());
  jest.spyOn(HTMLCanvasElement.prototype, 'toDataURL').mockReturnValue('data:image/png;base64,qr');
});

afterEach(() => {
  jest.restoreAllMocks();
  delete process.env.NEXT_PUBLIC_SITE_URL;
});

it('renders the protected dashboard page with the tenant menu link and QR value', async () => {
  mockTenantService.getTenantInfo.mockResolvedValue({
    id: 'tenant-1',
    businessName: 'Bistro One',
    slug: 'bistro-one',
    status: 'active',
    createdAt: '2026-05-26T00:00:00Z',
  });

  render(<MenuQrPage />);

  const expectedUrl = 'https://shop.example.com/menu/bistro-one';

  expect(await screen.findByDisplayValue(expectedUrl)).toBeInTheDocument();
  expect(screen.getByTestId('qr-code')).toHaveAttribute('data-value', expectedUrl);
  expect(screen.getByTestId('protected-route')).toHaveAttribute(
    'data-allowed-roles',
    'owner,manager,cashier'
  );
  expect(screen.getByTestId('dashboard-layout')).toBeInTheDocument();
});

it('copies, opens, and downloads the menu QR', async () => {
  const user = userEvent.setup();
  const writeText = jest.fn().mockResolvedValue(undefined);
  Object.defineProperty(window.navigator, 'clipboard', {
    configurable: true,
    value: {
      writeText,
    },
  });

  mockTenantService.getTenantInfo.mockResolvedValue({
    id: 'tenant-1',
    businessName: 'Bistro One',
    slug: 'bistro-one',
    status: 'active',
    createdAt: '2026-05-26T00:00:00Z',
  });

  render(<MenuQrPage />);

  const expectedUrl = 'https://shop.example.com/menu/bistro-one';
  await screen.findByDisplayValue(expectedUrl);

  await user.click(screen.getByRole('button', { name: /copy link/i }));
  await waitFor(() => {
    expect(writeText).toHaveBeenCalledWith(expectedUrl);
  });
  expect(screen.getByRole('button', { name: /copied/i })).toBeInTheDocument();

  await user.click(screen.getByRole('button', { name: /open menu/i }));
  expect(window.open).toHaveBeenCalledWith(expectedUrl, '_blank', 'noopener,noreferrer');

  const anchor = document.createElement('a');
  const click = jest.fn();
  anchor.click = click;
  const originalCreateElement = Document.prototype.createElement;
  const createElement = jest.spyOn(document, 'createElement').mockImplementation((tagName, options) => {
    if (tagName === 'a') {
      return anchor;
    }
    return originalCreateElement.call(document, tagName, options);
  });

  await user.click(screen.getByRole('button', { name: /download png/i }));

  expect(createElement).toHaveBeenCalledWith('a');
  expect(HTMLCanvasElement.prototype.toDataURL).toHaveBeenCalledWith('image/png');
  expect(anchor).toHaveAttribute('href', 'data:image/png;base64,qr');
  expect(anchor).toHaveAttribute('download', 'bistro-one-menu-qr.png');
  expect(click).toHaveBeenCalled();
});
