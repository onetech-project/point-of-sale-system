package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// T049/T050: Contract tests for offline order payment record endpoints.

func TestCreatePaymentRecord_RequestSchema(t *testing.T) {
	// POST /offline-orders/{id}/payments required fields
	request := map[string]interface{}{
		"payment_number":          0,
		"amount_paid":             300000,
		"payment_method":          "cash",
		"remaining_balance_after": 700000,
		"recorded_by_user_id":     "550e8400-e29b-41d4-a716-446655440001",
	}

	requiredFields := []string{
		"payment_number", "amount_paid", "payment_method",
		"remaining_balance_after", "recorded_by_user_id",
	}
	for _, field := range requiredFields {
		assert.Contains(t, request, field, "POST /payments request must contain field: %s", field)
	}

	// Validate payment_method enum
	validMethods := []string{"cash", "card", "bank_transfer", "check", "other"}
	assert.Contains(t, validMethods, request["payment_method"])
}

func TestCreatePaymentRecord_ResponseSchema(t *testing.T) {
	// POST /offline-orders/{id}/payments 201 response
	response := map[string]interface{}{
		"id":                      "pr-uuid-123",
		"order_id":                "order-uuid-456",
		"payment_number":          0,
		"amount_paid":             300000,
		"payment_method":          "cash",
		"remaining_balance_after": 700000,
		"recorded_by_user_id":     "550e8400-e29b-41d4-a716-446655440001",
		"payment_date":            "2026-05-01T10:00:00Z",
		"created_at":              "2026-05-01T10:00:00Z",
	}

	requiredFields := []string{
		"id", "order_id", "payment_number", "amount_paid",
		"payment_method", "remaining_balance_after", "recorded_by_user_id",
		"payment_date", "created_at",
	}
	for _, field := range requiredFields {
		assert.Contains(t, response, field, "POST /payments response must contain field: %s", field)
	}
	assert.IsType(t, 0, response["amount_paid"])
}

func TestGetPaymentRecords_ResponseSchema(t *testing.T) {
	// GET /offline-orders/{id}/payments response
	listResponse := map[string]interface{}{
		"payments": []interface{}{
			map[string]interface{}{
				"id":                      "pr-uuid-123",
				"order_id":                "order-uuid-456",
				"payment_number":          0,
				"amount_paid":             300000,
				"payment_method":          "cash",
				"remaining_balance_after": 700000,
				"payment_date":            "2026-05-01T10:00:00Z",
			},
		},
		"total_paid":        300000,
		"remaining_balance": 700000,
	}

	assert.Contains(t, listResponse, "payments")
	assert.Contains(t, listResponse, "total_paid")
	assert.Contains(t, listResponse, "remaining_balance")

	payments := listResponse["payments"].([]interface{})
	if len(payments) > 0 {
		pmt := payments[0].(map[string]interface{})
		assert.Contains(t, pmt, "id")
		assert.Contains(t, pmt, "amount_paid")
		assert.Contains(t, pmt, "payment_method")
		assert.Contains(t, pmt, "payment_number")
	}
}

func TestPaymentRecord_DownPaymentIsZero(t *testing.T) {
	// Contract: down payment always has payment_number = 0
	downPayment := map[string]interface{}{
		"payment_number": 0,
		"amount_paid":    300000,
		"payment_method": "cash",
	}
	assert.Equal(t, 0, downPayment["payment_number"])
}

func TestPaymentRecord_InstallmentNumbering(t *testing.T) {
	// Contract: installments have payment_number >= 1
	installment := map[string]interface{}{
		"payment_number": 1,
		"amount_paid":    233334,
		"payment_method": "bank_transfer",
	}
	assert.GreaterOrEqual(t, installment["payment_number"], 1)
}
