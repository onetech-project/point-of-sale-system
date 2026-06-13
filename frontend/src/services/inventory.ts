import apiClient from './api';
import {
  Bundle,
  BundleCost,
  BundleListParams,
  CreateBundleRequest,
  CreateConversionRequest,
  CreateIngredientRequest,
  CreateUOMRequest,
  Ingredient,
  IngredientListParams,
  IngredientUOMConversion,
  InventoryPaginationParams,
  InventoryValuation,
  ManualAdjustmentRequest,
  PaginatedInventoryResponse,
  StockMovement,
  StockMutationRequest,
  UpdateBundleRequest,
  UOM,
  UpdateConversionRequest,
  UpdateIngredientRequest,
  UpdateUOMRequest,
} from '../types/inventory';

const INGREDIENTS_BASE = '/api/v1/ingredients';
const UOMS_BASE = '/api/v1/uoms';
const INVENTORY_BASE = '/api/v1/inventory';
const BUNDLES_BASE = `${INVENTORY_BASE}/bundles`;

interface ListResponse<TDataKey extends string, TItem> {
  total?: number;
  limit?: number;
  offset?: number;
  [key: string]: TItem[] | number | undefined;
}

const pageToOffset = (page = 1, limit = 20) => Math.max(0, page - 1) * limit;

const toPaginatedResponse = <TDataKey extends string, TItem>(
  response: ListResponse<TDataKey, TItem>,
  dataKey: TDataKey,
  requestedPage: number,
  requestedLimit: number
): PaginatedInventoryResponse<TItem> => {
  const limit = response.limit || requestedLimit;
  const offset = response.offset || 0;
  const total = response.total || 0;
  const page = Math.floor(offset / limit) + 1 || requestedPage;

  return {
    data: (response[dataKey] as TItem[] | undefined) || [],
    total,
    page,
    limit,
    offset,
    total_pages: Math.max(1, Math.ceil(total / limit)),
  };
};

const appendPagination = (
  queryParams: URLSearchParams,
  params: InventoryPaginationParams | undefined,
  defaultLimit = 20
) => {
  const page = params?.page || 1;
  const limit = params?.limit || defaultLimit;
  queryParams.append('limit', limit.toString());
  queryParams.append('offset', pageToOffset(page, limit).toString());
  return { page, limit };
};

class InventoryService {
  async getUOMs(includeInactive = false): Promise<UOM[]> {
    const queryParams = new URLSearchParams();
    if (includeInactive) {
      queryParams.append('include_inactive', 'true');
    }

    const url = queryParams.toString() ? `${UOMS_BASE}?${queryParams}` : UOMS_BASE;
    const response = await apiClient.get<{ uoms: UOM[] }>(url);
    return response.uoms || [];
  }

  async createUOM(data: CreateUOMRequest): Promise<UOM> {
    return apiClient.post<UOM>(UOMS_BASE, data);
  }

  async updateUOM(id: string, data: UpdateUOMRequest): Promise<UOM> {
    return apiClient.put<UOM>(`${UOMS_BASE}/${id}`, data);
  }

  async deleteUOM(id: string): Promise<void> {
    return apiClient.delete(`${UOMS_BASE}/${id}`);
  }

  async getIngredients(
    params: IngredientListParams = {}
  ): Promise<PaginatedInventoryResponse<Ingredient>> {
    const queryParams = new URLSearchParams();
    const { page, limit } = appendPagination(queryParams, params);

    if (params.search?.trim()) {
      queryParams.append('search', params.search.trim());
    }
    if (params.includeInactive) {
      queryParams.append('include_inactive', 'true');
    }

    const response = await apiClient.get<ListResponse<'ingredients', Ingredient>>(
      `${INGREDIENTS_BASE}?${queryParams}`
    );

    return toPaginatedResponse(response, 'ingredients', page, limit);
  }

  async getIngredient(id: string): Promise<Ingredient> {
    return apiClient.get<Ingredient>(`${INGREDIENTS_BASE}/${id}`);
  }

  async createIngredient(data: CreateIngredientRequest): Promise<Ingredient> {
    return apiClient.post<Ingredient>(INGREDIENTS_BASE, data);
  }

  async updateIngredient(id: string, data: UpdateIngredientRequest): Promise<Ingredient> {
    return apiClient.put<Ingredient>(`${INGREDIENTS_BASE}/${id}`, data);
  }

  async deleteIngredient(id: string): Promise<void> {
    return apiClient.delete(`${INGREDIENTS_BASE}/${id}`);
  }

