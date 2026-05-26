export interface Product {
  id: string;
  tenant_id: string;
  sku: string;
  name: string;
  description?: string;
  category_id?: string;
  category_name?: string;
  selling_price: number;
  cost_price: number;
  tax_rate: number;
  stock_quantity: number;
  photo_path?: string;
  photo_size?: number;
  archived_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Category {
  id: string;
  tenant_id: string;
  name: string;
  display_order: number;
  created_at: string;
  updated_at: string;
}

export interface StockAdjustment {
  id: string;
  tenant_id: string;
  product_id: string;
  user_id: string;
  previous_quantity: number;
  new_quantity: number;
  quantity_delta: number;
  reason: StockAdjustmentReason;
  notes?: string;
  created_at: string;
  adjusted_by?: string;
}

export type StockAdjustmentReason =
  | 'supplier_delivery'
  | 'physical_count'
  | 'shrinkage'
  | 'damage'
  | 'return'
  | 'correction'
  | 'sale';

export interface CreateProductRequest {
  sku: string;
  name: string;
  description?: string;
  category_id?: string;
  selling_price: number;
  cost_price: number;
  tax_rate?: number;
  stock_quantity?: number;
}

export interface UpdateProductRequest {
  sku?: string;
  name?: string;
  description?: string;
  category_id?: string;
  selling_price?: number;
  cost_price?: number;
  tax_rate?: number;
}

export interface CreateCategoryRequest {
  name: string;
  display_order?: number;
}

export interface UpdateCategoryRequest {
  name?: string;
  display_order?: number;
}

export interface StockAdjustmentRequest {
  new_quantity: number;
  reason: StockAdjustmentReason;
  notes?: string;
}

export interface ProductListParams {
  page?: number;
  limit?: number;
  search?: string;
  category_id?: string;
  low_stock?: boolean;
  archived?: boolean;
}

export interface InventorySummary {
  total_products: number;
  total_value: number;
  low_stock_count: number;
  out_of_stock_count: number;
  categories_count: number;
}

export type ProductImportTemplateFormat = 'csv' | 'xlsx';

export type ProductImportStatus =
  | 'pending'
  | 'queued'
  | 'processing'
  | 'completed'
  | 'success'
  | 'succeeded'
  | 'completed_with_errors'
  | 'partial_success'
  | 'failed'
  | 'error'
  | 'cancelled'
  | 'canceled'
  | (string & {});

export interface ProductImportStartResponse {
  import_id: string;
  status: ProductImportStatus;
}

export interface ProductImportTemplateDownload {
  blob: Blob;
  filename: string;
}

export interface ProductImportSummary {
  total_rows?: number;
  processed_rows?: number;
  imported_rows?: number;
  imported_count?: number;
  created_count?: number;
  updated_count?: number;
  skipped_count?: number;
  failed_rows?: number;
  failed_count?: number;
  error_count?: number;
  warning_count?: number;
  [key: string]: string | number | boolean | null | undefined;
}

export interface ProductImportRowIssue {
  row?: number;
  row_number?: number;
  line?: number;
  sku?: string;
  field?: string;
  value?: string;
  message?: string;
  error?: string;
  warning?: string;
  reason?: string;
  [key: string]: string | number | boolean | null | undefined;
}

export interface ProductImportStatusResponse {
  id?: string;
  import_id?: string;
  status: ProductImportStatus;
  summary?: ProductImportSummary | null;
  row_errors?: ProductImportRowIssue[] | null;
  errors?: ProductImportRowIssue[] | null;
  warnings?: ProductImportRowIssue[] | null;
  message?: string;
  created_at?: string;
  updated_at?: string;
  completed_at?: string;
}

export interface ProductImportUploadOptions {
  onUploadProgress?: (progress: number) => void;
  signal?: AbortSignal;
}

export interface ProductImportPollOptions {
  intervalMs?: number;
  timeoutMs?: number;
  signal?: AbortSignal;
  onStatus?: (status: ProductImportStatusResponse) => void;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}
