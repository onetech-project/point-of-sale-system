package unit

import (
	"testing"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T118: Unit tests calling actual PaymentCalculator service code for coverage.

func TestPaymentCalculatorService_CalculateInstallmentSchedule(t *testing.T) {
	calc := services.NewPaymentCalculator()
	startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("3 equal installments", func(t *testing.T) {
		schedule, err := calc.CalculateInstallmentSchedule(1000000, 300000, 3, startDate, 30)
		require.NoError(t, err)
		require.Len(t, schedule, 3)

		// Remaining = 700000; base = 233333, remainder = 1 → first gets +1
		assert.Equal(t, 233334, schedule[0].Amount)
		assert.Equal(t, 233333, schedule[1].Amount)
		assert.Equal(t, 233333, schedule[2].Amount)

		total := 300000
		for _, inst := range schedule {
			total += inst.Amount
		}
		assert.Equal(t, 1000000, total)
	})

	t.Run("Installment numbers and due dates", func(t *testing.T) {
		schedule, err := calc.CalculateInstallmentSchedule(600000, 0, 2, startDate, 30)
		require.NoError(t, err)
		assert.Equal(t, 1, schedule[0].InstallmentNumber)
		assert.Equal(t, 2, schedule[1].InstallmentNumber)
		assert.Equal(t, "2026-01-31", schedule[0].DueDate)
		assert.Equal(t, "2026-03-02", schedule[1].DueDate)
		assert.Equal(t, "pending", schedule[0].Status)
	})

	t.Run("Invalid total amount", func(t *testing.T) {
		_, err := calc.CalculateInstallmentSchedule(0, 0, 3, startDate, 30)
		require.Error(t, err)
	})

	t.Run("Down payment >= total amount", func(t *testing.T) {
		_, err := calc.CalculateInstallmentSchedule(100000, 100000, 3, startDate, 30)
		require.Error(t, err)
	})

	t.Run("Negative down payment", func(t *testing.T) {
		_, err := calc.CalculateInstallmentSchedule(100000, -1, 3, startDate, 30)
		require.Error(t, err)
	})

	t.Run("Zero installment count", func(t *testing.T) {
		_, err := calc.CalculateInstallmentSchedule(100000, 0, 0, startDate, 30)
		require.Error(t, err)
	})

	t.Run("Zero interval days", func(t *testing.T) {
		_, err := calc.CalculateInstallmentSchedule(100000, 0, 3, startDate, 0)
		require.Error(t, err)
	})
}

func TestPaymentCalculatorService_CalculateRemainingBalance(t *testing.T) {
	calc := services.NewPaymentCalculator()

	t.Run("Valid payment reduces balance", func(t *testing.T) {
		balance, err := calc.CalculateRemainingBalance(500000, 200000)
		require.NoError(t, err)
		assert.Equal(t, 300000, balance)
	})

	t.Run("Full payment clears balance", func(t *testing.T) {
		balance, err := calc.CalculateRemainingBalance(500000, 500000)
		require.NoError(t, err)
		assert.Equal(t, 0, balance)
	})

	t.Run("Payment exceeds balance", func(t *testing.T) {
		_, err := calc.CalculateRemainingBalance(100000, 200000)
		require.Error(t, err)
	})

	t.Run("Negative payment amount", func(t *testing.T) {
		_, err := calc.CalculateRemainingBalance(100000, -1)
		require.Error(t, err)
	})
}

func TestPaymentCalculatorService_ValidatePaymentAmount(t *testing.T) {
	calc := services.NewPaymentCalculator()

	t.Run("Valid amount", func(t *testing.T) {
		err := calc.ValidatePaymentAmount(50000, 100000)
		assert.NoError(t, err)
	})

	t.Run("Zero amount is invalid", func(t *testing.T) {
		err := calc.ValidatePaymentAmount(0, 100000)
		require.Error(t, err)
	})

	t.Run("Amount exceeds balance", func(t *testing.T) {
		err := calc.ValidatePaymentAmount(150000, 100000)
		require.Error(t, err)
	})

	t.Run("Exact balance is valid", func(t *testing.T) {
		err := calc.ValidatePaymentAmount(100000, 100000)
		assert.NoError(t, err)
	})
}

func TestPaymentCalculatorService_CalculateNextPaymentNumber(t *testing.T) {
	calc := services.NewPaymentCalculator()

	t.Run("No existing payments returns 0", func(t *testing.T) {
		n := calc.CalculateNextPaymentNumber(nil)
		assert.Equal(t, 0, n)
	})

	t.Run("Returns max+1", func(t *testing.T) {
		payments := []models.PaymentRecord{
			{PaymentNumber: 0},
			{PaymentNumber: 1},
			{PaymentNumber: 2},
		}
		n := calc.CalculateNextPaymentNumber(payments)
		assert.Equal(t, 3, n)
	})

	t.Run("Single payment returns 1", func(t *testing.T) {
		payments := []models.PaymentRecord{{PaymentNumber: 0}}
		n := calc.CalculateNextPaymentNumber(payments)
		assert.Equal(t, 1, n)
	})
}

func TestPaymentCalculatorService_SumPaymentAmounts(t *testing.T) {
	calc := services.NewPaymentCalculator()

	t.Run("Sum of multiple payments", func(t *testing.T) {
		payments := []models.PaymentRecord{
			{AmountPaid: 100000},
			{AmountPaid: 200000},
			{AmountPaid: 300000},
		}
		sum := calc.SumPaymentAmounts(payments)
		assert.Equal(t, 600000, sum)
	})

	t.Run("Empty payments returns 0", func(t *testing.T) {
		sum := calc.SumPaymentAmounts(nil)
		assert.Equal(t, 0, sum)
	})
}

func TestPaymentCalculatorService_IsFullyPaid(t *testing.T) {
	calc := services.NewPaymentCalculator()

	t.Run("Fully paid", func(t *testing.T) {
		payments := []models.PaymentRecord{{AmountPaid: 500000}}
		assert.True(t, calc.IsFullyPaid(500000, payments))
	})

	t.Run("Overpaid counts as fully paid", func(t *testing.T) {
		payments := []models.PaymentRecord{{AmountPaid: 600000}}
		assert.True(t, calc.IsFullyPaid(500000, payments))
	})

	t.Run("Partial payment not fully paid", func(t *testing.T) {
		payments := []models.PaymentRecord{{AmountPaid: 300000}}
		assert.False(t, calc.IsFullyPaid(500000, payments))
	})

	t.Run("No payments not fully paid", func(t *testing.T) {
		assert.False(t, calc.IsFullyPaid(500000, nil))
	})
}

func TestPaymentCalculatorService_GetOverdueInstallments(t *testing.T) {
	calc := services.NewPaymentCalculator()
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	schedule := []models.Installment{
		{InstallmentNumber: 1, DueDate: "2026-01-01", Status: "pending"}, // overdue
		{InstallmentNumber: 2, DueDate: "2026-02-01", Status: "paid"},    // paid, skip
		{InstallmentNumber: 3, DueDate: "2026-06-01", Status: "pending"}, // future
	}

	overdue := calc.GetOverdueInstallments(schedule, now)
	require.Len(t, overdue, 1)
	assert.Equal(t, 1, overdue[0].InstallmentNumber)
}

func TestPaymentCalculatorService_GetOverdueInstallments_BadDate(t *testing.T) {
	calc := services.NewPaymentCalculator()
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	// Invalid date format should be skipped gracefully
	schedule := []models.Installment{
		{InstallmentNumber: 1, DueDate: "not-a-date", Status: "pending"},
	}
	overdue := calc.GetOverdueInstallments(schedule, now)
	assert.Len(t, overdue, 0)
}
