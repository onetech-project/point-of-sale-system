package services

import (
	"bytes"
	"context"
	"crypto/sha512"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"
	"github.com/rs/zerolog/log"

	"github.com/point-of-sale-system/order-service/src/config"
	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/repository"
)

// PaymentService handles payment operations with Midtrans integration
type PaymentService struct {
	db               *sql.DB
	snapClient       *snap.Client
	coreAPIClient    *coreapi.Client
	serverKey        string
	paymentRepo      *repository.PaymentRepository
	orderRepo        *repository.OrderRepository
	inventoryService *InventoryService
	orderService     *OrderService
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	db *sql.DB,
	paymentRepo *repository.PaymentRepository,
	orderRepo *repository.OrderRepository,
	inventoryService *InventoryService,
	orderService *OrderService,
) *PaymentService {
	return &PaymentService{
		db:               db,
		snapClient:       config.GetSnapClient(),
		coreAPIClient:    config.GetCoreAPIClient(),
		serverKey:        config.GetMidtransServerKey(),
		paymentRepo:      paymentRepo,
		orderRepo:        orderRepo,
		inventoryService: inventoryService,
		orderService:     orderService,
	}
}

// QRISChargeResponse represents Midtrans QRIS charge API response
type QRISChargeResponse struct {
	StatusCode        string   `json:"status_code"`
	StatusMessage     string   `json:"status_message"`
	TransactionID     string   `json:"transaction_id"`
	OrderID           string   `json:"order_id"`
	MerchantID        string   `json:"merchant_id"`
	GrossAmount       string   `json:"gross_amount"`
	Currency          string   `json:"currency"`
	PaymentType       string   `json:"payment_type"`
	TransactionTime   string   `json:"transaction_time"`
	TransactionStatus string   `json:"transaction_status"`
	FraudStatus       string   `json:"fraud_status"`
	Actions           []Action `json:"actions"`
	Acquirer          string   `json:"acquirer"`
	QRString          string   `json:"qr_string"`
	ExpiryTime        string   `json:"expiry_time"`
}

// Action represents available actions for the transaction
type Action struct {
	Name   string `json:"name"`
	Method string `json:"method"`
	URL    string `json:"url"`
}

