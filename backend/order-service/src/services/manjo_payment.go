package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/point-of-sale-system/order-service/src/config"
	"github.com/point-of-sale-system/order-service/src/manjo"
	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/repository"
)

const (
	// manjoQRValidity matches the 15 minute inventory reservation.
	manjoQRValidity = 15 * time.Minute
	// manjoPollInterval limits how often a pending payment is queried at Manjo
	// when the guest order page polls.
	manjoPollInterval = 20 * time.Second
	// manjoExpiryGrace lets a payment made at the last second settle before an
	// expired QR is treated as unpaid.
	manjoExpiryGrace = 2 * time.Minute
)

// manjoGateway is the slice of the Manjo integration PaymentService uses,
// replaceable in tests.
type manjoGateway interface {
	Credentials(ctx context.Context, tenantID string) (manjo.Credentials, *manjo.Client, error)
}

type defaultManjoGateway struct{}

func (defaultManjoGateway) Credentials(ctx context.Context, tenantID string) (manjo.Credentials, *manjo.Client, error) {
	cfg, err := config.GetManjoConfigForTenant(ctx, tenantID)
	if err != nil {
		return manjo.Credentials{}, nil, err
	}
	if !cfg.IsConfigured {
		return manjo.Credentials{}, nil, fmt.Errorf("Manjo is not configured for tenant %s", tenantID)
	}
	client, err := config.GetManjoClient(cfg.Environment)
	if err != nil {
		return manjo.Credentials{}, nil, err
	}
	return cfg.Credentials(), client, nil
}

// createManjoQRISCharge generates a dynamic QRIS code at Manjo.
func (s *PaymentService) createManjoQRISCharge(ctx context.Context, order *models.GuestOrder) (*QRISCharge, error) {
	creds, client, err := s.manjo.Credentials(ctx, order.TenantID)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", order.TenantID).Msg("Failed to get Manjo credentials")
		return nil, fmt.Errorf("failed to get Manjo configuration: %w", err)
	}

	resp, err := client.Generate(ctx, creds, manjo.GenerateRequest{
		PartnerReferenceNo: order.OrderReference,
		Amount:             manjo.Amount{Value: manjo.FormatAmount(order.TotalAmount), Currency: "IDR"},
		ValidityPeriod:     fmt.Sprintf("%d", int(manjoQRValidity.Seconds())),
		AdditionalInfo: manjo.GenerateAdditionalInfo{
			ProdDesc: truncate("Order "+order.OrderReference, 100),
		},
	})
	if err != nil {
		log.Error().Err(err).
			Str("tenant_id", order.TenantID).
			Str("order_reference", order.OrderReference).
			Msg("Failed to generate Manjo QRIS")
		return nil, fmt.Errorf("failed to generate Manjo QRIS: %w", err)
	}

	// expiry_time stores WIB wall-clock time (see GetPublicOrder), so keep the
	// WIB clock reading and drop the zone.
	expiresAt, ok := resp.ExpiresAt(client.Location())
	if !ok {
		expiresAt = time.Now().In(client.Location()).Add(manjoQRValidity)
	}
	expiryWallClock := time.Date(expiresAt.Year(), expiresAt.Month(), expiresAt.Day(),
		expiresAt.Hour(), expiresAt.Minute(), expiresAt.Second(), 0, time.UTC)

	log.Info().
		Str("tenant_id", order.TenantID).
		Str("order_id", order.ID).
		Str("order_reference", order.OrderReference).
		Str("reference_no", resp.ReferenceNo).
		Msg("Manjo QRIS generated")

	return &QRISCharge{
		Gateway:           config.PaymentGatewayManjo,
		TransactionID:     resp.ReferenceNo,
		MerchantReference: order.OrderReference,
		PaymentType:       "qris",
		TransactionStatus: manjo.StatusPending,
		QRString:          resp.QRContent,
		ExpiryTime:        expiryWallClock,
		ExternalID:        resp.ExternalID,
		RawResponse:       resp,
	}, nil
}

