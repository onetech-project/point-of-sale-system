package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"

	"github.com/point-of-sale-system/order-service/src/manjo"
	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/repository"
)

type staticManjoGateway struct {
	creds  manjo.Credentials
	client *manjo.Client
}

func (g staticManjoGateway) Credentials(context.Context, string) (manjo.Credentials, *manjo.Client, error) {
	return g.creds, g.client, nil
}

// newManjoServer answers access token requests and hands every other request to handler.
func newManjoServer(t *testing.T, handler func(path string, body map[string]interface{}) (int, string)) staticManjoGateway {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if r.URL.Path == manjo.PathAccessToken {
			_, _ = io.WriteString(w, `{"responseCode":"2007300","accessToken":"tok","tokenType":"Bearer","expiresIn":"900"}`)
			return
		}
		var body map[string]interface{}
		_ = json.Unmarshal(raw, &body)
		status, response := handler(r.URL.Path, body)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(server.Close)

	return staticManjoGateway{
		creds:  manjo.Credentials{ClientID: "EP1", ClientSecret: "secret", PrivateKey: keyPEM, MerchantID: "MT1"},
		client: manjo.NewClient(server.URL, manjo.WithHTTPClient(server.Client())),
	}
}

const (
	manjoOrderID  = "11111111-1111-1111-1111-111111111111"
	manjoTenantID = "22222222-2222-2222-2222-222222222222"
	manjoPayID    = "55555555-5555-5555-5555-555555555555"
)

func newManjoTestService(t *testing.T, gateway manjoGateway) (*PaymentService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	orderRepo := repository.NewOrderRepository(db, prefixEncryptor{})
	paymentRepo := repository.NewPaymentRepository(db)
	return &PaymentService{
		db:           db,
		paymentRepo:  paymentRepo,
		refundRepo:   repository.NewPaymentRefundRepository(db),
		orderRepo:    orderRepo,
		orderService: &OrderService{db: db, orderRepo: orderRepo},
		manjo:        gateway,
		syncThrottle: newSyncThrottle(time.Minute),
	}, mock
}

var paymentColumns = []string{
	"id", "order_id", "midtrans_transaction_id", "midtrans_order_id",
	"amount", "payment_type", "transaction_status", "fraud_status",
	"notification_payload", "signature_key", "signature_verified",
	"qr_code_url", "qr_string", "expiry_time",
	"created_at", "notification_received_at", "settled_at", "idempotency_key",
	"payment_gateway", "gateway_reference_no", "gateway_external_id",
}

func expectPaymentByOrderID(mock sqlmock.Sqlmock, gateway, status string, expiry time.Time) {
	mock.ExpectQuery(`FROM payment_transactions\s+WHERE order_id = \$1`).
		WithArgs(manjoOrderID).
		WillReturnRows(sqlmock.NewRows(paymentColumns).AddRow(
			manjoPayID, manjoOrderID, "A0001", "GO-0001",
			50000, "qris", status, nil,
			[]byte(`{}`), nil, false,
			nil, "000201", expiry,
			time.Now(), nil, nil, "A0001:"+status,
			gateway, "A0001", "EXT-1",
		))
}

func manjoOrder(status models.OrderStatus) *models.GuestOrder {
	return &models.GuestOrder{
		ID:             manjoOrderID,
		TenantID:       manjoTenantID,
		OrderReference: "GO-0001",
		Status:         status,
		TotalAmount:    50000,
	}
}