// CreateQRISCharge creates a QRIS payment charge using Midtrans Core API
// Implements integration with /v2/charge endpoint for QRIS generation
func (s *PaymentService) CreateQRISCharge(ctx context.Context, order *models.GuestOrder) (*QRISChargeResponse, error) {
	// Fetch tenant-specific Midtrans configuration
	midtransConfig, err := config.GetMidtransConfigForTenant(ctx, order.TenantID)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", order.TenantID).Msg("Failed to fetch tenant Midtrans config")
		return nil, fmt.Errorf("failed to get Midtrans configuration: %w", err)
	}

	if !midtransConfig.IsConfigured {
		log.Error().Str("tenant_id", order.TenantID).Msg("Midtrans not configured for tenant")
		return nil, fmt.Errorf("Midtrans is not configured for this tenant")
	}

	// Build charge request payload
	chargeReq := map[string]interface{}{
		"payment_type": "qris",
		"transaction_details": map[string]interface{}{
			"order_id":     order.OrderReference,
			"gross_amount": order.TotalAmount,
		},
	}

	// Marshal request to JSON
	requestBody, err := json.Marshal(chargeReq)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal QRIS charge request")
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Determine API base URL based on environment
	apiBaseURL := "https://api.sandbox.midtrans.com"
	if midtransConfig.Environment == "production" {
		apiBaseURL = "https://api.midtrans.com"
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiBaseURL+"/v2/charge", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create HTTP request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers with tenant-specific server key
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(midtransConfig.ServerKey+":"))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Add webhook URL to header
	webhookURL := config.GetWebhookURL()
	req.Header.Set("X-Override-Notification", webhookURL)

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute QRIS charge request")
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read response body")
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("response_body", string(body)).
			Msg("QRIS charge request failed")
		return nil, fmt.Errorf("charge request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var chargeResp QRISChargeResponse
	if err := json.Unmarshal(body, &chargeResp); err != nil {
		log.Error().
			Err(err).
			Str("response_body", string(body)).
			Msg("Failed to unmarshal QRIS charge response")
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	log.Info().
		Str("tenant_id", order.TenantID).
		Str("order_id", order.ID).
		Str("order_reference", order.OrderReference).
		Str("transaction_id", chargeResp.TransactionID).
		Str("qr_code_url", chargeResp.Actions[0].URL).
		Str("expiry_time", chargeResp.ExpiryTime).
		Msg("QRIS charge created successfully with tenant-specific credentials")

	return &chargeResp, nil
}

// SaveQRISPaymentInfo saves QRIS payment information to database
func (s *PaymentService) SaveQRISPaymentInfo(ctx context.Context, orderID string, amount int, chargeResp *QRISChargeResponse) error {
	// Parse expiry time - Midtrans returns time in Asia/Jakarta timezone (WIB)
	// Since our database column is TIMESTAMP WITHOUT TIME ZONE, we need to convert to UTC
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Error().Err(err).Msg("Failed to load Asia/Jakarta location, using UTC fallback")
		loc = time.UTC
	}
	expiryTimeWIB, err := time.ParseInLocation("2006-01-02 15:04:05", chargeResp.ExpiryTime, loc)
	if err != nil {
		log.Error().Err(err).Str("expiry_time", chargeResp.ExpiryTime).Msg("Failed to parse expiry time")
		// Continue with nil expiry time rather than failing
	}

	// Convert to UTC for storage (since DB column is WITHOUT TIME ZONE, it stores as-is)
	expiryTime := expiryTimeWIB.UTC()

	// Get QR code URL from actions array (first action)
	var qrCodeURL string
	if len(chargeResp.Actions) > 0 {
		qrCodeURL = chargeResp.Actions[0].URL
	}

	// Create payment transaction record
	transactionID := chargeResp.TransactionID
	paymentType := chargeResp.PaymentType
	transactionStatus := chargeResp.TransactionStatus
	fraudStatus := chargeResp.FraudStatus

	// Marshal charge response to JSON for notification_payload
	chargeJSON, err := json.Marshal(chargeResp)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal charge response")
		chargeJSON = json.RawMessage(`{}`) // Use empty JSON object as fallback
	}

	// Generate idempotency key for initial pending status
	idempotencyKey := transactionID + ":" + strings.ToLower(transactionStatus)

	payment := &models.PaymentTransaction{
		OrderID:               orderID,
		MidtransTransactionID: &transactionID,
		MidtransOrderID:       chargeResp.OrderID,
		Amount:                amount,
		PaymentType:           &paymentType,
		TransactionStatus:     &transactionStatus,
		FraudStatus:           &fraudStatus,
		NotificationPayload:   chargeJSON,
		QRCodeURL:             &qrCodeURL,
		QRString:              &chargeResp.QRString,
		ExpiryTime:            &expiryTime,
		SignatureVerified:     false, // Will be verified on webhook
		IdempotencyKey:        &idempotencyKey,
	}

	err = s.paymentRepo.CreatePaymentTransaction(ctx, payment)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Str("transaction_id", chargeResp.TransactionID).
			Msg("Failed to save QRIS payment info")
		return fmt.Errorf("failed to save payment info: %w", err)
	}

	log.Info().
		Str("order_id", orderID).
		Str("transaction_id", chargeResp.TransactionID).
		Str("qr_code_url", qrCodeURL).
		Msg("QRIS payment info saved successfully")

	return nil
}