// ManjoNotificationResult tells the webhook handler how to answer Manjo.
type ManjoNotificationResult int

const (
	ManjoNotificationProcessed ManjoNotificationResult = iota
	ManjoNotificationInvalidBody
	ManjoNotificationInvalidSignature
)

// ProcessManjoNotification verifies a Manjo payment notification and applies
// the payment status. The notification's own status is not trusted (the docs
// disagree on its codes): the status is confirmed with the query API.
func (s *PaymentService) ProcessManjoNotification(ctx context.Context, header map[string][]string, requestPath string, body []byte) (ManjoNotificationResult, error) {
	notification, err := manjo.ParseNotification(body)
	if err != nil || notification.OriginalPartnerReferenceNo == "" {
		return ManjoNotificationInvalidBody, fmt.Errorf("invalid Manjo notification body")
	}

	order, err := s.orderRepo.GetOrderByReference(ctx, notification.OriginalPartnerReferenceNo)
	if err != nil {
		return ManjoNotificationProcessed, fmt.Errorf("failed to get order: %w", err)
	}
	if order == nil {
		return ManjoNotificationInvalidBody, fmt.Errorf("order %s not found", notification.OriginalPartnerReferenceNo)
	}

	payment, err := s.paymentRepo.GetPaymentByOrderID(ctx, order.ID)
	if err != nil {
		return ManjoNotificationProcessed, fmt.Errorf("failed to get payment: %w", err)
	}
	if payment == nil || payment.Gateway() != config.PaymentGatewayManjo {
		return ManjoNotificationInvalidBody, fmt.Errorf("order %s has no Manjo payment", order.OrderReference)
	}
	if notification.OriginalReferenceNo != "" && payment.GatewayReferenceNo != nil &&
		*payment.GatewayReferenceNo != notification.OriginalReferenceNo {
		return ManjoNotificationInvalidBody, fmt.Errorf("reference number does not match order %s", order.OrderReference)
	}

	creds, client, err := s.manjo.Credentials(ctx, order.TenantID)
	if err != nil {
		return ManjoNotificationProcessed, fmt.Errorf("failed to get Manjo configuration: %w", err)
	}
	if !manjo.VerifyNotificationSignature(header, requestPath, body, creds.ClientSecret) {
		log.Warn().
			Str("tenant_id", order.TenantID).
			Str("order_reference", order.OrderReference).
			Msg("Invalid Manjo notification signature")
		return ManjoNotificationInvalidSignature, errors.New("invalid signature")
	}

	if _, err := s.syncManjoPayment(ctx, order, payment, creds, client, body); err != nil {
		return ManjoNotificationProcessed, err
	}
	return ManjoNotificationProcessed, nil
}

// SyncPendingManjoPayment queries Manjo for a pending payment, so a guest sees
// their payment even when the notification is lost. Calls are throttled per
// payment. It returns true when the order status changed.
func (s *PaymentService) SyncPendingManjoPayment(ctx context.Context, order *models.GuestOrder, payment *models.PaymentTransaction) (bool, error) {
	if order == nil || payment == nil || payment.Gateway() != config.PaymentGatewayManjo ||
		order.Status != models.OrderStatusPending || !s.syncThrottle.allow(payment.ID) {
		return false, nil
	}

	creds, client, err := s.manjo.Credentials(ctx, order.TenantID)
	if err != nil {
		return false, err
	}
	return s.syncManjoPayment(ctx, order, payment, creds, client, nil)
}

