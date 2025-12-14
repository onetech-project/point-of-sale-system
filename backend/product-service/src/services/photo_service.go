package services

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/pos/backend/product-service/src/models"
	"github.com/pos/backend/product-service/src/repository"
	"github.com/rs/zerolog/log"
)

// PhotoService handles business logic for product photos
type PhotoService struct {
	photoRepo           *repository.PhotoRepository
	storageService      *StorageService
	imageProcessor      *ImageProcessor
	retryQueue          *RetryQueue
	maxPhotosPerProduct int
}

// NewPhotoService creates a new PhotoService
func NewPhotoService(
	photoRepo *repository.PhotoRepository,
	storageService *StorageService,
	imageProcessor *ImageProcessor,
	retryQueue *RetryQueue,
	maxPhotosPerProduct int,
) *PhotoService {
	return &PhotoService{
		photoRepo:           photoRepo,
		storageService:      storageService,
		imageProcessor:      imageProcessor,
		retryQueue:          retryQueue,
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
		log.Error().
			Err(err).
			Str("tenant_id", tenantID.String()).
			Str("product_id", productID.String()).
			Str("photo_id", photoID.String()).
			Msg("Failed to update tenant storage usage after photo upload")
	}

	// 10. Generate presigned URL for response
	photoURL, err := s.storageService.GetPhotoURL(ctx, storageKey)
	if err != nil {
		// Log error but don't fail - URL can be generated later
		log.Warn().
			Err(err).
			Str("storage_key", storageKey).
			Msg("Failed to generate photo URL after upload")
	} else {
		photo.PhotoURL = photoURL
	}

	// Audit log: successful photo upload
	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("product_id", productID.String()).
		Str("photo_id", photoID.String()).
		Str("filename", sanitizedFilename).
		Int("file_size", int(metadata.Size)).
		Bool("is_primary", isPrimary).
		Msg("Photo uploaded successfully")

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
			// Log error but continue - frontend will show placeholder
			log.Warn().
				Err(err).
				Str("photo_id", photo.ID.String()).
				Str("storage_key", photo.StorageKey).
				Msg("Failed to generate URL for photo, client will use placeholder")
			photo.PhotoURL = "" // Empty URL signals frontend to use placeholder
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
		log.Warn().
			Err(err).
			Str("photo_id", photoID.String()).
			Str("storage_key", photo.StorageKey).
			Msg("Failed to generate URL for photo, client will use placeholder")
		photo.PhotoURL = "" // Empty URL signals frontend to use placeholder
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

	// Audit log: successful metadata update
	logEvent := log.Info().
		Str("tenant_id", tenantID.String()).
		Str("photo_id", photoID.String()).
		Str("product_id", photo.ProductID.String())

	if displayOrder != nil {
		logEvent = logEvent.Int("new_display_order", *displayOrder)
	}
	if isPrimary != nil {
		logEvent = logEvent.Bool("new_is_primary", *isPrimary)
	}

	logEvent.Msg("Photo metadata updated successfully")

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
		// Enqueue for background retry with max 5 attempts
		if s.retryQueue != nil {
			s.retryQueue.Enqueue(tenantID.String(), photo.StorageKey, 5)
		}

		log.Error().
			Err(err).
			Str("tenant_id", tenantID.String()).
			Str("photo_id", photoID.String()).
			Str("storage_key", photo.StorageKey).
			Msg("Failed to delete photo from S3 storage, enqueued for retry")
	} else {
		log.Debug().
			Str("tenant_id", tenantID.String()).
			Str("photo_id", photoID.String()).
			Str("storage_key", photo.StorageKey).
			Msg("Photo deleted from S3 storage successfully")
	}

	// Update tenant storage usage
	err = s.photoRepo.UpdateTenantStorageUsage(ctx, tenantID, -int64(photo.FileSizeBytes))
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID.String()).
			Str("photo_id", photoID.String()).
			Msg("Failed to update tenant storage usage after photo deletion")
	}

	// Audit log: successful photo deletion
	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("photo_id", photoID.String()).
		Str("product_id", photo.ProductID.String()).
		Int("file_size", photo.FileSizeBytes).
		Msg("Photo deleted successfully")

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

	err := s.photoRepo.ReorderPhotos(ctx, tenantID, orders)
	if err != nil {
		return err
	}

	// Audit log: successful reordering
	log.Info().
		Str("tenant_id", tenantID.String()).
		Int("photo_count", len(orders)).
		Msg("Photos reordered successfully")

	return nil
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
			// Enqueue for background retry
			if s.retryQueue != nil {
				s.retryQueue.Enqueue(tenantID.String(), existingPhoto.StorageKey, 5)
			}

			log.Warn().
				Err(err).
				Str("tenant_id", tenantID.String()).
				Str("photo_id", photoID.String()).
				Str("storage_key", existingPhoto.StorageKey).
				Msg("Failed to delete old photo from storage after replacement, enqueued for retry")
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
			log.Error().
				Err(err).
				Str("tenant_id", tenantID.String()).
				Str("photo_id", photoID.String()).
				Int64("net_change", netSizeChange).
				Msg("Failed to update tenant storage usage after photo replacement")
		}
	}

	// 10. Generate presigned URL for response
	photoURL, err := s.storageService.GetPhotoURL(ctx, storageKey)
	if err != nil {
		log.Warn().
			Err(err).
			Str("storage_key", storageKey).
			Msg("Failed to generate photo URL after replacement")
	} else {
		updatedPhoto.PhotoURL = photoURL
	}

	// Audit log: successful photo replacement
	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("photo_id", photoID.String()).
		Str("product_id", existingPhoto.ProductID.String()).
		Str("old_filename", existingPhoto.OriginalFilename).
		Str("new_filename", sanitizedFilename).
		Int("old_size", existingPhoto.FileSizeBytes).
		Int("new_size", int(metadata.Size)).
		Int64("net_change", netSizeChange).
		Msg("Photo replaced successfully")

	return updatedPhoto, nil
}

