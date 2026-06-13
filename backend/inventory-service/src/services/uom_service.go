package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/repository"
)

type UOMService struct {
	repo *repository.UOMRepository
}

func NewUOMService(repo *repository.UOMRepository) *UOMService {
	return &UOMService{repo: repo}
}

func (s *UOMService) List(ctx context.Context, tenantID uuid.UUID, includeInactive bool) ([]models.UOM, error) {
	return s.repo.List(ctx, tenantID, includeInactive)
}

func (s *UOMService) Get(ctx context.Context, tenantID, id uuid.UUID) (*models.UOM, error) {
	uom, err := s.repo.FindByID(ctx, tenantID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return uom, err
}

func (s *UOMService) Create(ctx context.Context, tenantID uuid.UUID, req models.CreateUOMRequest) (*models.UOM, error) {
	code := strings.TrimSpace(strings.ToLower(req.Code))
	name := strings.TrimSpace(req.Name)
	category := strings.TrimSpace(strings.ToLower(req.Category))
	if code == "" || name == "" {
		return nil, fmt.Errorf("%w: code and name are required", ErrInvalidInput)
	}
	if !validUOMCategory(category) {
		return nil, fmt.Errorf("%w: invalid uom category", ErrInvalidInput)
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	uom := &models.UOM{
		Code:       code,
		Name:       name,
		Category:   category,
		IsBaseUnit: req.IsBaseUnit,
		IsActive:   isActive,
	}
	if err := s.repo.Create(ctx, tenantID, uom); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, fmt.Errorf("%w: duplicate uom code", ErrConflict)
		}
		return nil, err
	}
	return uom, nil
}

func (s *UOMService) Update(ctx context.Context, tenantID, id uuid.UUID, req models.UpdateUOMRequest) (*models.UOM, error) {
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
		}
		req.Name = &name
	}
	if req.Category != nil {
		category := strings.TrimSpace(strings.ToLower(*req.Category))
		if !validUOMCategory(category) {
			return nil, fmt.Errorf("%w: invalid uom category", ErrInvalidInput)
		}
		req.Category = &category
	}
	uom, err := s.repo.Update(ctx, tenantID, id, req)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil && strings.Contains(err.Error(), "global uoms are read-only") {
		return nil, ErrReadOnlyGlobalUOM
	}
	return uom, err
}

func (s *UOMService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	err := s.repo.Delete(ctx, tenantID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func validUOMCategory(category string) bool {
	switch category {
	case "weight", "volume", "count", "packaging":
		return true
	default:
		return false
	}
}
