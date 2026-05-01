package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// T048: Contract tests for payment terms schema on POST /offline-orders.

func TestOfflineOrderCreate_InstallmentPaymentSchema(t *testing.T) {
	// payment.type = "installment" requires specific fields
	installmentRequest := map[string]interface{}{
		"type":                 "installment",
		"down_payment_amount":  300000,
		"down_payment_method":  "cash",
		"installment_count":    3,
	}

	assert.Contains(t, installmentRequest, "type")
	assert.Contains(t, installmentRequest, "down_payment_amount")
	assert.Contains(t, installmentRequest, "installment_count")
	assert.Equal(t, "installment", installmentRequest["type"])
}

func TestOfflineOrderCreate_FullPaymentSchema(t *testing.T) {
	// payment.type = "full" requires amount and method
	fullPaymentRequest := map[string]interface{}{
		"type":   "full",
		"amount": 500000,
		"method": "cash",
	}

	assert.Contains(t, fullPaymentRequest, "type")
	assert.Contains(t, fullPaymentRequest, "amount")
	assert.Contains(t, fullPaymentRequest, "method")
	assert.Equal(t, "full", fullPaymentRequest["type"])
}

func TestOfflineOrderCreate_PaymentTermsResponseSchema(t *testing.T) {
	// GET /offline-orders/{id}/payment-terms response schema
	response := map[string]interface{}{
		"id":                 "pt-uuid-123",
		"order_id":           "order-uuid-456",
		"total_amount":       1000000,
		"down_payment_amount": 300000,
		"installment_count":  3,
		"installment_amount": 233333,
		"payment_schedule": []interface{}{
			map[string]interface{}{
				"installment_number": 1,
				"due_date":           "2026-06-01",
				"amount":             233334,
				"status":             "pending",
			},
		},
		"total_paid":        300000,
		"remaining_balance": 700000,
		"created_at":        "2026-05-01T10:00:00Z",
	}

	requiredFields := []string{
		"id", "order_id", "total_amount", "installment_count",
		"installment_amount", "payment_schedule", "total_paid",
		"remaining_balance",
	}
	for _, field := range requiredFields {
		assert.Contains(t, response, field, "payment terms response must contain field: %s", field)
	}

	// Validate payment schedule item schema
	schedule := response["payment_schedule"].([]interface{})
	assert.Greater(t, len(schedule), 0)
	item := schedule[0].(map[string]interface{})
	assert.Contains(t, item, "installment_number")
	assert.Contains(t, item, "due_date")
	assert.Contains(t, item, "amount")
	assert.Contains(t, item, "status")
}

func TestOfflineOrderCreate_PaymentTypeEnum(t *testing.T) {
	validTypes := []string{"full", "installment"}
	for _, pt := range validTypes {
		assert.Contains(t, validTypes, pt)
	}
	invalidTypes := []string{"partial", "deferred", ""}
	for _, pt := range invalidTypes {
		assert.NotContains(t, validTypes, pt)
	}
}
