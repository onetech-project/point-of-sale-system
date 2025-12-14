import apiClient from './api';

export interface ProductPhoto {
  id: string;
  product_id: string;
  tenant_id: string;
  storage_key: string;
  original_filename: string;
  file_size_bytes: number;
  mime_type: string;
  width_px?: number;
  height_px?: number;
  display_order: number;
  is_primary: boolean;
  photo_url?: string;
  created_at: string;
  updated_at: string;
}

export interface StorageQuota {
  tenant_id: string;
  storage_used_bytes: number;
  storage_quota_bytes: number;
  available_bytes: number;
  usage_percentage: number;
  photo_count: number;
  approaching_limit: boolean;
  quota_exceeded: boolean;
}

export interface UploadPhotoParams {
  productId: string;
  file: File;
  displayOrder?: number;
  isPrimary?: boolean;
  onProgress?: (progress: number) => void;
}

export interface UpdatePhotoMetadataParams {
  photoId: string;
  displayOrder?: number;
  isPrimary?: boolean;
}

export interface ReorderPhotosParams {
  photos: Array<{ photo_id: string; display_order: number }>;
}

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
        onUploadProgress: (progressEvent) => {
          if (onProgress && progressEvent.total) {
            const percentComplete = Math.round(
              (progressEvent.loaded * 100) / progressEvent.total
            );
            onProgress(percentComplete);
          }
        },
      }
    );

    return response.data.data;
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
        onUploadProgress: (progressEvent) => {
          if (onProgress && progressEvent.total) {
            const percentComplete = Math.round(
              (progressEvent.loaded * 100) / progressEvent.total
            );
            onProgress(percentComplete);
          }
        },
      }
    );

    return response.data.data;
  }

  /**
   * List all photos for a product
   */
  async listPhotos(productId: string): Promise<ProductPhoto[]> {
    const response = await apiClient.get<{ data: ProductPhoto[] }>(
      `/api/v1/products/${productId}/photos`
    );
    return response.data.data;
  }

  /**
   * Get a single photo by ID
   */
  async getPhoto(productId: string, photoId: string): Promise<ProductPhoto> {
    const response = await apiClient.get<{ data: ProductPhoto }>(
      `/api/v1/products/${productId}/photos/${photoId}`
    );
    return response.data.data;
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
      '/api/v1/tenants/storage-quota'
    );
    return response.data.data;
  }

  /**
   * Validate file before upload (client-side validation)
   */
  validateFile(file: File): { valid: boolean; error?: string } {
    const maxSize = 10 * 1024 * 1024; // 10MB
    const allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];

    if (!allowedTypes.includes(file.type)) {
      return {
        valid: false,
        error: 'Invalid file type. Only JPEG, PNG, GIF, and WebP images are allowed.',
      };
    }

    if (file.size > maxSize) {
      return {
        valid: false,
        error: 'File size exceeds 10MB limit.',
      };
    }

    return { valid: true };
  }

  /**
   * Format file size for display
   */
  formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
  }
}

export default new PhotoService();