// DeleteAllTenantPhotos deletes all photos for a tenant (cascade delete)
func (s *PhotoService) DeleteAllTenantPhotos(ctx context.Context, tenantID uuid.UUID) error {
	// 1. List all photos for the tenant
	photos, err := s.photoRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to list tenant photos: %w", err)
	}

	// 2. Delete each photo from S3 (continue on error to cleanup as much as possible)
	deletedCount := 0
	failedKeys := []string{}

	for _, photo := range photos {
		err := s.storageService.DeletePhoto(ctx, photo.StorageKey)
		if err != nil {
			// Enqueue for background retry
			if s.retryQueue != nil {
				s.retryQueue.Enqueue(tenantID.String(), photo.StorageKey, 5)
			}

			log.Error().
				Err(err).
				Str("tenant_id", tenantID.String()).
				Str("storage_key", photo.StorageKey).
				Msg("Failed to delete photo from S3 during tenant cascade delete, enqueued for retry")
			failedKeys = append(failedKeys, photo.StorageKey)
		} else {
			deletedCount++
		}
	}

	// 3. Delete all photos from database
	err = s.photoRepo.DeleteAllByTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete tenant photos from database: %w", err)
	}

	// 4. Audit log for tenant cascade delete
	logEvent := log.Info().
		Str("tenant_id", tenantID.String()).
		Int("total_photos", len(photos)).
		Int("deleted_from_s3", deletedCount).
		Int("failed_s3_deletes", len(failedKeys))

	if len(failedKeys) > 0 {
		logEvent = logEvent.Strs("failed_keys", failedKeys)
	}

	logEvent.Msg("Tenant photos cascade delete completed")

	return nil
}

// GetStorageQuota retrieves storage quota information for a tenant
func (s *PhotoService) GetStorageQuota(ctx context.Context, tenantID uuid.UUID) (*models.StorageQuotaResponse, error) {
	return s.photoRepo.GetTenantStorageQuota(ctx, tenantID)
}
