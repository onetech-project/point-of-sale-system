package e2e_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pos/notification-service/api"
)

func makeTestToken(secret string, payload map[string]interface{}) string {
	hdr := map[string]interface{}{"alg": "HS256", "typ": "JWT"}
	hb, _ := json.Marshal(hdr)
	pb, _ := json.Marshal(payload)
	h := base64.RawURLEncoding.EncodeToString(hb)
	p := base64.RawURLEncoding.EncodeToString(pb)
	signing := h + "." + p
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signing))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return signing + "." + sig
}

func TestSSEAuthContract(t *testing.T) {
	e := echo.New()
	e.GET("/api/v1/sse", api.SSEHandler, api.SSEAuthMiddleware)
	ts := httptest.NewServer(e)
	defer ts.Close()

	client := &http.Client{Timeout: 2 * time.Second}

	// Missing Authorization -> 401
	resp, err := client.Get(ts.URL + "/api/v1/sse")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing auth; got %d", resp.StatusCode)
	}

	// Malformed token -> 401
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/sse", nil)
	req.Header.Set("Authorization", "Bearer malformed.token")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for malformed token; got %d", resp.StatusCode)
	}

	secret := "test-secret"
	os.Setenv("TEST_JWT_SECRET", secret)
	defer os.Unsetenv("TEST_JWT_SECRET")

	// Valid token but tenant mismatch -> 403
	token := makeTestToken(secret, map[string]interface{}{"tenant_id": "other-tenant"})
	req, _ = http.NewRequest("GET", ts.URL+"/api/v1/sse", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for tenant mismatch; got %d", resp.StatusCode)
	}

	// Valid token and tenant match -> 200 and heartbeat body
	os.Setenv("ALLOWED_TENANT", "tenant:demo")
	tokenOk := makeTestToken(secret, map[string]interface{}{"tenant_id": "tenant:demo"})
	req, _ = http.NewRequest("GET", ts.URL+"/api/v1/sse", nil)
	req.Header.Set("Authorization", "Bearer "+tokenOk)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for valid token; got %d", resp.StatusCode)
	}
	buf := make([]byte, 64)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])
	if !strings.Contains(body, "heartbeat") {
		t.Fatalf("expected heartbeat in body; got %q", body)
	}
	// ensure test doesn't hang
	time.Sleep(10 * time.Millisecond)
}
