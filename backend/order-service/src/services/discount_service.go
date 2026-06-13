package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
)

type discountRepository interface {
	Create(ctx context.Context, rule *models.DiscountRule) (*models.DiscountRule, error)
	GetByID(ctx context.Context, tenantID, id string) (*models.DiscountRule, error)
	List(ctx context.Context, filter models.ListDiscountRulesFilter) ([]models.DiscountRule, error)
	Update(ctx context.Context, rule *models.DiscountRule) (*models.DiscountRule, error)
	Delete(ctx context.Context, tenantID, id string) error
	FindApplicable(ctx context.Context, tenantID string, targetType models.DiscountTargetType, targetID string, at time.Time) ([]models.DiscountRule, error)
}

type DiscountService struct {
	repo discountRepository
}

func NewDiscountService(repo discountRepository) *DiscountService {
	return &DiscountService{repo: repo}
}

func (s *DiscountService) Create(ctx context.Context, tenantID string, req *models.CreateDiscountRuleRequest) (*models.DiscountRule, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	exclusive := true
	if req.Exclusive != nil {
		exclusive = *req.Exclusive
	}
	priority := 100
	if req.Priority != nil {
		priority = *req.Priority
	}

	rule := &models.DiscountRule{
		TenantID:      tenantID,
		Name:          strings.TrimSpace(req.Name),
		Description:   req.Description,
		TargetType:    req.TargetType,
		TargetID:      strings.TrimSpace(req.TargetID),
		DiscountType:  req.DiscountType,
		DiscountValue: req.DiscountValue,
		StartsAt:      req.StartsAt,
		EndsAt:        req.EndsAt,
		IsActive:      isActive,
		Exclusive:     exclusive,
		Priority:      priority,
	}

	if err := validateDiscountRule(rule); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, rule)
}

func (s *DiscountService) GetByID(ctx context.Context, tenantID, id string) (*models.DiscountRule, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *DiscountService) List(ctx context.Context, filter models.ListDiscountRulesFilter) ([]models.DiscountRule, error) {
	return s.repo.List(ctx, filter)
}