// syncManjoPayment queries the payment at Manjo and applies its status.
// notificationBody is stored as the payload when the sync comes from a notification.
func (s *PaymentService) syncManjoPayment(
	ctx context.Context,
	order *models.GuestOrder,
	payment *models.PaymentTransaction,
	creds manjo.Credentials,
	client *manjo.Client,
	notificationBody []byte,
) (bool, error) {
	query := manjo.QueryRequest{
		OriginalReferenceNo:        derefString(payment.GatewayReferenceNo),
		OriginalPartnerReferenceNo: payment.MidtransOrderID,
		OriginalExternalID:         derefString(payment.GatewayExternalID),
	}
	resp, err := client.Query(ctx, creds, query)
	if err != nil {
		return false, fmt.Errorf("failed to query Manjo payment: %w", err)
	}

	status := manjo.StatusFromQuery(resp.LatestTransactionStatus)
	if status == manjo.StatusSettlement {
		if err := checkPaidAmount(resp.Amount, payment.Amount); err != nil {
			log.Error().Err(err).
				Str("order_reference", order.OrderReference).
				Msg("Manjo paid amount does not match the order - not marking it paid")
			return false, err
		}
	}
	if status == manjo.StatusPending && isExpired(payment.ExpiryTime) {
		status = manjo.StatusExpire
	}

	payload := notificationBody
	if payload == nil {
		payload, _ = json.Marshal(resp)
	}
	if err := s.recordGatewayStatus(ctx, payment, status, payload, notificationBody != nil); err != nil {
		return false, err
	}

	switch status {
	case manjo.StatusSettlement:
		if order.Status != models.OrderStatusPending {
			return false, nil
		}
		return true, s.handlePaymentSuccess(ctx, order.ID, order.TenantID, order.OrderReference, derefString(payment.MidtransTransactionID))
	case manjo.StatusCancel, manjo.StatusFailure, manjo.StatusExpire:
		if order.Status != models.OrderStatusPending {
			return false, nil
		}
		return true, s.handlePaymentFailure(ctx, order.ID, order.TenantID, order.OrderReference, status)
	default:
		return false, nil
	}
}

// recordGatewayStatus stores the latest gateway status of a payment. A status
// already recorded is left alone, so repeated notifications do not rewrite it.
func (s *PaymentService) recordGatewayStatus(ctx context.Context, payment *models.PaymentTransaction, status string, payload []byte, fromNotification bool) error {
	if payment.TransactionStatus != nil && *payment.TransactionStatus == status {
		return nil
	}

	var settledAt *time.Time
	if status == manjo.StatusSettlement {
		now := time.Now()
		settledAt = &now
	}
	idempotencyKey := derefString(payment.MidtransTransactionID) + ":" + status

	_, err := s.db.ExecContext(ctx, `
		UPDATE payment_transactions
		SET transaction_status = $2,
		    settled_at = COALESCE(settled_at, $3),
		    notification_payload = $4,
		    notification_received_at = CASE WHEN $6 THEN NOW() ELSE notification_received_at END,
		    idempotency_key = $5,
		    signature_verified = signature_verified OR $6
		WHERE id = $1`,
		payment.ID, status, settledAt, payload, idempotencyKey, fromNotification,
	)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}
	payment.TransactionStatus = &status
	return nil
}

func checkPaidAmount(amount *manjo.Amount, expected int) error {
	if amount == nil {
		return nil
	}
	paid, err := manjo.ParseAmount(amount.Value)
	if err != nil {
		return err
	}
	if paid != expected {
		return fmt.Errorf("paid amount %d does not match order total %d", paid, expected)
	}
	return nil
}

// isExpired reports whether an expiry_time (WIB wall-clock time, see
// GetPublicOrder) passed more than manjoExpiryGrace ago.
func isExpired(expiry *time.Time) bool {
	if expiry == nil {
		return false
	}
	wib := time.FixedZone("WIB", 7*60*60)
	expiresAt := time.Date(expiry.Year(), expiry.Month(), expiry.Day(),
		expiry.Hour(), expiry.Minute(), expiry.Second(), 0, wib)
	return time.Now().After(expiresAt.Add(manjoExpiryGrace))
}

