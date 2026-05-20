package services

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/phpdave11/gofpdf"
	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/queue"
)

type OrderDocumentSource string

const (
	OrderDocumentSourceAny     OrderDocumentSource = "any"
	OrderDocumentSourceGuest   OrderDocumentSource = "guest"
	OrderDocumentSourceOffline OrderDocumentSource = "offline"
)

type OrderDocumentType string

const (
	OrderDocumentTypeInvoice OrderDocumentType = "invoice"
	OrderDocumentTypeReceipt OrderDocumentType = "receipt"
)

const (
	OrderDocumentErrorInvalidType   = "invalid_document_type"
	OrderDocumentErrorNotFound      = "order_not_found"
	OrderDocumentErrorForbidden     = "forbidden"
	OrderDocumentErrorIneligible    = "document_ineligible"
	OrderDocumentErrorMissingEmail  = "missing_customer_email"
	OrderDocumentErrorInvalidEmail  = "invalid_customer_email"
	OrderDocumentErrorInvalidBatch  = "invalid_batch"
	OrderDocumentMaxBatchOrderCount = 100
)

type OrderDocumentError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	OrderID string `json:"order_id,omitempty"`
}

func (e *OrderDocumentError) Error() string {
	if e.OrderID == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.OrderID, e.Message)
}

type BatchOrderDocumentError struct {
	Errors []OrderDocumentError `json:"errors"`
}

func (e *BatchOrderDocumentError) Error() string {
	return "one or more selected orders cannot generate the requested document"
}

type OrderDocumentItem struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int    `json:"unit_price"`
	TotalPrice  int    `json:"total_price"`
}

type OrderDocumentPaymentRecord struct {
	PaymentNumber         int       `json:"payment_number"`
	AmountPaid            int       `json:"amount_paid"`
	PaymentDate           time.Time `json:"payment_date"`
	PaymentMethod         string    `json:"payment_method"`
	RemainingBalanceAfter int       `json:"remaining_balance_after"`
	ReceiptNumber         string    `json:"receipt_number,omitempty"`
}

type OrderDocumentSnapshot struct {
	Source               OrderDocumentSource          `json:"source"`
	DocumentType         OrderDocumentType            `json:"document_type"`
	TenantID             string                       `json:"tenant_id"`
	TenantName           string                       `json:"tenant_name"`
	OrderID              string                       `json:"order_id"`
	OrderReference       string                       `json:"order_reference"`
	Status               models.OrderStatus           `json:"status"`
	CustomerName         string                       `json:"customer_name"`
	CustomerPhone        string                       `json:"customer_phone"`
	CustomerEmail        string                       `json:"customer_email,omitempty"`
	DeliveryType         string                       `json:"delivery_type"`
	TableNumber          string                       `json:"table_number,omitempty"`
	Notes                string                       `json:"notes,omitempty"`
	Items                []OrderDocumentItem          `json:"items"`
	SubtotalAmount       int                          `json:"subtotal_amount"`
	DeliveryFee          int                          `json:"delivery_fee"`
	TotalAmount          int                          `json:"total_amount"`
	CreatedAt            time.Time                    `json:"created_at"`
	PaidAt               *time.Time                   `json:"paid_at,omitempty"`
	PaymentMethod        string                       `json:"payment_method,omitempty"`
	PaymentTransactionID string                       `json:"payment_transaction_id,omitempty"`
	PaymentRecords       []OrderDocumentPaymentRecord `json:"payment_records,omitempty"`
}

type GeneratedOrderDocument struct {
	Filename string
	Content  []byte
}

type OrderDocumentService struct {
	db                  *sql.DB
	orderService        *OrderService
	offlineOrderService *OfflineOrderService
	paymentRepo         interface {
		GetPaymentByOrderID(ctx context.Context, orderID string) (*models.PaymentTransaction, error)
		GetPaymentHistory(ctx context.Context, orderID string) ([]models.PaymentRecord, error)
	}
	notificationProducer *queue.KafkaProducer
}

func NewOrderDocumentService(
	db *sql.DB,
	orderService *OrderService,
	offlineOrderService *OfflineOrderService,
	paymentRepo interface {
		GetPaymentByOrderID(ctx context.Context, orderID string) (*models.PaymentTransaction, error)
		GetPaymentHistory(ctx context.Context, orderID string) ([]models.PaymentRecord, error)
	},
	notificationProducer *queue.KafkaProducer,
) *OrderDocumentService {
	return &OrderDocumentService{
		db:                   db,
		orderService:         orderService,
		offlineOrderService:  offlineOrderService,
		paymentRepo:          paymentRepo,
		notificationProducer: notificationProducer,
	}
}

