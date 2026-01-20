/**
 * Tenant Types for UU PDP Compliance
 * Includes business profile, team members, and configuration data
 */

export interface Tenant {
  id: string;
  business_name: string;
  slug: string;
  status: 'active' | 'inactive' | 'suspended' | 'deleted';
  settings?: Record<string, any>;
  storage_used_bytes?: number;
  storage_quota_bytes?: number;
  created_at: string;
  updated_at?: string;
}

export interface TenantInfo {
  id: string;
  businessName: string;
  slug: string;
  status: string;
  createdAt: string;
}

export interface TeamMember {
  id: string;
  email: string;
  role: 'owner' | 'manager' | 'cashier';
  status: 'active' | 'invited' | 'suspended' | 'deleted';
  first_name?: string;
  last_name?: string;
  locale: string;
  created_at: string;
  last_login_at?: string;
}

export interface TenantConfiguration {
  enabled_delivery_types: string[];
  service_area: Record<string, any>;
  delivery_fee_config: Record<string, any>;
  auto_calculate_fees: boolean;
  midtrans_configured: boolean;
  midtrans_environment: 'sandbox' | 'production';
}

export interface TenantData {
  tenant: Tenant;
  team_members: TeamMember[];
  configuration: TenantConfiguration;
}

export interface TenantConfig {
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

export interface DeleteUserResponse {
  message: string;
  user_id: string;
  delete_type: 'soft' | 'hard';
  retention_days?: number;
  permanent_deletion_after?: string;
}
