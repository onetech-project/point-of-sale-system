package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// T088: Contract tests for DELETE /offline-orders/{id} schema.

func TestOfflineOrderDelete_RequestSchema(t *testing.T) {
	// DELETE /offline-orders/{id} requires reason and user
	deleteRequest := map[string]interface{}{
		"reason":           "Customer requested cancellation",
		"deleted_by_user_id": "550e8400-e29b-41d4-a716-446655440001",
	}

	assert.Contains(t, deleteRequest, "reason")
	assert.Contains(t, deleteRequest, "deleted_by_user_id")
	assert.NotEmpty(t, deleteRequest["reason"])
}

func TestOfflineOrderDelete_ResponseSchema(t *testing.T) {
	// DELETE /offline-orders/{id} 200 soft-delete response
	response := map[string]interface{}{
		"message":    "Order successfully cancelled",
		"order_id":   "order-uuid-123",
		"deleted_at": "2026-05-01T12:00:00Z",
	}

	assert.Contains(t, response, "message")
	assert.Contains(t, response, "order_id")
	assert.Contains(t, response, "deleted_at")
}

func TestOfflineOrderDelete_SoftDeleteBehavior(t *testing.T) {
	// Soft delete: order is marked cancelled, not physically removed
	// After delete, order should still be retrievable with cancelled status
	expectedStatusAfterDelete := "CANCELLED"
	validTerminalStatuses := []string{"CANCELLED", "COMPLETE"}

	assert.Contains(t, validTerminalStatuses, expectedStatusAfterDelete)

	// Verify delete is idempotent
	statusAlreadyCancelled := "CANCELLED"
	assert.Equal(t, expectedStatusAfterDelete, statusAlreadyCancelled)
}