func TestRefundOrderPaymentSuccess(t *testing.T) {
	var refundBody map[string]interface{}
	gateway := newManjoServer(t, func(path string, body map[string]interface{}) (int, string) {
		if path != manjo.PathRefund {
			t.Fatalf("unexpected path %s", path)
		}
		refundBody = body
		return http.StatusOK, `{"responseCode":"2007800","responseMessage":"Successful","refundNo":"RF-1"}`
	})
	svc, mock := newManjoTestService(t, gateway)

	expectPaymentByOrderID(mock, "manjo", manjo.StatusSettlement, time.Now())
	mock.ExpectQuery(`INSERT INTO payment_refunds`).
		WithArgs(manjoPayID, manjoOrderID, manjoTenantID, "manjo", sqlmock.AnyArg(), 50000, "Customer complain",
			models.RefundStatusProcessing, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("refund-1", time.Now(), time.Now()))
	mock.ExpectExec(`UPDATE payment_refunds`).
		WithArgs("refund-1", models.RefundStatusSuccess, "RF-1", nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE payment_transactions`).
		WithArgs(manjoPayID, manjo.StatusRefund, nil, sqlmock.AnyArg(), "A0001:refund", false).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO order_notes`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("note-1", time.Now()))

	refund, err := svc.RefundOrderPayment(context.Background(), manjoOrder(models.OrderStatusPaid), RefundRequest{
		Reason: "Customer complain", UserName: "Owner",
	})
	if err != nil {
		t.Fatalf("RefundOrderPayment returned error: %v", err)
	}
	if refund.Status != models.RefundStatusSuccess {
		t.Fatalf("status = %s, want success", refund.Status)
	}
	if refundBody["originalExternalId"] != "EXT-1" || refundBody["originalReferenceNo"] != "A0001" ||
		refundBody["originalPartnerReferenceNo"] != "GO-0001" {
		t.Fatalf("unexpected refund request: %v", refundBody)
	}
	amount := refundBody["refundAmount"].(map[string]interface{})
	if amount["value"] != "50000.00" {
		t.Fatalf("refund amount = %v", amount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestRefundOrderPaymentRejectedByManjoIsRecordedAsFailed(t *testing.T) {
	gateway := newManjoServer(t, func(string, map[string]interface{}) (int, string) {
		return http.StatusNotFound, `{"responseCode":"4047813","responseMessage":"Invalid Amount"}`
	})
	svc, mock := newManjoTestService(t, gateway)

	expectPaymentByOrderID(mock, "manjo", manjo.StatusSettlement, time.Now())
	mock.ExpectQuery(`INSERT INTO payment_refunds`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("refund-1", time.Now(), time.Now()))
	mock.ExpectExec(`UPDATE payment_refunds`).
		WithArgs("refund-1", models.RefundStatusFailed, nil, "Invalid Amount", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO order_notes`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("note-1", time.Now()))

	_, err := svc.RefundOrderPayment(context.Background(), manjoOrder(models.OrderStatusComplete), RefundRequest{Reason: "x"})
	var gatewayErr *RefundGatewayError
	if !errors.As(err, &gatewayErr) {
		t.Fatalf("err = %v, want RefundGatewayError", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestRefundOrderPaymentInProgressStaysProcessing(t *testing.T) {
	gateway := newManjoServer(t, func(string, map[string]interface{}) (int, string) {
		return http.StatusAccepted, `{"responseCode":"2027800","responseMessage":"Request In Progress"}`
	})
	svc, mock := newManjoTestService(t, gateway)

	expectPaymentByOrderID(mock, "manjo", manjo.StatusSettlement, time.Now())
	mock.ExpectQuery(`INSERT INTO payment_refunds`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("refund-1", time.Now(), time.Now()))
	mock.ExpectExec(`UPDATE payment_refunds`).
		WithArgs("refund-1", models.RefundStatusProcessing, nil, nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO order_notes`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("note-1", time.Now()))

	refund, err := svc.RefundOrderPayment(context.Background(), manjoOrder(models.OrderStatusPaid), RefundRequest{Reason: "x"})
	if err != nil || refund.Status != models.RefundStatusProcessing {
		t.Fatalf("refund = %+v, err = %v; want processing", refund, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestRefundOrderPaymentGuards(t *testing.T) {
	tests := []struct {
		name    string
		gateway string
		status  string
		order   models.OrderStatus
		want    error
	}{
		{name: "midtrans payment", gateway: "midtrans", status: manjo.StatusSettlement, order: models.OrderStatusPaid, want: ErrRefundUnsupported},
		{name: "unpaid order", gateway: "manjo", status: manjo.StatusPending, order: models.OrderStatusPending, want: ErrRefundNotAllowed},
		{name: "already refunded", gateway: "manjo", status: manjo.StatusRefund, order: models.OrderStatusPaid, want: ErrRefundNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, mock := newManjoTestService(t, staticManjoGateway{})
			expectPaymentByOrderID(mock, tt.gateway, tt.status, time.Now())
			_, err := svc.RefundOrderPayment(context.Background(), manjoOrder(tt.order), RefundRequest{Reason: "x"})
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestRefundOrderPaymentRejectsSecondActiveRefund(t *testing.T) {
	gateway := newManjoServer(t, func(string, map[string]interface{}) (int, string) {
		t.Fatal("Manjo must not be called when a refund is already active")
		return 0, ""
	})
	svc, mock := newManjoTestService(t, gateway)
	expectPaymentByOrderID(mock, "manjo", manjo.StatusSettlement, time.Now())
	mock.ExpectQuery(`INSERT INTO payment_refunds`).
		WillReturnError(&pqUniqueViolation)

	_, err := svc.RefundOrderPayment(context.Background(), manjoOrder(models.OrderStatusPaid), RefundRequest{Reason: "x"})
	if !errors.Is(err, ErrRefundInProgress) {
		t.Fatalf("err = %v, want ErrRefundInProgress", err)
	}
}

func TestSyncPendingManjoPaymentRejectsAmountMismatch(t *testing.T) {
	gateway := newManjoServer(t, func(path string, _ map[string]interface{}) (int, string) {
		return http.StatusOK, `{"responseCode":"2005100","latestTransactionStatus":"00","amount":{"value":"1000.00","currency":"IDR"}}`
	})
	svc, mock := newManjoTestService(t, gateway)

	payment := &models.PaymentTransaction{
		ID: manjoPayID, OrderID: manjoOrderID, PaymentGateway: "manjo", Amount: 50000, MidtransOrderID: "GO-0001",
		GatewayReferenceNo: strPtr("A0001"), MidtransTransactionID: strPtr("A0001"), TransactionStatus: strPtr("pending"),
	}
	changed, err := svc.SyncPendingManjoPayment(context.Background(), manjoOrder(models.OrderStatusPending), payment)
	if err == nil || changed {
		t.Fatalf("changed = %v, err = %v; want amount mismatch error", changed, err)
	}
	// No status update may be written.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected sql: %v", err)
	}
}

func TestSyncPendingManjoPaymentIsThrottledAndSkipsOtherGateways(t *testing.T) {
	calls := 0
	gateway := newManjoServer(t, func(string, map[string]interface{}) (int, string) {
		calls++
		return http.StatusOK, `{"responseCode":"2005100","latestTransactionStatus":"03"}`
	})
	svc, _ := newManjoTestService(t, gateway)

	future := time.Now().Add(time.Hour)
	payment := &models.PaymentTransaction{
		ID: manjoPayID, PaymentGateway: "manjo", Amount: 50000, MidtransOrderID: "GO-0001",
		GatewayReferenceNo: strPtr("A0001"), MidtransTransactionID: strPtr("A0001"),
		TransactionStatus: strPtr("pending"), ExpiryTime: &future,
	}
	for i := 0; i < 3; i++ {
		if _, err := svc.SyncPendingManjoPayment(context.Background(), manjoOrder(models.OrderStatusPending), payment); err != nil {
			t.Fatalf("sync returned error: %v", err)
		}
	}
	if calls != 1 {
		t.Fatalf("Manjo query calls = %d, want 1 (throttled)", calls)
	}

	midtrans := &models.PaymentTransaction{ID: "other", PaymentGateway: "midtrans"}
	if changed, err := svc.SyncPendingManjoPayment(context.Background(), manjoOrder(models.OrderStatusPending), midtrans); changed || err != nil {
		t.Fatalf("Midtrans payment was synced at Manjo: changed=%v err=%v", changed, err)
	}
}

func TestIsExpiredUsesWIBWallClock(t *testing.T) {
	wib := time.FixedZone("WIB", 7*60*60)
	nowWIB := time.Now().In(wib)
	asStored := func(t time.Time) *time.Time {
		v := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
		return &v
	}
	if isExpired(asStored(nowWIB.Add(10 * time.Minute))) {
		t.Fatal("QR expiring in 10 minutes reported expired")
	}
	if isExpired(asStored(nowWIB.Add(-time.Minute))) {
		t.Fatal("QR expired 1 minute ago reported expired before the grace period")
	}
	if !isExpired(asStored(nowWIB.Add(-10 * time.Minute))) {
		t.Fatal("QR expired 10 minutes ago not reported expired")
	}
}

func strPtr(s string) *string { return &s }

// pqUniqueViolation mimics the partial unique index rejecting a second active refund.
var pqUniqueViolation = pq.Error{Code: "23505", Constraint: "uq_payment_refunds_active_per_payment"}
