package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/point-of-sale-system/order-service/src/manjo"
	"github.com/point-of-sale-system/order-service/src/models"
)

type memoryManjoTokenStore struct {
	mu          sync.Mutex
	tokens      map[string]ManjoTokenHolder
	externalIDs map[string]bool
}

func newMemoryManjoTokenStore() *memoryManjoTokenStore {
	return &memoryManjoTokenStore{tokens: map[string]ManjoTokenHolder{}, externalIDs: map[string]bool{}}
}

func (m *memoryManjoTokenStore) SaveToken(_ context.Context, token string, holder ManjoTokenHolder, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[token] = holder
	return nil
}

func (m *memoryManjoTokenStore) Token(_ context.Context, token string) (*ManjoTokenHolder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	holder, ok := m.tokens[token]
	if !ok {
		return nil, nil
	}
	return &holder, nil
}

func (m *memoryManjoTokenStore) ClaimExternalID(_ context.Context, scope, externalID string, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := scope + ":" + externalID
	if m.externalIDs[key] {
		return false, nil
	}
	m.externalIDs[key] = true
	return true, nil
}

var manjoTestNow = time.Date(2026, 11, 9, 10, 0, 0, 0, time.FixedZone("WIB", 7*60*60))

func newManjoKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func tokenRequestHeader(t *testing.T, key *rsa.PrivateKey, clientKey string, at time.Time) http.Header {
	t.Helper()
	timestamp := at.Format("2006-01-02T15:04:05-07:00")
	signature, err := manjo.SignAccessToken(key, clientKey, timestamp)
	if err != nil {
		t.Fatal(err)
	}
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("X-CLIENT-KEY", clientKey)
	h.Set("X-TIMESTAMP", timestamp)
	h.Set("X-SIGNATURE", signature)
	return h
}

const tokenRequestBody = `{"grantType":"client_credentials"}`

func TestIssueManjoAccessToken(t *testing.T) {
	sandboxKey, productionKey, otherKey := newManjoKey(t), newManjoKey(t), newManjoKey(t)
	store := newMemoryManjoTokenStore()
	svc := (&PaymentService{now: func() time.Time { return manjoTestNow }}).WithManjoInbound(store, map[string]*rsa.PublicKey{
		"sandbox":    &sandboxKey.PublicKey,
		"production": &productionKey.PublicKey,
	})
	ctx := context.Background()

	status, resp := svc.IssueManjoAccessToken(ctx, tokenRequestHeader(t, productionKey, "EP1", manjoTestNow), []byte(tokenRequestBody))
	if status != http.StatusOK || resp.ResponseCode != "2007300" || resp.TokenType != "Bearer" || resp.ExpiresIn != "900" || resp.AccessToken == "" {
		t.Fatalf("status %d, response %+v", status, resp)
	}
	holder, _ := store.Token(ctx, resp.AccessToken)
	if holder == nil || holder.ClientID != "EP1" || holder.Environment != "production" {
		t.Fatalf("stored holder = %+v, want EP1/production", holder)
	}

	cases := []struct {
		name   string
		header http.Header
		body   string
		status int
		code   string
	}{
		{"signed with an unknown key", tokenRequestHeader(t, otherKey, "EP1", manjoTestNow), tokenRequestBody, 401, "4017300"},
		{"stale timestamp", tokenRequestHeader(t, sandboxKey, "EP1", manjoTestNow.Add(-10*time.Minute)), tokenRequestBody, 401, "4017300"},
		{"wrong grant type", tokenRequestHeader(t, sandboxKey, "EP1", manjoTestNow), `{"grantType":"password"}`, 400, "4007301"},
		{"missing body", tokenRequestHeader(t, sandboxKey, "EP1", manjoTestNow), ``, 400, "4007302"},
		{"missing signature", func() http.Header {
			h := tokenRequestHeader(t, sandboxKey, "EP1", manjoTestNow)
			h.Del("X-SIGNATURE")
			return h
		}(), tokenRequestBody, 400, "4007302"},
	}
	for _, c := range cases {
		status, resp := svc.IssueManjoAccessToken(ctx, c.header, []byte(c.body))
		if status != c.status || resp.ResponseCode != c.code || resp.AccessToken != "" {
			t.Errorf("%s: status %d, response %+v; want %d %s", c.name, status, resp, c.status, c.code)
		}
	}
}

var guestOrderColumns = []string{
	"id", "order_reference", "tenant_id", "status", "subtotal_amount", "delivery_fee", "total_amount",
	"customer_name", "customer_phone", "customer_email", "delivery_type", "table_number", "notes",
	"created_at", "paid_at", "completed_at", "cancelled_at", "session_id", "ip_address", "user_agent", "is_anonymized",
	"anonymized_at", "tenant_slug", "order_type",
}

