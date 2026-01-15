package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/repository"
	"github.com/point-of-sale-system/order-service/src/utils"
)

// GuestDataService handles guest customer data access for UU PDP compliance
// Implements right to access personal data (UU PDP Article 4)
type GuestDataService struct {
	orderRepo   *repository.OrderRepository
	addressRepo *repository.AddressRepository
	db          *sql.DB
	encryptor   utils.Encryptor
}

// NewGuestDataService creates a new guest data service
func NewGuestDataService(db *sql.DB, encryptor utils.Encryptor) *GuestDataService {
	orderRepo := repository.NewOrderRepository(db, encryptor)
	addressRepo := repository.NewAddressRepository(db, encryptor)
	return &GuestDataService{
		orderRepo:   orderRepo,
		addressRepo: addressRepo,
		db:          db,
		encryptor:   encryptor,
	}
}

// GuestDataResponse represents all personal data associated with a guest order
type GuestDataResponse struct {
	OrderReference  string                  `json:"order_reference"`
	CustomerInfo    *CustomerInfo           `json:"customer_info"`
	OrderDetails    *OrderDetails           `json:"order_details"`
	DeliveryAddress *models.DeliveryAddress `json:"delivery_address,omitempty"`
	IsAnonymized    bool                    `json:"is_anonymized"`
	AnonymizedAt    *string                 `json:"anonymized_at,omitempty"`
}

// CustomerInfo represents decrypted customer personal information
type CustomerInfo struct {
	Name  string  `json:"name"`
	Phone string  `json:"phone"`
	Email *string `json:"email,omitempty"`
}

// OrderDetails represents order transaction details
type OrderDetails struct {
	OrderReference string              `json:"order_reference"`
	Status         string              `json:"status"`
	TotalAmount    int                 `json:"total_amount"`
	DeliveryFee    int                 `json:"delivery_fee"`
	SubtotalAmount int                 `json:"subtotal_amount"`
	DeliveryType   string              `json:"delivery_type"`
	TableNumber    *string             `json:"table_number,omitempty"`
	Notes          *string             `json:"notes,omitempty"`
	CreatedAt      string              `json:"created_at"`
	PaidAt         *string             `json:"paid_at,omitempty"`
	Items          []models.OrderItem  `json:"items"`
}

// GetGuestOrderData retrieves and decrypts all personal data for a guest order (T139)
// Returns decrypted customer PII, order details, and delivery address
func (s *GuestDataService) GetGuestOrderData(ctx context.Context, orderReference string) (*GuestDataResponse, error) {
	// Get order with all related entities
	order, err := s.orderRepo.GetOrderByReference(ctx, orderReference)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Check if order data has been anonymized
	if order.IsAnonymized {
		return &GuestDataResponse{
			OrderReference: orderReference,
			IsAnonymized:   true,
			AnonymizedAt:   formatTime(order.AnonymizedAt),
			CustomerInfo: &CustomerInfo{
				Name:  "Deleted User",
				Phone: "***",
				Email: nil,
			},
			OrderDetails: &OrderDetails{
				OrderReference: order.OrderReference,
				Status:         string(order.Status),
				TotalAmount:    order.TotalAmount,
				DeliveryFee:    order.DeliveryFee,
				SubtotalAmount: order.SubtotalAmount,
				DeliveryType:   string(order.DeliveryType),
				TableNumber:    order.TableNumber,
				Notes:          order.Notes,
				CreatedAt:      order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				PaidAt:         formatTime(order.PaidAt),
			},
		}, nil
	}

	// Decrypt customer PII
	customerName, err := s.encryptor.Decrypt(ctx, order.CustomerName)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt customer name: %w", err)
	}

	customerPhone, err := s.encryptor.Decrypt(ctx, order.CustomerPhone)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt customer phone: %w", err)
	}

	var customerEmail *string
	if order.CustomerEmail != nil && *order.CustomerEmail != "" {
		decryptedEmail, err := s.encryptor.Decrypt(ctx, *order.CustomerEmail)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt customer email: %w", err)
		}
		customerEmail = &decryptedEmail
	}

	// Get order items
	items, err := s.orderRepo.GetOrderItemsByOrderID(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	// Get delivery address if exists
	var deliveryAddress *models.DeliveryAddress
	if order.DeliveryType == models.DeliveryTypeDelivery {
		deliveryAddress, err = s.addressRepo.GetByOrderID(ctx, order.ID)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to get delivery address: %w", err)
		}

		// Decrypt delivery address if exists
		if deliveryAddress != nil && !order.IsAnonymized {
			decryptedAddress, err := s.encryptor.Decrypt(ctx, deliveryAddress.FullAddress)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt delivery address: %w", err)
			}
			deliveryAddress.FullAddress = decryptedAddress
		}
	}

	response := &GuestDataResponse{
		OrderReference: orderReference,
		CustomerInfo: &CustomerInfo{
			Name:  customerName,
			Phone: customerPhone,
			Email: customerEmail,
		},
		OrderDetails: &OrderDetails{
			OrderReference: order.OrderReference,
			Status:         string(order.Status),
			TotalAmount:    order.TotalAmount,
			DeliveryFee:    order.DeliveryFee,
			SubtotalAmount: order.SubtotalAmount,
			DeliveryType:   string(order.DeliveryType),
			TableNumber:    order.TableNumber,
			Notes:          order.Notes,
			CreatedAt:      order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			PaidAt:         formatTime(order.PaidAt),
			Items:          items,
		},
		DeliveryAddress: deliveryAddress,
		IsAnonymized:    order.IsAnonymized,
	}

	return response, nil
}

// VerifyGuestAccess verifies that the provided email or phone matches the order's customer data
// Used for verification-based access control (T146)
func (s *GuestDataService) VerifyGuestAccess(ctx context.Context, orderReference string, email *string, phone *string) (bool, error) {
	if email == nil && phone == nil {
		return false, fmt.Errorf("either email or phone must be provided for verification")
	}

	// Get order
	order, err := s.orderRepo.GetOrderByReference(ctx, orderReference)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("order not found")
		}
		return false, fmt.Errorf("failed to get order: %w", err)
	}

	// If anonymized, deny access
	if order.IsAnonymized {
		return false, nil
	}

	// Verify email if provided
	if email != nil {
		if order.CustomerEmail == nil || *order.CustomerEmail == "" {
			return false, nil
		}

		decryptedEmail, err := s.encryptor.Decrypt(ctx, *order.CustomerEmail)
		if err != nil {
			return false, fmt.Errorf("failed to decrypt customer email: %w", err)
		}

		if decryptedEmail != *email {
			return false, nil
		}
	}

	// Verify phone if provided
	if phone != nil {
		decryptedPhone, err := s.encryptor.Decrypt(ctx, order.CustomerPhone)
		if err != nil {
			return false, fmt.Errorf("failed to decrypt customer phone: %w", err)
		}

		if decryptedPhone != *phone {
			return false, nil
		}
	}

	return true, nil
}

// Helper function to format time pointers
func formatTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format("2006-01-02T15:04:05Z07:00")
	return &formatted
}