// CreateSnapTransaction creates a Midtrans Snap transaction for QRIS payment
// Implements T057-T058: Snap transaction creation with QRIS method
func (s *PaymentService) CreateSnapTransaction(ctx context.Context, order *models.GuestOrder) (*snap.Response, error) {
	// Build Snap request
	snapReq := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  order.OrderReference, // Use order reference as Midtrans order_id
			GrossAmt: int64(order.TotalAmount),
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: order.CustomerName,
			Phone: order.CustomerPhone,
		},
		EnabledPayments: []snap.SnapPaymentType{
			snap.PaymentTypeGopay, // QRIS is provided through GoPay
		},
		CreditCard: &snap.CreditCardDetails{
			Secure: true,
		},
	} // Add items to Snap request
	items := []midtrans.ItemDetails{}
	// Note: Items will be populated from order_items in checkout handler
	snapReq.Items = &items

	// Create Snap transaction
	snapResp, err := s.snapClient.CreateTransaction(snapReq)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", order.ID).
			Str("order_reference", order.OrderReference).
			Msg("Failed to create Snap transaction")
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	log.Info().
		Str("order_id", order.ID).
		Str("order_reference", order.OrderReference).
		Str("token", snapResp.Token).
		Str("redirect_url", snapResp.RedirectURL).
		Msg("Snap transaction created successfully")

	return snapResp, nil
}

// VerifySignature verifies Midtrans webhook signature using tenant-specific server key
// Implements T059: SHA512 signature verification
func (s *PaymentService) VerifySignature(ctx context.Context, tenantID, orderID, statusCode, grossAmount, signatureKey string) bool {
	// Fetch tenant-specific Midtrans server key
	serverKey, err := config.GetMidtransServerKeyForTenant(ctx, tenantID)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID).
			Str("order_id", orderID).
			Msg("Failed to fetch tenant Midtrans server key for signature verification")
		return false
	}

	// Build signature string: order_id + status_code + gross_amount + server_key
	signatureString := orderID + statusCode + grossAmount + serverKey

	// Calculate SHA512 hash
	hash := sha512.New()
	hash.Write([]byte(signatureString))
	calculatedSignature := hex.EncodeToString(hash.Sum(nil))

	// Compare signatures
	isValid := calculatedSignature == signatureKey

	if !isValid {
		log.Warn().
			Str("tenant_id", tenantID).
			Str("order_id", orderID).
			Str("expected_signature", calculatedSignature).
			Str("received_signature", signatureKey).
			Msg("Signature verification failed")
	}

	return isValid
}