func (s *DiscountService) Update(ctx context.Context, tenantID, id string, req *models.UpdateDiscountRuleRequest) (*models.DiscountRule, error) {
	existing, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	if req.Name != nil {
		existing.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		existing.Description = req.Description
	}
	if req.TargetType != nil {
		existing.TargetType = *req.TargetType
	}
	if req.TargetID != nil {
		existing.TargetID = strings.TrimSpace(*req.TargetID)
	}
	if req.DiscountType != nil {
		existing.DiscountType = *req.DiscountType
	}
	if req.DiscountValue != nil {
		existing.DiscountValue = *req.DiscountValue
	}
	if req.StartsAt != nil {
		existing.StartsAt = req.StartsAt
	}
	if req.EndsAt != nil {
		existing.EndsAt = req.EndsAt
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if req.Exclusive != nil {
		existing.Exclusive = *req.Exclusive
	}
	if req.Priority != nil {
		existing.Priority = *req.Priority
	}

	if err := validateDiscountRule(existing); err != nil {
		return nil, err
	}

	return s.repo.Update(ctx, existing)
}

func (s *DiscountService) Delete(ctx context.Context, tenantID, id string) error {
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *DiscountService) PreviewPricing(ctx context.Context, req *models.PricingPreviewRequest) (*models.PricingResult, error) {
	if err := validatePricingRequest(req); err != nil {
		return nil, err
	}

	at := time.Now().UTC()
	if req.At != nil {
		at = req.At.UTC()
	}

	var selected *models.DiscountRule
	if req.DiscountRuleID != nil && *req.DiscountRuleID != "" {
		rule, err := s.repo.GetByID(ctx, req.TenantID, *req.DiscountRuleID)
		if err != nil {
			return nil, err
		}
		if rule == nil {
			return nil, fmt.Errorf("discount rule not found")
		}
		if err := validateRuleApplies(rule, req, at); err != nil {
			return nil, err
		}
		selected = rule
	} else if req.ApplyDiscount {
		rules, err := s.repo.FindApplicable(ctx, req.TenantID, req.ItemType, pricingTargetID(req), at)
		if err != nil {
			return nil, err
		}
		if len(rules) > 0 {
			selected = &rules[0]
		}
	}

	return buildPricingResult(req, selected, at)
}

func validateDiscountRule(rule *models.DiscountRule) error {
	if strings.TrimSpace(rule.TenantID) == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if strings.TrimSpace(rule.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if rule.TargetType != models.DiscountTargetProduct && rule.TargetType != models.DiscountTargetBundle {
		return fmt.Errorf("target_type must be product or bundle")
	}
	if strings.TrimSpace(rule.TargetID) == "" {
		return fmt.Errorf("target_id is required")
	}
	if rule.DiscountType != models.DiscountTypePercentage && rule.DiscountType != models.DiscountTypeFixedAmount {
		return fmt.Errorf("discount_type must be percentage or fixed_amount")
	}
	if rule.DiscountType == models.DiscountTypePercentage && (rule.DiscountValue <= 0 || rule.DiscountValue > 100) {
		return fmt.Errorf("percentage discount_value must be greater than 0 and less than or equal to 100")
	}
	if rule.DiscountType == models.DiscountTypeFixedAmount && rule.DiscountValue <= 0 {
		return fmt.Errorf("fixed_amount discount_value must be greater than 0")
	}
	if rule.StartsAt != nil && rule.EndsAt != nil && !rule.StartsAt.Before(*rule.EndsAt) {
		return fmt.Errorf("starts_at must be before ends_at")
	}
	if rule.Priority < 0 {
		return fmt.Errorf("priority cannot be negative")
	}
	return nil
}

func validatePricingRequest(req *models.PricingPreviewRequest) error {
	if req == nil {
		return fmt.Errorf("pricing request is required")
	}
	if strings.TrimSpace(req.TenantID) == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if req.ItemType == "" {
		req.ItemType = models.DiscountTargetProduct
	}
	if req.ItemType != models.DiscountTargetProduct && req.ItemType != models.DiscountTargetBundle {
		return fmt.Errorf("item_type must be product or bundle")
	}
	if req.ItemType == models.DiscountTargetProduct && (req.ProductID == nil || strings.TrimSpace(*req.ProductID) == "") {
		return fmt.Errorf("product_id is required for product pricing")
	}
	if req.ItemType == models.DiscountTargetBundle && (req.BundleID == nil || strings.TrimSpace(*req.BundleID) == "") {
		return fmt.Errorf("bundle_id is required for bundle pricing")
	}
	if req.Quantity <= 0 {
		return fmt.Errorf("quantity must be greater than 0")
	}
	if req.UnitPrice < 0 {
		return fmt.Errorf("unit_price cannot be negative")
	}
	return nil
}

func validateRuleApplies(rule *models.DiscountRule, req *models.PricingPreviewRequest, at time.Time) error {
	if !rule.IsActive {
		return fmt.Errorf("discount rule is inactive")
	}
	if rule.TargetType != req.ItemType || rule.TargetID != pricingTargetID(req) {
		return fmt.Errorf("discount rule does not apply to this item")
	}
	if rule.StartsAt != nil && rule.StartsAt.After(at) {
		return fmt.Errorf("discount rule has not started")
	}
	if rule.EndsAt != nil && rule.EndsAt.Before(at) {
		return fmt.Errorf("discount rule has expired")
	}
	return nil
}

func pricingTargetID(req *models.PricingPreviewRequest) string {
	if req.ItemType == models.DiscountTargetBundle && req.BundleID != nil {
		return *req.BundleID
	}
	if req.ProductID != nil {
		return *req.ProductID
	}
	return ""
}

func buildPricingResult(req *models.PricingPreviewRequest, rule *models.DiscountRule, at time.Time) (*models.PricingResult, error) {
	unitDiscount := 0
	var ruleID *string
	var discountType *models.DiscountType
	var discountValue *float64
	var discountName *string

	if rule != nil {
		unitDiscount = calculateUnitDiscount(req.UnitPrice, rule)
		if unitDiscount > req.UnitPrice {
			unitDiscount = req.UnitPrice
		}
		ruleID = &rule.ID
		discountType = &rule.DiscountType
		discountValue = &rule.DiscountValue
		discountName = &rule.Name
	}

	finalUnitPrice := req.UnitPrice - unitDiscount
	if finalUnitPrice < 0 {
		finalUnitPrice = 0
	}

	result := &models.PricingResult{
		ItemType:       req.ItemType,
		ProductID:      req.ProductID,
		BundleID:       req.BundleID,
		Quantity:       req.Quantity,
		ListUnitPrice:  req.UnitPrice,
		ListTotalPrice: req.UnitPrice * req.Quantity,
		UnitPrice:      finalUnitPrice,
		TotalPrice:     finalUnitPrice * req.Quantity,
		DiscountAmount: unitDiscount * req.Quantity,
		DiscountRuleID: ruleID,
		DiscountType:   discountType,
		DiscountValue:  discountValue,
		DiscountName:   discountName,
	}

	snapshot := map[string]interface{}{
		"item_type":          result.ItemType,
		"target_id":          pricingTargetID(req),
		"list_unit_price":    result.ListUnitPrice,
		"list_total_price":   result.ListTotalPrice,
		"unit_price":         result.UnitPrice,
		"total_price":        result.TotalPrice,
		"discount_amount":    result.DiscountAmount,
		"discount_applied":   rule != nil,
		"pricing_applied_at": at.Format(time.RFC3339),
	}
	if rule != nil {
		snapshot["discount_rule"] = map[string]interface{}{
			"id":             rule.ID,
			"name":           rule.Name,
			"discount_type":  rule.DiscountType,
			"discount_value": rule.DiscountValue,
			"exclusive":      rule.Exclusive,
			"priority":       rule.Priority,
		}
	}

	raw, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	result.PricingSnapshot = raw

	return result, nil
}

func calculateUnitDiscount(unitPrice int, rule *models.DiscountRule) int {
	switch rule.DiscountType {
	case models.DiscountTypePercentage:
		return int(math.Round(float64(unitPrice) * rule.DiscountValue / 100))
	case models.DiscountTypeFixedAmount:
		return int(math.Round(rule.DiscountValue))
	default:
		return 0
	}
}
