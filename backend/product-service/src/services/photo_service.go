package services

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/pos/backend/product-service/src/models"
	"github.com/pos/backend/product-service/src/repository"
)

// PhotoService handles business logic for product photos
type PhotoService struct {
	photoRepo           *repository.PhotoRepository
	storageService      *StorageService
	imageProcessor      *ImageProcessor
	maxPhotosPerProduct int
}

// NewPhotoService creates a new PhotoService
func NewPhotoService(
	photoRepo *repository.PhotoRepository,
	storageService *StorageService,
	imageProcessor *ImageProcessor,
	maxPhotosPerProduct int,
) *PhotoService {
	return &PhotoService{
		photoRepo:           photoRepo,
		storageService:      storageService,
		imageProcessor:      imageProcessor,
		maxPhotosPerProduct: maxPhotosPerProduct,
	}
}

// UploadPhoto handles the complete photo upload process
func (s *PhotoService) UploadPhoto(
	ctx context.Context,
	productID, tenantID uuid.UUID,
	filename string,
	fileReader io.Reader,
	displayOrder int,
	isPrimary bool,
) (*models.ProductPhoto, error) {
	// 1. Check if product has reached max photos limit
	photoCount, err := s.photoRepo.CountByProduct(ctx, productID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to count existing photos: %w", err)
	}

	if photoCount >= s.maxPhotosPerProduct {
		return nil, models.ErrMaxPhotosReached
	}

	// 2. Validate and process image
	metadata, imageData, err := s.imageProcessor.ValidateImage(fileReader)
	if err != nil {
		return nil, fmt.Errorf("image validation failed: %w", err)
	}

	// 3. Check storage quota
	quota, err := s.photoRepo.GetTenantStorageQuota(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check storage quota: %w", err)
	}

	if quota.StorageUsedBytes+metadata.Size > quota.StorageQuotaBytes {
		return nil, models.ErrQuotaExceeded
	}

	// 4. Optimize image (currently a pass-through)
	optimizedData, err := s.imageProcessor.OptimizeImage(imageData, metadata.MimeType)
	if err != nil {
		return nil, fmt.Errorf("image optimization failed: %w", err)
	}

	// 5. Generate storage key and photo ID
	photoID := uuid.New()
	sanitizedFilename := SanitizeFilename(filename)
	storageKey := GenerateStorageKey(tenantID, productID, photoID, sanitizedFilename)

	// 6. Upload to object storage
	err = s.storageService.UploadPhoto(
		ctx,
		storageKey,
		bytes.NewReader(optimizedData),
		int64(len(optimizedData)),
		metadata.MimeType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload photo to storage: %w", err)
	}

	// 7. If this should be primary, clear existing primary photo
	if isPrimary {
		err = s.photoRepo.ClearPrimaryPhoto(ctx, productID, tenantID)
		if err != nil {
			// Try to cleanup uploaded photo
			_ = s.storageService.DeletePhoto(ctx, storageKey)
			return nil, fmt.Errorf("failed to clear existing primary photo: %w", err)
		}
	}

	// 8. Create database record
	photo := &models.ProductPhoto{
		ID:               photoID,
		ProductID:        productID,
		TenantID:         tenantID,
		StorageKey:       storageKey,
		OriginalFilename: sanitizedFilename,
		FileSizeBytes:    int(metadata.Size),
		MimeType:         metadata.MimeType,
		WidthPx:          &metadata.Width,
		HeightPx:         &metadata.Height,
		DisplayOrder:     displayOrder,
		IsPrimary:        isPrimary,
	}

	err = s.photoRepo.Create(ctx, photo)
	if err != nil {
		// Cleanup: Delete uploaded photo from storage
		_ = s.storageService.DeletePhoto(ctx, storageKey)
		return nil, fmt.Errorf("failed to save photo metadata: %w", err)
	}

	// 9. Update tenant storage usage
	err = s.photoRepo.UpdateTenantStorageUsage(ctx, tenantID, metadata.Size)
	if err != nil {
		// Log error but don't fail the upload (can be corrected later)
		// TODO: Add proper logging here
		fmt.Printf("Warning: failed to update tenant storage usage: %v\n", err)
	}

	// 10. Generate presigned URL for response
	photoURL, err := s.storageService.GetPhotoURL(ctx, storageKey)
	if err != nil {
		// Log error but don't fail - URL can be generated later
		fmt.Printf("Warning: failed to generate photo URL: %v\n", err)
	} else {
		photo.PhotoURL = photoURL
	}

	return photo, nil
}

