import apiClient from './api';

export interface MidtransConfig {
  tenant_id: string;
  server_key: string;
  client_key: string;
  merchant_id: string;
  environment: 'sandbox' | 'production';
  is_configured: boolean;
}

export interface UpdateMidtransConfigRequest {
  server_key: string;
  client_key: string;
  merchant_id: string;
  environment: 'sandbox' | 'production';
}

class PaymentService {
  /**
   * Get Midtrans configuration for a tenant
   */
  async getMidtransConfig(tenantId: string): Promise<MidtransConfig> {
    return apiClient.get<MidtransConfig>(
      `/api/v1/admin/tenants/${tenantId}/midtrans-config`
    );
  }

  /**
   * Update Midtrans configuration for a tenant
   */
  async updateMidtransConfig(
    tenantId: string,
    config: UpdateMidtransConfigRequest
  ): Promise<{ success: boolean; message: string }> {
    return apiClient.patch(
      `/api/v1/admin/tenants/${tenantId}/midtrans-config`,
      config
    );
  }
}

export default new PaymentService();
