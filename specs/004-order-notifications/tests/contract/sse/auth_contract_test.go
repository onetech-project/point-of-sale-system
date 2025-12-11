package sse_contract_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// makeTestJWT creates a simple HMAC-SHA256 signed JWT using the TEST_JWT_SECRET
// environment variable (default "test-secret"). This is a test-only helper
// intended for contract checks; CI should provide a compatible test signing key.
func makeTestJWT(tenant string, roles []string) string {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	payload := map[string]interface{}{
		"tenant_id": tenant,
		"roles":     roles,
		"iat":       time.Now().Unix(),
	}

	hEnc := base64.RawURLEncoding.EncodeToString(mustMarshal(header))
	pEnc := base64.RawURLEncoding.EncodeToString(mustMarshal(payload))
	signingInput := hEnc + "." + pEnc

	secret := os.Getenv("TEST_JWT_SECRET")
	if secret == "" {
		secret = "test-secret"
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	sig := mac.Sum(nil)
	sEnc := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + sEnc
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func baseURL() string {
	if u := os.Getenv("SSE_BASE_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "http://localhost:8085"
}

// TestSSEAuthContract performs three contract checks:
// 1) connection without token should be rejected (401)
// 2) connection with token for a different tenant should be rejected (403)
// 3) connection with a valid token containing tenant_id and roles should be accepted (200) and produce data/heartbeat
// Note: This is a contract test and is expected to fail until T024 (SSE auth) is implemented.
func TestSSEAuthContract(t *testing.T) {
	url := baseURL() + "/api/v1/sse"
	client := &http.Client{Timeout: 10 * time.Second}

	t.Logf("Attempting unauthenticated request to %s", url)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Unauthenticated request failed (connection error): %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		t.Log("OK: unauthenticated request rejected with 401")
	} else {
		t.Fatalf("Expected 401 for unauthenticated request, got %d", resp.StatusCode)
	}

	t.Log("Attempting request with wrong-tenant token (expect 403)")
	wrong := makeTestJWT("tenant:other", []string{"Cashier"})
	req, _ = http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+wrong)
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("Wrong-tenant request failed (connection error): %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode == http.StatusForbidden || resp2.StatusCode == http.StatusUnauthorized {
		t.Logf("OK: wrong-tenant token rejected with %d", resp2.StatusCode)
	} else {
		t.Fatalf("Expected 403/401 for wrong-tenant token, got %d", resp2.StatusCode)
	}

	t.Log("Attempting request with valid token (expect 200 and SSE data/heartbeat)")
	good := makeTestJWT("tenant:demo", []string{"Owner", "Manager"})
	req, _ = http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+good)
	resp3, err := client.Do(req)
	if err != nil {
		t.Fatalf("Authenticated request failed (connection error): %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 for valid token, got %d", resp3.StatusCode)
	}

	// If we reach here, server returned 200. Try to read a small chunk to ensure SSE stream or heartbeat exists.
	buf := make([]byte, 64)
	n, _ := resp3.Body.Read(buf)
	if n == 0 {
		t.Fatalf("Connected with valid token but no data/heartbeat received within timeout")
	}
	t.Logf("Received %d bytes from SSE stream (ok)", n)
}
