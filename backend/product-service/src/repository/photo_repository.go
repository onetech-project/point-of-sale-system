package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pos/backend/product-service/src/models"
)

// PhotoRepository handles database operations for product photos
type PhotoRepository struct {
	db *sql.DB
}

// NewPhotoRepository creates a new PhotoRepository
func NewPhotoRepository(db *sql.DB) *PhotoRepository {
	return &PhotoRepository{db: db}
}

// Create inserts a new product photo into the database
func (r *PhotoRepository) Create(ctx context.Context, photo *models.ProductPhoto) error {
	query := `
		INSERT INTO product_photos (
			id, product_id, tenant_id, storage_key, original_filename,
			file_size_bytes, mime_type, width_px, height_px,
			display_order, is_primary, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(
		ctx, query,
		photo.ID, photo.ProductID, photo.TenantID, photo.StorageKey,
		photo.OriginalFilename, photo.FileSizeBytes, photo.MimeType,
		photo.WidthPx, photo.HeightPx, photo.DisplayOrder, photo.IsPrimary,
		time.Now(), time.Now(),
	).Scan(&photo.ID, &photo.CreatedAt, &photo.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create product photo: %w", err)
	}

	return nil
}

// GetByProduct retrieves all photos for a specific product
func (r *PhotoRepository) GetByProduct(ctx context.Context, productID, tenantID uuid.UUID) ([]*models.ProductPhoto, error) {
	query := `
		SELECT id, product_id, tenant_id, storage_key, original_filename,
		       file_size_bytes, mime_type, width_px, height_px,
		       display_order, is_primary, created_at, updated_at
		FROM product_photos
		WHERE product_id = $1 AND tenant_id = $2
		ORDER BY display_order ASC, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, productID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query product photos: %w", err)
	}
	defer rows.Close()

	var photos []*models.ProductPhoto
	for rows.Next() {
		photo := &models.ProductPhoto{}
		err := rows.Scan(
			&photo.ID, &photo.ProductID, &photo.TenantID, &photo.StorageKey,
			&photo.OriginalFilename, &photo.FileSizeBytes, &photo.MimeType,
			&photo.WidthPx, &photo.HeightPx, &photo.DisplayOrder, &photo.IsPrimary,
			&photo.CreatedAt, &photo.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product photo: %w", err)
		}
		photos = append(photos, photo)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product photos: %w", err)
	}

	return photos, nil
}

// GetByID retrieves a single photo by ID with tenant validation
func (r *PhotoRepository) GetByID(ctx context.Context, photoID, tenantID uuid.UUID) (*models.ProductPhoto, error) {
	query := `
		SELECT id, product_id, tenant_id, storage_key, original_filename,
		       file_size_bytes, mime_type, width_px, height_px,
		       display_order, is_primary, created_at, updated_at
		FROM product_photos
		WHERE id = $1 AND tenant_id = $2
	`

	photo := &models.ProductPhoto{}
	err := r.db.QueryRowContext(ctx, query, photoID, tenantID).Scan(
		&photo.ID, &photo.ProductID, &photo.TenantID, &photo.StorageKey,
		&photo.OriginalFilename, &photo.FileSizeBytes, &photo.MimeType,
		&photo.WidthPx, &photo.HeightPx, &photo.DisplayOrder, &photo.IsPrimary,
		&photo.CreatedAt, &photo.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, models.ErrPhotoNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get product photo: %w", err)
	}

	return photo, nil
}

// UpdateMetadata updates display_order and is_primary for a photo
func (r *PhotoRepository) UpdateMetadata(ctx context.Context, photoID, tenantID uuid.UUID, displayOrder *int, isPrimary *bool) error {
	// Build dynamic update query
	query := "UPDATE product_photos SET updated_at = $1"
	args := []interface{}{time.Now()}
	argPos := 2

	if displayOrder != nil {
		query += fmt.Sprintf(", display_order = $%d", argPos)
		args = append(args, *displayOrder)
		argPos++
	}

	if isPrimary != nil {
		query += fmt.Sprintf(", is_primary = $%d", argPos)
		args = append(args, *isPrimary)
		argPos++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND tenant_id = $%d", argPos, argPos+1)
	args = append(args, photoID, tenantID)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update photo metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return models.ErrPhotoNotFound
	}

	return nil
}

// Update replaces all fields of a photo (used for photo replacement)
func (r *PhotoRepository) Update(ctx context.Context, photo *models.ProductPhoto) error {
	query := `
		UPDATE product_photos 
		SET storage_key = $1, original_filename = $2, file_size_bytes = $3,
		    mime_type = $4, width_px = $5, height_px = $6, updated_at = $7
		WHERE id = $8 AND tenant_id = $9
	`

	result, err := r.db.ExecContext(
		ctx, query,
		photo.StorageKey, photo.OriginalFilename, photo.FileSizeBytes,
		photo.MimeType, photo.WidthPx, photo.HeightPx, time.Now(),
		photo.ID, photo.TenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to update product photo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return models.ErrPhotoNotFound
	}

	return nil
}

