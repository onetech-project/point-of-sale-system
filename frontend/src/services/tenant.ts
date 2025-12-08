import apiClient from './api';
import { TenantConfig } from '../types/cart';

export interface TenantInfo {
  tenant_id: string;
  tenant_name: string;
  logo_url?: string;
  description?: string;
  enabled_delivery_types: string[];
  auto_calculate_fees: boolean;
  service_area?: Record<string, any>;
  delivery_fee_config?: Record<string, any>;
  default_delivery_fee?: number;
  min_order_amount?: number;
  estimated_prep_time?: number;
  charge_delivery_fee?: boolean;
}

export const tenant = {
  /**
   * Get tenant configuration by ID
   * Includes delivery types, branding, and operational settings
   */
  async getTenantConfig(tenantId: string): Promise<TenantInfo> {
    const response = await apiClient.get<TenantInfo>(
      `/api/public/tenants/${tenantId}/config`
    );
    return response;
  },

  /**
   * Check if tenant is active and accessible
   */
  async validateTenant(tenantId: string): Promise<boolean> {
    try {
      await this.getTenantConfig(tenantId);
      return true;
    } catch (error: any) {
      if (error.response?.status === 404 || error.response?.status === 403) {
        return false;
      }
      throw error;
    }
  },
};
