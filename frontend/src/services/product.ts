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
  // Product Management
  async getProducts(params?: ProductListParams): Promise<PaginatedResponse<Product>> {
    const queryParams = new URLSearchParams();
    if (params?.page) queryParams.append('page', params.page.toString());
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.search) queryParams.append('search', params.search);
    if (params?.category_id) queryParams.append('category_id', params.category_id);
    if (params?.low_stock !== undefined) queryParams.append('low_stock', params.low_stock.toString());
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

  async getProduct(id: string): Promise<Product> {
    return apiClient.get<Product>(`${PRODUCTS_BASE}/${id}`);
  }

  async createProduct(data: CreateProductRequest): Promise<Product> {
    return apiClient.post<Product>(PRODUCTS_BASE, data);
  }

  async updateProduct(id: string, data: UpdateProductRequest): Promise<Product> {
    return apiClient.put<Product>(`${PRODUCTS_BASE}/${id}`, data);
  }

  async deleteProduct(id: string): Promise<void> {
    return apiClient.delete(`${PRODUCTS_BASE}/${id}`);
  }

  async archiveProduct(id: string): Promise<Product> {
    return apiClient.patch<Product>(`${PRODUCTS_BASE}/${id}/archive`);
  }

  async restoreProduct(id: string): Promise<Product> {
    return apiClient.patch<Product>(`${PRODUCTS_BASE}/${id}/restore`);
  }

  // Photo Management
  async uploadPhoto(id: string, file: File): Promise<Product> {
    const formData = new FormData();
    formData.append('photo', file);

    return apiClient.post<Product>(`${PRODUCTS_BASE}/${id}/photo`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  }

  async deletePhoto(id: string): Promise<void> {
    return apiClient.delete(`${PRODUCTS_BASE}/${id}/photo`);
  }

  getPhotoUrl(id: string): string {
    return `${apiClient.getAxiosInstance().defaults.baseURL}${PRODUCTS_BASE}/${id}/photo`;
  }

  // Stock Management
  async adjustStock(id: string, data: StockAdjustmentRequest): Promise<Product> {
    return apiClient.post<Product>(`${PRODUCTS_BASE}/${id}/stock`, data);
  }

  async getStockAdjustments(id: string, page?: number, limit?: number): Promise<PaginatedResponse<StockAdjustment>> {
    const queryParams = new URLSearchParams();
    if (page) queryParams.append('page', page.toString());
    if (limit) queryParams.append('limit', limit.toString());

    const url = queryParams.toString() 
      ? `${PRODUCTS_BASE}/${id}/adjustments?${queryParams}` 
      : `${PRODUCTS_BASE}/${id}/adjustments`;
    
    return apiClient.get<PaginatedResponse<StockAdjustment>>(url);
  }

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

  // Inventory Summary
  async getInventorySummary(): Promise<InventorySummary> {
    return apiClient.get<InventorySummary>(`${INVENTORY_BASE}/summary`);
  }

  // Category Management
  async getCategories(): Promise<Category[]> {
    const response = await apiClient.get<{ categories: Category[] }>(CATEGORIES_BASE);
    return response.categories || [];
  }

  async getCategory(id: string): Promise<Category> {
    return apiClient.get<Category>(`${CATEGORIES_BASE}/${id}`);
  }

  async createCategory(data: CreateCategoryRequest): Promise<Category> {
    return apiClient.post<Category>(CATEGORIES_BASE, data);
  }

  async updateCategory(id: string, data: UpdateCategoryRequest): Promise<Category> {
    return apiClient.put<Category>(`${CATEGORIES_BASE}/${id}`, data);
  }

  async deleteCategory(id: string): Promise<void> {
    return apiClient.delete(`${CATEGORIES_BASE}/${id}`);
  }
}

const productService = new ProductService();

export default productService;