// ---------------------------------------------------------------------------
// Refunds
// ---------------------------------------------------------------------------

var (
	ErrRefundNotAllowed  = errors.New("payment cannot be refunded")
	ErrRefundUnsupported = errors.New("refunds are only supported for Manjo payments")
	ErrRefundInProgress  = repository.ErrRefundAlreadyExists
)

// RefundRequest is an owner's request to refund an order's payment.
type RefundRequest struct {
	OrderID  string
	TenantID string
	Reason   string
	UserID   string
	UserName string
}

// RefundGatewayError is a refund the gateway rejected.
type RefundGatewayError struct {
	Message string
}

func (e *RefundGatewayError) Error() string {
	return e.Message
}

// PaymentSummary is the payment state of an order shown to admins.
type PaymentSummary struct {
	Gateway           string `json:"gateway"`
	TransactionStatus string `json:"transaction_status"`
	Amount            int    `json:"amount"`
	Refundable        bool   `json:"refundable"`
	Refunded          bool   `json:"refunded"`
}

// GetPaymentDetails returns the payment summary and refund history of an order.
func (s *PaymentService) GetPaymentDetails(ctx context.Context, order *models.GuestOrder) (*PaymentSummary, []models.PaymentRefund, error) {
	payment, err := s.paymentRepo.GetPaymentByOrderID(ctx, order.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get payment: %w", err)
	}
	if payment == nil {
		return nil, []models.PaymentRefund{}, nil
	}
	refunds, err := s.refundRepo.ListByOrderID(ctx, order.ID)
	if err != nil {
		return nil, nil, err
	}

	summary := &PaymentSummary{
		Gateway:           payment.Gateway(),
		TransactionStatus: derefString(payment.TransactionStatus),
		Amount:            payment.Amount,
	}
	active := false
	for _, refund := range refunds {
		switch refund.Status {
		case models.RefundStatusSuccess:
			summary.Refunded = true
			active = true
		case models.RefundStatusProcessing:
			active = true
		}
	}
	summary.Refundable = refundable(order, payment) && !active
	return summary, refunds, nil
}

func refundable(order *models.GuestOrder, payment *models.PaymentTransaction) bool {
	return payment.Gateway() == config.PaymentGatewayManjo &&
		derefString(payment.TransactionStatus) == manjo.StatusSettlement &&
		(order.Status == models.OrderStatusPaid || order.Status == models.OrderStatusComplete)
}