func NormalizeOrderDocumentType(documentType string) (OrderDocumentType, error) {
	switch OrderDocumentType(strings.ToLower(strings.TrimSpace(documentType))) {
	case OrderDocumentTypeInvoice:
		return OrderDocumentTypeInvoice, nil
	case OrderDocumentTypeReceipt:
		return OrderDocumentTypeReceipt, nil
	default:
		return "", &OrderDocumentError{
			Code:    OrderDocumentErrorInvalidType,
			Message: "document_type must be invoice or receipt",
		}
	}
}

func ValidateOrderDocumentEligibility(orderStatus models.OrderStatus, documentType OrderDocumentType) error {
	switch documentType {
	case OrderDocumentTypeInvoice:
		if orderStatus == models.OrderStatusCancelled {
			return &OrderDocumentError{
				Code:    OrderDocumentErrorIneligible,
				Message: "invoice is not available for cancelled orders",
			}
		}
	case OrderDocumentTypeReceipt:
		if orderStatus != models.OrderStatusPaid && orderStatus != models.OrderStatusComplete {
			return &OrderDocumentError{
				Code:    OrderDocumentErrorIneligible,
				Message: "receipt is only available for paid or complete orders",
			}
		}
	default:
		return &OrderDocumentError{
			Code:    OrderDocumentErrorInvalidType,
			Message: "document_type must be invoice or receipt",
		}
	}

	return nil
}

func (s *OrderDocumentService) GeneratePDF(ctx context.Context, source OrderDocumentSource, tenantID, orderID, rawDocumentType string) (*GeneratedOrderDocument, error) {
	snapshot, err := s.BuildSnapshot(ctx, source, tenantID, orderID, rawDocumentType)
	if err != nil {
		return nil, err
	}

	content, err := renderOrderDocumentPDF(snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to render %s PDF: %w", snapshot.DocumentType, err)
	}

	return &GeneratedOrderDocument{
		Filename: orderDocumentFilename(snapshot),
		Content:  content,
	}, nil
}

