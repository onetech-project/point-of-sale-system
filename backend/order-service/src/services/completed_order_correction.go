package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/observability"
	"github.com/point-of-sale-system/order-service/src/repository"
	"github.com/point-of-sale-system/order-service/src/utils"
)

// Owner corrections of COMPLETE offline orders. A completed order is already counted in revenue,
// has taken its stock and has been paid in full, so editing or deleting it changes the books.
// These operations therefore require an explicit acknowledgement and a reason, and every change
// is written to the audit trail with before and after snapshots.

var (
	// ErrOrderNotFound is returned when the order does not exist, belongs to another tenant or was deleted
	ErrOrderNotFound = repository.ErrOfflineOrderNotFound
	// ErrOrderNotComplete is returned when a completed-order correction targets an order in another status
	ErrOrderNotComplete = errors.New("order is not complete")
	// ErrAccountingImpactNotAcknowledged is returned when the caller did not confirm the accounting impact
	ErrAccountingImpactNotAcknowledged = errors.New("accounting impact must be acknowledged")
	// ErrInvalidCorrection is returned when the correction request is incomplete or inconsistent
	ErrInvalidCorrection = errors.New("invalid correction")
)

const (
	minCorrectionReasonLength = 5
	maxCorrectionReasonLength = 500
	auditTimeout              = 5 * time.Second
)

// OrderCorrectionActor identifies who made a correction, for the audit trail
type OrderCorrectionActor struct {
	UserID    string
	Role      string
	UserAgent string
	RequestID string
}

// CompletedOrderCorrection holds what every completed-order correction needs
type CompletedOrderCorrection struct {
	OrderID                     string
	TenantID                    string
	Actor                       OrderCorrectionActor
	Reason                      string
	AcknowledgeAccountingImpact bool
}

// CompletedOrderEditRequest edits the details and items of a COMPLETE offline order
type CompletedOrderEditRequest struct {
	CompletedOrderCorrection
	Updates models.UpdateOfflineOrderRequest
}

// CompletedOrderDeleteRequest deletes a COMPLETE offline order
type CompletedOrderDeleteRequest struct {
	CompletedOrderCorrection
}

func (c *CompletedOrderCorrection) validate() error {
	if !c.AcknowledgeAccountingImpact {
		return ErrAccountingImpactNotAcknowledged
	}
	c.Reason = strings.TrimSpace(c.Reason)
	if len(c.Reason) < minCorrectionReasonLength || len(c.Reason) > maxCorrectionReasonLength {
		return fmt.Errorf("%w: reason must be between %d and %d characters",
			ErrInvalidCorrection, minCorrectionReasonLength, maxCorrectionReasonLength)
	}
	return nil
}

func validateCorrectionUpdates(updates *models.UpdateOfflineOrderRequest) error {
	if updates.DeliveryFee != nil && *updates.DeliveryFee < 0 {
		return fmt.Errorf("%w: delivery fee cannot be negative", ErrInvalidCorrection)
	}
	for _, item := range updates.Items {
		if item.ProductID == "" || item.ProductName == "" || item.Quantity < 1 || item.UnitPrice < 0 {
			return fmt.Errorf("%w: every item needs a product, a quantity of at least 1 and a non-negative price", ErrInvalidCorrection)
		}
	}
	return nil
}

