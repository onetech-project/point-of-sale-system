import apiClient from './api';
import type {
  ProductPhoto,
  StorageQuota,
  UploadPhotoParams,
  UpdatePhotoMetadataParams,
  ReorderPhotosParams,
} from '@/types/photo';
import { validateImageFile } from '@/utils/validation';
import { formatFileSize } from '@/utils/format';

class PhotoService {
  /**
   * Upload a new photo for a product
   */
  async uploadPhoto(params: UploadPhotoParams): Promise<ProductPhoto> {
    const { productId, file, displayOrder, isPrimary, onProgress } = params;

    const formData = new FormData();
    formData.append('photo', file);
    if (displayOrder !== undefined) {
      formData.append('display_order', displayOrder.toString());
    }
    if (isPrimary !== undefined) {
      formData.append('is_primary', isPrimary.toString());
    }

    const response = await apiClient.post<{ data: ProductPhoto }>(
      `/api/v1/products/${productId}/photos`,
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
        onUploadProgress: (progressEvent: ProgressEvent) => {
          if (onProgress && progressEvent.total) {
            const percentComplete = Math.round(
              (progressEvent.loaded * 100) / progressEvent.total
            );
            onProgress(percentComplete);
          }
        },
      }
    );

    return response.data;
  }

  /**
   * Replace an existing photo with a new one
   */
  async replacePhoto(
    productId: string,
    photoId: string,
    file: File,
    onProgress?: (progress: number) => void
  ): Promise<ProductPhoto> {
    const formData = new FormData();
    formData.append('photo', file);

    const response = await apiClient.put<{ data: ProductPhoto }>(
      `/api/v1/products/${productId}/photos/${photoId}`,
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
        onUploadProgress: (progressEvent: ProgressEvent) => {
          if (onProgress && progressEvent.total) {
            const percentComplete = Math.round(
              (progressEvent.loaded * 100) / progressEvent.total
            );
            onProgress(percentComplete);
          }
        },
      }
    );

    return response.data;
  }

  /**
   * List all photos for a product
   */
  async listPhotos(productId: string): Promise<ProductPhoto[]> {
    const response = await apiClient.get<{
      data: {
        photos: ProductPhoto[] | null;
        count: number;
      }
    }>(
      `/api/v1/products/${productId}/photos`
    );
    // Backend returns {photos: null, count: 0} when empty, convert null to empty array
    return response.data.photos || [];
  }

  /**
   * Get a single photo by ID
   */
  async getPhoto(productId: string, photoId: string): Promise<ProductPhoto> {
    const response = await apiClient.get<{ data: ProductPhoto }>(
      `/api/v1/products/${productId}/photos/${photoId}`
    );
    return response.data;
  }

  /**
   * Update photo metadata (display order, primary flag)
   */
  async updatePhotoMetadata(
    productId: string,
    photoId: string,
    params: UpdatePhotoMetadataParams
  ): Promise<void> {
    await apiClient.patch(
      `/api/v1/products/${productId}/photos/${photoId}`,
      params
    );
  }

  /**
   * Delete a photo
   */
  async deletePhoto(productId: string, photoId: string): Promise<void> {
    await apiClient.delete(`/api/v1/products/${productId}/photos/${photoId}`);
  }

  /**
   * Reorder multiple photos
   */
  async reorderPhotos(
    productId: string,
    params: ReorderPhotosParams
  ): Promise<void> {
    await apiClient.put(`/api/v1/products/${productId}/photos/reorder`, params);
  }

  /**
   * Get storage quota information for current tenant
   */
  async getStorageQuota(): Promise<StorageQuota> {
    const response = await apiClient.get<{ data: StorageQuota }>(
      '/api/v1/products/storage-quota'
    );
    return response.data;
  }

  /**
   * Validate file before upload (client-side validation)
   */
  validateFile = validateImageFile;

  /**
   * Format file size for display
   */
  formatFileSize = formatFileSize;
}

export default new PhotoService();
