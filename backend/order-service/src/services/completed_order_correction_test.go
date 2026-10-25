package services

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/repository"
	"github.com/point-of-sale-system/order-service/src/utils"
)

// prefixEncryptor marks values as encrypted so tests can tell plaintext from ciphertext
type prefixEncryptor struct{}

func (prefixEncryptor) Encrypt(_ context.Context, plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}
func (prefixEncryptor) EncryptWithContext(_ context.Context, plaintext, _ string) (string, error) {
	return "enc:" + plaintext, nil
}
func (prefixEncryptor) Decrypt(_ context.Context, ciphertext string) (string, error) {
	return strings.TrimPrefix(ciphertext, "enc:"), nil
}
func (prefixEncryptor) DecryptWithContext(_ context.Context, ciphertext, _ string) (string, error) {
	return strings.TrimPrefix(ciphertext, "enc:"), nil
}
func (prefixEncryptor) EncryptBatch(_ context.Context, plaintexts []string) ([]string, error) {
	return plaintexts, nil
}
func (prefixEncryptor) DecryptBatch(_ context.Context, ciphertexts []string) ([]string, error) {
	return ciphertexts, nil
}

type recordingAuditPublisher struct {
	events []*utils.AuditEvent
}

func (p *recordingAuditPublisher) Publish(_ context.Context, event *utils.AuditEvent) error {
	p.events = append(p.events, event)
	return nil
}
func (p *recordingAuditPublisher) PublishBatch(_ context.Context, events []*utils.AuditEvent) error {
	p.events = append(p.events, events...)
	return nil
}
func (p *recordingAuditPublisher) Close() error { return nil }

const (
	testOrderID  = "11111111-1111-1111-1111-111111111111"
	testTenantID = "22222222-2222-2222-2222-222222222222"
	testOwnerID  = "33333333-3333-3333-3333-333333333333"
	testProduct  = "44444444-4444-4444-4444-444444444444"
)

func newCorrectionTestService(t *testing.T) (*OfflineOrderService, sqlmock.Sqlmock, *recordingAuditPublisher) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	encryptor := prefixEncryptor{}
	orderRepo := repository.NewOrderRepository(db, encryptor)
	audit := &recordingAuditPublisher{}

	service := NewOfflineOrderService(
		db,
		repository.NewOfflineOrderRepository(db, encryptor, nil),
		orderRepo,
		repository.NewPaymentRepository(db),
		repository.NewOutboxRepository(db),
		NewEventPublisher(db, EventPublisherConfig{KafkaBrokers: []string{"localhost:9092"}}),
		NewPaymentCalculator(),
		NewInventoryService(db, nil),
		nil,
		audit,
	)

	return service, mock, audit
}

func correction(reason string, acknowledged bool) CompletedOrderCorrection {
	return CompletedOrderCorrection{
		OrderID:                     testOrderID,
		TenantID:                    testTenantID,
		Actor:                       OrderCorrectionActor{UserID: testOwnerID, Role: "owner"},
		Reason:                      reason,
		AcknowledgeAccountingImpact: acknowledged,
	}
}

func expectLock(mock sqlmock.Sqlmock, status models.OrderStatus) {
	mock.ExpectQuery(`SELECT status\s+FROM guest_orders.*deleted_at IS NULL\s+FOR UPDATE`).
		WithArgs(testOrderID, testTenantID).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(status))
}

func expectLoadOrder(mock sqlmock.Sqlmock, subtotal, deliveryFee int, items [][]driver.Value) {
	now := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`FROM guest_orders\s+WHERE id = \$1 AND tenant_id = \$2 AND order_type = 'offline'`).
		WithArgs(testOrderID, testTenantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "order_reference", "status", "order_type",
			"delivery_type", "customer_name", "customer_phone", "customer_email",
			"table_number", "notes",
			"subtotal_amount", "delivery_fee", "total_amount",
			"data_consent_given", "consent_method", "recorded_by_user_id",
			"last_modified_by_user_id", "last_modified_at",
			"created_at", "paid_at", "completed_at", "cancelled_at",
		}).AddRow(
			testOrderID, testTenantID, "GO-000001", "COMPLETE", "offline",
			"pickup", "enc:Budi", "enc:081234567890", "",
			nil, nil,
			subtotal, deliveryFee, subtotal+deliveryFee,
			true, "verbal", testOwnerID,
			nil, nil,
			now, now, now, nil,
		))

	rows := sqlmock.NewRows([]string{"id", "order_id", "product_id", "product_name", "unit_price", "quantity", "total_price"})
	for _, item := range items {
		rows.AddRow(item...)
	}
	mock.ExpectQuery(`FROM order_items\s+WHERE order_id = \$1`).WithArgs(testOrderID).WillReturnRows(rows)
}

