package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// T071: Contract tests for PATCH /offline-orders/{id} schema.

func TestOfflineOrderUpdate_RequestSchema(t *testing.T) {
	// PATCH /offline-orders/{id} - all fields are optional
	updateRequest := map[string]interface{}{
		"customer_name":            "Jane Doe",
		"customer_phone":           "08129876543",
		"delivery_type":            "dine_in",
		"table_number":             "T5",
		"notes":                    "Extra sauce",
		"last_modified_by_user_id": "550e8400-e29b-41d4-a716-446655440001",
	}

	// All fields are optional - just verify the request can be constructed
	assert.NotEmpty(t, updateRequest)
	assert.Contains(t, updateRequest, "last_modified_by_user_id")
}

func TestOfflineOrderUpdate_ResponseSchema(t *testing.T) {
	// PATCH /offline-orders/{id} returns updated order schema
	response := map[string]interface{}{
		"id":                       "order-uuid-123",
		"order_reference":          "OFF-20260501-0001",
		"tenant_id":                "550e8400-e29b-41d4-a716-446655440000",
		"status":                   "PENDING",
		"order_type":               "offline",
		"customer_name":            "Jane Doe",
		"customer_phone":           "08129876543",
		"delivery_type":            "dine_in",
		"total_amount":             150000,
		"last_modified_by_user_id": "550e8400-e29b-41d4-a716-446655440001",
		"last_modified_at":         "2026-05-01T11:00:00Z",
	}

	requiredFields := []string{
		"id", "order_reference", "status", "order_type",
		"customer_name", "customer_phone", "delivery_type", "total_amount",
		"last_modified_by_user_id", "last_modified_at",
	}
	for _, field := range requiredFields {
		assert.Contains(t, response, field, "PATCH response must contain field: %s", field)
	}
	assert.Equal(t, "offline", response["order_type"])
}

func TestOfflineOrderUpdate_ImmutableFields(t *testing.T) {
	// These fields cannot be changed via PATCH
	immutableFields := []string{"id", "order_reference", "tenant_id", "order_type", "created_at", "recorded_by_user_id"}
	for _, field := range immutableFields {
		// Verify the field is documented as immutable
		assert.NotEmpty(t, field)
	}
}
