package models

import (
	"encoding/json"
	"time"
)

// PaymentTransaction is a guest order QRIS payment at Midtrans or Manjo.
// The midtrans_* columns hold the gateway transaction ID and the merchant
// reference for both gateways.
type PaymentTransaction struct {
	ID                     string          `json:"id"`
	OrderID                string          `json:"order_id"`
	PaymentGateway         string          `json:"payment_gateway"`
	GatewayReferenceNo     *string         `json:"gateway_reference_no,omitempty"` // Manjo referenceNo
	GatewayExternalID      *string         `json:"gateway_external_id,omitempty"`  // Manjo X-EXTERNAL-ID of the generate call
	MidtransTransactionID  *string         `json:"midtrans_transaction_id,omitempty"`
	MidtransOrderID        string          `json:"midtrans_order_id"`
	Amount                 int             `json:"amount"`
	PaymentType            *string         `json:"payment_type,omitempty"`
	TransactionStatus      *string         `json:"transaction_status,omitempty"`
	FraudStatus            *string         `json:"fraud_status,omitempty"`
	NotificationPayload    json.RawMessage `json:"notification_payload,omitempty"`
	SignatureKey           *string         `json:"signature_key,omitempty"`
	SignatureVerified      bool            `json:"signature_verified"`
	QRCodeURL              *string         `json:"qr_code_url,omitempty"` // URL to QR code image
	QRString               *string         `json:"qr_string,omitempty"`   // Raw QRIS string
	ExpiryTime             *time.Time      `json:"expiry_time,omitempty"` // Payment expiration time
	CreatedAt              time.Time       `json:"created_at"`
	NotificationReceivedAt *time.Time      `json:"notification_received_at,omitempty"`
	SettledAt              *time.Time      `json:"settled_at,omitempty"`
	IdempotencyKey         *string         `json:"idempotency_key,omitempty"`
}

// GenerateIdempotencyKey creates a unique key for webhook deduplication
func (pt *PaymentTransaction) GenerateIdempotencyKey() string {
	if pt.MidtransTransactionID != nil && pt.TransactionStatus != nil {
		return *pt.MidtransTransactionID + ":" + *pt.TransactionStatus
	}
	return ""
}

// Gateway returns the payment gateway, defaulting to Midtrans for rows written
// before Manjo existed.
func (pt *PaymentTransaction) Gateway() string {
	if pt.PaymentGateway == "" {
		return "midtrans"
	}
	return pt.PaymentGateway
}
