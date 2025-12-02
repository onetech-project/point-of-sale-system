package services

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/pos/backend/product-service/src/models"
	"github.com/pos/backend/product-service/src/repository"
	"github.com/pos/backend/product-service/src/utils"
)

type ProductService struct {
	repo           repository.ProductRepository
	uploadDir      string
	maxPhotoSizeMB int
}

func NewProductService(repo repository.ProductRepository) *ProductService {
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	return &ProductService{
		repo:           repo,
		uploadDir:      uploadDir,
		maxPhotoSizeMB: 5,
	}
}

func (s *ProductService) CreateProduct(ctx context.Context, product *models.Product) error {
	utils.Log.Info("Creating product: name=%s, sku=%s", product.Name, product.SKU)

	existing, err := s.repo.FindAll(ctx, map[string]interface{}{"search": product.SKU}, 1, 0)
	if err != nil {
		utils.Log.Error("Failed to check SKU uniqueness: %v", err)
		return err
	}
	if len(existing) > 0 {
		for _, p := range existing {
			if p.SKU == product.SKU {
				utils.Log.Warn("SKU already exists: %s", product.SKU)
				return fmt.Errorf("SKU already exists")
			}
		}
	}

	if err := s.repo.Create(ctx, product); err != nil {
		utils.Log.Error("Failed to create product: %v", err)
		return err
	}

	utils.Log.Info("Product created successfully: id=%s, name=%s", product.ID, product.Name)
	return nil
}

func (s *ProductService) GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *ProductService) GetProducts(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]models.Product, int, error) {
	products, err := s.repo.FindAll(ctx, filters, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.Count(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	return products, count, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, product *models.Product) error {
	utils.Log.Info("Updating product: id=%s, name=%s", product.ID, product.Name)

	existing, err := s.repo.FindByID(ctx, product.ID)
	if err != nil {
		utils.Log.Error("Failed to find product for update: id=%s, error=%v", product.ID, err)
		return err
	}
	if existing == nil {
		utils.Log.Warn("Product not found for update: id=%s", product.ID)
		return fmt.Errorf("product not found")
	}

	if existing.SKU != product.SKU {
		allProducts, err := s.repo.FindAll(ctx, map[string]interface{}{}, 10000, 0)
		if err != nil {
			utils.Log.Error("Failed to check SKU uniqueness: %v", err)
			return err
		}
		for _, p := range allProducts {
			if p.SKU == product.SKU && p.ID != product.ID {
				utils.Log.Warn("SKU already exists: %s", product.SKU)
				return fmt.Errorf("SKU already exists")
			}
		}
	}

	if err := s.repo.Update(ctx, product); err != nil {
		utils.Log.Error("Failed to update product: id=%s, error=%v", product.ID, err)
		return err
	}

	utils.Log.Info("Product updated successfully: id=%s", product.ID)
	return nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	utils.Log.Info("Deleting product: id=%s", id)

	hasSales, err := s.repo.HasSalesHistory(ctx, id)
	if err != nil {
		utils.Log.Error("Failed to check sales history: id=%s, error=%v", id, err)
		return err
	}
	if hasSales {
		utils.Log.Warn("Cannot delete product with sales history: id=%s", id)
		return fmt.Errorf("cannot delete product with sales history")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		utils.Log.Error("Failed to delete product: id=%s, error=%v", id, err)
		return err
	}

	utils.Log.Info("Product deleted successfully: id=%s", id)
	return nil
}

func (s *ProductService) ArchiveProduct(ctx context.Context, id uuid.UUID) error {
	utils.Log.Info("Archiving product: id=%s", id)

	if err := s.repo.Archive(ctx, id); err != nil {
		utils.Log.Error("Failed to archive product: id=%s, error=%v", id, err)
		return err
	}

	utils.Log.Info("Product archived successfully: id=%s", id)
	return nil
}

func (s *ProductService) RestoreProduct(ctx context.Context, id uuid.UUID) error {
	utils.Log.Info("Restoring product: id=%s", id)

	if err := s.repo.Restore(ctx, id); err != nil {
		utils.Log.Error("Failed to restore product: id=%s, error=%v", id, err)
		return err
	}

	utils.Log.Info("Product restored successfully: id=%s", id)
	return nil
}

func (s *ProductService) GetInventorySummary(ctx context.Context) (map[string]interface{}, error) {
	allProducts, err := s.repo.FindAll(ctx, map[string]interface{}{}, 10000, 0)
	if err != nil {
		return nil, err
	}

	lowStock, err := s.repo.FindLowStock(ctx, 10)
	if err != nil {
		return nil, err
	}

	outOfStock := 0
	totalValue := 0.0
	categoryMap := make(map[uuid.UUID]bool)

	for _, p := range allProducts {
		if p.StockQuantity <= 0 {
			outOfStock++
		}
		// Calculate total inventory value (cost price * quantity)
		totalValue += p.CostPrice * float64(p.StockQuantity)

		// Track unique categories
		if p.CategoryID != nil {
			categoryMap[*p.CategoryID] = true
		}
	}

	summary := map[string]interface{}{
		"total_products":     len(allProducts),
		"total_value":        totalValue,
		"low_stock_count":    len(lowStock),
		"out_of_stock_count": outOfStock,
		"categories_count":   len(categoryMap),
	}

	return summary, nil
}

func (s *ProductService) UploadPhoto(ctx context.Context, productID uuid.UUID, tenantID uuid.UUID, file multipart.File, header *multipart.FileHeader) error {
	utils.Log.Info("Uploading photo for product: id=%s, filename=%s, size=%d", productID, header.Filename, header.Size)

	// Security: Validate file size
	if header.Size > int64(s.maxPhotoSizeMB*1024*1024) {
		utils.Log.Warn("Photo size exceeds limit: size=%d, limit=%dMB", header.Size, s.maxPhotoSizeMB)
		return fmt.Errorf("file size exceeds %dMB limit", s.maxPhotoSizeMB)
	}

	// Security: Validate file size is not zero
	if header.Size == 0 {
		utils.Log.Warn("Empty file uploaded")
		return fmt.Errorf("file cannot be empty")
	}

	// Security: Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
	}
	if !allowedExts[ext] {
		utils.Log.Warn("Invalid photo format: %s", ext)
		return fmt.Errorf("invalid file format, only JPEG, PNG, WebP allowed")
	}

	// Security: Sanitize filename to prevent directory traversal
	sanitizedFilename := filepath.Base(header.Filename)
	if sanitizedFilename != header.Filename {
		utils.Log.Warn("Potentially malicious filename detected: %s", header.Filename)
	}

	// Security: Read first 512 bytes to detect actual MIME type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		utils.Log.Error("Failed to read file for MIME detection: %v", err)
		return fmt.Errorf("failed to read file")
	}

	// Security: Reset file pointer after reading
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, 0); err != nil {
			utils.Log.Error("Failed to reset file pointer: %v", err)
			return fmt.Errorf("failed to process file")
		}
	}

	// Security: Validate MIME type matches extension
	mimeType := http.DetectContentType(buffer[:n])
	validMimeTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}
	if !validMimeTypes[mimeType] {
		utils.Log.Warn("Invalid MIME type detected: %s (expected image type)", mimeType)
		return fmt.Errorf("invalid file content, must be a valid image file")
	}

	uploadPath := filepath.Join(s.uploadDir, tenantID.String(), productID.String())
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return fmt.Errorf("failed to create upload directory: %w", err)
	}

	filename := "photo" + ext
	filePath := filepath.Join(uploadPath, filename)

	dst, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	img, err := imaging.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}

	img = imaging.Resize(img, 800, 0, imaging.Lanczos)
	if err := imaging.Save(img, filePath); err != nil {
		return fmt.Errorf("failed to resize image: %w", err)
	}

	relativePath := filepath.Join("uploads", tenantID.String(), productID.String(), filename)

	product, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	if product == nil {
		return fmt.Errorf("product not found")
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	photoSize := int(fileInfo.Size())

	product.PhotoPath = &relativePath
	product.PhotoSize = &photoSize

	if err := s.repo.Update(ctx, product); err != nil {
		utils.Log.Error("Failed to update product with photo path: id=%s, error=%v", productID, err)
		return err
	}

	utils.Log.Info("Photo uploaded successfully: product_id=%s, path=%s", productID, relativePath)
	return nil
}