func (s *OrderDocumentService) GenerateBatchZIP(ctx context.Context, source OrderDocumentSource, tenantID string, orderIDs []string, rawDocumentType string) (*GeneratedOrderDocument, error) {
	documentType, err := NormalizeOrderDocumentType(rawDocumentType)
	if err != nil {
		return nil, err
	}

	if len(orderIDs) == 0 {
		return nil, &OrderDocumentError{
			Code:    OrderDocumentErrorInvalidBatch,
			Message: "order_ids is required",
		}
	}
	if len(orderIDs) > OrderDocumentMaxBatchOrderCount {
		return nil, &OrderDocumentError{
			Code:    OrderDocumentErrorInvalidBatch,
			Message: fmt.Sprintf("batch download supports at most %d orders", OrderDocumentMaxBatchOrderCount),
		}
	}

	documents := make([]*GeneratedOrderDocument, 0, len(orderIDs))
	batchErrors := make([]OrderDocumentError, 0)

	for _, orderID := range orderIDs {
		orderID = strings.TrimSpace(orderID)
		if orderID == "" {
			batchErrors = append(batchErrors, OrderDocumentError{
				Code:    OrderDocumentErrorInvalidBatch,
				Message: "order_id cannot be empty",
			})
			continue
		}

		document, err := s.GeneratePDF(ctx, source, tenantID, orderID, string(documentType))
		if err != nil {
			var docErr *OrderDocumentError
			if asOrderDocumentError(err, &docErr) {
				docErr.OrderID = orderID
				batchErrors = append(batchErrors, *docErr)
				continue
			}
			return nil, err
		}
		documents = append(documents, document)
	}

	if len(batchErrors) > 0 {
		return nil, &BatchOrderDocumentError{Errors: batchErrors}
	}

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	for _, document := range documents {
		fileWriter, err := zipWriter.Create(document.Filename)
		if err != nil {
			zipWriter.Close()
			return nil, fmt.Errorf("failed to add %s to ZIP: %w", document.Filename, err)
		}
		if _, err := fileWriter.Write(document.Content); err != nil {
			zipWriter.Close()
			return nil, fmt.Errorf("failed to write %s to ZIP: %w", document.Filename, err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize ZIP: %w", err)
	}

	return &GeneratedOrderDocument{
		Filename: batchOrderDocumentsFilename(source, documentType, time.Now().UTC()),
		Content:  buf.Bytes(),
	}, nil
}

func (s *OrderDocumentService) Resend(ctx context.Context, source OrderDocumentSource, tenantID, orderID, rawDocumentType, userID string) error {
	snapshot, err := s.BuildSnapshot(ctx, source, tenantID, orderID, rawDocumentType)
	if err != nil {
		return err
	}

	normalizedEmail, err := normalizePlainEmail(snapshot.CustomerEmail)
	if err != nil {
		return &OrderDocumentError{
			Code:    OrderDocumentErrorInvalidEmail,
			Message: "customer_email is invalid",
			OrderID: orderID,
		}
	}
	if normalizedEmail == "" {
		return &OrderDocumentError{
			Code:    OrderDocumentErrorMissingEmail,
			Message: "customer_email is required to resend documents",
			OrderID: orderID,
		}
	}
	snapshot.CustomerEmail = normalizedEmail

	if s.notificationProducer == nil {
		return fmt.Errorf("notification producer is not configured")
	}

	event := map[string]interface{}{
		"event_id":   fmt.Sprintf("order-document-resend-%s-%s-%d", snapshot.DocumentType, snapshot.OrderID, time.Now().UnixNano()),
		"event_type": "order.document.resend",
		"tenant_id":  tenantID,
		"user_id":    userID,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"data":       snapshot.toEventData(),
	}

	return s.notificationProducer.Publish(ctx, snapshot.OrderReference, event)
}

func (s *OrderDocumentService) BuildSnapshot(ctx context.Context, source OrderDocumentSource, tenantID, orderID, rawDocumentType string) (*OrderDocumentSnapshot, error) {
	documentType, err := NormalizeOrderDocumentType(rawDocumentType)
	if err != nil {
		return nil, err
	}

	actualSource := source
	var order *models.GuestOrder
	switch source {
	case OrderDocumentSourceAny:
		orderType, err := s.getOrderType(ctx, orderID)
		if err != nil {
			return nil, err
		}
		if orderType == models.OrderTypeOffline {
			actualSource = OrderDocumentSourceOffline
			order, err = s.offlineOrderService.GetOfflineOrderByID(ctx, orderID, tenantID)
		} else {
			actualSource = OrderDocumentSourceGuest
			order, err = s.orderService.GetOrderByID(ctx, orderID)
		}
	case OrderDocumentSourceGuest:
		orderType, err := s.getOrderType(ctx, orderID)
		if err != nil {
			return nil, err
		}
		if orderType == models.OrderTypeOffline {
			return nil, &OrderDocumentError{
				Code:    OrderDocumentErrorNotFound,
				Message: "order not found",
				OrderID: orderID,
			}
		}
		order, err = s.orderService.GetOrderByID(ctx, orderID)
	case OrderDocumentSourceOffline:
		order, err = s.offlineOrderService.GetOfflineOrderByID(ctx, orderID, tenantID)
	default:
		return nil, &OrderDocumentError{
			Code:    OrderDocumentErrorInvalidBatch,
			Message: "unknown document source",
			OrderID: orderID,
		}
	}
	if err != nil {
		if err == sql.ErrNoRows || strings.Contains(err.Error(), "not found") {
			return nil, &OrderDocumentError{
				Code:    OrderDocumentErrorNotFound,
				Message: "order not found",
				OrderID: orderID,
			}
		}
		return nil, err
	}
	if order == nil {
		return nil, &OrderDocumentError{
			Code:    OrderDocumentErrorNotFound,
			Message: "order not found",
			OrderID: orderID,
		}
	}
	if order.TenantID != tenantID {
		return nil, &OrderDocumentError{
			Code:    OrderDocumentErrorForbidden,
			Message: "order belongs to a different tenant",
			OrderID: orderID,
		}
	}

	if err := ValidateOrderDocumentEligibility(order.Status, documentType); err != nil {
		var docErr *OrderDocumentError
		if asOrderDocumentError(err, &docErr) {
			docErr.OrderID = orderID
		}
		return nil, err
	}

	items, err := s.orderService.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order items: %w", err)
	}

	tenantName, err := s.getTenantName(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tenant name: %w", err)
	}

	snapshot := &OrderDocumentSnapshot{
		Source:         actualSource,
		DocumentType:   documentType,
		TenantID:       tenantID,
		TenantName:     tenantName,
		OrderID:        order.ID,
		OrderReference: order.OrderReference,
		Status:         order.Status,
		CustomerName:   order.CustomerName,
		CustomerPhone:  order.CustomerPhone,
		DeliveryType:   string(order.DeliveryType),
		SubtotalAmount: order.SubtotalAmount,
		DeliveryFee:    order.DeliveryFee,
		TotalAmount:    order.TotalAmount,
		CreatedAt:      order.CreatedAt,
		PaidAt:         order.PaidAt,
	}
	if order.CustomerEmail != nil {
		snapshot.CustomerEmail = *order.CustomerEmail
	}
	if order.TableNumber != nil {
		snapshot.TableNumber = *order.TableNumber
	}
	if order.Notes != nil {
		snapshot.Notes = *order.Notes
	}

	snapshot.Items = make([]OrderDocumentItem, 0, len(items))
	for _, item := range items {
		snapshot.Items = append(snapshot.Items, OrderDocumentItem{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			TotalPrice:  item.TotalPrice,
		})
	}

	if snapshot.Source == OrderDocumentSourceOffline {
		paymentRecords, err := s.paymentRepo.GetPaymentHistory(ctx, orderID)
		if err != nil {
			return nil, fmt.Errorf("failed to load payment history: %w", err)
		}
		for _, record := range paymentRecords {
			entry := OrderDocumentPaymentRecord{
				PaymentNumber:         record.PaymentNumber,
				AmountPaid:            record.AmountPaid,
				PaymentDate:           record.PaymentDate,
				PaymentMethod:         string(record.PaymentMethod),
				RemainingBalanceAfter: record.RemainingBalanceAfter,
			}
			if record.ReceiptNumber != nil {
				entry.ReceiptNumber = *record.ReceiptNumber
			}
			snapshot.PaymentRecords = append(snapshot.PaymentRecords, entry)
		}
		if len(snapshot.PaymentRecords) > 0 {
			snapshot.PaymentMethod = snapshot.PaymentRecords[0].PaymentMethod
		}
	} else {
		payment, err := s.paymentRepo.GetPaymentByOrderID(ctx, orderID)
		if err != nil {
			return nil, fmt.Errorf("failed to load payment transaction: %w", err)
		}
		if payment != nil {
			if payment.PaymentType != nil {
				snapshot.PaymentMethod = *payment.PaymentType
			}
			if payment.MidtransTransactionID != nil {
				snapshot.PaymentTransactionID = *payment.MidtransTransactionID
			}
		}
	}

	return snapshot, nil
}

func (s *OrderDocumentService) getOrderType(ctx context.Context, orderID string) (models.OrderType, error) {
	var orderType string
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(order_type, 'online') FROM guest_orders WHERE id = $1`, orderID).Scan(&orderType)
	if err == sql.ErrNoRows {
		return "", &OrderDocumentError{
			Code:    OrderDocumentErrorNotFound,
			Message: "order not found",
			OrderID: orderID,
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to load order type: %w", err)
	}
	return models.OrderType(orderType), nil
}

func (s *OrderDocumentService) getTenantName(ctx context.Context, tenantID string) (string, error) {
	var tenantName string
	err := s.db.QueryRowContext(ctx, `SELECT business_name FROM tenants WHERE id = $1`, tenantID).Scan(&tenantName)
	if err != nil {
		return "", err
	}
	return tenantName, nil
}

func (snapshot *OrderDocumentSnapshot) toEventData() map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(snapshot.Items))
	for _, item := range snapshot.Items {
		items = append(items, map[string]interface{}{
			"product_id":   item.ProductID,
			"product_name": item.ProductName,
			"quantity":     item.Quantity,
			"unit_price":   item.UnitPrice,
			"total_price":  item.TotalPrice,
		})
	}

	paymentRecords := make([]map[string]interface{}, 0, len(snapshot.PaymentRecords))
	for _, record := range snapshot.PaymentRecords {
		paymentRecords = append(paymentRecords, map[string]interface{}{
			"payment_number":          record.PaymentNumber,
			"amount_paid":             record.AmountPaid,
			"payment_date":            record.PaymentDate.Format(time.RFC3339),
			"payment_method":          record.PaymentMethod,
			"remaining_balance_after": record.RemainingBalanceAfter,
			"receipt_number":          record.ReceiptNumber,
		})
	}

	data := map[string]interface{}{
		"source":                 snapshot.Source,
		"document_type":          snapshot.DocumentType,
		"tenant_name":            snapshot.TenantName,
		"order_id":               snapshot.OrderID,
		"order_reference":        snapshot.OrderReference,
		"status":                 snapshot.Status,
		"customer_name":          snapshot.CustomerName,
		"customer_phone":         snapshot.CustomerPhone,
		"customer_email":         snapshot.CustomerEmail,
		"delivery_type":          snapshot.DeliveryType,
		"table_number":           snapshot.TableNumber,
		"notes":                  snapshot.Notes,
		"items":                  items,
		"subtotal_amount":        snapshot.SubtotalAmount,
		"delivery_fee":           snapshot.DeliveryFee,
		"total_amount":           snapshot.TotalAmount,
		"created_at":             snapshot.CreatedAt.Format(time.RFC3339),
		"payment_method":         snapshot.PaymentMethod,
		"payment_transaction_id": snapshot.PaymentTransactionID,
		"payment_records":        paymentRecords,
	}
	if snapshot.PaidAt != nil {
		data["paid_at"] = snapshot.PaidAt.Format(time.RFC3339)
	}

	return data
}

func renderOrderDocumentPDF(snapshot *OrderDocumentSnapshot) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(14, 14, 14)
	pdf.SetAutoPageBreak(true, 14)
	pdf.AddPage()

	title := "Invoice"
	if snapshot.DocumentType == OrderDocumentTypeReceipt {
		title = "Receipt"
	}

	pdf.SetFont("Helvetica", "B", 20)
	pdf.CellFormat(0, 10, strings.ToUpper(title), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(0, 7, snapshot.TenantName, "", 1, "L", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(95, 8, "Order", "1", 0, "L", true, 0, "")
	pdf.CellFormat(85, 8, "Customer", "1", 1, "L", true, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	addTwoColumnRow(pdf, "Reference", snapshot.OrderReference, "Name", snapshot.CustomerName)
	addTwoColumnRow(pdf, "Status", string(snapshot.Status), "Phone", snapshot.CustomerPhone)
	addTwoColumnRow(pdf, "Created", snapshot.CreatedAt.Format("02 Jan 2006 15:04"), "Email", emptyDash(snapshot.CustomerEmail))
	addTwoColumnRow(pdf, "Delivery", formatDeliveryType(snapshot.DeliveryType), "Table", emptyDash(snapshot.TableNumber))
	if snapshot.DocumentType == OrderDocumentTypeReceipt {
		paidAt := "-"
		if snapshot.PaidAt != nil {
			paidAt = snapshot.PaidAt.Format("02 Jan 2006 15:04")
		}
		addTwoColumnRow(pdf, "Paid At", paidAt, "Payment", emptyDash(snapshot.PaymentMethod))
	}

	pdf.Ln(6)
	if snapshot.DocumentType == OrderDocumentTypeReceipt {
		pdf.SetTextColor(16, 128, 80)
		pdf.SetFont("Helvetica", "B", 24)
		pdf.CellFormat(0, 12, "PAID", "1", 1, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(3)
	}

	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, "Items", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(82, 8, "Item", "1", 0, "L", true, 0, "")
	pdf.CellFormat(20, 8, "Qty", "1", 0, "R", true, 0, "")
	pdf.CellFormat(39, 8, "Unit Price", "1", 0, "R", true, 0, "")
	pdf.CellFormat(39, 8, "Total", "1", 1, "R", true, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	for _, item := range snapshot.Items {
		pdf.CellFormat(82, 7, truncatePDFText(item.ProductName, 46), "1", 0, "L", false, 0, "")
		pdf.CellFormat(20, 7, strconv.Itoa(item.Quantity), "1", 0, "R", false, 0, "")
		pdf.CellFormat(39, 7, formatIDR(item.UnitPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(39, 7, formatIDR(item.TotalPrice), "1", 1, "R", false, 0, "")
	}

	pdf.Ln(4)
	addSummaryRow(pdf, "Subtotal", snapshot.SubtotalAmount, false)
	if snapshot.DeliveryFee > 0 {
		addSummaryRow(pdf, "Delivery Fee", snapshot.DeliveryFee, false)
	}
	addSummaryRow(pdf, "Total", snapshot.TotalAmount, true)

	if snapshot.DocumentType == OrderDocumentTypeReceipt && len(snapshot.PaymentRecords) > 0 {
		pdf.Ln(6)
		pdf.SetFont("Helvetica", "B", 12)
		pdf.CellFormat(0, 8, "Payment History", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(24, 8, "No.", "1", 0, "R", true, 0, "")
		pdf.CellFormat(40, 8, "Date", "1", 0, "L", true, 0, "")
		pdf.CellFormat(38, 8, "Method", "1", 0, "L", true, 0, "")
		pdf.CellFormat(39, 8, "Paid", "1", 0, "R", true, 0, "")
		pdf.CellFormat(39, 8, "Balance", "1", 1, "R", true, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		for _, record := range snapshot.PaymentRecords {
			pdf.CellFormat(24, 7, strconv.Itoa(record.PaymentNumber), "1", 0, "R", false, 0, "")
			pdf.CellFormat(40, 7, record.PaymentDate.Format("02 Jan 2006"), "1", 0, "L", false, 0, "")
			pdf.CellFormat(38, 7, strings.ReplaceAll(record.PaymentMethod, "_", " "), "1", 0, "L", false, 0, "")
			pdf.CellFormat(39, 7, formatIDR(record.AmountPaid), "1", 0, "R", false, 0, "")
			pdf.CellFormat(39, 7, formatIDR(record.RemainingBalanceAfter), "1", 1, "R", false, 0, "")
		}
	}

	if snapshot.Notes != "" {
		pdf.Ln(6)
		pdf.SetFont("Helvetica", "B", 12)
		pdf.CellFormat(0, 8, "Notes", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.MultiCell(0, 6, snapshot.Notes, "1", "L", false)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func addTwoColumnRow(pdf *gofpdf.Fpdf, leftLabel, leftValue, rightLabel, rightValue string) {
	pdf.CellFormat(30, 7, leftLabel, "1", 0, "L", false, 0, "")
	pdf.CellFormat(65, 7, leftValue, "1", 0, "L", false, 0, "")
	pdf.CellFormat(30, 7, rightLabel, "1", 0, "L", false, 0, "")
	pdf.CellFormat(55, 7, rightValue, "1", 1, "L", false, 0, "")
}

func addSummaryRow(pdf *gofpdf.Fpdf, label string, amount int, total bool) {
	style := ""
	if total {
		style = "B"
	}
	pdf.SetFont("Helvetica", style, 10)
	pdf.CellFormat(141, 8, label, "", 0, "R", false, 0, "")
	pdf.CellFormat(39, 8, formatIDR(amount), "1", 1, "R", false, 0, "")
}

func orderDocumentFilename(snapshot *OrderDocumentSnapshot) string {
	return fmt.Sprintf("%s-%s.pdf", snapshot.DocumentType, sanitizeFilename(snapshot.OrderReference))
}

func batchOrderDocumentsFilename(source OrderDocumentSource, documentType OrderDocumentType, timestamp time.Time) string {
	zipSource := string(source)
	if source == OrderDocumentSourceAny {
		zipSource = "orders"
	}
	return fmt.Sprintf("%s-%s-documents-%s.zip", zipSource, documentType, timestamp.UTC().Format("20060102-150405"))
}

func sanitizeFilename(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	lastHyphen := false
	for _, r := range value {
		keep := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
		if keep {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "order"
	}
	return result
}

func normalizePlainEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return "", nil
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil {
		return "", err
	}
	if parsed.Name != "" || parsed.Address != email {
		return "", fmt.Errorf("email must be a plain address")
	}
	return strings.ToLower(parsed.Address), nil
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func formatDeliveryType(value string) string {
	return strings.Title(strings.ReplaceAll(value, "_", " "))
}

func formatIDR(amount int) string {
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}

	digits := strconv.Itoa(amount)
	parts := make([]string, 0, len(digits)/3+1)
	for len(digits) > 3 {
		parts = append([]string{digits[len(digits)-3:]}, parts...)
		digits = digits[:len(digits)-3]
	}
	parts = append([]string{digits}, parts...)
	return "Rp " + sign + strings.Join(parts, ".")
}

func truncatePDFText(value string, maxRunes int) string {
	if len([]rune(value)) <= maxRunes {
		return value
	}
	runes := []rune(value)
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}

func asOrderDocumentError(err error, target **OrderDocumentError) bool {
	if docErr, ok := err.(*OrderDocumentError); ok {
		*target = docErr
		return true
	}
	return false
}
