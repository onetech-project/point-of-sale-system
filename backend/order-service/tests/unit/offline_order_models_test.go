package unit
package unit

import (
	"testing"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T118: Unit tests for offline order model methods.

func TestPaymentRecord_ValidatePayment(t *testing.T) {
	t.Run("Valid cash payment", func(t *testing.T) {
		pr := &models.PaymentRecord{
			AmountPaid:            100000,
			RemainingBalanceAfter: 0,
			PaymentMethod:         models.PaymentMethodCash,
		}
		assert.NoError(t, pr.ValidatePayment())
	})

	t.Run("Zero amount is invalid", func(t *testing.T) {
		pr := &models.PaymentRecord{
			AmountPaid:            0,
			RemainingBalanceAfter: 0,
			PaymentMethod:         models.PaymentMethodCash,
		}
		assert.ErrorIs(t, pr.ValidatePayment(), models.ErrInvalidPaymentAmount)
	})

	t.Run("Negative remaining balance is invalid", func(t *testing.T) {
		pr := &models.PaymentRecord{
			AmountPaid:            100000,
			RemainingBalanceAfter: -1,
			PaymentMethod:         models.PaymentMethodCash,
		}
		assert.ErrorIs(t, pr.ValidatePayment(), models.ErrNegativeBalance)
	})

	t.Run("Invalid payment method", func(t *testing.T) {
		pr := &models.PaymentRecord{
			AmountPaid:            100000,
			RemainingBalanceAfter: 0,
			PaymentMethod:         models.PaymentMethod("bitcoin"),
		}
		assert.ErrorIs(t, pr.ValidatePayment(), models.ErrInvalidPaymentMethod)
	})

	t.Run("All valid payment methods accepted", func(t *testing.T) {
		methods := []models.PaymentMethod{
			models.PaymentMethodCash,
			models.PaymentMethodCard,
			models.PaymentMethodBankTransfer,
			models.PaymentMethodCheck,
			models.PaymentMethodOther,
		}
		for _, method := range methods {
			pr := &models.PaymentRecord{
				AmountPaid:            1,
				RemainingBalanceAfter: 0,
				PaymentMethod:         method,
			}
			assert.NoError(t, pr.ValidatePayment(), "method %s should be valid", method)
		}
	})
}

func TestPaymentRecord_PaymentTypeChecks(t *testing.T) {
	t.Run("IsDownPayment returns true for payment_number=0", func(t *testing.T) {
		pr := &models.PaymentRecord{PaymentNumber: 0}
		assert.True(t, pr.IsDownPayment())
		assert.False(t, pr.IsInstallment())
	})

	t.Run("IsInstallment returns true for payment_number>=1", func(t *testing.T) {
		pr := &models.PaymentRecord{PaymentNumber: 1}
		assert.False(t, pr.IsDownPayment())
		assert.True(t, pr.IsInstallment())
	})

	t.Run("IsInstallment for higher numbers", func(t *testing.T) {
		pr := &models.PaymentRecord{PaymentNumber: 5}
		assert.True(t, pr.IsInstallment())
		assert.False(t, pr.IsDownPayment())
	})

	t.Run("IsFinalPayment when balance is zero", func(t *testing.T) {
		pr := &models.PaymentRecord{RemainingBalanceAfter: 0}
		assert.True(t, pr.IsFinalPayment())
	})

	t.Run("IsFinalPayment returns false when balance remains", func(t *testing.T) {
		pr := &models.PaymentRecord{RemainingBalanceAfter: 50000}
		assert.False(t, pr.IsFinalPayment())
	})
}

func TestPaymentTerms_Methods(t *testing.T) {
	t.Run("HasRemainingBalance true when balance > 0", func(t *testing.T) {
		pt := &models.PaymentTerms{RemainingBalance: 100000}
		assert.True(t, pt.HasRemainingBalance())
	})

	t.Run("HasRemainingBalance false when balance = 0", func(t *testing.T) {
		pt := &models.PaymentTerms{RemainingBalance: 0}
		assert.False(t, pt.HasRemainingBalance())
	})

	t.Run("IsFullyPaid when balance=0 and total_paid>=total_amount", func(t *testing.T) {
		pt := &models.PaymentTerms{
			TotalAmount:      500000,
			TotalPaid:        500000,
			RemainingBalance: 0,
		}
		assert.True(t, pt.IsFullyPaid())
	})

	t.Run("IsFullyPaid false when balance > 0", func(t *testing.T) {
		pt := &models.PaymentTerms{
			TotalAmount:      500000,
			TotalPaid:        200000,
			RemainingBalance: 300000,
		}
		assert.False(t, pt.IsFullyPaid())
	})

	t.Run("CalculateRemainingBalance", func(t *testing.T) {
		pt := &models.PaymentTerms{
			TotalAmount: 500000,
			TotalPaid:   150000,
		}
		assert.Equal(t, 350000, pt.CalculateRemainingBalance())
	})
}

func TestPaymentTerms_ValidatePaymentStructure(t *testing.T) {
	t.Run("Valid payment terms", func(t *testing.T) {
		pt := &models.PaymentTerms{
			TotalAmount:       500000,
			InstallmentCount:  3,
			InstallmentAmount: 100000,
		}
		assert.NoError(t, pt.ValidatePaymentStructure())
	})

	t.Run("Zero total amount is invalid", func(t *testing.T) {
		pt := &models.PaymentTerms{TotalAmount: 0}
		require.ErrorIs(t, pt.ValidatePaymentStructure(), models.ErrInvalidTotalAmount)
	})

	t.Run("Down payment >= total is invalid", func(t *testing.T) {
		downPayment := 500000
		pt := &models.PaymentTerms{
			TotalAmount:       500000,
			DownPaymentAmount: &downPayment,
		}
		require.ErrorIs(t, pt.ValidatePaymentStructure(), models.ErrDownPaymentTooLarge)
	})

	t.Run("Negative installment count is invalid", func(t *testing.T) {
		pt := &models.PaymentTerms{
			TotalAmount:      500000,
			InstallmentCount: -1,
		}
		require.ErrorIs(t, pt.ValidatePaymentStructure(), models.ErrInvalidInstallmentCount)
	})

	t.Run("Negative installment amount is invalid", func(t *testing.T) {
		pt := &models.PaymentTerms{
			TotalAmount:       500000,
			InstallmentAmount: -1,
		}
		require.ErrorIs(t, pt.ValidatePaymentStructure(), models.ErrInvalidInstallmentAmount)
	})
}

func TestPaymentSchedule_ScanValue(t *testing.T) {
	t.Run("Scan nil returns empty schedule", func(t *testing.T) {
		var ps models.PaymentSchedule
		err := ps.Scan(nil)
		assert.NoError(t, err)
		assert.Empty(t, ps)
	})

	t.Run("Value of nil returns nil", func(t *testing.T) {
		var ps models.PaymentSchedule
		val, err := ps.Value()
		assert.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("Scan valid JSON bytes", func(t *testing.T) {
		var ps models.PaymentSchedule
		jsonBytes := []byte(`[{"installment_number":1,"due_date":"2026-01-31","amount":100000,"status":"pending"}]`)
		err := ps.Scan(jsonBytes)
		require.NoError(t, err)
		require.Len(t, ps, 1)
		assert.Equal(t, 1, ps[0].InstallmentNumber)
		assert.Equal(t, 100000, ps[0].Amount)
	})

	t.Run("Value serializes to JSON", func(t *testing.T) {
		ps := models.PaymentSchedule{
			{InstallmentNumber: 1, DueDate: "2026-01-31", Amount: 100000, Status: "pending"},
		}
		val, err := ps.Value()
		require.NoError(t, err)
		assert.NotNil(t, val)
	})
}
