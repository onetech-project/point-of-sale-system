import apiClient from './api';

export interface PlatformAdmin {
  id: string;
  email: string;
  name: string;
  role: 'platform_owner' | 'platform_operator';
  status: 'active' | 'suspended';
}

export interface TenantHealthSummary {
  registered_tenants: number;
  active_tenants: number;
  trial_tenants: number;
  grace_period_tenants: number;
  will_expire_7: number;
  will_expire_14: number;
  will_expire_30: number;
  expired_tenants: number;
  cancelled_tenants: number;
  suspended_tenants: number;
  inactive_tenants: number;
  scheduled_deletes: number;
}

export interface RevenuePoint {
  date: string;
  paid_income_idr: number;
  invoice_count: number;
}

export interface TopPayingTenant {
  tenant_id: string;
  business_name: string;
  paid_income_idr: number;
  invoice_count: number;
}

export interface TenantInvoice {
  id: string;
  tenant_id: string;
  tenant_name?: string;
  invoice_number: string;
  amount_idr: number;
  billing_interval: string;
  period_start: string;
  period_end: string;
  due_at: string;
  status: string;
  paid_at?: string;
  midtrans_order_id?: string;
  payment_url?: string;
  payment_link_expires_at?: string;
  created_at: string;
  updated_at: string;
}

export interface PaymentAttempt {
  id: string;
  invoice_id: string;
  tenant_id: string;
  amount_idr: number;
  midtrans_order_id?: string;
  payment_method?: string;
  status: string;
  error_msg?: string;
  created_at: string;
}

export interface RevenueSummary {
  start_date: string;
  end_date: string;
  paid_income_idr: number;
  previous_paid_income_idr: number;
  period_delta_percent?: number;
  pending_income_idr: number;
  expired_income_idr: number;
  overdue_income_idr: number;
  invoice_count_by_status: Record<string, number>;
  billing_cycle_mix: Record<string, number>;
  revenue_timeseries: RevenuePoint[];
  top_paying_tenants: TopPayingTenant[];
  overdue_invoices: TenantInvoice[];
}

export interface UrgentItem {
  type: string;
  tenant_id: string;
  business_name: string;
  label: string;
  severity: 'default' | 'warning' | 'danger' | string;
  due_at?: string;
  amount_idr?: number;
}

export interface PlatformOverview {
  start_date: string;
  end_date: string;
  registered_tenants: number;
  active_tenants: number;
  active_users: number;
  recently_active_users: number;
  open_tickets: number;
  high_priority_tickets: number;
  grace_period_tenants: number;
  expired_tenants: number;
  trial_tenants: number;
  will_expire_7: number;
  will_expire_14: number;
  will_expire_30: number;
  storage_used_bytes: number;
  storage_quota_bytes: number;
  billing_income_idr: number;
  paid_income_idr: number;
  pending_income_idr: number;
  expired_income_idr: number;
  overdue_income_idr: number;
  previous_paid_income_idr: number;
  period_delta_percent?: number;
  invoice_count_by_status: Record<string, number>;
  billing_cycle_mix: Record<string, number>;
  revenue_timeseries: RevenuePoint[];
  top_paying_tenants: TopPayingTenant[];
  tenant_health: TenantHealthSummary;
  urgent_items: UrgentItem[];
  tenant_sales_gmv_idr: number;
  tickets_by_status: Record<string, number>;
  top_sales_tenants: Array<{
    tenant_id: string;
    business_name: string;
    tenant_sales_gmv_idr: number;
    orders: number;
  }>;
}

export interface PlatformTenant {
  id: string;
  business_name: string;
  slug: string;
  status: 'active' | 'inactive' | 'suspended' | 'deleted';
  owner_email?: string;
  subscription_plan: string;
  billing_cycle: string;
  subscription_status: string;
  trial_ends_at?: string;
  subscription_ends_at?: string;
  expires_at?: string;
  storage_used_bytes: number;
  storage_quota_bytes: number;
  user_count: number;
  active_user_count: number;
  paid_invoice_total_idr: number;
  open_ticket_count: number;
  last_active_at?: string;
  created_at: string;
  updated_at: string;
  suspended_at?: string;
  deactivated_at?: string;
  scheduled_delete_at?: string;
  status_reason?: string;
}

export interface TenantActivity {
  type: string;
  action: string;
  actor_type?: string;
  resource?: string;
  reason?: string;
  before_status?: string;
  after_status?: string;
  occurred_at: string;
  delete_after?: string;
}

export interface TenantNote {
  id: string;
  tenant_id: string;
  platform_admin_id?: string;
  body: string;
  created_at: string;
}

export interface TenantBillingDetail {
  invoices: TenantInvoice[];
  payment_attempts: PaymentAttempt[];
}

