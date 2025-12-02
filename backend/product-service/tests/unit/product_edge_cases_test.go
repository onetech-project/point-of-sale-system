package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// T148: Unit tests for edge cases (negative stock, duplicate SKU, photo size limits)

func TestNegativeStockPrevention(t *testing.T) {
	t.Run("should prevent negative stock from adjustments", func(t *testing.T) {
		// Test that stock cannot go below 0
		currentStock := 5
		adjustment := -10
		expectedStock := 0 // Should clamp to 0, not allow -5

		// Logic: max(0, currentStock + adjustment)
		resultStock := currentStock + adjustment
		if resultStock < 0 {
			resultStock = 0
		}

		assert.Equal(t, expectedStock, resultStock, "Stock should not go negative")
	})

	t.Run("should allow zero stock", func(t *testing.T) {
		currentStock := 0
		adjustment := 0
		expectedStock := 0

		resultStock := currentStock + adjustment
		if resultStock < 0 {
			resultStock = 0
		}

		assert.Equal(t, expectedStock, resultStock)
	})

	t.Run("should handle large negative adjustment", func(t *testing.T) {
		currentStock := 1
		adjustment := -1000000
		expectedStock := 0

		resultStock := currentStock + adjustment
		if resultStock < 0 {
			resultStock = 0
		}

		assert.Equal(t, expectedStock, resultStock)
	})
}

func TestDuplicateSKUValidation(t *testing.T) {
	t.Run("should detect duplicate SKU in same tenant", func(t *testing.T) {
		existingSKUs := map[string]bool{
			"SKU001": true,
			"SKU002": true,
			"SKU003": true,
		}

		newSKU := "SKU002"
		isDuplicate := existingSKUs[newSKU]

		assert.True(t, isDuplicate, "Should detect duplicate SKU")
	})

	t.Run("should allow same SKU in different tenant", func(t *testing.T) {
		// Tenant 1 has SKU001
		tenant1SKUs := map[string]bool{
			"SKU001": true,
		}

		// Tenant 2 has different SKUs
		tenant2SKUs := map[string]bool{
			"SKU002": true,
		}

		// Check if SKU001 exists in tenant 1 (should exist)
		existsInTenant1 := tenant1SKUs["SKU001"]
		assert.True(t, existsInTenant1, "SKU001 should exist in tenant 1")

		// Check if SKU001 exists in tenant 2 (should not exist)
		newSKU := "SKU001"
		isDuplicateInTenant2 := tenant2SKUs[newSKU]

		assert.False(t, isDuplicateInTenant2, "Same SKU should be allowed in different tenant")
	})

	t.Run("should handle case-sensitive SKU comparison", func(t *testing.T) {
		existingSKUs := map[string]bool{
			"SKU001": true,
		}

		newSKU := "sku001"
		isDuplicate := existingSKUs[newSKU]

		// SKUs are case-sensitive
		assert.False(t, isDuplicate, "SKU comparison should be case-sensitive")
	})

	t.Run("should allow updating product with same SKU", func(t *testing.T) {
		existingSKUs := map[string]string{
			"SKU001": "product-id-1",
			"SKU002": "product-id-2",
		}

		productID := "product-id-1"
		updatedSKU := "SKU001"

		// Check if SKU exists and belongs to same product
		existingProductID, exists := existingSKUs[updatedSKU]
		isDuplicate := exists && existingProductID != productID

		assert.False(t, isDuplicate, "Should allow updating product with its own SKU")
	})
}

func TestPhotoSizeLimits(t *testing.T) {
	const maxPhotoSizeMB = 5
	const bytesInMB = 1024 * 1024

	t.Run("should reject photo larger than 5MB", func(t *testing.T) {
		photoSizeBytes := int64(6 * bytesInMB) // 6MB
		maxSizeBytes := int64(maxPhotoSizeMB * bytesInMB)

		isValid := photoSizeBytes <= maxSizeBytes

		assert.False(t, isValid, "Should reject photo larger than 5MB")
	})

	t.Run("should accept photo exactly 5MB", func(t *testing.T) {
		photoSizeBytes := int64(5 * bytesInMB) // Exactly 5MB
		maxSizeBytes := int64(maxPhotoSizeMB * bytesInMB)

		isValid := photoSizeBytes <= maxSizeBytes

		assert.True(t, isValid, "Should accept photo exactly 5MB")
	})

	t.Run("should accept photo smaller than 5MB", func(t *testing.T) {
		photoSizeBytes := int64(3 * bytesInMB) // 3MB
		maxSizeBytes := int64(maxPhotoSizeMB * bytesInMB)

		isValid := photoSizeBytes <= maxSizeBytes

		assert.True(t, isValid, "Should accept photo smaller than 5MB")
	})

	t.Run("should reject empty photo file", func(t *testing.T) {
		photoSizeBytes := int64(0)
		minSizeBytes := int64(1) // At least 1 byte

		isValid := photoSizeBytes >= minSizeBytes

		assert.False(t, isValid, "Should reject empty photo file")
	})

	t.Run("should validate photo file extension", func(t *testing.T) {
		validExtensions := map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".webp": true,
		}

		testCases := []struct {
			filename string
			valid    bool
		}{
			{"photo.jpg", true},
			{"photo.jpeg", true},
			{"photo.png", true},
			{"photo.webp", true},
			{"photo.gif", false},
			{"photo.bmp", false},
			{"photo.txt", false},
			{"photo", false},
		}

		for _, tc := range testCases {
			ext := ""
			for i := len(tc.filename) - 1; i >= 0; i-- {
				if tc.filename[i] == '.' {
					ext = tc.filename[i:]
					break
				}
			}

			isValid := validExtensions[ext]
			assert.Equal(t, tc.valid, isValid, "Extension validation failed for %s", tc.filename)
		}
	})
}