  async getIngredientConversions(ingredientId: string): Promise<IngredientUOMConversion[]> {
    const response = await apiClient.get<{ conversions: IngredientUOMConversion[] }>(
      `${INGREDIENTS_BASE}/${ingredientId}/conversions`
    );
    return response.conversions || [];
  }

  async createIngredientConversion(
    ingredientId: string,
    data: CreateConversionRequest
  ): Promise<IngredientUOMConversion> {
    return apiClient.post<IngredientUOMConversion>(
      `${INGREDIENTS_BASE}/${ingredientId}/conversions`,
      data
    );
  }

  async updateIngredientConversion(
    ingredientId: string,
    conversionId: string,
    data: UpdateConversionRequest
  ): Promise<IngredientUOMConversion> {
    return apiClient.put<IngredientUOMConversion>(
      `${INGREDIENTS_BASE}/${ingredientId}/conversions/${conversionId}`,
      data
    );
  }

  async deleteIngredientConversion(ingredientId: string, conversionId: string): Promise<void> {
    return apiClient.delete(`${INGREDIENTS_BASE}/${ingredientId}/conversions/${conversionId}`);
  }

  async recordInitialStock(data: StockMutationRequest): Promise<StockMovement> {
    return apiClient.post<StockMovement>(`${INVENTORY_BASE}/initial-stock`, data);
  }

  async recordPurchase(data: StockMutationRequest): Promise<StockMovement> {
    return apiClient.post<StockMovement>(`${INVENTORY_BASE}/purchases`, data);
  }

  async recordWaste(data: StockMutationRequest): Promise<StockMovement> {
    return apiClient.post<StockMovement>(`${INVENTORY_BASE}/waste`, data);
  }

  async recordAdjustment(data: ManualAdjustmentRequest): Promise<StockMovement> {
    return apiClient.post<StockMovement>(`${INVENTORY_BASE}/adjustments`, data);
  }

  async getIngredientMovements(
    ingredientId: string,
    params: InventoryPaginationParams = {}
  ): Promise<PaginatedInventoryResponse<StockMovement>> {
    const queryParams = new URLSearchParams();
    const { page, limit } = appendPagination(queryParams, params);
    const response = await apiClient.get<ListResponse<'movements', StockMovement>>(
      `${INVENTORY_BASE}/ingredients/${ingredientId}/movements?${queryParams}`
    );

    return toPaginatedResponse(response, 'movements', page, limit);
  }

  async getLowStockIngredients(
    params: InventoryPaginationParams = {}
  ): Promise<PaginatedInventoryResponse<Ingredient>> {
    const queryParams = new URLSearchParams();
    const { page, limit } = appendPagination(queryParams, params);
    const response = await apiClient.get<ListResponse<'ingredients', Ingredient>>(
      `${INVENTORY_BASE}/low-stock?${queryParams}`
    );

    return toPaginatedResponse(response, 'ingredients', page, limit);
  }

  async getValuation(): Promise<InventoryValuation> {
    return apiClient.get<InventoryValuation>(`${INVENTORY_BASE}/valuation`);
  }

  async getBundles(params: BundleListParams = {}): Promise<PaginatedInventoryResponse<Bundle>> {
    const queryParams = new URLSearchParams();
    const { page, limit } = appendPagination(queryParams, params);

    if (params.search?.trim()) {
      queryParams.append('search', params.search.trim());
    }
    if (params.includeInactive) {
      queryParams.append('include_inactive', 'true');
    }

    const response = await apiClient.get<ListResponse<'bundles', Bundle>>(
      `${BUNDLES_BASE}?${queryParams}`
    );

    return toPaginatedResponse(response, 'bundles', page, limit);
  }

  async getBundle(id: string): Promise<Bundle> {
    return apiClient.get<Bundle>(`${BUNDLES_BASE}/${id}`);
  }

  async createBundle(data: CreateBundleRequest): Promise<Bundle> {
    return apiClient.post<Bundle>(BUNDLES_BASE, data);
  }

  async updateBundle(id: string, data: UpdateBundleRequest): Promise<Bundle> {
    return apiClient.put<Bundle>(`${BUNDLES_BASE}/${id}`, data);
  }

  async deleteBundle(id: string): Promise<void> {
    return apiClient.delete(`${BUNDLES_BASE}/${id}`);
  }

  async getBundleCost(id: string): Promise<BundleCost> {
    return apiClient.get<BundleCost>(`${BUNDLES_BASE}/${id}/cost`);
  }
}

export const inventory = new InventoryService();
