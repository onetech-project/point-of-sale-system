import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import PlatformDashboardPage from './dashboard/page';
import PlatformTenantsPage from './tenants/page';
import PlatformTenantDetailPage from './tenants/[tenantId]/page';
import PlatformRevenuePage from './revenue/page';
import PlatformAuditPage from './audit/page';
import { platformService } from '@/services/platform';

jest.mock('next/navigation', () => ({
  useParams: () => ({ tenantId: 'tenant-1' }),
  usePathname: () => '/platform/tenants/tenant-1',
  useRouter: () => ({ replace: jest.fn() }),
}));

jest.mock('@/components/platform/PlatformShell', () => ({
  PlatformShell: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  PlatformMetric: ({ label, value }: { label: string; value: string | number }) => (
    <div>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  ),
}));

jest.mock('@/services/platform', () => ({
  platformService: {
    getOverview: jest.fn(),
    getRevenue: jest.fn(),
    listTenants: jest.fn(),
    getTenant: jest.fn(),
    tenantAction: jest.fn(),
    listAuditEvents: jest.fn(),
  },
}));

const mockPlatformService = platformService as jest.Mocked<typeof platformService>;

const overview = {
  start_date: '2026-05-01',
  end_date: '2026-05-20',
  registered_tenants: 8,
  active_tenants: 5,
  active_users: 12,
  recently_active_users: 7,
  open_tickets: 2,
  high_priority_tickets: 1,
  grace_period_tenants: 1,
  expired_tenants: 1,
  trial_tenants: 2,
  will_expire_7: 1,
  will_expire_14: 3,
  will_expire_30: 4,
  storage_used_bytes: 1024,
  storage_quota_bytes: 2048,
  billing_income_idr: 300000,
  paid_income_idr: 300000,
  pending_income_idr: 100000,
  expired_income_idr: 0,
  overdue_income_idr: 50000,
  previous_paid_income_idr: 200000,
  period_delta_percent: 50,
  invoice_count_by_status: { paid: 3 },
  billing_cycle_mix: { monthly: 2, annual: 1 },
  revenue_timeseries: [{ date: '2026-05-20', paid_income_idr: 300000, invoice_count: 3 }],
  top_paying_tenants: [{ tenant_id: 'tenant-1', business_name: 'Bistro One', paid_income_idr: 300000, invoice_count: 3 }],
  tenant_health: {
    registered_tenants: 8,
    active_tenants: 5,
    trial_tenants: 2,
    grace_period_tenants: 1,
    will_expire_7: 1,
    will_expire_14: 3,
    will_expire_30: 4,
    expired_tenants: 1,
    cancelled_tenants: 0,
    suspended_tenants: 1,
    inactive_tenants: 1,
    scheduled_deletes: 0,
  },
  urgent_items: [],
  tenant_sales_gmv_idr: 1000000,
  tickets_by_status: { open: 2 },
  top_sales_tenants: [],
};

const tenant = {
  id: 'tenant-1',
  business_name: 'Bistro One',
  slug: 'bistro-one',
  status: 'active',
  owner_email: 'ow***@example.com',
  subscription_plan: 'starter',
  billing_cycle: 'monthly',
  subscription_status: 'active',
  subscription_ends_at: '2026-06-20T00:00:00Z',
  expires_at: '2026-06-20T00:00:00Z',
  storage_used_bytes: 1024,
  storage_quota_bytes: 2048,
  user_count: 4,
  active_user_count: 3,
  paid_invoice_total_idr: 300000,
  open_ticket_count: 1,
  last_active_at: '2026-05-20T00:00:00Z',
  created_at: '2026-05-01T00:00:00Z',
  updated_at: '2026-05-20T00:00:00Z',
} as const;

beforeEach(() => {
  jest.clearAllMocks();
  mockPlatformService.getOverview.mockResolvedValue(overview as any);
  mockPlatformService.getRevenue.mockResolvedValue({
    start_date: '2026-05-01',
    end_date: '2026-05-20',
    paid_income_idr: 300000,
    previous_paid_income_idr: 200000,
    period_delta_percent: 50,
    pending_income_idr: 100000,
    expired_income_idr: 0,
    overdue_income_idr: 50000,
    invoice_count_by_status: { paid: 3, pending: 1 },
    billing_cycle_mix: { monthly: 2, annual: 1 },
    revenue_timeseries: [{ date: '2026-05-20', paid_income_idr: 300000, invoice_count: 3 }],
    top_paying_tenants: [{ tenant_id: 'tenant-1', business_name: 'Bistro One', paid_income_idr: 300000, invoice_count: 3 }],
    overdue_invoices: [],
  } as any);
  mockPlatformService.listTenants.mockResolvedValue({ tenants: [tenant as any], total: 1, limit: 50, offset: 0 });
  mockPlatformService.getTenant.mockResolvedValue({
    tenant,
    billing: {
      invoices: [
        {
          id: 'invoice-1',
          tenant_id: 'tenant-1',
          tenant_name: 'Bistro One',
          invoice_number: 'INV-202605-000001-LONG-REFERENCE',
          amount_idr: 300000,
          billing_interval: 'monthly',
          period_start: '2026-05-01T00:00:00Z',
          period_end: '2026-06-01T00:00:00Z',
          due_at: '2026-05-20T00:00:00Z',
          status: 'pending',
          created_at: '2026-05-01T00:00:00Z',
          updated_at: '2026-05-20T00:00:00Z',
        },
      ],
      payment_attempts: [],
    },
    activity: [],
    tickets: [],
    notes: [],
  } as any);
  mockPlatformService.tenantAction.mockResolvedValue({});
  mockPlatformService.listAuditEvents.mockResolvedValue({
    events: [
      {
        type: 'audit',
        action: 'LOGIN',
        actor_type: 'user',
        resource: 'Bistro One - authentication:user-1',
        occurred_at: '2026-05-20T00:00:00Z',
      },
    ],
    limit: 100,
    offset: 0,
  } as any);
});

