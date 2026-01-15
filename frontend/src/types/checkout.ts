export interface CheckoutData {
  delivery_type: string;
  customer_name: string;
  customer_phone: string;
  customer_email?: string;
  delivery_address?: string;
  table_number?: string;
  notes?: string;
  consents?: string[]
}

export interface TenantConfig {
  tenant_id: string;
  enabled_delivery_types: string[];
  auto_calculate_fees: boolean;
  charge_delivery_fee?: boolean;
  default_delivery_fee?: number;
}
