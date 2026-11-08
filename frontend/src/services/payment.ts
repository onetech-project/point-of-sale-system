import apiClient from './api';

export type PaymentEnvironment = 'sandbox' | 'production';
export type PaymentGatewayName = 'midtrans' | 'manjo';

export interface MidtransConfig {
  tenant_id: string;
  server_key: string;
  client_key: string;
  merchant_id: string;
  environment: PaymentEnvironment;
  is_configured: boolean;
}

export interface UpdateMidtransConfigRequest {
  server_key: string;
  client_key: string;
  merchant_id: string;
  environment: PaymentEnvironment;
}

/** Manjo credentials as returned by the API. Secrets are never sent back. */
export interface ManjoConfig {
  tenant_id: string;
  client_id: string;
  merchant_id: string;
  partner_id: string;
  environment: PaymentEnvironment;
  has_client_secret: boolean;
  has_private_key: boolean;
  is_configured: boolean;
}

/** Empty or omitted client_secret / private_key keep the stored value. */
export interface UpdateManjoConfigRequest {
  client_id: string;
  merchant_id: string;
  partner_id: string;
  environment: PaymentEnvironment;
  client_secret?: string;
  private_key?: string;
}

export interface GatewayStatus {
  configured: boolean;
  environment: PaymentEnvironment;
}

export interface PaymentGatewaySettings {
  tenant_id: string;
  default_gateway: PaymentGatewayName;
  gateways: Record<PaymentGatewayName, GatewayStatus>;
}

class PaymentService {
  /**
   * Get Midtrans configuration for a tenant
   */
  async getMidtransConfig(tenantId: string): Promise<MidtransConfig> {
    return apiClient.get<MidtransConfig>(`/api/v1/admin/tenants/${tenantId}/midtrans-config`);
  }

  /**
   * Update Midtrans configuration for a tenant
   */
  async updateMidtransConfig(
    tenantId: string,
    config: UpdateMidtransConfigRequest
  ): Promise<{ success: boolean; message: string }> {
    return apiClient.patch(`/api/v1/admin/tenants/${tenantId}/midtrans-config`, config);
  }

  /**
   * Get Manjo configuration for a tenant
   */
  async getManjoConfig(tenantId: string): Promise<ManjoConfig> {
    return apiClient.get<ManjoConfig>(`/api/v1/admin/tenants/${tenantId}/manjo-config`);
  }

  /**
   * Update Manjo configuration for a tenant
   */
  async updateManjoConfig(
    tenantId: string,
    config: UpdateManjoConfigRequest
  ): Promise<{ message: string }> {
    return apiClient.patch(`/api/v1/admin/tenants/${tenantId}/manjo-config`, config);
  }

  /**
   * Get the default gateway and the configuration status of every gateway
   */
  async getPaymentGateway(tenantId: string): Promise<PaymentGatewaySettings> {
    return apiClient.get<PaymentGatewaySettings>(
      `/api/v1/admin/tenants/${tenantId}/payment-gateway`
    );
  }

  /**
   * Choose the gateway used for new guest QRIS payments
   */
  async updateDefaultGateway(
    tenantId: string,
    defaultGateway: PaymentGatewayName
  ): Promise<{ message: string }> {
    return apiClient.patch(`/api/v1/admin/tenants/${tenantId}/payment-gateway`, {
      default_gateway: defaultGateway,
    });
  }
}

export default new PaymentService();
