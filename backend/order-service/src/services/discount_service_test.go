package services

import (
	"context"
	"testing"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/stretchr/testify/require"
)

type fakeDiscountRepository struct {
	rule            *models.DiscountRule
	applicableRules []models.DiscountRule
}

func (f *fakeDiscountRepository) Create(ctx context.Context, rule *models.DiscountRule) (*models.DiscountRule, error) {
	return rule, nil
}

func (f *fakeDiscountRepository) GetByID(ctx context.Context, tenantID, id string) (*models.DiscountRule, error) {
	return f.rule, nil
}

func (f *fakeDiscountRepository) List(ctx context.Context, filter models.ListDiscountRulesFilter) ([]models.DiscountRule, error) {
	return f.applicableRules, nil
}

func (f *fakeDiscountRepository) Update(ctx context.Context, rule *models.DiscountRule) (*models.DiscountRule, error) {
	return rule, nil
}

func (f *fakeDiscountRepository) Delete(ctx context.Context, tenantID, id string) error {
	return nil
}

func (f *fakeDiscountRepository) FindApplicable(ctx context.Context, tenantID string, targetType models.DiscountTargetType, targetID string, at time.Time) ([]models.DiscountRule, error) {
	return f.applicableRules, nil
}

func TestDiscountServicePreviewPricingNoDiscountPreservesPrice(t *testing.T) {
	productID := "product-1"
	service := NewDiscountService(&fakeDiscountRepository{})

	result, err := service.PreviewPricing(context.Background(), &models.PricingPreviewRequest{
		TenantID:  "tenant-1",
		ItemType:  models.DiscountTargetProduct,
		ProductID: &productID,
		Quantity:  3,
		UnitPrice: 10000,
	})

	require.NoError(t, err)
	require.Equal(t, 10000, result.UnitPrice)
	require.Equal(t, 30000, result.TotalPrice)
	require.Equal(t, 0, result.DiscountAmount)
	require.Nil(t, result.DiscountRuleID)
	require.NotEmpty(t, result.PricingSnapshot)
}

func TestDiscountServicePreviewPricingExplicitPercentageRule(t *testing.T) {
	productID := "product-1"
	ruleID := "discount-1"
	service := NewDiscountService(&fakeDiscountRepository{
		rule: &models.DiscountRule{
			ID:            ruleID,
			TenantID:      "tenant-1",
			Name:          "Ten percent off",
			TargetType:    models.DiscountTargetProduct,
			TargetID:      productID,
			DiscountType:  models.DiscountTypePercentage,
			DiscountValue: 10,
			IsActive:      true,
			Exclusive:     true,
			Priority:      100,
		},
	})

	result, err := service.PreviewPricing(context.Background(), &models.PricingPreviewRequest{
		TenantID:       "tenant-1",
		ItemType:       models.DiscountTargetProduct,
		ProductID:      &productID,
		Quantity:       2,
		UnitPrice:      10000,
		DiscountRuleID: &ruleID,
	})

	require.NoError(t, err)
	require.Equal(t, 9000, result.UnitPrice)
	require.Equal(t, 18000, result.TotalPrice)
	require.Equal(t, 2000, result.DiscountAmount)
	require.Equal(t, ruleID, *result.DiscountRuleID)
	require.Equal(t, models.DiscountTypePercentage, *result.DiscountType)
}

func TestDiscountServicePreviewPricingFixedAmountCapsAtZero(t *testing.T) {
	productID := "product-1"
	ruleID := "discount-1"
	service := NewDiscountService(&fakeDiscountRepository{
		rule: &models.DiscountRule{
			ID:            ruleID,
			TenantID:      "tenant-1",
			Name:          "Free item",
			TargetType:    models.DiscountTargetProduct,
			TargetID:      productID,
			DiscountType:  models.DiscountTypeFixedAmount,
			DiscountValue: 12000,
			IsActive:      true,
			Exclusive:     true,
			Priority:      100,
		},
	})

	result, err := service.PreviewPricing(context.Background(), &models.PricingPreviewRequest{
		TenantID:       "tenant-1",
		ItemType:       models.DiscountTargetProduct,
		ProductID:      &productID,
		Quantity:       2,
		UnitPrice:      10000,
		DiscountRuleID: &ruleID,
	})

	require.NoError(t, err)
	require.Equal(t, 0, result.UnitPrice)
	require.Equal(t, 0, result.TotalPrice)
	require.Equal(t, 20000, result.DiscountAmount)
}

func TestDiscountServicePreviewPricingRejectsWrongTarget(t *testing.T) {
	productID := "product-1"
	otherProductID := "product-2"
	ruleID := "discount-1"
	service := NewDiscountService(&fakeDiscountRepository{
		rule: &models.DiscountRule{
			ID:            ruleID,
			TenantID:      "tenant-1",
			Name:          "Wrong target",
			TargetType:    models.DiscountTargetProduct,
			TargetID:      otherProductID,
			DiscountType:  models.DiscountTypePercentage,
			DiscountValue: 10,
			IsActive:      true,
			Exclusive:     true,
			Priority:      100,
		},
	})

	_, err := service.PreviewPricing(context.Background(), &models.PricingPreviewRequest{
		TenantID:       "tenant-1",
		ItemType:       models.DiscountTargetProduct,
		ProductID:      &productID,
		Quantity:       1,
		UnitPrice:      10000,
		DiscountRuleID: &ruleID,
	})

	require.ErrorContains(t, err, "does not apply")
}

func TestDiscountServicePreviewPricingCanApplyApplicableBundleRule(t *testing.T) {
	bundleID := "bundle-1"
	service := NewDiscountService(&fakeDiscountRepository{
		applicableRules: []models.DiscountRule{
			{
				ID:            "discount-1",
				TenantID:      "tenant-1",
				Name:          "Bundle discount",
				TargetType:    models.DiscountTargetBundle,
				TargetID:      bundleID,
				DiscountType:  models.DiscountTypeFixedAmount,
				DiscountValue: 5000,
				IsActive:      true,
				Exclusive:     true,
				Priority:      100,
			},
		},
	})

	result, err := service.PreviewPricing(context.Background(), &models.PricingPreviewRequest{
		TenantID:      "tenant-1",
		ItemType:      models.DiscountTargetBundle,
		BundleID:      &bundleID,
		Quantity:      1,
		UnitPrice:     25000,
		ApplyDiscount: true,
	})

	require.NoError(t, err)
	require.Equal(t, 20000, result.UnitPrice)
	require.Equal(t, 5000, result.DiscountAmount)
	require.Equal(t, "discount-1", *result.DiscountRuleID)
}
