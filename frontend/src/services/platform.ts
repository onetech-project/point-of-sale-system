import apiClient from './api';

export interface PlatformAdmin {
  id: string;
  email: string;
  name: string;
  role: 'platform_owner' | 'platform_operator';
  status: 'active' | 'suspended';
}

export interface PlatformOverview {
  start_date: string;
  end_date: string;
  active_tenants: number;
  active_users: number;
  recently_active_users: number;
  grace_period_tenants: number;
  expired_tenants: number;
  billing_income_idr: number;
  tenant_sales_gmv_idr: number;
  open_tickets: number;
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
  subscription_plan: string;
  billing_cycle: string;
  subscription_status: string;
  trial_ends_at?: string;
  subscription_ends_at?: string;
  storage_used_bytes: number;
  storage_quota_bytes: number;
  user_count: number;
  paid_invoice_total_idr: number;
  open_ticket_count: number;
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

  async getOverview(): Promise<PlatformOverview> {
    return apiClient.get('/api/v1/platform/analytics/overview');
  },

  async listTenants(params?: {
    status?: string;
    subscription_status?: string;
    q?: string;
  }): Promise<{ tenants: PlatformTenant[] }> {
    const query = new URLSearchParams();
    if (params?.status) query.set('status', params.status);
    if (params?.subscription_status) query.set('subscription_status', params.subscription_status);
    if (params?.q) query.set('q', params.q);
    const suffix = query.toString() ? `?${query.toString()}` : '';
    return apiClient.get(`/api/v1/platform/tenants${suffix}`);
  },

  async getTenant(id: string): Promise<{ tenant: PlatformTenant; activity: TenantActivity[] }> {
    return apiClient.get(`/api/v1/platform/tenants/${id}`);
  },

  async tenantAction(
    id: string,
    action: 'suspend' | 'deactivate' | 'delete' | 'reactivate' | 'cancel-delete',
    reason: string
  ): Promise<{ tenant_id: string; status: string; delete_after?: string }> {
    return apiClient.post(`/api/v1/platform/tenants/${id}/actions/${action}`, { reason });
  },

  async listTickets(params?: { status?: string; tenant_id?: string }): Promise<{ tickets: PlatformTicket[] }> {
    const query = new URLSearchParams();
    if (params?.status) query.set('status', params.status);
    if (params?.tenant_id) query.set('tenant_id', params.tenant_id);
    const suffix = query.toString() ? `?${query.toString()}` : '';
    return apiClient.get(`/api/v1/platform/tickets${suffix}`);
  },

  async updateTicket(id: string, data: { status?: string; priority?: string }): Promise<void> {
    await apiClient.patch(`/api/v1/platform/tickets/${id}`, data);
  },

  async addTicketNote(id: string, body: string): Promise<void> {
    await apiClient.post(`/api/v1/platform/tickets/${id}/notes`, { body });
  },
};
