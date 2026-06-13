package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/repository"
	"github.com/shopspring/decimal"
)

type ConversionService struct {
	repo           *repository.ConversionRepository
	uomRepo        *repository.UOMRepository
	ingredientRepo *repository.IngredientRepository
}

func NewConversionService(repo *repository.ConversionRepository, uomRepo *repository.UOMRepository, ingredientRepo *repository.IngredientRepository) *ConversionService {
	return &ConversionService{repo: repo, uomRepo: uomRepo, ingredientRepo: ingredientRepo}
}

func (s *ConversionService) List(ctx context.Context, tenantID, ingredientID uuid.UUID) ([]models.ConversionResponse, error) {
	var conversions []models.IngredientUOMConversion
	err := repository.WithTenantTx(ctx, s.repo.DB(), tenantID, nil, func(tx *sql.Tx) error {
		if _, err := s.ingredientRepo.FindByIDTx(ctx, tx, tenantID, ingredientID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		var err error
		conversions, err = s.repo.ListByIngredientTx(ctx, tx, tenantID, ingredientID)
		return err
	})
	if err != nil {
		return nil, err
	}
	responses := make([]models.ConversionResponse, 0, len(conversions))
	for i := range conversions {
		responses = append(responses, models.NewConversionResponse(&conversions[i]))
	}
	return responses, nil
}

func (s *ConversionService) Create(ctx context.Context, tenantID, ingredientID uuid.UUID, req models.CreateConversionRequest) (*models.ConversionResponse, error) {
	if req.Multiplier.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("%w: multiplier must be greater than zero", ErrInvalidInput)
	}
	conversion := &models.IngredientUOMConversion{
		TenantID:     &tenantID,
		IngredientID: &ingredientID,
		FromUOMID:    req.FromUOMID,
		ToUOMID:      req.ToUOMID,
		Multiplier:   req.Multiplier.Decimal,
	}

	err := repository.WithTenantTx(ctx, s.repo.DB(), tenantID, nil, func(tx *sql.Tx) error {
		if err := s.validateConversionTx(ctx, tx, tenantID, ingredientID, req.FromUOMID, req.ToUOMID); err != nil {
			return err
		}
		if err := s.repo.CreateTx(ctx, tx, conversion); err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				return fmt.Errorf("%w: duplicate conversion", ErrConflict)
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	response := models.NewConversionResponse(conversion)
	return &response, nil
}

func (s *ConversionService) Update(ctx context.Context, tenantID, ingredientID, id uuid.UUID, req models.UpdateConversionRequest) (*models.ConversionResponse, error) {
	var updated *models.IngredientUOMConversion
	err := repository.WithTenantTx(ctx, s.repo.DB(), tenantID, nil, func(tx *sql.Tx) error {
		conversion, err := s.repo.FindByIDTx(ctx, tx, tenantID, ingredientID, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if req.FromUOMID != nil {
			conversion.FromUOMID = *req.FromUOMID
		}
		if req.ToUOMID != nil {
			conversion.ToUOMID = *req.ToUOMID
		}
		if req.Multiplier != nil {
			if req.Multiplier.LessThanOrEqual(decimal.Zero) {
				return fmt.Errorf("%w: multiplier must be greater than zero", ErrInvalidInput)
			}
			conversion.Multiplier = req.Multiplier.Decimal
		}
		if err := s.validateConversionTx(ctx, tx, tenantID, ingredientID, conversion.FromUOMID, conversion.ToUOMID); err != nil {
			return err
		}
		if err := s.repo.UpdateTx(ctx, tx, conversion); err != nil {
			return err
		}
		updated, err = s.repo.FindByIDTx(ctx, tx, tenantID, ingredientID, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	response := models.NewConversionResponse(updated)
	return &response, nil
}

func (s *ConversionService) Delete(ctx context.Context, tenantID, ingredientID, id uuid.UUID) error {
	return repository.WithTenantTx(ctx, s.repo.DB(), tenantID, nil, func(tx *sql.Tx) error {
		err := s.repo.DeleteTx(ctx, tx, tenantID, ingredientID, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
}

func (s *ConversionService) NormalizeQuantity(ctx context.Context, tenantID uuid.UUID, ingredient *models.Ingredient, quantity decimal.Decimal, fromUOMID uuid.UUID) (decimal.Decimal, error) {
	if fromUOMID == ingredient.BaseUOMID {
		return quantity, nil
	}
	conversion, err := s.repo.FindMultiplier(ctx, tenantID, &ingredient.ID, fromUOMID, ingredient.BaseUOMID)
	return normalizedQuantity(conversion, err, quantity)
}

func (s *ConversionService) NormalizeQuantityTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, ingredient *models.Ingredient, quantity decimal.Decimal, fromUOMID uuid.UUID) (decimal.Decimal, error) {
	if fromUOMID == ingredient.BaseUOMID {
		return quantity, nil
	}
	conversion, err := s.repo.FindMultiplierTx(ctx, tx, tenantID, &ingredient.ID, fromUOMID, ingredient.BaseUOMID)
	return normalizedQuantity(conversion, err, quantity)
}

func normalizedQuantity(conversion *models.IngredientUOMConversion, err error, quantity decimal.Decimal) (decimal.Decimal, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return decimal.Zero, ErrMissingConversion
	}
	if err != nil {
		return decimal.Zero, err
	}
	return quantity.Mul(conversion.Multiplier), nil
}

func (s *ConversionService) validateConversion(ctx context.Context, tenantID, ingredientID, fromUOMID, toUOMID uuid.UUID) error {
	return repository.WithTenantTx(ctx, s.repo.DB(), tenantID, nil, func(tx *sql.Tx) error {
		return s.validateConversionTx(ctx, tx, tenantID, ingredientID, fromUOMID, toUOMID)
	})
}

func (s *ConversionService) validateConversionTx(ctx context.Context, tx *sql.Tx, tenantID, ingredientID, fromUOMID, toUOMID uuid.UUID) error {
	if fromUOMID == toUOMID {
		return fmt.Errorf("%w: conversion units must be different", ErrInvalidInput)
	}
	if _, err := s.ingredientRepo.FindByIDTx(ctx, tx, tenantID, ingredientID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	from, err := s.uomRepo.FindByIDTx(ctx, tx, tenantID, fromUOMID)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: from uom not found", ErrInvalidInput)
	}
	if err != nil {
		return err
	}
	to, err := s.uomRepo.FindByIDTx(ctx, tx, tenantID, toUOMID)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: to uom not found", ErrInvalidInput)
	}
	if err != nil {
		return err
	}
	if !from.IsActive || !to.IsActive {
		return fmt.Errorf("%w: inactive uom", ErrInvalidInput)
	}
	if from.Category != to.Category && from.Category != "packaging" && to.Category != "packaging" {
		return fmt.Errorf("%w: cross-category conversion requires ingredient-specific packaging", ErrInvalidInput)
	}
	return nil
}