func expectPayments(mock sqlmock.Sqlmock, amount int) {
	mock.ExpectQuery(`FROM payment_records`).WithArgs(testOrderID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_id", "payment_terms_id", "payment_number",
			"amount_paid", "payment_date", "payment_method",
			"remaining_balance_after", "recorded_by_user_id",
			"notes", "receipt_number", "created_at",
		}).AddRow("pay-1", testOrderID, nil, 0, amount, time.Now(), "cash", 0, testOwnerID, nil, nil, time.Now()))
}

func TestCompletedOrderCorrectionRequiresAcknowledgementAndReason(t *testing.T) {
	service, mock, audit := newCorrectionTestService(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		correction CompletedOrderCorrection
		wantErr    error
	}{
		{name: "not acknowledged", correction: correction("Salah input harga", false), wantErr: ErrAccountingImpactNotAcknowledged},
		{name: "reason too short", correction: correction(" abc ", true), wantErr: ErrInvalidCorrection},
		{name: "reason too long", correction: correction(strings.Repeat("a", 501), true), wantErr: ErrInvalidCorrection},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteCompletedOfflineOrder(ctx, &CompletedOrderDeleteRequest{CompletedOrderCorrection: tt.correction})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("delete error = %v, want %v", err, tt.wantErr)
			}

			name := "Budi Santoso"
			_, err = service.UpdateCompletedOfflineOrder(ctx, &CompletedOrderEditRequest{
				CompletedOrderCorrection: tt.correction,
				Updates:                  models.UpdateOfflineOrderRequest{CustomerName: &name},
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("update error = %v, want %v", err, tt.wantErr)
			}
		})
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("validation must fail before touching the database: %v", err)
	}
	if len(audit.events) != 0 {
		t.Fatalf("audit events = %d, want 0", len(audit.events))
	}
}

