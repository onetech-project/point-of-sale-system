package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Payment terms validation errors
var (
	ErrInvalidTotalAmount      = errors.New("total amount must be greater than 0")
	ErrDownPaymentTooLarge     = errors.New("down payment amount must be less than total amount")
	ErrInvalidInstallmentCount = errors.New("installment count must be 0 or greater")
	ErrInvalidInstallmentAmount = errors.New("installment amount must be 0 or greater")
)

// PaymentTerms represents the payment schedule for an offline order with installments
type PaymentTerms struct {
	ID                 string                 `json:"id"`
	OrderID            string                 `json:"order_id"`
	TotalAmount        int                    `json:"total_amount"`         // In smallest currency unit (IDR cents)
	DownPaymentAmount  *int                   `json:"down_payment_amount"`  // Optional down payment
	InstallmentCount   int                    `json:"installment_count"`    // Number of installments
	InstallmentAmount  int                    `json:"installment_amount"`   // Amount per installment
	PaymentSchedule    PaymentSchedule        `json:"payment_schedule"`     // JSONB: Array of installment details
	TotalPaid          int                    `json:"total_paid"`           // Running total of payments received
	RemainingBalance   int                    `json:"remaining_balance"`    // Computed: total_amount - total_paid
	CreatedAt          time.Time              `json:"created_at"`
	CreatedByUserID    string                 `json:"created_by_user_id"`   // Staff who created the payment terms
}

// PaymentSchedule represents the JSONB array of installment schedules
type PaymentSchedule []Installment

// Installment represents a single installment in the payment schedule
type Installment struct {
	InstallmentNumber int       `json:"installment_number"`
	DueDate           string    `json:"due_date"` // ISO 8601 date string (YYYY-MM-DD)
	Amount            int       `json:"amount"`   // Amount in smallest currency unit
	Status            string    `json:"status"`   // "pending", "paid", "overdue"
}

// Scan implements sql.Scanner for PaymentSchedule (JSONB)
func (ps *PaymentSchedule) Scan(value interface{}) error {
	if value == nil {
		*ps = PaymentSchedule{}
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	
	return json.Unmarshal(bytes, ps)
}

// Value implements driver.Valuer for PaymentSchedule (JSONB)
func (ps PaymentSchedule) Value() (driver.Value, error) {
	if ps == nil {
		return nil, nil
	}
	return json.Marshal(ps)
}

// CreatePaymentTermsRequest represents the request to create payment terms for an order
type CreatePaymentTermsRequest struct {
	OrderID            string              `json:"order_id" validate:"required,uuid"`
	TotalAmount        int                 `json:"total_amount" validate:"required,min=1"`
	DownPaymentAmount  *int                `json:"down_payment_amount,omitempty" validate:"omitempty,min=0"`
	InstallmentCount   int                 `json:"installment_count" validate:"required,min=0"`
	InstallmentAmount  int                 `json:"installment_amount" validate:"required,min=0"`
	PaymentSchedule    []Installment       `json:"payment_schedule" validate:"required,min=1,dive"`
	CreatedByUserID    string              `json:"created_by_user_id" validate:"required,uuid"`
}

// UpdatePaymentTermsRequest represents the request to update payment terms
type UpdatePaymentTermsRequest struct {
	TotalPaid        int `json:"total_paid" validate:"required,min=0"`
	RemainingBalance int `json:"remaining_balance" validate:"required,min=0"`
}

// HasRemainingBalance checks if there is an outstanding balance
func (pt *PaymentTerms) HasRemainingBalance() bool {
	return pt.RemainingBalance > 0
}

// IsFullyPaid checks if the order is fully paid
func (pt *PaymentTerms) IsFullyPaid() bool {
	return pt.RemainingBalance == 0 && pt.TotalPaid >= pt.TotalAmount
}

// CalculateRemainingBalance computes the remaining balance
func (pt *PaymentTerms) CalculateRemainingBalance() int {
	return pt.TotalAmount - pt.TotalPaid
}

// ValidatePaymentStructure checks if the payment terms are valid
func (pt *PaymentTerms) ValidatePaymentStructure() error {
	if pt.TotalAmount <= 0 {
		return ErrInvalidTotalAmount
	}
	
	if pt.DownPaymentAmount != nil && *pt.DownPaymentAmount >= pt.TotalAmount {
		return ErrDownPaymentTooLarge
	}
	
	if pt.InstallmentCount < 0 {
		return ErrInvalidInstallmentCount
	}
	
	if pt.InstallmentAmount < 0 {
		return ErrInvalidInstallmentAmount
	}
	
	return nil
}
