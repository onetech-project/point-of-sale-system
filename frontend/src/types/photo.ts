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
  displayOrder?: number;
  isPrimary?: boolean;
}

export interface ReorderPhotosParams {
  photos: Array<{ photo_id: string; display_order: number }>;
}
