import apiClient from './api';
import {
  Product,
  Category,
  CreateProductRequest,
  UpdateProductRequest,
  StockAdjustmentRequest,
  StockAdjustment,
  ProductListParams,
  PaginatedResponse,
  InventorySummary,
  CreateCategoryRequest,
  UpdateCategoryRequest,
} from '../types/product';

const PRODUCTS_BASE = '/api/v1/products';
const CATEGORIES_BASE = '/api/v1/categories';
const INVENTORY_BASE = '/api/v1/inventory';

class ProductService {
  // ==================== Product Management ====================

  /**
   * Retrieves a paginated list of products
   * @param params - Query parameters for filtering and pagination
   * @returns Paginated list of products with metadata
   */
  async getProducts(params?: ProductListParams): Promise<PaginatedResponse<Product>> {
    const queryParams = new URLSearchParams();
    if (params?.page) queryParams.append('page', params.page.toString());
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.search) queryParams.append('search', params.search);
    if (params?.category_id) queryParams.append('category_id', params.category_id);
    if (params?.low_stock !== undefined)
      queryParams.append('low_stock', params.low_stock.toString());
    if (params?.archived !== undefined) queryParams.append('archived', params.archived.toString());

    const url = queryParams.toString() ? `${PRODUCTS_BASE}?${queryParams}` : PRODUCTS_BASE;
    const response = await apiClient.get<{
      products: Product[];
      total: number;
      limit: number;
      offset: number;
    }>(url);

    // Transform backend response to frontend format
    const limit = response.limit || 20;
    const total = response.total || 0;
    const page = params?.page || 1;

