package models

import (
	"errors"
	"time"
)

// PaymentMethod represents the method used for payment
type PaymentMethod string

const (
	PaymentMethodCash         PaymentMethod = "cash"
	PaymentMethodCard         PaymentMethod = "card"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodCheck        PaymentMethod = "check"
	PaymentMethodOther        PaymentMethod = "other"
)

// PaymentRecord represents a single payment transaction for an offline order
type PaymentRecord struct {
	ID                    string        `json:"id"`
	OrderID               string        `json:"order_id"`
	PaymentTermsID        *string       `json:"payment_terms_id,omitempty"` // NULL for full payment orders
	PaymentNumber         int           `json:"payment_number"`              // 0 = down payment, 1+ = installment number
	AmountPaid            int           `json:"amount_paid"`                 // In smallest currency unit (IDR cents)
	PaymentDate           time.Time     `json:"payment_date"`
	PaymentMethod         PaymentMethod `json:"payment_method"`
	RemainingBalanceAfter int           `json:"remaining_balance_after"`     // Outstanding balance after this payment
	RecordedByUserID      string        `json:"recorded_by_user_id"`         // Staff who recorded the payment
	Notes                 *string       `json:"notes,omitempty"`
	ReceiptNumber         *string       `json:"receipt_number,omitempty"`
	CreatedAt             time.Time     `json:"created_at"`
}

// Payment record validation errors
var (
	ErrInvalidPaymentAmount = errors.New("payment amount must be greater than 0")
	ErrInvalidPaymentMethod = errors.New("invalid payment method")
	ErrNegativeBalance      = errors.New("remaining balance cannot be negative")
)

// Scan implements sql.Scanner for PaymentMethod
func (pm *PaymentMethod) Scan(value interface{}) error {
	if value == nil {
		*pm = PaymentMethodCash
		return nil
	}
	*pm = PaymentMethod(value.(string))
	return nil
}

// CreatePaymentRecordRequest represents the request to record a payment
type CreatePaymentRecordRequest struct {
	OrderID               string        `json:"order_id" validate:"required,uuid"`
	PaymentTermsID        *string       `json:"payment_terms_id,omitempty" validate:"omitempty,uuid"`
	PaymentNumber         int           `json:"payment_number" validate:"required,min=0"`
	AmountPaid            int           `json:"amount_paid" validate:"required,min=1"`
	PaymentMethod         PaymentMethod `json:"payment_method" validate:"required,oneof=cash card bank_transfer check other"`
	RemainingBalanceAfter int           `json:"remaining_balance_after" validate:"required,min=0"`
	RecordedByUserID      string        `json:"recorded_by_user_id" validate:"required,uuid"`
	Notes                 *string       `json:"notes,omitempty" validate:"omitempty,max=1000"`
	ReceiptNumber         *string       `json:"receipt_number,omitempty" validate:"omitempty,max=100"`
}

// PaymentRecordResponse represents a payment record with additional context
type PaymentRecordResponse struct {
	*PaymentRecord
	OrderReference string `json:"order_reference,omitempty"`
}

// ValidatePayment checks if the payment record is valid
func (pr *PaymentRecord) ValidatePayment() error {
	if pr.AmountPaid <= 0 {
		return ErrInvalidPaymentAmount
	}
	
	if pr.RemainingBalanceAfter < 0 {
		return ErrNegativeBalance
	}
	
	validMethods := []PaymentMethod{
		PaymentMethodCash,
		PaymentMethodCard,
		PaymentMethodBankTransfer,
		PaymentMethodCheck,
		PaymentMethodOther,
	}
	
	isValid := false
	for _, method := range validMethods {
		if pr.PaymentMethod == method {
			isValid = true
			break
		}
	}
	
	if !isValid {
		return ErrInvalidPaymentMethod
	}
	
	return nil
}

// IsDownPayment checks if this is a down payment (payment_number = 0)
func (pr *PaymentRecord) IsDownPayment() bool {
	return pr.PaymentNumber == 0
}

// IsInstallment checks if this is an installment payment (payment_number >= 1)
func (pr *PaymentRecord) IsInstallment() bool {
	return pr.PaymentNumber >= 1
}

// IsFinalPayment checks if this payment brings the balance to zero
func (pr *PaymentRecord) IsFinalPayment() bool {
	return pr.RemainingBalanceAfter == 0
}
