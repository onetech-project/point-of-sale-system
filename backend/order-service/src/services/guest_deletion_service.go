package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/repository"
	"github.com/point-of-sale-system/order-service/src/utils"
)

// GuestDeletionService handles guest customer data anonymization for UU PDP compliance
// Implements right to deletion while preserving order records for business continuity (UU PDP Article 5)
type GuestDeletionService struct {
	orderRepo      *repository.OrderRepository
	addressRepo    *repository.AddressRepository
	db             *sql.DB
	encryptor      utils.Encryptor
	auditPublisher *utils.AuditPublisher
}

// NewGuestDeletionService creates a new guest deletion service
func NewGuestDeletionService(db *sql.DB, encryptor utils.Encryptor, auditPublisher *utils.AuditPublisher) *GuestDeletionService {
	orderRepo := repository.NewOrderRepository(db, encryptor)
	addressRepo := repository.NewAddressRepository(db, encryptor)
	return &GuestDeletionService{
		orderRepo:      orderRepo,
		addressRepo:    addressRepo,
		db:             db,
		encryptor:      encryptor,
		auditPublisher: auditPublisher,
	}
}

// AnonymizeGuestData anonymizes all personal data for a guest order (T140-T143)
// Sets is_anonymized=TRUE, replaces PII with generic values, anonymizes delivery address
// Preserves order record for merchant's business continuity
func (s *GuestDeletionService) AnonymizeGuestData(ctx context.Context, orderReference string) error {
	// Get order
	order, err := s.orderRepo.GetOrderByReference(ctx, orderReference)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("order not found")
		}
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Check if already anonymized
	if order.IsAnonymized {
		return fmt.Errorf("order data already anonymized")
	}

	// Start transaction for atomic anonymization
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Encrypt generic "Deleted User" value
	deletedUserEncrypted, err := s.encryptor.Encrypt(ctx, "Deleted User")
	if err != nil {
		return fmt.Errorf("failed to encrypt deleted user placeholder: %w", err)
	}

	now := time.Now()

	// T141: Anonymize order PII - replace with generic values
	anonymizeOrderQuery := `
		UPDATE guest_orders
		SET customer_name_encrypted = $1,
		    customer_phone_encrypted = NULL,
		    customer_email_encrypted = NULL,
		    ip_address_encrypted = NULL,
		    is_anonymized = TRUE,
		    anonymized_at = $2,
		    updated_at = $2
		WHERE order_reference = $3
	`

	_, err = tx.ExecContext(ctx, anonymizeOrderQuery, deletedUserEncrypted, now, orderReference)
	if err != nil {
		return fmt.Errorf("failed to anonymize order: %w", err)
	}

	// T142: Anonymize delivery address if exists
	if order.DeliveryType == models.DeliveryTypeDelivery {
		deletedAddressEncrypted, err := s.encryptor.Encrypt(ctx, "Address Deleted")
		if err != nil {
			return fmt.Errorf("failed to encrypt deleted address placeholder: %w", err)
		}

		anonymizeAddressQuery := `
			UPDATE delivery_addresses
			SET full_address_encrypted = $1,
			    latitude = NULL,
			    longitude = NULL,
			    updated_at = $2
			WHERE order_id = $3
		`

		_, err = tx.ExecContext(ctx, anonymizeAddressQuery, deletedAddressEncrypted, now, order.ID)
		if err != nil {
			// Non-critical if address doesn't exist
			if err != sql.ErrNoRows {
				return fmt.Errorf("failed to anonymize delivery address: %w", err)
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit anonymization transaction: %w", err)
	}

	// T143: Publish GuestDataAnonymizedEvent to audit topic
	auditEvent := &utils.AuditEvent{
		EventID:      uuid.New().String(),
		TenantID:     order.TenantID,
		Action:       "GUEST_DATA_ANONYMIZED",
		ActorType:    "guest",
		ActorID:      nil, // Guest user, no actor ID
		ActorEmail:   nil,
		ResourceType: "guest_order",
		ResourceID:   order.ID,
		Timestamp:    now,
		Metadata: map[string]interface{}{
			"order_reference": orderReference,
			"anonymized_at":   now.Format(time.RFC3339),
			"compliance":      "UU_PDP_Article_5",
			"fields_anonymized": []string{
				"customer_name",
				"customer_phone",
				"customer_email",
				"ip_address",
				"delivery_address",
			},
		},
	}

	if err := s.auditPublisher.Publish(ctx, auditEvent); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to publish guest data anonymized event: %v\n", err)
	}

	return nil
}

// CanAnonymizeOrder checks if an order can be anonymized (not already anonymized)
func (s *GuestDeletionService) CanAnonymizeOrder(ctx context.Context, orderReference string) (bool, error) {
	order, err := s.orderRepo.GetOrderByReference(ctx, orderReference)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("order not found")
		}
		return false, fmt.Errorf("failed to get order: %w", err)
	}

	return !order.IsAnonymized, nil
}