// ListPhotos retrieves all photos for a product with presigned URLs
func (s *PhotoService) ListPhotos(ctx context.Context, productID, tenantID uuid.UUID) ([]*models.ProductPhoto, error) {
	photos, err := s.photoRepo.GetByProduct(ctx, productID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list photos: %w", err)
	}

	// Generate presigned URLs for all photos
	for _, photo := range photos {
		url, err := s.storageService.GetPhotoURL(ctx, photo.StorageKey)
		if err != nil {
			// Log error but continue with other photos
			fmt.Printf("Warning: failed to generate URL for photo %s: %v\n", photo.ID, err)
			photo.PhotoURL = "" // Placeholder/fallback will be used by client
		} else {
			photo.PhotoURL = url
		}
	}

	return photos, nil
}

// GetPhoto retrieves a single photo by ID
func (s *PhotoService) GetPhoto(ctx context.Context, photoID, tenantID uuid.UUID) (*models.ProductPhoto, error) {
	photo, err := s.photoRepo.GetByID(ctx, photoID, tenantID)
	if err != nil {
		return nil, err
	}

	// Generate presigned URL
	url, err := s.storageService.GetPhotoURL(ctx, photo.StorageKey)
	if err != nil {
		fmt.Printf("Warning: failed to generate URL for photo %s: %v\n", photo.ID, err)
	} else {
		photo.PhotoURL = url
	}

	return photo, nil
}

// UpdatePhotoMetadata updates display order and primary flag
func (s *PhotoService) UpdatePhotoMetadata(
	ctx context.Context,
	photoID, tenantID uuid.UUID,
	displayOrder *int,
	isPrimary *bool,
) error {
	// Get existing photo to validate it exists and belongs to tenant
	photo, err := s.photoRepo.GetByID(ctx, photoID, tenantID)
	if err != nil {
		return err
	}

	// If setting as primary, clear existing primary photo
	if isPrimary != nil && *isPrimary {
		err = s.photoRepo.ClearPrimaryPhoto(ctx, photo.ProductID, tenantID)
		if err != nil {
			return fmt.Errorf("failed to clear existing primary photo: %w", err)
		}
	}

	// Update metadata
	err = s.photoRepo.UpdateMetadata(ctx, photoID, tenantID, displayOrder, isPrimary)
	if err != nil {
		return fmt.Errorf("failed to update photo metadata: %w", err)
	}

	return nil
}

// DeletePhoto removes a photo and updates storage usage
func (s *PhotoService) DeletePhoto(ctx context.Context, photoID, tenantID uuid.UUID) error {
	// Get photo metadata (includes file size for quota adjustment)
	photo, err := s.photoRepo.Delete(ctx, photoID, tenantID)
	if err != nil {
		return err
	}

	// Delete from object storage
	err = s.storageService.DeletePhoto(ctx, photo.StorageKey)
	if err != nil {
		// Log error but don't fail - database record is already deleted
		// Consider implementing retry mechanism for failed deletions
		fmt.Printf("Warning: failed to delete photo from storage: %v\n", err)
	}

	// Update tenant storage usage
	err = s.photoRepo.UpdateTenantStorageUsage(ctx, tenantID, -int64(photo.FileSizeBytes))
	if err != nil {
		fmt.Printf("Warning: failed to update tenant storage usage after deletion: %v\n", err)
	}

	return nil
}

// ReorderPhotos updates display order for multiple photos
func (s *PhotoService) ReorderPhotos(ctx context.Context, tenantID uuid.UUID, orders []models.PhotoOrder) error {
	// Validate that all display orders are non-negative and unique
	orderMap := make(map[int]bool)
	for _, order := range orders {
		if order.DisplayOrder < 0 {
			return models.ErrInvalidDisplayOrder
		}
		if orderMap[order.DisplayOrder] {
			return fmt.Errorf("duplicate display order: %d", order.DisplayOrder)
		}
		orderMap[order.DisplayOrder] = true
	}

	return s.photoRepo.ReorderPhotos(ctx, tenantID, orders)
}