it('renders platform dashboard command-center metrics', async () => {
  render(<PlatformDashboardPage />);

  expect(await screen.findByText('Platform Command Center')).toBeInTheDocument();
  expect(screen.getByText('MTD subscription income')).toBeInTheDocument();
  expect(screen.getByText('Will expire in 14d')).toBeInTheDocument();
  expect(screen.getByText('No urgent platform work right now.')).toBeInTheDocument();
  expect(mockPlatformService.getOverview).toHaveBeenCalledWith({ range: 'month' });
});

it('applies tenant registry filters through the platform service', async () => {
  const user = userEvent.setup();
  render(<PlatformTenantsPage />);

  await screen.findByText('Bistro One');
  await user.type(screen.getByPlaceholderText('Search business or slug'), 'Bistro');
  await user.selectOptions(screen.getByLabelText('Subscription'), 'grace_period');
  await user.selectOptions(screen.getByLabelText('Plan'), 'professional');
  await user.selectOptions(screen.getByLabelText('Expires'), '14');
  await user.click(screen.getByRole('button', { name: /apply filters/i }));

  await waitFor(() => {
    expect(mockPlatformService.listTenants).toHaveBeenLastCalledWith(
      expect.objectContaining({
        q: 'Bistro',
        subscription_status: 'grace_period',
        plan: 'professional',
        expires_within_days: '14',
        offset: 0,
      })
    );
  });
});

it('confirms tenant lifecycle actions with an audit reason', async () => {
  const user = userEvent.setup();
  render(<PlatformTenantDetailPage />);

  await screen.findByText('Bistro One');
  await user.click(screen.getByRole('button', { name: /^suspend$/i }));
  expect(screen.getByRole('dialog', { name: /suspend tenant/i })).toBeInTheDocument();
  expect(screen.getByText(/terminates active sessions/i)).toBeInTheDocument();

  await user.type(screen.getByPlaceholderText(/required reason/i), 'Billing abuse review');
  await user.click(screen.getByRole('button', { name: /confirm/i }));

  await waitFor(() => {
    expect(mockPlatformService.tenantAction).toHaveBeenCalledWith('tenant-1', 'suspend', {
      reason: 'Billing abuse review',
    });
  });
});

it('renders tenant detail when optional collections are null or missing', async () => {
  mockPlatformService.getTenant.mockResolvedValueOnce({
    tenant,
    billing: {
      invoices: null,
      payment_attempts: null,
    },
    activity: null,
    tickets: null,
    notes: null,
  } as any);

  render(<PlatformTenantDetailPage />);

  expect(await screen.findByText('Bistro One')).toBeInTheDocument();
  expect(screen.getByText('No platform notes yet.')).toBeInTheDocument();
  expect(screen.getByText('No invoices.')).toBeInTheDocument();
  expect(screen.getByText('No support tickets.')).toBeInTheDocument();
  expect(screen.getByText('No lifecycle activity yet.')).toBeInTheDocument();
});

it('requests year-to-date revenue when the YTD filter is selected', async () => {
  const user = userEvent.setup();
  render(<PlatformRevenuePage />);

  await screen.findByText('Subscription Revenue');
  expect(mockPlatformService.getRevenue).toHaveBeenCalledWith({ range: 'month' });

  await user.click(screen.getByRole('button', { name: 'YTD' }));

  await waitFor(() => {
    expect(mockPlatformService.getRevenue).toHaveBeenLastCalledWith({ range: 'year' });
  });
});

it('renders immutable audit events on the platform audit page', async () => {
  render(<PlatformAuditPage />);

  expect(await screen.findByText('LOGIN')).toBeInTheDocument();
  expect(screen.getByText('Bistro One - authentication:user-1')).toBeInTheDocument();
  expect(screen.getByText('user')).toBeInTheDocument();
  expect(mockPlatformService.listAuditEvents).toHaveBeenCalledWith({
    tenant_id: '',
    admin_id: '',
    action: '',
    limit: 100,
  });
});
