package services

import (
	"context"
	"fmt"
	"os"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"

	"github.com/pos/billing-service/src/models"
)

// SnapPaymentResult holds the result of a Midtrans Snap payment creation.
type SnapPaymentResult struct {
	Token      string
	PaymentURL string
	OrderID    string
}

// PaymentService handles Midtrans Snap payment creation.
type PaymentService struct {
	serverKey string
	env       midtrans.EnvironmentType
}

// NewPaymentService creates a PaymentService from environment variables.
func NewPaymentService() *PaymentService {
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	env := midtrans.Sandbox
	if os.Getenv("MIDTRANS_ENV") == "production" {
		env = midtrans.Production
	}
	return &PaymentService{serverKey: serverKey, env: env}
}

// CreateSnapPayment creates a Midtrans Snap transaction for a billing invoice.
// Order ID format: BILL-YYYYMM-{invoice_number_suffix}
func (p *PaymentService) CreateSnapPayment(_ context.Context, inv *models.BillingInvoice, tenantEmail, tenantBusinessName string) (*SnapPaymentResult, error) {
	var client snap.Client
	client.New(p.serverKey, p.env)

	// Build order ID from the invoice number suffix (last 6 digits)
	suffix := inv.InvoiceNumber
	if len(suffix) > 6 {
		suffix = suffix[len(suffix)-6:]
	}
	orderID := fmt.Sprintf("BILL-%s-%s", inv.PeriodStart.Format("200601"), suffix)

	intervalLabel := "Monthly"
	if inv.BillingInterval == "annual" {
		intervalLabel = "Annual"
	}
	itemName := fmt.Sprintf("POS Subscription - %s - %s to %s",
		intervalLabel,
		inv.PeriodStart.Format("2006-01-02"),
		inv.PeriodEnd.Format("2006-01-02"),
	)

	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  orderID,
			GrossAmt: int64(inv.AmountIDR),
		},
		CustomerDetail: &midtrans.CustomerDetails{
			Email: tenantEmail,
			FName: tenantBusinessName,
		},
		Items: &[]midtrans.ItemDetails{
			{
				ID:    inv.InvoiceNumber,
				Price: int64(inv.AmountIDR),
				Qty:   1,
				Name:  itemName,
			},
		},
		Expiry: &snap.ExpiryDetails{
			Duration: 24,
			Unit:     "hour",
		},
	}

	resp, midErr := client.CreateTransaction(req)
	if midErr != nil {
		return nil, fmt.Errorf("midtrans snap error: %s", midErr.Message)
	}

	return &SnapPaymentResult{
		Token:      resp.Token,
		PaymentURL: resp.RedirectURL,
		OrderID:    orderID,
	}, nil
}
