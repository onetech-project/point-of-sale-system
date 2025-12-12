package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos/backend/product-service/src/config"
	"github.com/pos/backend/product-service/src/models"
	"github.com/pos/backend/product-service/src/repository"
)

type CategoryService struct {
	repo repository.CategoryRepository
}

func NewCategoryService(repo repository.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

// invalidateCategoryCache removes category cache for a tenant
func (s *CategoryService) invalidateCategoryCache(ctx context.Context, tenantID uuid.UUID) error {
	cacheKey := fmt.Sprintf("categories:tenant:%s", tenantID.String())
	return config.RedisClient.Del(ctx, cacheKey).Err()
}

func (s *CategoryService) CreateCategory(ctx context.Context, category *models.Category) error {
	// Check for name uniqueness within tenant
	existing, err := s.repo.FindAll(ctx, category.TenantID)
	if err != nil {
		return fmt.Errorf("failed to check category uniqueness: %w", err)
	}

	for _, cat := range existing {
		if cat.Name == category.Name && cat.TenantID == category.TenantID {
			return fmt.Errorf("category name already exists")
		}
	}

	if err := s.repo.Create(ctx, category); err != nil {
		return err
	}

	// Invalidate cache after creating category
	s.invalidateCategoryCache(ctx, category.TenantID)

	return nil
}

func (s *CategoryService) GetCategories(ctx context.Context, tenantID uuid.UUID) ([]models.Category, error) {
	return s.repo.FindAll(ctx, tenantID)
}

func (s *CategoryService) GetCategory(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*models.Category, error) {
	return s.repo.FindByID(ctx, tenantID, id)
}

func (s *CategoryService) UpdateCategory(ctx context.Context, category *models.Category) error {
	// Check for name uniqueness within tenant (excluding current category)
	existing, err := s.repo.FindAll(ctx, category.TenantID)
	if err != nil {
		return fmt.Errorf("failed to check category uniqueness: %w", err)
	}

	for _, cat := range existing {
		if cat.ID != category.ID && cat.Name == category.Name && cat.TenantID == category.TenantID {
			return fmt.Errorf("category name already exists")
		}
	}

	if err := s.repo.Update(ctx, category); err != nil {
		return err
	}

	// Invalidate cache after updating category
	s.invalidateCategoryCache(ctx, category.TenantID)

	return nil
}

func (s *CategoryService) DeleteCategory(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	// Get category to access tenant ID for cache invalidation
	category, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("category not found: %w", err)
	}

	hasProducts, err := s.repo.HasProducts(ctx, id)
	if err != nil {
		return err
	}
	if hasProducts {
		return fmt.Errorf("cannot delete category with assigned products")
	}

	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		return err
	}

	// Invalidate cache after deleting category
	s.invalidateCategoryCache(ctx, category.TenantID)

	return nil
}
