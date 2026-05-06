import apiClient from './api';

export interface BillingSubscription {
  subscription_status: 'trial' | 'active' | 'grace_period' | 'expired';
  subscription_plan: string;
  billing_cycle: 'monthly' | 'annual';
  trial_ends_at?: string;
  subscription_ends_at?: string;
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
  status: 'paid' | 'pending' | 'expired' | 'cancelled';
  paid_at?: string;
  payment_url?: string;
  created_at: string;
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
  ): Promise<{ invoice: BillingInvoice; payment_url: string; snap_token: string }> {
    const response = await apiClient.post<{
      invoice: BillingInvoice;
      payment_url: string;
      snap_token: string;
    }>('/api/v1/billing/subscription/upgrade', { billing_interval: billingInterval });
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
};
