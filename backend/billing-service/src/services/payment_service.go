package services

import (
	"context"
	"fmt"
	"net/mail"
	"os"
	"strings"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"

	"github.com/pos/billing-service/src/models"
)

const (
	midtransItemNameMaxLength     = 50
	midtransCustomerNameMaxLength = 255
)

// SnapPaymentResult holds the result of a Midtrans Snap payment creation.
type SnapPaymentResult struct {
	Token      string
	PaymentURL string
	OrderID    string
}

// PaymentService handles Midtrans Snap payment creation.
type PaymentService struct {
	serverKey      string
	env            midtrans.EnvironmentType
	webhookURL     string
	frontendDomain string
}

// NewPaymentService creates a PaymentService from environment variables.
func NewPaymentService() *PaymentService {
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	env := midtrans.Sandbox
	if os.Getenv("MIDTRANS_ENV") == "production" {
		env = midtrans.Production
	}
	webhookURL := strings.TrimSpace(os.Getenv("MIDTRANS_WEBHOOK_URL"))
	frontendDomain := strings.TrimSpace(os.Getenv("FRONTEND_DOMAIN"))
	return &PaymentService{
		serverKey:      serverKey,
		env:            env,
		webhookURL:     webhookURL,
		frontendDomain: frontendDomain,
	}
}

// CreateSnapPayment creates a Midtrans Snap transaction for a billing invoice.
// Order ID format: BILL-YYYYMM-{invoice_number_suffix}.
func (p *PaymentService) CreateSnapPayment(_ context.Context, inv *models.BillingInvoice, tenantEmail, tenantBusinessName string) (*SnapPaymentResult, error) {
	if inv == nil {
		return nil, fmt.Errorf("invoice is required")
	}
	if inv.AmountIDR <= 0 {
		return nil, fmt.Errorf("invoice amount must be greater than zero")
	}

	var client snap.Client
	client.New(p.serverKey, p.env)
	p.configureSnapNotifications(&client)

	intervalLabel := "Monthly"
	if inv.BillingInterval == "annual" {
		intervalLabel = "Annual"
	}

	item := buildSubscriptionItem(inv, intervalLabel)

	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  buildBillingOrderID(inv),
			GrossAmt: int64(inv.AmountIDR),
		},
		Items: &[]midtrans.ItemDetails{
			item,
		},
		Expiry:    buildSnapExpiryDetails(),
		Callbacks: buildSnapCallbacks(p.frontendDomain),
	}
	if customer := buildCustomerDetails(tenantEmail, tenantBusinessName); customer != nil {
		req.CustomerDetail = customer
	}

	resp, midErr := client.CreateTransaction(req)
	if midErr != nil {
		return nil, fmt.Errorf("midtrans snap error: %s", midErr.Message)
	}

	return &SnapPaymentResult{
		Token:      resp.Token,
		PaymentURL: resp.RedirectURL,
		OrderID:    req.TransactionDetails.OrderID,
	}, nil
}

func (p *PaymentService) configureSnapNotifications(client *snap.Client) {
	if p == nil || client == nil || p.webhookURL == "" {
		return
	}
	client.Options.SetPaymentOverrideNotification(p.webhookURL)
}

func buildSnapExpiryDetails() *snap.ExpiryDetails {
	return &snap.ExpiryDetails{
		Duration: PaymentLinkTTLMinutes,
		Unit:     "minute",
	}
}

func buildSnapCallbacks(frontendDomain string) *snap.Callbacks {
	returnURL := buildSubscriptionReturnURL(frontendDomain)
	if returnURL == "" {
		return nil
	}
	return &snap.Callbacks{Finish: returnURL}
}

func buildSubscriptionReturnURL(frontendDomain string) string {
	frontendDomain = strings.TrimRight(strings.TrimSpace(frontendDomain), "/")
	if frontendDomain == "" {
		return ""
	}
	return frontendDomain + "/subscription?payment_return=midtrans"
}

func buildBillingOrderID(inv *models.BillingInvoice) string {
	return fmt.Sprintf("BILL-%s-%s", inv.PeriodStart.Format("200601"), invoiceNumberSuffix(inv.InvoiceNumber))
}

func invoiceNumberSuffix(invoiceNumber string) string {
	parts := strings.Split(invoiceNumber, "-")
	if len(parts) == 0 {
		return invoiceNumber
	}
	suffix := strings.TrimSpace(parts[len(parts)-1])
	if suffix == "" {
		return invoiceNumber
	}
	return suffix
}

func buildSubscriptionItem(inv *models.BillingInvoice, intervalLabel string) midtrans.ItemDetails {
	name := truncateForMidtrans(fmt.Sprintf("Posku %s Subscription", intervalLabel), midtransItemNameMaxLength)
	return midtrans.ItemDetails{
		ID:    inv.InvoiceNumber,
		Price: int64(inv.AmountIDR),
		Qty:   1,
		Name:  name,
	}
}

func buildCustomerDetails(email, businessName string) *midtrans.CustomerDetails {
	customer := &midtrans.CustomerDetails{
		FName: truncateForMidtrans(strings.TrimSpace(businessName), midtransCustomerNameMaxLength),
	}
	if validEmail := normalizeEmail(email); validEmail != "" {
		customer.Email = validEmail
	}
	if customer.FName == "" && customer.Email == "" {
		return nil
	}
	return customer
}

func normalizeEmail(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil {
		return ""
	}
	return parsed.Address
}

func truncateForMidtrans(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