func expectOrderByReference(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`FROM guest_orders od.*WHERE order_reference = \$1`).
		WithArgs("GO-0001").
		WillReturnRows(sqlmock.NewRows(guestOrderColumns).AddRow(
			manjoOrderID, "GO-0001", manjoTenantID, string(models.OrderStatusPending), 50000, 0, 50000,
			nil, nil, nil, "pickup", nil, nil,
			time.Now(), nil, nil, nil, nil, nil, nil, false,
			nil, "shop", "online",
		))
}

const notifyPath = "/api/v1/webhooks/payments/manjo/v1.0/qr/qr-mpm-notify"

var notifyBody = []byte(`{"originalReferenceNo":"A0001","originalPartnerReferenceNo":"GO-0001","latestTransactionStatus":"00","amount":{"value":"50000.00","currency":"IDR"}}`)

func notifyHeader(t *testing.T, token, externalID, secret string) http.Header {
	t.Helper()
	timestamp := manjoTestNow.Format("2006-01-02T15:04:05-07:00")
	signature, err := manjo.SignRequest(secret, http.MethodPost, manjo.PathNotify, token, notifyBody, timestamp)
	if err != nil {
		t.Fatal(err)
	}
	h := http.Header{}
	h.Set("Authorization", "Bearer "+token)
	h.Set("X-TIMESTAMP", timestamp)
	h.Set("X-SIGNATURE", signature)
	h.Set("X-PARTNER-ID", "EP1")
	h.Set("X-EXTERNAL-ID", externalID)
	h.Set("CHANNEL-ID", "05")
	return h
}

func TestProcessManjoNotificationFlow(t *testing.T) {
	queries := 0
	gateway := newManjoServer(t, func(path string, _ map[string]interface{}) (int, string) {
		if path != manjo.PathQuery {
			t.Fatalf("unexpected path %s", path)
		}
		queries++
		return http.StatusOK, `{"responseCode":"2005100","latestTransactionStatus":"03"}`
	})
	gateway.creds.Environment = "sandbox"
	svc, mock := newManjoTestService(t, gateway)
	store := newMemoryManjoTokenStore()
	svc.now = func() time.Time { return manjoTestNow }
	svc.WithManjoInbound(store, nil)
	ctx := context.Background()
	_ = store.SaveToken(ctx, "tok-ep1", ManjoTokenHolder{ClientID: "EP1", Environment: "sandbox"}, time.Minute)
	_ = store.SaveToken(ctx, "tok-other", ManjoTokenHolder{ClientID: "EP-OTHER", Environment: "sandbox"}, time.Minute)
	_ = store.SaveToken(ctx, "tok-prod", ManjoTokenHolder{ClientID: "EP1", Environment: "production"}, time.Minute)
	future := time.Now().Add(time.Hour)

	process := func(header http.Header) ManjoNotificationResult {
		t.Helper()
		result, _ := svc.ProcessManjoNotification(ctx, header, notifyPath, notifyBody)
		return result
	}

	// Rejected before anything is looked up.
	noToken := notifyHeader(t, "", "ext-0", "secret")
	noToken.Del("Authorization")
	if got := process(noToken); got != ManjoNotificationInvalidToken {
		t.Fatalf("no token: result %d", got)
	}
	if got := process(notifyHeader(t, "unknown", "ext-0", "secret")); got != ManjoNotificationInvalidToken {
		t.Fatalf("unknown token: result %d", got)
	}

	// A token issued to another tenant's client ID, or for another environment.
	for _, token := range []string{"tok-other", "tok-prod"} {
		expectOrderByReference(mock)
		expectPaymentByOrderID(mock, "manjo", manjo.StatusPending, future)
		if got := process(notifyHeader(t, token, "ext-"+token, "secret")); got != ManjoNotificationInvalidToken {
			t.Fatalf("%s: result %d, want invalid token", token, got)
		}
	}

	// Wrong signature.
	expectOrderByReference(mock)
	expectPaymentByOrderID(mock, "manjo", manjo.StatusPending, future)
	if got := process(notifyHeader(t, "tok-ep1", "ext-1", "wrong-secret")); got != ManjoNotificationInvalidSignature {
		t.Fatalf("wrong secret: result %d", got)
	}

	// Valid notification: payment confirmed through the query API.
	expectOrderByReference(mock)
	expectPaymentByOrderID(mock, "manjo", manjo.StatusPending, future)
	if got := process(notifyHeader(t, "tok-ep1", "ext-2", "secret")); got != ManjoNotificationProcessed {
		t.Fatalf("valid notification: result %d", got)
	}
	if queries != 1 {
		t.Fatalf("query calls = %d, want 1", queries)
	}

	// Same X-EXTERNAL-ID again the same day.
	expectOrderByReference(mock)
	expectPaymentByOrderID(mock, "manjo", manjo.StatusPending, future)
	if got := process(notifyHeader(t, "tok-ep1", "ext-2", "secret")); got != ManjoNotificationDuplicate {
		t.Fatalf("replayed external ID: result %d", got)
	}
	if queries != 1 {
		t.Fatalf("query calls = %d after replay, want 1", queries)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