// ReplacePhoto replaces an existing photo (delete old, upload new)
func (s *PhotoService) ReplacePhoto(
	ctx context.Context,
	photoID, tenantID uuid.UUID,
	filename string,
	fileReader io.Reader,
) (*models.ProductPhoto, error) {
	// 1. Get existing photo to validate it exists and belongs to tenant
	existingPhoto, err := s.photoRepo.GetByID(ctx, photoID, tenantID)
	if err != nil {
		return nil, err
	}

	// 2. Validate and process new image
	metadata, imageData, err := s.imageProcessor.ValidateImage(fileReader)
	if err != nil {
		return nil, fmt.Errorf("image validation failed: %w", err)
	}

	// 3. Check storage quota (consider the size difference)
	quota, err := s.photoRepo.GetTenantStorageQuota(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check storage quota: %w", err)
	}

	// Calculate net storage change (new size - old size)
	netSizeChange := metadata.Size - int64(existingPhoto.FileSizeBytes)
	if netSizeChange > 0 && quota.StorageUsedBytes+netSizeChange > quota.StorageQuotaBytes {
		return nil, models.ErrQuotaExceeded
	}

	// 4. Optimize image
	optimizedData, err := s.imageProcessor.OptimizeImage(imageData, metadata.MimeType)
	if err != nil {
		return nil, fmt.Errorf("image optimization failed: %w", err)
	}

	// 5. Generate new storage key (keep same photo ID but new filename)
	sanitizedFilename := SanitizeFilename(filename)
	storageKey := GenerateStorageKey(tenantID, existingPhoto.ProductID, photoID, sanitizedFilename)

	// 6. Upload new photo to object storage
	err = s.storageService.UploadPhoto(
		ctx,
		storageKey,
		bytes.NewReader(optimizedData),
		int64(len(optimizedData)),
		metadata.MimeType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload replacement photo to storage: %w", err)
	}

	// 7. Delete old photo from storage (best effort)
	if existingPhoto.StorageKey != storageKey {
		err = s.storageService.DeletePhoto(ctx, existingPhoto.StorageKey)
		if err != nil {
			// Log warning but continue - old file can be cleaned up later
			fmt.Printf("Warning: failed to delete old photo from storage: %v\n", err)
		}
	}

	// 8. Update database record with new metadata
	updatedPhoto := &models.ProductPhoto{
		ID:               photoID,
		ProductID:        existingPhoto.ProductID,
		TenantID:         tenantID,
		StorageKey:       storageKey,
		OriginalFilename: sanitizedFilename,
		FileSizeBytes:    int(metadata.Size),
		MimeType:         metadata.MimeType,
		WidthPx:          &metadata.Width,
		HeightPx:         &metadata.Height,
		DisplayOrder:     existingPhoto.DisplayOrder, // Keep existing order
		IsPrimary:        existingPhoto.IsPrimary,    // Keep existing primary status
	}

	err = s.photoRepo.Update(ctx, updatedPhoto)
	if err != nil {
		// Cleanup: Try to delete newly uploaded photo
		_ = s.storageService.DeletePhoto(ctx, storageKey)
		return nil, fmt.Errorf("failed to update photo metadata: %w", err)
	}

	// 9. Update tenant storage usage with net change
	if netSizeChange != 0 {
		err = s.photoRepo.UpdateTenantStorageUsage(ctx, tenantID, netSizeChange)
		if err != nil {
			fmt.Printf("Warning: failed to update tenant storage usage: %v\n", err)
		}
	}

	// 10. Generate presigned URL for response
	photoURL, err := s.storageService.GetPhotoURL(ctx, storageKey)
	if err != nil {
		fmt.Printf("Warning: failed to generate photo URL: %v\n", err)
	} else {
		updatedPhoto.PhotoURL = photoURL
	}

	return updatedPhoto, nil
}

// GetStorageQuota retrieves storage quota information for a tenant
func (s *PhotoService) GetStorageQuota(ctx context.Context, tenantID uuid.UUID) (*models.StorageQuotaResponse, error) {
	return s.photoRepo.GetTenantStorageQuota(ctx, tenantID)
}
