package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestSubscriptionEnforcerStatuses(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		wantCode      int
		wantWarning   string
		nextShouldRun bool
	}{
		{
			name:          "trial passes",
			status:        "trial",
			wantCode:      http.StatusNoContent,
			nextShouldRun: true,
		},
		{
			name:          "active passes",
			status:        "active",
			wantCode:      http.StatusNoContent,
			nextShouldRun: true,
		},
		{
			name:          "grace period passes with warning",
			status:        "grace_period",
			wantCode:      http.StatusNoContent,
			wantWarning:   "grace_period",
			nextShouldRun: true,
		},
		{
			name:          "expired is blocked",
			status:        "expired",
			wantCode:      http.StatusPaymentRequired,
			nextShouldRun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			billingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/internal/subscription/tenant-1" {
					t.Fatalf("unexpected billing path: %s", r.URL.Path)
				}
				_, _ = fmt.Fprintf(w, `{"subscription_status":%q}`, tt.status)
			}))
			defer billingServer.Close()

			enforcer := &SubscriptionEnforcer{
				billingURL: billingServer.URL,
				httpClient: billingServer.Client(),
			}

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("tenant_id", "tenant-1")

			nextRan := false
			next := func(c echo.Context) error {
				nextRan = true
				return c.NoContent(http.StatusNoContent)
			}

			if err := enforcer.EnforceSubscription()(next)(c); err != nil {
				t.Fatalf("middleware returned error: %v", err)
			}
			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantCode, rec.Body.String())
			}
			if nextRan != tt.nextShouldRun {
				t.Fatalf("nextRan = %v, want %v", nextRan, tt.nextShouldRun)
			}
			if got := rec.Header().Get("X-Subscription-Warning"); got != tt.wantWarning {
				t.Fatalf("X-Subscription-Warning = %q, want %q", got, tt.wantWarning)
			}
		})
	}
}

func TestSubscriptionEnforcerFailsOpenWhenBillingUnavailable(t *testing.T) {
	enforcer := &SubscriptionEnforcer{
		billingURL: "http://127.0.0.1:1",
		httpClient: &http.Client{Timeout: time.Millisecond},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "tenant-1")

	nextRan := false
	next := func(c echo.Context) error {
		nextRan = true
		return c.NoContent(http.StatusNoContent)
	}

	if err := enforcer.EnforceSubscription()(next)(c); err != nil {
		t.Fatalf("middleware returned error: %v", err)
	}
	if !nextRan {
		t.Fatal("expected middleware to fail open and call next")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestSubscriptionCacheTTL(t *testing.T) {
	tests := []struct {
		status string
		want   time.Duration
	}{
		{status: "active", want: 5 * time.Minute},
		{status: "trial", want: 5 * time.Minute},
		{status: "grace_period", want: time.Minute},
		{status: "expired", want: 30 * time.Second},
		{status: "cancelled", want: 30 * time.Second},
	}

	for _, tt := range tests {
		if got := subscriptionCacheTTL(tt.status); got != tt.want {
			t.Fatalf("subscriptionCacheTTL(%q) = %s, want %s", tt.status, got, tt.want)
		}
	}
}