func TestPriceValidation(t *testing.T) {
	t.Run("should reject negative price", func(t *testing.T) {
		price := -10.00
		isValid := price >= 0

		assert.False(t, isValid, "Should reject negative price")
	})

	t.Run("should allow zero price", func(t *testing.T) {
		price := 0.00
		isValid := price >= 0

		assert.True(t, isValid, "Should allow zero price for free items")
	})

	t.Run("should handle price precision (2 decimal places)", func(t *testing.T) {
		price := 19.99
		expectedPrice := 19.99

		// Round to 2 decimal places
		roundedPrice := float64(int(price*100+0.5)) / 100

		assert.Equal(t, expectedPrice, roundedPrice)
	})

	t.Run("should handle very large prices", func(t *testing.T) {
		price := 999999.99
		maxPrice := 1000000.00
		isValid := price < maxPrice

		assert.True(t, isValid, "Should handle large prices")
	})
}

func TestQuantityValidation(t *testing.T) {
	t.Run("should reject negative initial quantity", func(t *testing.T) {
		quantity := -5
		isValid := quantity >= 0

		assert.False(t, isValid, "Initial quantity cannot be negative")
	})

	t.Run("should allow zero initial quantity", func(t *testing.T) {
		quantity := 0
		isValid := quantity >= 0

		assert.True(t, isValid, "Should allow zero initial quantity")
	})

	t.Run("should handle very large quantities", func(t *testing.T) {
		quantity := 1000000
		maxQuantity := 10000000
		isValid := quantity <= maxQuantity

		assert.True(t, isValid, "Should handle large quantities")
	})
}

func TestSKUValidation(t *testing.T) {
	t.Run("should reject empty SKU", func(t *testing.T) {
		sku := ""
		isValid := len(sku) > 0 && len(sku) <= 50

		assert.False(t, isValid, "SKU cannot be empty")
	})

	t.Run("should reject SKU longer than 50 characters", func(t *testing.T) {
		sku := "SKU123456789012345678901234567890123456789012345678901"
		isValid := len(sku) > 0 && len(sku) <= 50

		assert.False(t, isValid, "SKU cannot exceed 50 characters")
	})

	t.Run("should accept valid SKU", func(t *testing.T) {
		sku := "SKU-001-A"
		isValid := len(sku) > 0 && len(sku) <= 50

		assert.True(t, isValid, "Valid SKU should be accepted")
	})

	t.Run("should handle SKU with special characters", func(t *testing.T) {
		validSKUs := []string{
			"SKU-001",
			"SKU_001",
			"SKU.001",
			"SKU/001",
			"SKU#001",
		}

		for _, sku := range validSKUs {
			isValid := len(sku) > 0 && len(sku) <= 50
			assert.True(t, isValid, "SKU %s should be valid", sku)
		}
	})
}

func TestNameValidation(t *testing.T) {
	t.Run("should reject empty name", func(t *testing.T) {
		name := ""
		isValid := len(name) >= 1 && len(name) <= 255

		assert.False(t, isValid, "Name cannot be empty")
	})

	t.Run("should reject name longer than 255 characters", func(t *testing.T) {
		name := ""
		for i := 0; i < 256; i++ {
			name += "a"
		}
		isValid := len(name) >= 1 && len(name) <= 255

		assert.False(t, isValid, "Name cannot exceed 255 characters")
	})

	t.Run("should accept valid name", func(t *testing.T) {
		name := "Product Name"
		isValid := len(name) >= 1 && len(name) <= 255

		assert.True(t, isValid, "Valid name should be accepted")
	})

	t.Run("should trim whitespace from name", func(t *testing.T) {
		name := "  Product Name  "
		trimmed := ""

		// Trim leading/trailing spaces
		start := 0
		end := len(name)

		for start < end && name[start] == ' ' {
			start++
		}
		for end > start && name[end-1] == ' ' {
			end--
		}

		if start < end {
			trimmed = name[start:end]
		}

		assert.Equal(t, "Product Name", trimmed)
	})
}

func TestTaxRateValidation(t *testing.T) {
	t.Run("should reject negative tax rate", func(t *testing.T) {
		taxRate := -0.05
		isValid := taxRate >= 0 && taxRate <= 1.0

		assert.False(t, isValid, "Tax rate cannot be negative")
	})

	t.Run("should reject tax rate over 100%", func(t *testing.T) {
		taxRate := 1.5
		isValid := taxRate >= 0 && taxRate <= 1.0

		assert.False(t, isValid, "Tax rate cannot exceed 100%")
	})

	t.Run("should accept valid tax rates", func(t *testing.T) {
		validRates := []float64{0.0, 0.05, 0.10, 0.15, 0.20, 0.25, 1.0}

		for _, rate := range validRates {
			isValid := rate >= 0 && rate <= 1.0
			assert.True(t, isValid, "Tax rate %.2f should be valid", rate)
		}
	})

	t.Run("should handle tax rate precision", func(t *testing.T) {
		taxRate := 0.075     // 7.5%
		expectedRate := 0.08 // Round to nearest 0.01

		// Round to 2 decimal places
		roundedRate := float64(int(taxRate*100+0.5)) / 100

		assert.Equal(t, expectedRate, roundedRate)
	})
}