// UpdateCompletedOfflineOrder applies an owner's correction to a COMPLETE offline order.
// Totals are recalculated from the items, and stock the order took is swapped for the new items.
// The order stays COMPLETE; the gap between the new total and what was paid is reported in the audit trail.
func (s *OfflineOrderService) UpdateCompletedOfflineOrder(ctx context.Context, req *CompletedOrderEditRequest) (*models.GuestOrder, error) {
	ctx, span := s.tracer.Start(ctx, "UpdateCompletedOfflineOrder",
		trace.WithAttributes(
			attribute.String("order_id", req.OrderID),
			attribute.String("tenant_id", req.TenantID),
			attribute.String("modified_by", req.Actor.UserID),
		),
	)
	defer span.End()

	if err := req.validate(); err != nil {
		return nil, err
	}
	if err := validateCorrectionUpdates(&req.Updates); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Lock the order so its status cannot change while the correction is applied
	status, err := s.offlineOrderRepo.LockOfflineOrder(ctx, tx, req.OrderID, req.TenantID)
	if err != nil {
		return nil, err
	}
	if status != models.OrderStatusComplete {
		return nil, fmt.Errorf("%w: order status is %s", ErrOrderNotComplete, status)
	}

	before, beforeItems, err := s.loadOrderWithItems(ctx, req.OrderID, req.TenantID)
	if err != nil {
		return nil, err
	}

	changes := s.detectChanges(before, &UpdateOfflineOrderRequest{ModelUpdates: req.Updates})
	itemsChanged := len(req.Updates.Items) > 0 && !sameItems(beforeItems, req.Updates.Items)
	if len(changes) == 0 && !itemsChanged {
		return nil, fmt.Errorf("%w: no changes detected", ErrInvalidCorrection)
	}

	// Also stamps last_modified_by_user_id / last_modified_at
	if err := s.offlineOrderRepo.UpdateOfflineOrder(ctx, tx, req.OrderID, req.TenantID, &req.Updates, req.Actor.UserID); err != nil {
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	stockAdjusted := false
	if itemsChanged {
		// Give back the stock the old items took before taking stock for the new ones
		allocated, err := s.inventoryService.ReturnAllocatedStockTx(ctx, tx, req.TenantID, req.OrderID)
		if err != nil {
			return nil, err
		}
		if _, _, err := s.offlineOrderRepo.UpdateOrderItems(ctx, tx, req.OrderID, req.TenantID, req.Updates.Items); err != nil {
			return nil, fmt.Errorf("failed to update order items: %w", err)
		}
		// Orders recorded before stock tracking never took stock, so there is nothing to swap
		if len(allocated) > 0 {
			if err := s.inventoryService.ReserveStockForOrder(ctx, tx, req.TenantID, req.OrderID, toCartItems(req.Updates.Items), time.Now().Add(OfflineReservationTTL)); err != nil {
				span.RecordError(err)
				span.SetAttributes(attribute.String("error.type", "insufficient_stock"))
				return nil, err
			}
			if err := s.inventoryService.ConvertReservationsTx(ctx, tx, req.OrderID); err != nil {
				return nil, fmt.Errorf("failed to allocate stock: %w", err)
			}
			stockAdjusted = true
		}
	}

	subtotal, total, err := s.offlineOrderRepo.RecalculateOrderTotals(ctx, tx, req.OrderID, req.TenantID)
	if err != nil {
		return nil, err
	}

	changedFields := make([]string, 0, len(changes)+3)
	for field := range changes {
		changedFields = append(changedFields, field)
	}
	if itemsChanged {
		changedFields = append(changedFields, "items")
	}
	if subtotal != before.SubtotalAmount {
		changedFields = append(changedFields, "subtotal_amount")
	}
	if total != before.TotalAmount {
		changedFields = append(changedFields, "total_amount")
	}
	sort.Strings(changedFields)

	// The outbox payload carries field names and amounts only, the customer data stays out of it
	if err := s.createCorrectionOutboxEvent(ctx, tx, "offline_order.completed_updated", req.CompletedOrderCorrection, map[string]interface{}{
		"changed_fields":   changedFields,
		"old_total_amount": before.TotalAmount,
		"new_total_amount": total,
		"stock_adjusted":   stockAdjusted,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", "transaction_commit_failed"))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.analyticsCache.InvalidateTenant(ctx, req.TenantID)
	observability.OfflineOrderUpdatesTotal.WithLabelValues(req.TenantID).Inc()

	after, afterItems, err := s.loadOrderWithItems(ctx, req.OrderID, req.TenantID)
	if err != nil {
		return nil, err
	}

	totalPaid := s.totalPaid(ctx, req.OrderID)
	s.publishCorrectionAudit(req.CompletedOrderCorrection, "UPDATE",
		s.auditSnapshot(ctx, before, beforeItems),
		s.auditSnapshot(ctx, after, afterItems),
		map[string]interface{}{
			"change_type":      "completed_order_edit",
			"changed_fields":   changedFields,
			"stock_adjusted":   stockAdjusted,
			"total_paid":       totalPaid,
			"old_total_amount": before.TotalAmount,
			"new_total_amount": after.TotalAmount,
			// Positive: the customer owes more. Negative: the customer paid more than the new total.
			"outstanding_balance": after.TotalAmount - totalPaid,
		},
	)

	span.SetAttributes(attribute.Int("change_count", len(changedFields)))
	log.Info().
		Str("order_id", req.OrderID).
		Str("tenant_id", req.TenantID).
		Str("modified_by", req.Actor.UserID).
		Strs("changed_fields", changedFields).
		Msg("Completed offline order corrected")

	return after, nil
}

// DeleteCompletedOfflineOrder soft-deletes a COMPLETE offline order on an owner's request.
// The order is moved to CANCELLED so it drops out of revenue, and the stock it took is returned.
func (s *OfflineOrderService) DeleteCompletedOfflineOrder(ctx context.Context, req *CompletedOrderDeleteRequest) error {
	ctx, span := s.tracer.Start(ctx, "DeleteCompletedOfflineOrder",
		trace.WithAttributes(
			attribute.String("order_id", req.OrderID),
			attribute.String("tenant_id", req.TenantID),
			attribute.String("deleted_by", req.Actor.UserID),
		),
	)
	defer span.End()

	if err := req.validate(); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	status, err := s.offlineOrderRepo.LockOfflineOrder(ctx, tx, req.OrderID, req.TenantID)
	if err != nil {
		return err
	}
	if status != models.OrderStatusComplete {
		return fmt.Errorf("%w: order status is %s", ErrOrderNotComplete, status)
	}

	before, beforeItems, err := s.loadOrderWithItems(ctx, req.OrderID, req.TenantID)
	if err != nil {
		return err
	}

	returned, err := s.inventoryService.ReturnAllocatedStockTx(ctx, tx, req.TenantID, req.OrderID)
	if err != nil {
		return err
	}

	if err := s.offlineOrderRepo.SoftDeleteCompletedOfflineOrder(ctx, tx, req.OrderID, req.TenantID, req.Actor.UserID); err != nil {
		return err
	}

	if err := s.createCorrectionOutboxEvent(ctx, tx, "offline_order.completed_deleted", req.CompletedOrderCorrection, map[string]interface{}{
		"total_amount":   before.TotalAmount,
		"stock_returned": returned,
	}); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", "transaction_commit_failed"))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.analyticsCache.InvalidateTenant(ctx, req.TenantID)
	observability.OfflineOrderDeletionsTotal.WithLabelValues(req.TenantID, req.Actor.Role).Inc()

	s.publishCorrectionAudit(req.CompletedOrderCorrection, "DELETE",
		s.auditSnapshot(ctx, before, beforeItems),
		nil,
		map[string]interface{}{
			"change_type":         "completed_order_delete",
			"total_paid":          s.totalPaid(ctx, req.OrderID),
			"removed_revenue":     before.TotalAmount,
			"stock_returned":      returned,
			"status_after_delete": models.OrderStatusCancelled,
		},
	)

	log.Info().
		Str("order_id", req.OrderID).
		Str("tenant_id", req.TenantID).
		Str("deleted_by", req.Actor.UserID).
		Msg("Completed offline order deleted")

	return nil
}

func (s *OfflineOrderService) loadOrderWithItems(ctx context.Context, orderID, tenantID string) (*models.GuestOrder, []models.OrderItem, error) {
	order, err := s.offlineOrderRepo.GetOfflineOrderByID(ctx, orderID, tenantID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load order: %w", err)
	}
	items, err := s.orderItemRepo.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load order items: %w", err)
	}
	return order, items, nil
}

// totalPaid sums the payments recorded for an order; it is informational, so failures are logged and reported as 0
func (s *OfflineOrderService) totalPaid(ctx context.Context, orderID string) int {
	payments, err := s.paymentRepo.GetPaymentHistory(ctx, orderID)
	if err != nil {
		log.Warn().Err(err).Str("order_id", orderID).Msg("Failed to load payments for audit trail")
		return 0
	}
	return s.paymentCalculator.SumPaymentAmounts(payments)
}

// sameItems reports whether the requested items match the stored ones, in order
func sameItems(current []models.OrderItem, requested []models.OrderItemInput) bool {
	if len(current) != len(requested) {
		return false
	}
	for i, item := range current {
		in := requested[i]
		if item.ProductID != in.ProductID || item.ProductName != in.ProductName ||
			item.Quantity != in.Quantity || item.UnitPrice != in.UnitPrice {
			return false
		}
	}
	return true
}

// auditSnapshot captures an order for the audit trail, with customer data encrypted the same way
// as in guest_orders
func (s *OfflineOrderService) auditSnapshot(ctx context.Context, order *models.GuestOrder, items []models.OrderItem) map[string]interface{} {
	email := ""
	if order.CustomerEmail != nil {
		email = *order.CustomerEmail
	}

	snapshot := map[string]interface{}{
		"order_reference": order.OrderReference,
		"status":          order.Status,
		"delivery_type":   order.DeliveryType,
		"table_number":    order.TableNumber,
		"notes":           order.Notes,
		"subtotal_amount": order.SubtotalAmount,
		"delivery_fee":    order.DeliveryFee,
		"total_amount":    order.TotalAmount,
	}

	for field, value := range map[string]string{
		"customer_name":  order.CustomerName,
		"customer_phone": order.CustomerPhone,
		"customer_email": email,
	} {
		encrypted, err := s.offlineOrderRepo.EncryptPII(ctx, field, value)
		if err != nil {
			log.Warn().Err(err).Str("order_id", order.ID).Str("field", field).Msg("Failed to encrypt field for audit trail, omitting it")
			continue
		}
		snapshot[field] = encrypted
	}

	auditItems := make([]map[string]interface{}, len(items))
	for i, item := range items {
		auditItems[i] = map[string]interface{}{
			"product_id":   item.ProductID,
			"product_name": item.ProductName,
			"quantity":     item.Quantity,
			"unit_price":   item.UnitPrice,
			"total_price":  item.TotalPrice,
		}
	}
	snapshot["items"] = auditItems

	return snapshot
}

func (s *OfflineOrderService) createCorrectionOutboxEvent(ctx context.Context, tx *sql.Tx, eventType string, correction CompletedOrderCorrection, details map[string]interface{}) error {
	details["order_id"] = correction.OrderID
	details["tenant_id"] = correction.TenantID
	details["actor_user_id"] = correction.Actor.UserID
	details["reason"] = correction.Reason
	details["occurred_at"] = time.Now().Format(time.RFC3339)

	payload, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal %s event: %w", eventType, err)
	}

	if err := s.eventPublisher.CreateEvent(ctx, tx, &models.CreateEventOutboxRequest{
		EventType:    eventType,
		EventKey:     correction.OrderID,
		EventPayload: payload,
		Topic:        "order-events",
	}); err != nil {
		return fmt.Errorf("failed to record %s event: %w", eventType, err)
	}
	return nil
}

// publishCorrectionAudit sends the correction to the audit trail. It runs after the commit with its
// own timeout so a cancelled request does not drop the record; the outbox event written in the same
// transaction as the change remains as a durable record if publishing fails.
func (s *OfflineOrderService) publishCorrectionAudit(correction CompletedOrderCorrection, action string, before, after, metadata map[string]interface{}) {
	if s.auditPublisher == nil {
		log.Warn().Str("order_id", correction.OrderID).Msg("Audit publisher not configured, completed order correction not sent to audit trail")
		return
	}

	metadata["reason"] = correction.Reason
	metadata["actor_role"] = correction.Actor.Role
	metadata["accounting_impact_acknowledged"] = correction.AcknowledgeAccountingImpact
	metadata["order_type"] = models.OrderTypeOffline

	actorID := correction.Actor.UserID
	event := &utils.AuditEvent{
		TenantID:     correction.TenantID,
		ActorType:    "user",
		ActorID:      &actorID,
		Action:       action,
		ResourceType: "order",
		ResourceID:   correction.OrderID,
		UserAgent:    optionalString(correction.Actor.UserAgent),
		RequestID:    optionalString(correction.Actor.RequestID),
		BeforeValue:  before,
		AfterValue:   after,
		Metadata:     metadata,
	}

	auditCtx, cancel := context.WithTimeout(context.Background(), auditTimeout)
	defer cancel()

	if err := s.auditPublisher.Publish(auditCtx, event); err != nil {
		log.Error().Err(err).
			Str("order_id", correction.OrderID).
			Str("action", action).
			Msg("Failed to publish completed order correction to audit trail")
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
