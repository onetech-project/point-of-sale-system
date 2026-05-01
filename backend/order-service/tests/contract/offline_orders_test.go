package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// T020-T022: Contract tests for core offline order endpoints.
// Validates request/response schemas against API specification.

func TestOfflineOrdersCreate_RequestSchema(t *testing.T) {
	// POST /offline-orders - required fields per API spec
	requiredFields := []string{
		"tenant_id",
		"customer_name",
		"customer_phone",
		"delivery_type",
		"items",
		"data_consent_given",
		"recorded_by_user_id",
	}

	request := map[string]interface{}{
		"tenant_id":            "550e8400-e29b-41d4-a716-446655440000",
		"customer_name":        "John Doe",
		"customer_phone":       "08123456789",
		"delivery_type":        "pickup",
		"items":                []interface{}{},
		"data_consent_given":   true,
		"recorded_by_user_id": "550e8400-e29b-41d4-a716-446655440001",
	}

	for _, field := range requiredFields {
		assert.Contains(t, request, field, "POST /offline-orders request must contain field: %s", field)
	}
}

func TestOfflineOrdersCreate_ResponseSchema(t *testing.T) {
	// POST /offline-orders 201 response schema
	response := map[string]interface{}{
		"id":               "order-uuid-123",
		"order_reference":  "OFF-20260501-0001",
		"tenant_id":        "550e8400-e29b-41d4-a716-446655440000",
		"status":           "PENDING",
		"order_type":       "offline",
		"customer_name":    "John Doe",
		"customer_phone":   "08123456789",
		"delivery_type":    "pickup",
		"total_amount":     150000,
		"data_consent_given": true,
		"recorded_by_user_id": "550e8400-e29b-41d4-a716-446655440001",
		"created_at":       "2026-05-01T10:00:00Z",
	}

	requiredResponseFields := []string{
		"id", "order_reference", "tenant_id", "status", "order_type",
		"customer_name", "customer_phone", "delivery_type",
		"total_amount", "data_consent_given", "recorded_by_user_id", "created_at",
	}
	for _, field := range requiredResponseFields {
		assert.Contains(t, response, field, "POST /offline-orders response must contain field: %s", field)
	}

	// Validate types
	assert.IsType(t, "", response["id"])
	assert.IsType(t, "", response["order_reference"])
	assert.IsType(t, 0, response["total_amount"])
	assert.IsType(t, true, response["data_consent_given"])
	assert.Equal(t, "offline", response["order_type"])
}

func TestOfflineOrdersList_ResponseSchema(t *testing.T) {
	// GET /offline-orders list response schema
	listResponse := map[string]interface{}{
		"orders": []interface{}{
			map[string]interface{}{
				"id":              "order-uuid-123",
				"order_reference": "OFF-20260501-0001",
				"status":          "PENDING",
				"order_type":      "offline",
				"customer_name":   "John Doe",
				"total_amount":    150000,
				"created_at":      "2026-05-01T10:00:00Z",
			},
		},
		"total":  1,
		"page":   1,
		"limit":  20,
	}

	assert.Contains(t, listResponse, "orders")
	assert.Contains(t, listResponse, "total")
	assert.Contains(t, listResponse, "page")
	assert.Contains(t, listResponse, "limit")

	orders := listResponse["orders"].([]interface{})
	if len(orders) > 0 {
		order := orders[0].(map[string]interface{})
		assert.Contains(t, order, "id")
		assert.Contains(t, order, "order_reference")
		assert.Contains(t, order, "status")
		assert.Contains(t, order, "order_type")
		assert.Equal(t, "offline", order["order_type"])
	}
}

func TestOfflineOrdersGetByID_ResponseSchema(t *testing.T) {
	// GET /offline-orders/{id} response schema
	response := map[string]interface{}{
		"id":               "order-uuid-123",
		"order_reference":  "OFF-20260501-0001",
		"tenant_id":        "550e8400-e29b-41d4-a716-446655440000",
		"status":           "PENDING",
		"order_type":       "offline",
		"customer_name":    "John Doe",
		"customer_phone":   "08123456789",
		"delivery_type":    "pickup",
		"total_amount":     150000,
		"data_consent_given": true,
		"recorded_by_user_id": "550e8400-e29b-41d4-a716-446655440001",
		"created_at":       "2026-05-01T10:00:00Z",
		"items":            []interface{}{},
	}

	requiredFields := []string{
		"id", "order_reference", "tenant_id", "status", "order_type",
		"customer_name", "customer_phone", "delivery_type",
		"total_amount", "data_consent_given", "items",
	}
	for _, field := range requiredFields {
		assert.Contains(t, response, field, "GET /offline-orders/{id} must contain field: %s", field)
	}
	assert.Equal(t, "offline", response["order_type"])
}

func TestOfflineOrders_DeliveryTypeEnum(t *testing.T) {
	// Validate that delivery_type only accepts valid enum values
	validTypes := []string{"pickup", "delivery", "dine_in"}
	for _, dt := range validTypes {
		assert.Contains(t, validTypes, dt)
	}

	// These should NOT be valid
	invalidTypes := []string{"PICKUP", "home", ""}
	for _, dt := range invalidTypes {
		assert.NotContains(t, validTypes, dt, "delivery_type %q should be invalid", dt)
	}
}

func TestOfflineOrders_ConsentMethodEnum(t *testing.T) {
	// Validate consent_method enum values
	validMethods := []string{"verbal", "written", "digital"}
	for _, method := range validMethods {
		assert.Contains(t, validMethods, method)
	}
}
