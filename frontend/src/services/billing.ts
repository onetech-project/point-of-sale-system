import apiClient from './api';

export interface BillingSubscription {
  subscription_status: 'trial' | 'active' | 'grace_period' | 'expired' | 'cancelled';
  subscription_plan: string;
  billing_cycle: 'monthly' | 'annual';
  trial_ends_at?: string;
  subscription_ends_at?: string;
  retention_started_at?: string;
  retention_cleanup_at?: string;
  data_anonymized_at?: string;
  payment_due_at?: string;
  current_invoice?: BillingInvoice;
  is_active: boolean;
  days_remaining: number;
  tenant_id: string;
}

export interface BillingSubscriptionUI extends BillingSubscription {
  status: BillingSubscription['subscription_status'];
  billing_interval: BillingSubscription['billing_cycle'];
  trial_days_remaining?: number;
  invoice_count: number;
}

export interface BillingInvoice {
  id: string;
  invoice_number: string;
  amount_idr: number;
  billing_interval: 'monthly' | 'annual';
  period_start: string;
  period_end: string;
  due_at: string;
  status: 'paid' | 'pending' | 'expired' | 'cancelled';
  paid_at?: string;
  payment_url?: string;
  midtrans_order_id?: string;
  payment_link_expires_at?: string;
  created_at: string;
}

export interface BillingPaymentResponse {
  invoice: BillingInvoice;
  payment_url: string;
  snap_token: string;
}

export interface BillingCycleSwitchResponse extends BillingPaymentResponse {
  status: 'payment_required';
  subscription_status: BillingSubscription['subscription_status'];
  current_billing_interval: BillingSubscription['billing_cycle'];
  billing_interval: BillingSubscription['billing_cycle'];
  subscription_ends_at?: string;
  due_at: string;
  payment_link_expires_at?: string;
}

export interface BillingPaymentAttempt {
  id: string;
  invoice_id: string;
  tenant_id: string;
  amount_idr: number;
  midtrans_order_id?: string;
  payment_method?: string;
  status: 'pending' | 'completed' | 'failed';
  error_msg?: string;
  created_at: string;
}

export interface SupportTicketRequest {
  subject: string;
  description: string;
  category?: 'billing' | 'technical' | 'account' | 'other';
  priority?: 'low' | 'normal' | 'high' | 'urgent';
}

export interface PublicPlans {
  monthly_price_idr: number;
  annual_discount_pct: number;
  trial_days: number;
}

export const billingService = {
  async getSubscription(): Promise<BillingSubscriptionUI> {
    const response = await apiClient.get<BillingSubscription>('/api/v1/billing/subscription');
    return {
      ...response,
      status: response.subscription_status,
      billing_interval: response.billing_cycle,
      trial_days_remaining: response.days_remaining,
      invoice_count: 0,
    };
  },

  async upgradeSubscription(
    billingInterval: 'monthly' | 'annual'
  ): Promise<BillingPaymentResponse> {
    const response = await apiClient.post<{
      invoice?: BillingInvoice;
      invoice_id?: string;
      invoice_number?: string;
      amount_idr?: number;
      payment_url: string;
      snap_token: string;
    }>('/api/v1/billing/subscription/upgrade', { billing_interval: billingInterval });
    return {
      invoice: response.invoice ?? {
        id: response.invoice_id ?? '',
        invoice_number: response.invoice_number ?? '',
        amount_idr: response.amount_idr ?? 0,
        billing_interval: billingInterval,
        period_start: '',
        period_end: '',
        due_at: '',
        status: 'pending',
        payment_url: response.payment_url,
        created_at: '',
      },
      payment_url: response.payment_url,
      snap_token: response.snap_token,
    };
  },

  async switchBillingCycle(
    billingInterval: 'monthly' | 'annual'
  ): Promise<BillingCycleSwitchResponse> {
    const response = await apiClient.post<BillingCycleSwitchResponse>(
      '/api/v1/billing/subscription/cycle-switch',
      { billing_interval: billingInterval }
    );
    return response;
  },

  async updateBillingCycle(billingInterval: 'monthly' | 'annual'): Promise<void> {
    await apiClient.put('/api/v1/billing/subscription/cycle', {
      billing_interval: billingInterval,
    });
  },

  async getInvoices(): Promise<BillingInvoice[]> {
    const response = await apiClient.get<BillingInvoice[]>('/api/v1/billing/invoices');
    return response;
  },

  async getInvoice(id: string): Promise<BillingInvoice> {
    const response = await apiClient.get<BillingInvoice>(`/api/v1/billing/invoices/${id}`);
    return response;
  },

  async getInvoicePaymentAttempts(id: string): Promise<BillingPaymentAttempt[]> {
    const response = await apiClient.get<BillingPaymentAttempt[]>(
      `/api/v1/billing/invoices/${id}/payments`
    );
    return response;
  },

  async initiatePayment(invoiceId: string): Promise<{ payment_url: string; snap_token: string }> {
    const response = await apiClient.post<{ payment_url: string; snap_token: string }>(
      `/api/v1/billing/invoices/${invoiceId}/pay`,
      {}
    );
    return response;
  },

  async getPublicPlans(): Promise<PublicPlans> {
    const response = await apiClient.get<PublicPlans>('/api/v1/public/plans');
    return response;
  },

  async createSupportTicket(request: SupportTicketRequest): Promise<{ id: string }> {
    const response = await apiClient.post<{ id: string }>('/api/v1/platform/tickets', request);
    return response;
  },
};