func (s *ProductService) GetPhotoPath(ctx context.Context, productID uuid.UUID, tenantID uuid.UUID) (string, error) {
	product, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return "", err
	}
	if product == nil {
		return "", fmt.Errorf("product not found")
	}
	if product.PhotoPath == nil {
		return "", fmt.Errorf("photo not found")
	}

	// Convert relative path to absolute path
	absolutePath := filepath.Join(s.uploadDir, tenantID.String(), productID.String(), filepath.Base(*product.PhotoPath))

	// Verify file exists
	if _, err := os.Stat(absolutePath); os.IsNotExist(err) {
		return "", fmt.Errorf("photo file not found")
	}

	return absolutePath, nil
}

func (s *ProductService) DeletePhoto(ctx context.Context, productID uuid.UUID, tenantID uuid.UUID) error {
	product, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	if product == nil {
		return fmt.Errorf("product not found")
	}
	if product.PhotoPath == nil {
		return fmt.Errorf("photo not found")
	}

	// Delete physical file
	uploadPath := filepath.Join(s.uploadDir, tenantID.String(), productID.String())
	if err := os.RemoveAll(uploadPath); err != nil {
		return fmt.Errorf("failed to delete photo file: %w", err)
	}

	// Clear photo fields in database
	product.PhotoPath = nil
	product.PhotoSize = nil

	return s.repo.Update(ctx, product)
}

func (s *ProductService) AdjustStock(ctx context.Context, productID, tenantID, userID uuid.UUID, newQuantity int, reason, notes string) (*models.Product, error) {
	// Get current product
	product, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, fmt.Errorf("product not found")
	}

	// Create stock adjustment record
	adjustment := &models.StockAdjustment{
		TenantID:         tenantID,
		ProductID:        productID,
		UserID:           userID,
		PreviousQuantity: product.StockQuantity,
		NewQuantity:      newQuantity,
		Reason:           reason,
		Notes:            &notes,
	}

	if notes == "" {
		adjustment.Notes = nil
	}

	// Record adjustment
	if err := s.repo.CreateStockAdjustment(ctx, adjustment); err != nil {
		return nil, fmt.Errorf("failed to create stock adjustment: %w", err)
	}

	// Update product stock
	if err := s.repo.UpdateStock(ctx, productID, newQuantity); err != nil {
		return nil, fmt.Errorf("failed to update stock: %w", err)
	}

	// Return updated product
	return s.repo.FindByID(ctx, productID)
}
