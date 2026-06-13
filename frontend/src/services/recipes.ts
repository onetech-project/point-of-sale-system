import apiClient from './api';
import {
  CreateRecipeRequest,
  IngredientListParams,
  IngredientListResponse,
  Recipe,
  RecipeCost,
  RecipeVersionSummary,
  UOM,
} from '@/types/recipes';

const PRODUCTS_BASE = '/api/v1/products';
const INGREDIENTS_BASE = '/api/v1/ingredients';
const UOMS_BASE = '/api/v1/uoms';

class RecipeService {
  async getActiveRecipe(productId: string): Promise<Recipe> {
    return apiClient.get<Recipe>(`${PRODUCTS_BASE}/${productId}/recipe`);
  }

  async getRecipeCost(productId: string): Promise<RecipeCost> {
    return apiClient.get<RecipeCost>(`${PRODUCTS_BASE}/${productId}/recipe/cost`);
  }

  async listRecipeVersions(productId: string): Promise<RecipeVersionSummary[]> {
    const response = await apiClient.get<{ recipes: RecipeVersionSummary[] }>(
      `${PRODUCTS_BASE}/${productId}/recipes`
    );
    return response.recipes || [];
  }

  async getRecipeVersion(productId: string, version: number): Promise<Recipe> {
    return apiClient.get<Recipe>(`${PRODUCTS_BASE}/${productId}/recipes/${version}`);
  }

  async createRecipeVersion(productId: string, data: CreateRecipeRequest): Promise<Recipe> {
    return apiClient.post<Recipe>(`${PRODUCTS_BASE}/${productId}/recipes`, data);
  }

  async listIngredients(params?: IngredientListParams): Promise<IngredientListResponse> {
    const queryParams = new URLSearchParams();
    if (params?.search) queryParams.append('search', params.search);
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.offset) queryParams.append('offset', params.offset.toString());
    if (params?.include_inactive !== undefined) {
      queryParams.append('include_inactive', params.include_inactive.toString());
    }

    const url = queryParams.toString() ? `${INGREDIENTS_BASE}?${queryParams}` : INGREDIENTS_BASE;
    const response = await apiClient.get<IngredientListResponse>(url);

    return {
      ingredients: response.ingredients || [],
      total: response.total || 0,
      limit: response.limit || params?.limit || 0,
      offset: response.offset || params?.offset || 0,
    };
  }

  async listUOMs(params?: { include_inactive?: boolean }): Promise<UOM[]> {
    const queryParams = new URLSearchParams();
    if (params?.include_inactive !== undefined) {
      queryParams.append('include_inactive', params.include_inactive.toString());
    }

    const url = queryParams.toString() ? `${UOMS_BASE}?${queryParams}` : UOMS_BASE;
    const response = await apiClient.get<{ uoms: UOM[] }>(url);
    return response.uoms || [];
  }
}

export const recipes = new RecipeService();
export default recipes;
