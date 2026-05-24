import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import SubscriptionPage from './page';
import { billingService } from '@/services/billing';
import { useSubscription } from '@/store/subscription';
import { redirectToPayment } from '@/utils/paymentRedirect';
import { useAuth } from '@/store/auth';

jest.mock('next/navigation', () => ({
  useRouter: () => ({
    replace: jest.fn(),
  }),
}));

jest.mock('@/components/auth/ProtectedRoute', () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('../../src/components/layout/DashboardLayout', () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('@/store/auth', () => ({
  useAuth: jest.fn(),
}));

jest.mock('@/store/subscription', () => ({
  useSubscription: jest.fn(),
}));

jest.mock('@/services/billing', () => ({
  billingService: {
    getPublicPlans: jest.fn(),
    getInvoices: jest.fn(),
    switchBillingCycle: jest.fn(),
    upgradeSubscription: jest.fn(),
    initiatePayment: jest.fn(),
  },
}));

jest.mock('@/utils/paymentRedirect', () => ({
  redirectToPayment: jest.fn(),
}));

const monthlySubscription = {
  tenant_id: 'tenant-1',
  subscription_status: 'active',
  status: 'active',
  subscription_plan: 'starter',
  billing_cycle: 'monthly',
  billing_interval: 'monthly',
  subscription_ends_at: '2026-06-19T00:00:00Z',
  is_active: true,
  days_remaining: 30,
  invoice_count: 0,
} as const;

const publicPlans = {
  monthly_price_idr: 299000,
  annual_discount_pct: 20,
  trial_days: 7,
};

const mockUseSubscription = useSubscription as jest.MockedFunction<typeof useSubscription>;
const mockUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;
const mockBillingService = billingService as jest.Mocked<typeof billingService>;
const mockRedirectToPayment = redirectToPayment as jest.MockedFunction<typeof redirectToPayment>;
let refreshSubscriptionMock: jest.Mock;

function renderMonthlySubscriptionPage(invoices: any[] = []) {
  refreshSubscriptionMock = jest.fn().mockResolvedValue(monthlySubscription);
  mockUseSubscription.mockReturnValue({
    subscription: monthlySubscription as any,
    isLoading: false,
    error: null,
    isExpired: false,
    setExpired: jest.fn(),
    refreshSubscription: refreshSubscriptionMock,
    invalidateSubscription: jest.fn(),
    clearSubscription: jest.fn(),
  });
  mockBillingService.getPublicPlans.mockResolvedValue(publicPlans);
  mockBillingService.getInvoices.mockResolvedValue(invoices);

  render(<SubscriptionPage />);
}

describe('SubscriptionPage billing payments', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    window.history.pushState({}, '', '/subscription');
    mockUseAuth.mockReturnValue({
      logout: jest.fn(),
      user: { role: 'manager' },
    } as any);
  });

  it('opens confirmation before switching monthly tenants to yearly', async () => {
    const user = userEvent.setup();
    renderMonthlySubscriptionPage();

    await screen.findByText('Subscription & Billing');
    await user.click(screen.getByRole('button', { name: /annual/i }));
    await user.click(screen.getByRole('button', { name: /^switch to yearly$/i }));

    expect(screen.getByRole('dialog', { name: /switch to yearly/i })).toBeInTheDocument();
    expect(screen.getByText(/monthly subscription stays active/i)).toBeInTheDocument();
    expect(mockBillingService.switchBillingCycle).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: /cancel/i }));

    expect(screen.queryByRole('dialog', { name: /switch to yearly/i })).not.toBeInTheDocument();
    expect(mockBillingService.switchBillingCycle).not.toHaveBeenCalled();
  });

  it('confirms yearly switch and redirects to Midtrans in the current tab', async () => {
    const user = userEvent.setup();
    mockBillingService.switchBillingCycle.mockResolvedValue({
      status: 'payment_required',
      subscription_status: 'active',
      current_billing_interval: 'monthly',
      billing_interval: 'annual',
      invoice: {} as any,
      payment_url: 'https://snap.midtrans.example/pay',
      snap_token: 'snap-token',
      due_at: '2026-05-19T00:00:00Z',
    });
    renderMonthlySubscriptionPage();

    await screen.findByText('Subscription & Billing');
    await user.click(screen.getByRole('button', { name: /annual/i }));
    await user.click(screen.getByRole('button', { name: /^switch to yearly$/i }));
    await user.click(screen.getByRole('button', { name: /confirm and pay/i }));

    await waitFor(() => {
      expect(mockBillingService.switchBillingCycle).toHaveBeenCalledWith('annual');
    });
    expect(mockRedirectToPayment).toHaveBeenCalledWith('https://snap.midtrans.example/pay');
  });

  it('opens outstanding invoice payment in the current tab', async () => {
    const user = userEvent.setup();
    mockBillingService.initiatePayment.mockResolvedValue({
      payment_url: 'https://snap.midtrans.example/invoice',
      snap_token: 'snap-token',
    });
    renderMonthlySubscriptionPage([
      {
        id: 'invoice-1',
        invoice_number: 'INV-202605-000001',
        amount_idr: 299000,
        billing_interval: 'monthly',
        period_start: '2026-05-19T00:00:00Z',
        period_end: '2026-06-19T00:00:00Z',
        due_at: '2026-05-19T00:00:00Z',
        status: 'pending',
        created_at: '2026-05-19T00:00:00Z',
      },
    ]);

    await screen.findByText('INV-202605-000001');
    await user.click(screen.getByRole('button', { name: /pay now/i }));

    await waitFor(() => {
      expect(mockBillingService.initiatePayment).toHaveBeenCalledWith('invoice-1');
    });
    expect(mockRedirectToPayment).toHaveBeenCalledWith('https://snap.midtrans.example/invoice');
  });

  it('renders long invoice numbers inside summary cards without truncating them', async () => {
    const pendingInvoiceNumber = 'INV-202605-1779202312863';
    const paidInvoiceNumber = 'INV-202605-1779202312864';

    renderMonthlySubscriptionPage([
      {
        id: 'invoice-pending',
        invoice_number: pendingInvoiceNumber,
        amount_idr: 299000,
        billing_interval: 'monthly',
        period_start: '2026-05-19T00:00:00Z',
        period_end: '2026-06-19T00:00:00Z',
        due_at: '2026-05-19T00:00:00Z',
        status: 'pending',
        created_at: '2026-05-19T00:00:00Z',
      },
      {
        id: 'invoice-paid',
        invoice_number: paidInvoiceNumber,
        amount_idr: 299000,
        billing_interval: 'monthly',
        period_start: '2026-04-19T00:00:00Z',
        period_end: '2026-05-19T00:00:00Z',
        due_at: '2026-04-19T00:00:00Z',
        status: 'paid',
        paid_at: '2026-04-19T00:00:00Z',
        created_at: '2026-04-19T00:00:00Z',
      },
    ]);

    const pendingInvoiceLink = await screen.findByText(pendingInvoiceNumber);
    const paidInvoiceLink = await screen.findByText(paidInvoiceNumber);

    expect(pendingInvoiceLink).toHaveClass('max-w-full', 'min-w-0', 'break-all', 'font-mono');
    expect(paidInvoiceLink).toHaveClass('max-w-full', 'min-w-0', 'break-all', 'font-mono');
  });

  it('shows a neutral Midtrans return message and refreshes billing data', async () => {
    window.history.pushState({}, '', '/subscription?payment_return=midtrans');
    renderMonthlySubscriptionPage();

    expect(await screen.findByText(/payment returned from midtrans/i)).toBeInTheDocument();
    expect(mockBillingService.getInvoices).toHaveBeenCalled();
    expect(refreshSubscriptionMock).toHaveBeenCalledWith({ force: true });
  });

  it('lets cashiers view billing without payment or plan-change actions', async () => {
    mockUseAuth.mockReturnValue({
      logout: jest.fn(),
      user: { role: 'cashier' },
    } as any);

    renderMonthlySubscriptionPage([
      {
        id: 'invoice-1',
        invoice_number: 'INV-202605-000001',
        amount_idr: 299000,
        billing_interval: 'monthly',
        period_start: '2026-05-19T00:00:00Z',
        period_end: '2026-06-19T00:00:00Z',
        due_at: '2026-05-19T00:00:00Z',
        status: 'pending',
        created_at: '2026-05-19T00:00:00Z',
      },
    ]);

    expect(await screen.findByText('INV-202605-000001')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /pay now/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /annual/i })).not.toBeInTheDocument();
    expect(screen.getByText(/managed by owners and managers/i)).toBeInTheDocument();
  });
});
