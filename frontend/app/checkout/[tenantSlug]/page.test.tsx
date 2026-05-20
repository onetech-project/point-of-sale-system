import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import CheckoutPage from './page';
import { tenant } from '../../../src/services/tenant';
import { TENANT_UNAVAILABLE_MESSAGE } from '../../../src/utils/tenantAvailability';

jest.mock('next/navigation', () => ({
  useParams: () => ({ tenantSlug: 'bistro-one' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (_key: string, fallback?: string) => fallback ?? _key,
  }),
}));

jest.mock('../../../src/components/layout/PublicLayout', () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

jest.mock('../../../src/services/tenant', () => ({
  tenant: {
    getTenantConfig: jest.fn(),
  },
}));

jest.mock('../../../src/services/cart', () => ({
  cart: {
    getCart: jest.fn(),
  },
}));

jest.mock('../../../src/services/order', () => ({
  order: {
    createOrder: jest.fn(),
  },
}));

const mockTenant = tenant as jest.Mocked<typeof tenant>;

it('renders unavailable copy when checkout tenant config is blocked', async () => {
  mockTenant.getTenantConfig.mockRejectedValueOnce({
    response: {
      status: 403,
      data: {
        error: 'Tenant currently unavailable',
        message: TENANT_UNAVAILABLE_MESSAGE,
        status: 'active',
        subscription_status: 'grace_period',
      },
    },
  });

  render(<CheckoutPage />);

  expect(await screen.findByText(TENANT_UNAVAILABLE_MESSAGE)).toBeInTheDocument();
});