// Delete removes a photo from the database
func (r *PhotoRepository) Delete(ctx context.Context, photoID, tenantID uuid.UUID) (*models.ProductPhoto, error) {
	// Get photo before deletion (for storage cleanup)
	photo, err := r.GetByID(ctx, photoID, tenantID)
	if err != nil {
		return nil, err
	}

	query := "DELETE FROM product_photos WHERE id = $1 AND tenant_id = $2"
	result, err := r.db.ExecContext(ctx, query, photoID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete photo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, models.ErrPhotoNotFound
	}

	return photo, nil
}

// CountByProduct counts photos for a specific product
func (r *PhotoRepository) CountByProduct(ctx context.Context, productID, tenantID uuid.UUID) (int, error) {
	query := "SELECT COUNT(*) FROM product_photos WHERE product_id = $1 AND tenant_id = $2"

	var count int
	err := r.db.QueryRowContext(ctx, query, productID, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count product photos: %w", err)
	}

	return count, nil
}

// UpdateTenantStorageUsage updates tenant storage usage (add or subtract)
func (r *PhotoRepository) UpdateTenantStorageUsage(ctx context.Context, tenantID uuid.UUID, deltaBytes int64) error {
	query := `
		UPDATE tenants 
		SET storage_used_bytes = storage_used_bytes + $1
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, deltaBytes, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update tenant storage usage: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return models.ErrInvalidTenantID
	}

	return nil
}

// GetTenantStorageQuota retrieves storage quota information for a tenant
func (r *PhotoRepository) GetTenantStorageQuota(ctx context.Context, tenantID uuid.UUID) (*models.StorageQuotaResponse, error) {
	query := `
		SELECT 
			t.id,
			COALESCE(t.storage_used_bytes, 0),
			COALESCE(t.storage_quota_bytes, 5368709120),
			COUNT(p.id)
		FROM tenants t
		LEFT JOIN product_photos p ON p.tenant_id = t.id
		WHERE t.id = $1
		GROUP BY t.id, t.storage_used_bytes, t.storage_quota_bytes
	`

	var quota models.StorageQuotaResponse
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&quota.TenantID,
		&quota.StorageUsedBytes,
		&quota.StorageQuotaBytes,
		&quota.PhotoCount,
	)

	if err == sql.ErrNoRows {
		return nil, models.ErrInvalidTenantID
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant storage quota: %w", err)
	}

	// Calculate derived fields
	quota.AvailableBytes = quota.StorageQuotaBytes - quota.StorageUsedBytes
	if quota.AvailableBytes < 0 {
		quota.AvailableBytes = 0
	}

	if quota.StorageQuotaBytes > 0 {
		quota.UsagePercentage = (float64(quota.StorageUsedBytes) / float64(quota.StorageQuotaBytes)) * 100
	}

	quota.ApproachingLimit = quota.UsagePercentage >= 80.0
	quota.QuotaExceeded = quota.StorageUsedBytes >= quota.StorageQuotaBytes

	return &quota, nil
}

// ClearPrimaryPhoto removes primary flag from all photos of a product
func (r *PhotoRepository) ClearPrimaryPhoto(ctx context.Context, productID, tenantID uuid.UUID) error {
	query := `
		UPDATE product_photos 
		SET is_primary = false, updated_at = $1
		WHERE product_id = $2 AND tenant_id = $3 AND is_primary = true
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), productID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to clear primary photo: %w", err)
	}

	return nil
}

// ReorderPhotos updates display order for multiple photos in a transaction
func (r *PhotoRepository) ReorderPhotos(ctx context.Context, tenantID uuid.UUID, orders []models.PhotoOrder) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := "UPDATE product_photos SET display_order = $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4"

	for _, order := range orders {
		result, err := tx.ExecContext(ctx, query, order.DisplayOrder, time.Now(), order.PhotoID, tenantID)
		if err != nil {
			return fmt.Errorf("failed to reorder photo %s: %w", order.PhotoID, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return fmt.Errorf("photo %s not found or unauthorized", order.PhotoID)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListByTenant retrieves all photos for a specific tenant (for cascade deletion)
func (r *PhotoRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.ProductPhoto, error) {
	query := `
		SELECT id, product_id, tenant_id, storage_key, original_filename,
			   file_size_bytes, mime_type, width_px, height_px,
			   display_order, is_primary, created_at, updated_at
		FROM product_photos
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant photos: %w", err)
	}
	defer rows.Close()

	var photos []*models.ProductPhoto
	for rows.Next() {
		photo := &models.ProductPhoto{}
		err := rows.Scan(
			&photo.ID, &photo.ProductID, &photo.TenantID, &photo.StorageKey,
			&photo.OriginalFilename, &photo.FileSizeBytes, &photo.MimeType,
			&photo.WidthPx, &photo.HeightPx, &photo.DisplayOrder, &photo.IsPrimary,
			&photo.CreatedAt, &photo.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan photo: %w", err)
		}
		photos = append(photos, photo)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating photos: %w", err)
	}

	return photos, nil
}

// DeleteAllByTenant deletes all photos for a tenant (for cascade deletion)
func (r *PhotoRepository) DeleteAllByTenant(ctx context.Context, tenantID uuid.UUID) error {
	query := "DELETE FROM product_photos WHERE tenant_id = $1"

	result, err := r.db.ExecContext(ctx, query, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete tenant photos: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log the number of photos deleted (caller should log this)
	_ = rowsAffected

	return nil
}