// RefundOrderPayment fully refunds a paid Manjo payment. The refund is recorded
// before Manjo is called, so a concurrent second request is rejected and a
// crash leaves a "processing" refund rather than an unrecorded one. The order
// status is not changed: the owner decides whether to cancel the order.
func (s *PaymentService) RefundOrderPayment(ctx context.Context, order *models.GuestOrder, req RefundRequest) (*models.PaymentRefund, error) {
	payment, err := s.paymentRepo.GetPaymentByOrderID(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	if payment == nil {
		return nil, ErrRefundNotAllowed
	}
	if payment.Gateway() != config.PaymentGatewayManjo {
		return nil, ErrRefundUnsupported
	}
	if !refundable(order, payment) {
		return nil, ErrRefundNotAllowed
	}
	if payment.GatewayExternalID == nil || payment.GatewayReferenceNo == nil {
		return nil, fmt.Errorf("payment %s is missing Manjo references", payment.ID)
	}

	creds, client, err := s.manjo.Credentials(ctx, order.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Manjo configuration: %w", err)
	}

	refund := &models.PaymentRefund{
		PaymentTransactionID: payment.ID,
		OrderID:              order.ID,
		TenantID:             order.TenantID,
		PaymentGateway:       config.PaymentGatewayManjo,
		PartnerRefundNo:      "RFD-" + strings.ReplaceAll(uuid.NewString(), "-", ""),
		Amount:               payment.Amount,
		Reason:               req.Reason,
		Status:               models.RefundStatusProcessing,
		RequestedByUserID:    optionalString(req.UserID),
		RequestedByName:      optionalString(req.UserName),
	}
	if err := s.refundRepo.Create(ctx, refund); err != nil {
		return nil, err
	}

	resp, refundErr := client.Refund(ctx, creds, manjo.RefundRequest{
		OriginalPartnerReferenceNo: payment.MidtransOrderID,
		OriginalReferenceNo:        *payment.GatewayReferenceNo,
		OriginalExternalID:         *payment.GatewayExternalID,
		PartnerRefundNo:            refund.PartnerRefundNo,
		RefundAmount:               manjo.Amount{Value: manjo.FormatAmount(refund.Amount), Currency: "IDR"},
		Reason:                     truncate(req.Reason, 256),
	})

	var apiErr *manjo.APIError
	switch {
	case refundErr == nil:
		refund.ResponsePayload, _ = json.Marshal(resp)
		refund.GatewayRefundNo = optionalString(resp.RefundNo)
		if !resp.InProgress {
			refund.Status = models.RefundStatusSuccess
		}
	case errors.As(refundErr, &apiErr):
		// Manjo answered and refused: nothing was refunded.
		refund.Status = models.RefundStatusFailed
		message := apiErr.ResponseMessage
		if message == "" {
			message = apiErr.Error()
		}
		refund.FailureReason = &message
	default:
		// Network error or unreadable answer: the refund may or may not have
		// happened, so it stays "processing" for manual follow-up.
		log.Error().Err(refundErr).
			Str("order_id", order.ID).
			Str("partner_refund_no", refund.PartnerRefundNo).
			Msg("Manjo refund outcome unknown - left as processing")
	}

	if err := s.refundRepo.UpdateResult(ctx, refund); err != nil {
		log.Error().Err(err).Str("refund_id", refund.ID).Msg("Failed to record Manjo refund result")
	}

	if refund.Status == models.RefundStatusSuccess {
		if err := s.recordGatewayStatus(ctx, payment, manjo.StatusRefund, refund.ResponsePayload, false); err != nil {
			log.Error().Err(err).Str("payment_id", payment.ID).Msg("Failed to mark payment refunded")
		}
	}

	note := fmt.Sprintf("Payment refund of Rp%d %s. Reason: %s", refund.Amount, refundNoteStatus(refund.Status), req.Reason)
	if err := s.orderService.AddOrderNote(ctx, order.ID, note, req.UserName); err != nil {
		log.Error().Err(err).Str("order_id", order.ID).Msg("Failed to add refund note")
	}

	log.Info().
		Str("order_id", order.ID).
		Str("refund_id", refund.ID).
		Str("status", refund.Status).
		Msg("Manjo refund requested")

	if refund.Status == models.RefundStatusFailed {
		return refund, &RefundGatewayError{Message: "Manjo rejected the refund: " + derefString(refund.FailureReason)}
	}
	return refund, nil
}

func refundNoteStatus(status string) string {
	switch status {
	case models.RefundStatusSuccess:
		return "completed"
	case models.RefundStatusFailed:
		return "failed"
	default:
		return "is being processed by the payment gateway"
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func truncate(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

// syncThrottle allows one action per key per interval.
type syncThrottle struct {
	interval time.Duration
	mu       sync.Mutex
	last     map[string]time.Time
}

func newSyncThrottle(interval time.Duration) *syncThrottle {
	return &syncThrottle{interval: interval, last: map[string]time.Time{}}
}

func (t *syncThrottle) allow(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	if last, ok := t.last[key]; ok && now.Sub(last) < t.interval {
		return false
	}
	t.last[key] = now
	// Drop stale entries so the map stays small.
	if len(t.last) > 1000 {
		for k, v := range t.last {
			if now.Sub(v) > t.interval {
				delete(t.last, k)
			}
		}
	}
	return true
}
