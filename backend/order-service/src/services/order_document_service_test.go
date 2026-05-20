package services

import (
	"strings"
	"testing"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
)

func TestValidateOrderDocumentEligibility(t *testing.T) {
	tests := []struct {
		name         string
		status       models.OrderStatus
		documentType OrderDocumentType
		wantErr      bool
	}{
		{name: "invoice for pending order", status: models.OrderStatusPending, documentType: OrderDocumentTypeInvoice},
		{name: "invoice for cancelled order", status: models.OrderStatusCancelled, documentType: OrderDocumentTypeInvoice, wantErr: true},
		{name: "receipt for paid order", status: models.OrderStatusPaid, documentType: OrderDocumentTypeReceipt},
		{name: "receipt for complete order", status: models.OrderStatusComplete, documentType: OrderDocumentTypeReceipt},
		{name: "receipt for pending order", status: models.OrderStatusPending, documentType: OrderDocumentTypeReceipt, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOrderDocumentEligibility(tt.status, tt.documentType)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateOrderDocumentEligibility() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRenderOrderDocumentPDF(t *testing.T) {
	paidAt := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	snapshot := &OrderDocumentSnapshot{
		Source:         OrderDocumentSourceOffline,
		DocumentType:   OrderDocumentTypeReceipt,
		TenantID:       "tenant-1",
		TenantName:     "Test Store",
		OrderID:        "order-1",
		OrderReference: "GO-0001",
		Status:         models.OrderStatusPaid,
		CustomerName:   "Customer",
		CustomerPhone:  "+628123456789",
		CustomerEmail:  "customer@example.com",
		DeliveryType:   "pickup",
		Items: []OrderDocumentItem{
			{
				ProductID:   "product-1",
				ProductName: "Coffee",
				Quantity:    2,
				UnitPrice:   15000,
				TotalPrice:  30000,
			},
		},
		SubtotalAmount: 30000,
		TotalAmount:    30000,
		CreatedAt:      paidAt.Add(-time.Hour),
		PaidAt:         &paidAt,
		PaymentMethod:  "cash",
		PaymentRecords: []OrderDocumentPaymentRecord{
			{
				PaymentNumber:         1,
				AmountPaid:            30000,
				PaymentDate:           paidAt,
				PaymentMethod:         "cash",
				RemainingBalanceAfter: 0,
			},
		},
	}

	content, err := renderOrderDocumentPDF(snapshot)
	if err != nil {
		t.Fatalf("renderOrderDocumentPDF() error = %v", err)
	}
	if len(content) == 0 {
		t.Fatal("renderOrderDocumentPDF() returned empty content")
	}
	if !strings.HasPrefix(string(content), "%PDF") {
		t.Fatalf("renderOrderDocumentPDF() content does not look like a PDF")
	}
}

func TestOrderDocumentFilenameSanitizesReference(t *testing.T) {
	filename := orderDocumentFilename(&OrderDocumentSnapshot{
		DocumentType:   OrderDocumentTypeInvoice,
		OrderReference: "GO/2026 0001",
	})

	if filename != "invoice-GO-2026-0001.pdf" {
		t.Fatalf("filename = %q, want invoice-GO-2026-0001.pdf", filename)
	}
}

func TestBatchOrderDocumentsFilenameUsesOrdersPrefixForAutoSource(t *testing.T) {
	filename := batchOrderDocumentsFilename(
		OrderDocumentSourceAny,
		OrderDocumentTypeReceipt,
		time.Date(2026, 5, 20, 12, 30, 45, 0, time.FixedZone("WIB", 7*60*60)),
	)

	if filename != "orders-receipt-documents-20260520-053045.zip" {
		t.Fatalf("filename = %q, want orders-receipt-documents-20260520-053045.zip", filename)
	}
}

func TestNormalizePlainEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		want    string
		wantErr bool
	}{
		{name: "empty", email: "", want: ""},
		{name: "plain email", email: "CUSTOMER@example.com", want: "customer@example.com"},
		{name: "display name rejected", email: "Customer <customer@example.com>", wantErr: true},
		{name: "invalid email rejected", email: "invalid-email", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizePlainEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizePlainEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("normalizePlainEmail() = %q, want %q", got, tt.want)
			}
		})
	}
}
