package models

import (
	"encoding/json"
	"time"
)

// PaymentTransaction represents a Midtrans payment transaction
type PaymentTransaction struct {
	ID                     string          `json:"id"`
	OrderID                string          `json:"order_id"`
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
