package models

import (
	"encoding/json"
	"time"
)

// Refund statuses.
const (
	RefundStatusProcessing = "processing"
	RefundStatusSuccess    = "success"
	RefundStatusFailed     = "failed"
)

// PaymentRefund is a refund of a guest order QRIS payment.
type PaymentRefund struct {
	ID                   string          `json:"id"`
	PaymentTransactionID string          `json:"payment_transaction_id"`
	OrderID              string          `json:"order_id"`
	TenantID             string          `json:"tenant_id"`
	PaymentGateway       string          `json:"payment_gateway"`
	PartnerRefundNo      string          `json:"partner_refund_no"`
	GatewayRefundNo      *string         `json:"gateway_refund_no,omitempty"`
	Amount               int             `json:"amount"`
	Reason               string          `json:"reason"`
	Status               string          `json:"status"`
	FailureReason        *string         `json:"failure_reason,omitempty"`
	ResponsePayload      json.RawMessage `json:"-"`
	RequestedByUserID    *string         `json:"requested_by_user_id,omitempty"`
	RequestedByName      *string         `json:"requested_by_name,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}