func TestDeleteCompletedOfflineOrderRejectsOtherStatuses(t *testing.T) {
	service, mock, audit := newCorrectionTestService(t)

	mock.ExpectBegin()
	expectLock(mock, models.OrderStatusPaid)
	mock.ExpectRollback()

	err := service.DeleteCompletedOfflineOrder(context.Background(), &CompletedOrderDeleteRequest{
		CompletedOrderCorrection: correction("Pesanan dobel", true),
	})
	if !errors.Is(err, ErrOrderNotComplete) {
		t.Fatalf("error = %v, want ErrOrderNotComplete", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	if len(audit.events) != 0 {
		t.Fatalf("audit events = %d, want 0", len(audit.events))
	}
}

func TestDeleteCompletedOfflineOrderMissingOrder(t *testing.T) {
	service, mock, _ := newCorrectionTestService(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT status\s+FROM guest_orders`).
		WithArgs(testOrderID, testTenantID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	err := service.DeleteCompletedOfflineOrder(context.Background(), &CompletedOrderDeleteRequest{
		CompletedOrderCorrection: correction("Pesanan dobel", true),
	})
	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("error = %v, want ErrOrderNotFound", err)
	}
}

func TestDeleteCompletedOfflineOrderReturnsStockAndAudits(t *testing.T) {
	service, mock, audit := newCorrectionTestService(t)

	mock.ExpectBegin()
	expectLock(mock, models.OrderStatusComplete)
	expectLoadOrder(mock, 30000, 0, [][]driver.Value{
		{"item-1", testOrderID, testProduct, "Kopi", 15000, 2, 30000},
	})
	mock.ExpectQuery(`SELECT product_id, SUM\(quantity\)\s+FROM inventory_reservations\s+WHERE order_id = \$1 AND status = 'converted'`).
		WithArgs(testOrderID).
		WillReturnRows(sqlmock.NewRows([]string{"product_id", "sum"}).AddRow(testProduct, 2))
	mock.ExpectExec(`UPDATE products\s+SET stock_quantity = stock_quantity \+ \$1`).
		WithArgs(2, testProduct, testTenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE inventory_reservations\s+SET status = 'released', released_at = NOW\(\)\s+WHERE order_id = \$1 AND status = 'converted'`).
		WithArgs(testOrderID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE inventory_reservations\s+SET status = 'released', released_at = NOW\(\)\s+WHERE order_id = \$1 AND status = 'active'`).
		WithArgs(testOrderID).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`UPDATE guest_orders\s+SET\s+deleted_at = NOW\(\).*status = 'CANCELLED'.*AND status = 'COMPLETE'`).
		WithArgs(testOwnerID, testOrderID, testTenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO event_outbox`).
		WithArgs("offline_order.completed_deleted", testOrderID, sqlmock.AnyArg(), "order-events", sqlmock.AnyArg(), 0).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("event-1"))
	mock.ExpectCommit()
	expectPayments(mock, 30000)

	err := service.DeleteCompletedOfflineOrder(context.Background(), &CompletedOrderDeleteRequest{
		CompletedOrderCorrection: correction("  Pesanan dobel tercatat  ", true),
	})
	if err != nil {
		t.Fatalf("DeleteCompletedOfflineOrder() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	if len(audit.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(audit.events))
	}
	event := audit.events[0]
	if event.Action != "DELETE" || event.ResourceType != "order" || event.ResourceID != testOrderID {
		t.Fatalf("audit event = %s %s %s, want DELETE order %s", event.Action, event.ResourceType, event.ResourceID, testOrderID)
	}
	if event.ActorID == nil || *event.ActorID != testOwnerID || event.TenantID != testTenantID {
		t.Fatalf("audit actor/tenant = %v/%s", event.ActorID, event.TenantID)
	}
	if event.Metadata["reason"] != "Pesanan dobel tercatat" {
		t.Fatalf("reason = %v, want trimmed reason", event.Metadata["reason"])
	}
	if event.Metadata["removed_revenue"] != 30000 || event.Metadata["total_paid"] != 30000 {
		t.Fatalf("metadata = %+v", event.Metadata)
	}
	if event.Metadata["accounting_impact_acknowledged"] != true {
		t.Fatalf("acknowledgement not recorded: %+v", event.Metadata)
	}
	if event.BeforeValue["customer_name"] != "enc:Budi" {
		t.Fatalf("customer_name in audit = %v, want encrypted value", event.BeforeValue["customer_name"])
	}
	if event.AfterValue != nil {
		t.Fatalf("after value = %+v, want nil for a deletion", event.AfterValue)
	}
}

func TestUpdateCompletedOfflineOrderRejectsNoChanges(t *testing.T) {
	service, mock, audit := newCorrectionTestService(t)

	mock.ExpectBegin()
	expectLock(mock, models.OrderStatusComplete)
	expectLoadOrder(mock, 30000, 0, [][]driver.Value{
		{"item-1", testOrderID, testProduct, "Kopi", 15000, 2, 30000},
	})
	mock.ExpectRollback()

	sameName := "Budi"
	_, err := service.UpdateCompletedOfflineOrder(context.Background(), &CompletedOrderEditRequest{
		CompletedOrderCorrection: correction("Koreksi data", true),
		Updates: models.UpdateOfflineOrderRequest{
			CustomerName: &sameName,
			Items:        []models.OrderItemInput{{ProductID: testProduct, ProductName: "Kopi", Quantity: 2, UnitPrice: 15000}},
		},
	})
	if !errors.Is(err, ErrInvalidCorrection) {
		t.Fatalf("error = %v, want ErrInvalidCorrection", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	if len(audit.events) != 0 {
		t.Fatalf("audit events = %d, want 0", len(audit.events))
	}
}

func TestUpdateCompletedOfflineOrderRejectsInvalidItems(t *testing.T) {
	service, mock, _ := newCorrectionTestService(t)

	_, err := service.UpdateCompletedOfflineOrder(context.Background(), &CompletedOrderEditRequest{
		CompletedOrderCorrection: correction("Koreksi data", true),
		Updates: models.UpdateOfflineOrderRequest{
			Items: []models.OrderItemInput{{ProductID: testProduct, ProductName: "Kopi", Quantity: 0, UnitPrice: 15000}},
		},
	})
	if !errors.Is(err, ErrInvalidCorrection) {
		t.Fatalf("error = %v, want ErrInvalidCorrection", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateCompletedOfflineOrderSwapsStockAndAuditsBalance(t *testing.T) {
	service, mock, audit := newCorrectionTestService(t)

	oldItems := [][]driver.Value{{"item-1", testOrderID, testProduct, "Kopi", 15000, 2, 30000}}
	newItems := [][]driver.Value{{"item-2", testOrderID, testProduct, "Kopi", 15000, 3, 45000}}

	mock.ExpectBegin()
	expectLock(mock, models.OrderStatusComplete)
	expectLoadOrder(mock, 30000, 0, oldItems)

	// Header update stamps the modifying owner
	mock.ExpectExec(`UPDATE guest_orders SET last_modified_by_user_id = \$1, last_modified_at = \$2 WHERE id = \$3 AND tenant_id = \$4 AND order_type = \$5`).
		WithArgs(testOwnerID, sqlmock.AnyArg(), testOrderID, testTenantID, models.OrderTypeOffline).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Old stock goes back
	mock.ExpectQuery(`SELECT product_id, SUM\(quantity\)`).WithArgs(testOrderID).
		WillReturnRows(sqlmock.NewRows([]string{"product_id", "sum"}).AddRow(testProduct, 2))
	mock.ExpectExec(`UPDATE products\s+SET stock_quantity = stock_quantity \+ \$1`).
		WithArgs(2, testProduct, testTenantID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`WHERE order_id = \$1 AND status = 'converted'`).WithArgs(testOrderID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`WHERE order_id = \$1 AND status = 'active'`).WithArgs(testOrderID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Items are replaced
	mock.ExpectExec(`DELETE FROM order_items`).WithArgs(testOrderID, testTenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO order_items`).
		WithArgs(testOrderID, testProduct, "Kopi", 3, 15000, 45000).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT delivery_fee FROM guest_orders`).WithArgs(testOrderID, testTenantID).
		WillReturnRows(sqlmock.NewRows([]string{"delivery_fee"}).AddRow(0))
	mock.ExpectExec(`UPDATE guest_orders\s+SET subtotal_amount = \$1, total_amount = \$2`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// New stock is taken permanently
	mock.ExpectQuery(`SELECT stock_quantity\s+FROM products\s+WHERE tenant_id = \$1 AND id = \$2\s+FOR UPDATE`).
		WithArgs(testTenantID, testProduct).
		WillReturnRows(sqlmock.NewRows([]string{"stock_quantity"}).AddRow(10))
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(quantity\), 0\)\s+FROM inventory_reservations`).WithArgs(testProduct).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(0))
	mock.ExpectQuery(`INSERT INTO inventory_reservations`).
		WithArgs(testOrderID, testProduct, 3, models.ReservationStatusActive, sqlmock.AnyArg(), nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("res-2", time.Now()))
	mock.ExpectQuery(`WHERE order_id = \$1 AND status = 'active'\s+ORDER BY created_at ASC\s+FOR UPDATE`).WithArgs(testOrderID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "order_id", "product_id", "quantity", "status", "created_at", "expires_at", "released_at"}).
			AddRow("res-2", testOrderID, testProduct, 3, "active", time.Now(), time.Now().Add(time.Hour), nil))
	mock.ExpectExec(`SET status = 'converted'`).WithArgs("res-2").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE products\s+SET stock_quantity = stock_quantity - \$1`).WithArgs(3, testProduct).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(regexp.QuoteMeta(`RETURNING g.subtotal_amount, g.total_amount`)).WithArgs(testOrderID, testTenantID).
		WillReturnRows(sqlmock.NewRows([]string{"subtotal_amount", "total_amount"}).AddRow(45000, 45000))
	mock.ExpectQuery(`INSERT INTO event_outbox`).
		WithArgs("offline_order.completed_updated", testOrderID, sqlmock.AnyArg(), "order-events", sqlmock.AnyArg(), 0).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("event-1"))
	mock.ExpectCommit()

	expectLoadOrder(mock, 45000, 0, newItems)
	expectPayments(mock, 30000)

	order, err := service.UpdateCompletedOfflineOrder(context.Background(), &CompletedOrderEditRequest{
		CompletedOrderCorrection: correction("Jumlah kopi salah input", true),
		Updates: models.UpdateOfflineOrderRequest{
			Items: []models.OrderItemInput{{ProductID: testProduct, ProductName: "Kopi", Quantity: 3, UnitPrice: 15000}},
		},
	})
	if err != nil {
		t.Fatalf("UpdateCompletedOfflineOrder() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	if order.TotalAmount != 45000 {
		t.Fatalf("total = %d, want 45000", order.TotalAmount)
	}

	if len(audit.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(audit.events))
	}
	event := audit.events[0]
	if event.Action != "UPDATE" {
		t.Fatalf("action = %s, want UPDATE", event.Action)
	}
	if event.Metadata["outstanding_balance"] != 15000 {
		t.Fatalf("outstanding_balance = %v, want 15000", event.Metadata["outstanding_balance"])
	}
	if event.Metadata["stock_adjusted"] != true {
		t.Fatalf("stock_adjusted = %v, want true", event.Metadata["stock_adjusted"])
	}
	fields, _ := event.Metadata["changed_fields"].([]string)
	if strings.Join(fields, ",") != "items,subtotal_amount,total_amount" {
		t.Fatalf("changed_fields = %v", fields)
	}
	if event.BeforeValue["total_amount"] != 30000 || event.AfterValue["total_amount"] != 45000 {
		t.Fatalf("before/after totals = %v/%v", event.BeforeValue["total_amount"], event.AfterValue["total_amount"])
	}
}
