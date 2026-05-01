package services

import (
	"fmt"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
)

// PaymentCalculator provides utilities for calculating payment schedules
// Implements T060: Payment schedule calculator
type PaymentCalculator struct{}

// NewPaymentCalculator creates a new payment calculator
func NewPaymentCalculator() *PaymentCalculator {
	return &PaymentCalculator{}
}

// CalculateInstallmentSchedule generates a payment schedule for installment payments
// Distributes remaining balance across installments with due dates
func (pc *PaymentCalculator) CalculateInstallmentSchedule(
	totalAmount int,
	downPaymentAmount int,
	installmentCount int,
	startDate time.Time,
	intervalDays int,
) ([]models.Installment, error) {
	if totalAmount <= 0 {
		return nil, fmt.Errorf("total amount must be greater than 0")
	}
	if downPaymentAmount < 0 || downPaymentAmount >= totalAmount {
		return nil, fmt.Errorf("down payment must be between 0 and total amount")
	}
	if installmentCount <= 0 {
		return nil, fmt.Errorf("installment count must be greater than 0")
	}
	if intervalDays <= 0 {
		return nil, fmt.Errorf("interval days must be greater than 0")
	}

	remainingAmount := totalAmount - downPaymentAmount
	baseInstallmentAmount := remainingAmount / installmentCount
	remainder := remainingAmount % installmentCount

	schedule := make([]models.Installment, installmentCount)
	for i := 0; i < installmentCount; i++ {
		// Distribute remainder across first few installments
		installmentAmount := baseInstallmentAmount
		if i < remainder {
			installmentAmount++
		}

		// Calculate due date (first installment is intervalDays from start date)
		dueDate := startDate.AddDate(0, 0, (i+1)*intervalDays)

		schedule[i] = models.Installment{
			InstallmentNumber: i + 1,
			DueDate:           dueDate.Format("2006-01-02"),
			Amount:            installmentAmount,
			Status:            "pending",
		}
	}

	return schedule, nil
}

// CalculateRemainingBalance computes remaining balance after a payment
func (pc *PaymentCalculator) CalculateRemainingBalance(currentBalance int, paymentAmount int) (int, error) {
	if paymentAmount < 0 {
		return 0, fmt.Errorf("payment amount cannot be negative")
	}
	if paymentAmount > currentBalance {
		return 0, fmt.Errorf("payment amount exceeds remaining balance")
	}
	return currentBalance - paymentAmount, nil
}

// ValidatePaymentAmount checks if payment amount is valid for the current balance
func (pc *PaymentCalculator) ValidatePaymentAmount(paymentAmount int, remainingBalance int) error {
	if paymentAmount <= 0 {
		return fmt.Errorf("payment amount must be greater than 0")
	}
	if paymentAmount > remainingBalance {
		return fmt.Errorf("payment amount (%d) exceeds remaining balance (%d)", paymentAmount, remainingBalance)
	}
	return nil
}

// CalculateNextPaymentNumber determines the next payment number based on existing payments
// Returns 0 for down payment, 1+ for installments
func (pc *PaymentCalculator) CalculateNextPaymentNumber(existingPayments []models.PaymentRecord) int {
	if len(existingPayments) == 0 {
		return 0 // First payment (down payment or full payment)
	}

	maxPaymentNumber := 0
	for _, payment := range existingPayments {
		if payment.PaymentNumber > maxPaymentNumber {
			maxPaymentNumber = payment.PaymentNumber
		}
	}

	return maxPaymentNumber + 1
}

// SumPaymentAmounts calculates total paid amount from payment records
func (pc *PaymentCalculator) SumPaymentAmounts(payments []models.PaymentRecord) int {
	total := 0
	for _, payment := range payments {
		total += payment.AmountPaid
	}
	return total
}

// IsFullyPaid checks if an order is fully paid based on payment records
func (pc *PaymentCalculator) IsFullyPaid(totalAmount int, payments []models.PaymentRecord) bool {
	totalPaid := pc.SumPaymentAmounts(payments)
	return totalPaid >= totalAmount
}

// GetOverdueInstallments returns installments that are past their due date
func (pc *PaymentCalculator) GetOverdueInstallments(schedule []models.Installment, currentDate time.Time) []models.Installment {
	var overdue []models.Installment
	for _, installment := range schedule {
		if installment.Status != "paid" {
			dueDate, err := time.Parse("2006-01-02", installment.DueDate)
			if err != nil {
				continue
			}
			if dueDate.Before(currentDate) {
				overdue = append(overdue, installment)
			}
		}
	}
	return overdue
}