// MidtransNotification represents the webhook notification from Midtrans
type MidtransNotification struct {
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	TransactionID     string `json:"transaction_id"`
	StatusMessage     string `json:"status_message"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	PaymentType       string `json:"payment_type"`
	OrderID           string `json:"order_id"`
	MerchantID        string `json:"merchant_id"`
	GrossAmount       string `json:"gross_amount"`
	FraudStatus       string `json:"fraud_status"`
	Currency          string `json:"currency"`
}

// ProcessNotification processes Midtrans webhook notification
// Implements T060: Notification processing with idempotency, signature validation, status mapping
func (s *PaymentService) ProcessNotification(ctx context.Context, notification *MidtransNotification) error {
	// Step 1: Check idempotency - have we processed this exact notification before?
	idempotencyKey := notification.TransactionID + ":" + strings.ToLower(notification.TransactionStatus)
	existing, err := s.paymentRepo.GetPaymentByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		log.Error().
			Err(err).
			Str("idempotency_key", idempotencyKey).
			Msg("Failed to check idempotency")
		return fmt.Errorf("failed to check idempotency: %w", err)
	}

	if existing != nil {
		log.Info().
			Str("idempotency_key", idempotencyKey).
			Str("order_reference", notification.OrderID).
			Msg("Notification already processed (idempotent)")
		return nil // Already processed, return success
	}

	// Step 2: Get order by order reference (need tenant ID for signature verification)
	order, err := s.orderRepo.GetOrderByReference(ctx, notification.OrderID)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_reference", notification.OrderID).
			Msg("Failed to get order")
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		log.Error().
			Str("order_reference", notification.OrderID).
			Msg("Order not found")
		return fmt.Errorf("order not found")
	}

	// Step 3: Verify signature using tenant-specific server key
	isValid := s.VerifySignature(
		ctx,
		order.TenantID,
		notification.OrderID,
		notification.StatusCode,
		notification.GrossAmount,
		notification.SignatureKey,
	)

	if !isValid {
		log.Error().
			Str("tenant_id", order.TenantID).
			Str("order_reference", notification.OrderID).
			Str("transaction_id", notification.TransactionID).
			Msg("Invalid signature - rejecting notification")
		return fmt.Errorf("invalid signature")
	}

	// Step 4: Map Midtrans transaction status to order status and process
	log.Info().
		Str("order_reference", notification.OrderID).
		Str("order_id", order.ID).
		Str("transaction_id", notification.TransactionID).
		Str("transaction_status", notification.TransactionStatus).
		Str("fraud_status", notification.FraudStatus).
		Msg("Processing payment notification")

	// Store notification payload as JSON
	notificationJSON, _ := json.Marshal(notification)

	// Create or update payment transaction record
	err = s.createOrUpdatePaymentTransaction(ctx, order.ID, notification, notificationJSON, idempotencyKey)
	if err != nil {
		return fmt.Errorf("failed to save payment transaction: %w", err)
	}

	// Process based on transaction status
	switch strings.ToLower(notification.TransactionStatus) {
	case "settlement", "capture":
		// Payment successful - update order to PAID and convert inventory reservations
		return s.handlePaymentSuccess(ctx, order.ID, order.TenantID, notification)

	case "pending":
		// Payment still pending - keep reservation active
		log.Info().
			Str("order_id", order.ID).
			Str("order_reference", notification.OrderID).
			Msg("Payment pending - reservation remains active")
		return nil

	case "cancel", "deny", "expire":
		// Payment failed or expired - release inventory reservations
		return s.handlePaymentFailure(ctx, order.ID, order.TenantID, notification)

	default:
		log.Warn().
			Str("order_id", order.ID).
			Str("transaction_status", notification.TransactionStatus).
			Msg("Unknown transaction status - no action taken")
		return nil
	}
}

// handlePaymentSuccess handles successful payment
// Implements T061: Order status update for settlement
// Implements T062: Inventory reservation conversion
func (s *PaymentService) handlePaymentSuccess(ctx context.Context, orderID, tenantID string, notification *MidtransNotification) error {
	// Step 1: Update order status to PAID using OrderService
	// This will handle the transaction, timestamp updates, AND publish order.paid event to Kafka
	err := s.orderService.UpdateOrderStatus(ctx, orderID, models.OrderStatusPaid)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Msg("Failed to update order status to PAID")
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Step 2: Convert inventory reservations to permanent allocations
	// This decrements product quantity and marks reservations as 'converted'
	err = s.inventoryService.ConvertReservationsToPermanent(ctx, orderID)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Str("tenant_id", tenantID).
			Msg("Failed to convert inventory reservations - order is PAID but inventory not updated")
		// Note: Order is already PAID, so we log error but don't fail the webhook
		// This should trigger an alert for manual intervention
		return nil
	}

	log.Info().
		Str("order_id", orderID).
		Str("order_reference", notification.OrderID).
		Str("transaction_id", notification.TransactionID).
		Msg("Payment successful - order PAID and inventory converted")

	return nil
}

// handlePaymentFailure handles payment failure, cancellation, or expiration
// Implements T061: Order status update for failed payments with reservation release
func (s *PaymentService) handlePaymentFailure(ctx context.Context, orderID, tenantID string, notification *MidtransNotification) error {
	// Step 1: Release inventory reservations
	// This marks reservations as 'expired' and increments Redis cache
	err := s.inventoryService.ReleaseReservations(ctx, orderID)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Str("tenant_id", tenantID).
			Msg("Failed to release inventory reservations")
		// Continue with order status update even if release fails
	}

	// Step 2: Update order status to CANCELLED using OrderService
	// This will handle the transaction and timestamp updates
	err = s.orderService.UpdateOrderStatus(ctx, orderID, models.OrderStatusCancelled)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Msg("Failed to update order status to CANCELLED")
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Step 3: Add system note explaining the cancellation reason
	var noteMessage string
	switch strings.ToLower(notification.TransactionStatus) {
	case "expire":
		noteMessage = "Order automatically cancelled due to payment expiration. Payment was not completed within the allocated time."
	case "cancel":
		noteMessage = "Order cancelled due to payment cancellation by customer or payment gateway."
	case "deny":
		noteMessage = "Order cancelled due to payment denial by payment gateway."
	default:
		noteMessage = fmt.Sprintf("Order cancelled due to payment failure (status: %s).", notification.TransactionStatus)
	}

	err = s.orderService.AddOrderNote(ctx, orderID, noteMessage, "System")
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Msg("Failed to add system note for payment failure")
		// Don't fail the webhook if note creation fails
	}

	log.Info().
		Str("order_id", orderID).
		Str("order_reference", notification.OrderID).
		Str("transaction_status", notification.TransactionStatus).
		Msg("Payment failed/cancelled/expired - order CANCELLED and inventory released")

	return nil
}

// createOrUpdatePaymentTransaction creates or updates payment transaction record
func (s *PaymentService) createOrUpdatePaymentTransaction(
	ctx context.Context,
	orderID string,
	notification *MidtransNotification,
	notificationJSON []byte,
	idempotencyKey string,
) error {
	// Check if payment transaction exists for this order
	existing, err := s.paymentRepo.GetPaymentByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to check existing payment: %w", err)
	}

	transactionID := notification.TransactionID
	paymentType := notification.PaymentType
	transactionStatus := notification.TransactionStatus
	fraudStatus := notification.FraudStatus
	signatureKey := notification.SignatureKey

	if existing == nil {
		// Create new payment transaction record
		payment := &models.PaymentTransaction{
			OrderID:               orderID,
			MidtransTransactionID: &transactionID,
			MidtransOrderID:       notification.OrderID,
			Amount:                0, // Will be set from order
			PaymentType:           &paymentType,
			TransactionStatus:     &transactionStatus,
			FraudStatus:           &fraudStatus,
			SignatureKey:          &signatureKey,
			SignatureVerified:     true,
			NotificationPayload:   notificationJSON,
			IdempotencyKey:        &idempotencyKey,
		}

		err = s.paymentRepo.CreatePaymentTransaction(ctx, payment)
		if err != nil {
			log.Error().
				Err(err).
				Str("order_id", orderID).
				Msg("Failed to create payment transaction")
			return fmt.Errorf("failed to create payment transaction: %w", err)
		}
	} else {
		// Update existing payment transaction with new status and idempotency key
		now := time.Now()
		var settledAt *time.Time
		if strings.ToLower(notification.TransactionStatus) == "settlement" || strings.ToLower(notification.TransactionStatus) == "capture" {
			settledAt = &now
		}

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()

		// Update payment status and idempotency key
		updateQuery := `
			UPDATE payment_transactions
			SET transaction_status = $2,
			    settled_at = $3,
			    notification_payload = $4,
			    notification_received_at = NOW(),
			    idempotency_key = $5,
			    signature_key = $6,
			    signature_verified = true
			WHERE midtrans_transaction_id = $1
		`
		_, err = tx.ExecContext(ctx, updateQuery, transactionID, transactionStatus, settledAt, notificationJSON, idempotencyKey, signatureKey)
		if err != nil {
			log.Error().
				Err(err).
				Str("transaction_id", transactionID).
				Msg("Failed to update payment transaction")
			return fmt.Errorf("failed to update payment transaction: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit payment update: %w", err)
		}
	}

	return nil
}