    return {
      data: response.products || [],
      total: total,
      page: page,
      limit: limit,
      total_pages: Math.ceil(total / limit),
    };
  }

  /**
   * Retrieves a single product by ID
   * @param id - Product UUID
   * @returns Product details
   */
  async getProduct(id: string): Promise<Product> {
    return apiClient.get<Product>(`${PRODUCTS_BASE}/${id}`);
  }

  /**
   * Creates a new product
   * @param data - Product creation data including name, SKU, price, etc.
   * @returns Created product with generated ID
   */
  async createProduct(data: CreateProductRequest): Promise<Product> {
    return apiClient.post<Product>(PRODUCTS_BASE, data);
  }

  /**
   * Updates an existing product
   * @param id - Product UUID
   * @param data - Updated product fields
   * @returns Updated product
   */
  async updateProduct(id: string, data: UpdateProductRequest): Promise<Product> {
    return apiClient.put<Product>(`${PRODUCTS_BASE}/${id}`, data);
  }

  /**
   * Permanently deletes a product (only if no sales history)
   * @param id - Product UUID
   * @throws Error if product has sales history
   */
  async deleteProduct(id: string): Promise<void> {
    return apiClient.delete(`${PRODUCTS_BASE}/${id}`);
  }

  /**
   * Archives a product (soft delete)
   * @param id - Product UUID
   * @returns Archived product with archived_at timestamp
   */
  async archiveProduct(id: string): Promise<Product> {
    return apiClient.patch<Product>(`${PRODUCTS_BASE}/${id}/archive`);
  }

  /**
   * Restores an archived product
   * @param id - Product UUID
   * @returns Restored product with archived_at cleared
   */
  async restoreProduct(id: string): Promise<Product> {
    return apiClient.patch<Product>(`${PRODUCTS_BASE}/${id}/restore`);
  }

  // ==================== Photo Management ====================

  /**
   * Uploads a photo for a product
   * @param id - Product UUID
   * @param file - Image file (JPEG, PNG, WebP, max 5MB)
   * @returns Updated product with photo_path set
   * @throws Error if file size exceeds limit or format is invalid
   */
  async uploadPhoto(id: string, file: File): Promise<Product> {
    const formData = new FormData();
    formData.append('photo', file);

    return apiClient.post<Product>(`${PRODUCTS_BASE}/${id}/photo`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  }

  /**
   * Deletes a product photo
   * @param id - Product UUID
   */
  async deletePhoto(id: string): Promise<void> {
    return apiClient.delete(`${PRODUCTS_BASE}/${id}/photo`);
  }

  /**
   * Gets the URL for a product photo
   * @param id - Product UUID
   * @param tenantId - Optional tenant ID for public access
   * @returns Full URL to product photo endpoint
   */
  getPhotoUrl(id: string, tenantId?: string): string {
    if (tenantId) {
      // Public photo URL for guest ordering
      return `${apiClient.getAxiosInstance().defaults.baseURL}/api/public/products/${tenantId}/${id}/photo`;
    }
    // Authenticated photo URL
    return `${apiClient.getAxiosInstance().defaults.baseURL}${PRODUCTS_BASE}/${id}/photo`;
  }

  // ==================== Stock Management ====================

  /**
   * Adjusts stock quantity for a product
   * @param id - Product UUID
   * @param data - Stock adjustment details (new quantity, reason, notes)
   * @returns Updated product with new stock quantity and audit log created
   */
  async adjustStock(id: string, data: StockAdjustmentRequest): Promise<Product> {
    return apiClient.post<Product>(`${PRODUCTS_BASE}/${id}/stock`, data);
  }

  /**
   * Retrieves stock adjustment history for a specific product
   * @param id - Product UUID
   * @param page - Page number for pagination (default: 1)
   * @param limit - Results per page (default: 20)
   * @returns Paginated list of stock adjustments
   */
  async getStockAdjustments(
    id: string,
    page?: number,
    limit?: number
  ): Promise<PaginatedResponse<StockAdjustment>> {
    const queryParams = new URLSearchParams();
    if (page) queryParams.append('page', page.toString());
    if (limit) queryParams.append('limit', limit.toString());

    const url = queryParams.toString()
      ? `${PRODUCTS_BASE}/${id}/adjustments?${queryParams}`
      : `${PRODUCTS_BASE}/${id}/adjustments`;

    return apiClient.get<PaginatedResponse<StockAdjustment>>(url);
  }

  /**
   * Retrieves all stock adjustments across all products with optional filters
   * @param params - Query parameters for filtering and pagination
   * @returns Paginated list of stock adjustments
   */
  async getAllStockAdjustments(params?: {
    page?: number;
    limit?: number;
    product_id?: string;
    reason?: string;
    start_date?: string;
    end_date?: string;
  }): Promise<PaginatedResponse<StockAdjustment>> {
    const queryParams = new URLSearchParams();
    if (params?.page) queryParams.append('page', params.page.toString());
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.product_id) queryParams.append('product_id', params.product_id);
    if (params?.reason) queryParams.append('reason', params.reason);
    if (params?.start_date) queryParams.append('start_date', params.start_date);
    if (params?.end_date) queryParams.append('end_date', params.end_date);

    const url = queryParams.toString()
      ? `${INVENTORY_BASE}/adjustments?${queryParams}`
      : `${INVENTORY_BASE}/adjustments`;

    return apiClient.get<PaginatedResponse<StockAdjustment>>(url);
  }

  // ==================== Inventory Summary ====================

  /**
   * Retrieves inventory summary statistics
   * @returns Summary including total products, total value, low/out of stock counts
   */
  async getInventorySummary(): Promise<InventorySummary> {
    return apiClient.get<InventorySummary>(`${INVENTORY_BASE}/summary`);
  }

  // ==================== Category Management ====================

  /**
   * Retrieves all categories (cached in Redis for 5 minutes)
   * @returns List of all categories
   */
  async getCategories(): Promise<Category[]> {
    const response = await apiClient.get<{ categories: Category[] }>(CATEGORIES_BASE);
    return response.categories || [];
  }

  /**
   * Retrieves a single category by ID
   * @param id - Category UUID
   * @returns Category details
   */
  async getCategory(id: string): Promise<Category> {
    return apiClient.get<Category>(`${CATEGORIES_BASE}/${id}`);
  }

  /**
   * Creates a new category
   * @param data - Category data (name, description, display order)
   * @returns Created category with generated ID
   * @throws Error if category name already exists for tenant
   */
  async createCategory(data: CreateCategoryRequest): Promise<Category> {
    return apiClient.post<Category>(CATEGORIES_BASE, data);
  }

  /**
   * Updates an existing category
   * @param id - Category UUID
   * @param data - Updated category fields
   * @returns Updated category
   */
  async updateCategory(id: string, data: UpdateCategoryRequest): Promise<Category> {
    return apiClient.put<Category>(`${CATEGORIES_BASE}/${id}`, data);
  }

  /**
   * Deletes a category
   * @param id - Category UUID
   * @throws Error if category has products assigned
   */
  async deleteCategory(id: string): Promise<void> {
    return apiClient.delete(`${CATEGORIES_BASE}/${id}`);
  }

  // ==================== Public Menu (Guest Ordering) ====================

  /**
   * Get public menu for guest ordering
   * @param tenantId - Tenant UUID
   * @param params - Query parameters for filtering
   * @returns List of available products
   */
  async getPublicMenu(
    tenantId: string,
    params?: {
      category?: string;
      available_only?: boolean;
    }
  ): Promise<{ products: any[] }> {
    const queryParams = new URLSearchParams();
    if (params?.category && params.category !== 'all') {
      queryParams.append('category', params.category);
    }
    if (params?.available_only !== undefined) {
      queryParams.append('available_only', params.available_only.toString());
    }

    const url = queryParams.toString()
      ? `/api/public/menu/${tenantId}/products?${queryParams}`
      : `/api/public/menu/${tenantId}/products`;

    return apiClient.get<{ products: any[] }>(url);
  }
}

export const product = new ProductService();