export interface PlatformTenantDetail {
  tenant: PlatformTenant;
  billing: TenantBillingDetail;
  activity: TenantActivity[];
  tickets: PlatformTicket[];
  notes: TenantNote[];
}

export interface PlatformTicket {
  id: string;
  tenant_id: string;
  tenant_name?: string;
  subject: string;
  description: string;
  category: string;
  priority: string;
  status: string;
  assigned_to_platform_admin_id?: string;
  created_at: string;
  updated_at: string;
  last_activity_at: string;
  resolved_at?: string;
}

export interface TenantActionPayload {
  reason: string;
  subscription_plan?: string;
  billing_cycle?: string;
  extend_days?: number;
  note?: string;
  ticket_id?: string;
  assigned_to_platform_admin_id?: string;
}

export type TenantAction =
  | 'suspend'
  | 'deactivate'
  | 'delete'
  | 'reactivate'
  | 'cancel-delete'
  | 'change-plan'
  | 'change-billing-cycle'
  | 'extend-trial'
  | 'extend-grace'
  | 'regenerate-payment-link'
  | 'add-note'
  | 'assign-support-owner';

function queryString(params?: Record<string, string | number | undefined>): string {
  const query = new URLSearchParams();
  Object.entries(params ?? {}).forEach(([key, value]) => {
    if (value !== undefined && value !== '') query.set(key, String(value));
  });
  const suffix = query.toString();
  return suffix ? `?${suffix}` : '';
}

function normalizeTenantDetail(response: PlatformTenantDetail): PlatformTenantDetail {
  const billing = response.billing ?? { invoices: [], payment_attempts: [] };
  return {
    ...response,
    billing: {
      ...billing,
      invoices: billing.invoices ?? [],
      payment_attempts: billing.payment_attempts ?? [],
    },
    activity: response.activity ?? [],
    tickets: response.tickets ?? [],
    notes: response.notes ?? [],
  };
}

export const platformService = {
  async login(email: string, password: string): Promise<{ admin: PlatformAdmin }> {
    return apiClient.post('/api/v1/platform/auth/login', { email, password });
  },

  async session(): Promise<{ admin: PlatformAdmin }> {
    return apiClient.get('/api/v1/platform/auth/session');
  },

  async logout(): Promise<void> {
    await apiClient.post('/api/v1/platform/auth/logout');
  },

  async getOverview(params?: { range?: string; start_date?: string; end_date?: string }): Promise<PlatformOverview> {
    return apiClient.get(`/api/v1/platform/analytics/overview${queryString(params)}`);
  },

  async getRevenue(params?: { range?: string; start_date?: string; end_date?: string }): Promise<RevenueSummary> {
    return apiClient.get(`/api/v1/platform/analytics/revenue${queryString(params)}`);
  },

  async getTenantHealth(): Promise<TenantHealthSummary> {
    return apiClient.get('/api/v1/platform/analytics/tenant-health');
  },

  async listTenants(params?: {
    status?: string;
    subscription_status?: string;
    plan?: string;
    expires_within_days?: number | string;
    q?: string;
    limit?: number;
    offset?: number;
    sort?: string;
  }): Promise<{ tenants: PlatformTenant[]; total: number; limit: number; offset: number }> {
    return apiClient.get(`/api/v1/platform/tenants${queryString(params)}`);
  },

  async getTenant(id: string): Promise<PlatformTenantDetail> {
    const response = await apiClient.get<PlatformTenantDetail>(`/api/v1/platform/tenants/${id}`);
    return normalizeTenantDetail(response);
  },

  async getTenantBilling(id: string): Promise<TenantBillingDetail> {
    return apiClient.get(`/api/v1/platform/tenants/${id}/billing`);
  },

  async tenantAction(id: string, action: TenantAction, data: TenantActionPayload): Promise<Record<string, unknown>> {
    return apiClient.post(`/api/v1/platform/tenants/${id}/actions/${action}`, data);
  },

  async listAuditEvents(params?: {
    tenant_id?: string;
    admin_id?: string;
    action?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ events: TenantActivity[]; limit: number; offset: number }> {
    return apiClient.get(`/api/v1/platform/audit-events${queryString(params)}`);
  },

  async listTickets(params?: { status?: string; tenant_id?: string }): Promise<{ tickets: PlatformTicket[] }> {
    return apiClient.get(`/api/v1/platform/tickets${queryString(params)}`);
  },

  async updateTicket(
    id: string,
    data: { status?: string; priority?: string; assigned_to_platform_admin_id?: string }
  ): Promise<void> {
    await apiClient.patch(`/api/v1/platform/tickets/${id}`, data);
  },

  async addTicketNote(id: string, body: string): Promise<void> {
    await apiClient.post(`/api/v1/platform/tickets/${id}/notes`, { body });
  },
};
