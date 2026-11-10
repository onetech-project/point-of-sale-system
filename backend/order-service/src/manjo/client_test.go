package manjo

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type testKey struct {
	key *rsa.PrivateKey
	pem string
}

func newTestKey(t *testing.T) testKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	return testKey{key: key, pem: string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}))}
}

// fakeManjo is a Manjo API double that checks signatures the way Manjo would.
type fakeManjo struct {
	t            *testing.T
	creds        Credentials
	publicKey    *rsa.PublicKey
	tokenCalls   atomic.Int32
	expireTokens atomic.Bool // answer 401 on the next transactional request
	handler      func(w http.ResponseWriter, path string, body map[string]interface{})
}

func (f *fakeManjo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path == PathAccessToken {
		f.tokenCalls.Add(1)
		digest := sha256.Sum256([]byte(r.Header.Get("X-CLIENT-KEY") + "|" + r.Header.Get("X-TIMESTAMP")))
		signature, err := hex.DecodeString(r.Header.Get("X-SIGNATURE"))
		if err != nil || rsa.VerifyPKCS1v15(f.publicKey, crypto.SHA256, digest[:], signature) != nil {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"responseCode":"4017300","responseMessage":"Unauthorized. Signature"}`)
			return
		}
		if string(body) != `{"grantType":"client_credentials"}` {
			f.t.Errorf("unexpected access token body: %s", body)
		}
		_, _ = io.WriteString(w, `{"responseCode":"2007300","responseMessage":"Successful","tokenType":"Bearer","accessToken":"token-`+
			string(rune('0'+f.tokenCalls.Load()))+`","expiresIn":"900"}`)
		return
	}

	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if f.expireTokens.CompareAndSwap(true, false) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"responseCode":"4014701","responseMessage":"Invalid token"}`)
		return
	}
	expected, err := SignRequest(f.creds.ClientSecret, r.Method, r.URL.Path, token, body, r.Header.Get("X-TIMESTAMP"))
	if err != nil || r.Header.Get("X-SIGNATURE") != expected {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"responseCode":"4014700","responseMessage":"Unauthorized. Signature"}`)
		return
	}
	for _, h := range []string{"X-PARTNER-ID", "X-EXTERNAL-ID", "CHANNEL-ID", "X-IP-ADDRESS", "X-TIMESTAMP"} {
		if r.Header.Get(h) == "" {
			f.t.Errorf("%s: missing header %s", r.URL.Path, h)
		}
	}
	if r.Header.Get("X-PARTNER-ID") != f.creds.partnerID() {
		f.t.Errorf("X-PARTNER-ID = %q, want %q", r.Header.Get("X-PARTNER-ID"), f.creds.partnerID())
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		f.t.Fatalf("invalid JSON body: %v", err)
	}
	f.handler(w, r.URL.Path, decoded)
}

func newFakeManjo(t *testing.T, handler func(w http.ResponseWriter, path string, body map[string]interface{})) (*fakeManjo, *Client, Credentials) {
	t.Helper()
	key := newTestKey(t)
	creds := Credentials{ClientID: "EP9613058999", ClientSecret: "secret", MerchantID: "MT60169117"}
	fake := &fakeManjo{t: t, creds: creds, publicKey: &key.key.PublicKey, handler: handler}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	return fake, NewClient(server.URL, WithHTTPClient(server.Client()), WithPrivateKey(key.key)), creds
}

func TestGenerateSignsRequestsAndReturnsQRContent(t *testing.T) {
	var gotBody map[string]interface{}
	fake, client, creds := newFakeManjo(t, func(w http.ResponseWriter, path string, body map[string]interface{}) {
		if path != PathGenerate {
			t.Fatalf("path = %s", path)
		}
		gotBody = body
		_, _ = io.WriteString(w, `{"responseCode":"2004700","responseMessage":"Successful","referenceNo":"A0000001703",
			"partnerReferenceNo":"ORD-1","qrContent":"000201010212","qrUrl":"-","qrImage":"-",
			"additionalInfo":{"paymentId":"99","expiryDuration":"900","expireDate":"20251127151448","amount":"50000.00"}}`)
	})

	resp, err := client.Generate(context.Background(), creds, GenerateRequest{
		PartnerReferenceNo: "ORD-1",
		Amount:             Amount{Value: FormatAmount(50000)},
		ValidityPeriod:     "900",
		AdditionalInfo:     GenerateAdditionalInfo{ProdDesc: "Order ORD-1"},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if resp.ReferenceNo != "A0000001703" || resp.QRContent != "000201010212" || resp.ExternalID == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if gotBody["merchantId"] != "MT60169117" {
		t.Fatalf("merchantId = %v", gotBody["merchantId"])
	}
	info := gotBody["additionalInfo"].(map[string]interface{})
	if info["paymentId"] != "99" || info["dynamicAmount"] != "N" {
		t.Fatalf("additionalInfo = %v", info)
	}
	amount := gotBody["amount"].(map[string]interface{})
	if amount["value"] != "50000.00" || amount["currency"] != "IDR" {
		t.Fatalf("amount = %v", amount)
	}
	expiresAt, ok := resp.ExpiresAt(nil)
	if !ok || expiresAt.UTC().Format(time.RFC3339) != "2025-11-27T08:14:48Z" {
		t.Fatalf("ExpiresAt = %v, %v", expiresAt, ok)
	}
	if fake.tokenCalls.Load() != 1 {
		t.Fatalf("token calls = %d, want 1", fake.tokenCalls.Load())
	}
}

func TestAccessTokenIsCachedAndRenewedAfter401(t *testing.T) {
	fake, client, creds := newFakeManjo(t, func(w http.ResponseWriter, path string, body map[string]interface{}) {
		_, _ = io.WriteString(w, `{"responseCode":"2005100","responseMessage":"Successful","latestTransactionStatus":"03"}`)
	})
	query := QueryRequest{OriginalReferenceNo: "A1", OriginalPartnerReferenceNo: "ORD-1"}

	for i := 0; i < 2; i++ {
		if _, err := client.Query(context.Background(), creds, query); err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
	}
	if fake.tokenCalls.Load() != 1 {
		t.Fatalf("token calls = %d, want 1 (cached)", fake.tokenCalls.Load())
	}

	fake.expireTokens.Store(true)
	if _, err := client.Query(context.Background(), creds, query); err != nil {
		t.Fatalf("Query after token expiry returned error: %v", err)
	}
	if fake.tokenCalls.Load() != 2 {
		t.Fatalf("token calls = %d, want 2 (renewed after 401)", fake.tokenCalls.Load())
	}
}

func TestQueryReportsNotFoundAsStatus(t *testing.T) {
	_, client, creds := newFakeManjo(t, func(w http.ResponseWriter, path string, body map[string]interface{}) {
		if body["serviceCode"] != "99" || body["merchantId"] != "MT60169117" {
			t.Errorf("unexpected query body: %v", body)
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"responseCode":"4045101","responseMessage":"Transaction Not Found","latestTransactionStatus":"07"}`)
	})

	resp, err := client.Query(context.Background(), creds, QueryRequest{OriginalReferenceNo: "A1", OriginalPartnerReferenceNo: "ORD-1"})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if StatusFromQuery(resp.LatestTransactionStatus) != StatusPending {
		t.Fatalf("status = %s", resp.LatestTransactionStatus)
	}
}

func TestRefund(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		body           string
		wantInProgress bool
		wantCaseCode   string
	}{
		{name: "success", status: http.StatusOK, body: `{"responseCode":"2007800","responseMessage":"Successful","refundNo":"RF-1","partnerRefundNo":"RFD-1"}`},
		{name: "in progress", status: http.StatusAccepted, body: `{"responseCode":"2027800","responseMessage":"Request In Progress","partnerRefundNo":"RFD-1"}`, wantInProgress: true},
		{name: "invalid amount", status: http.StatusNotFound, body: `{"responseCode":"4047813","responseMessage":"Invalid Amount"}`, wantCaseCode: "13"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, client, creds := newFakeManjo(t, func(w http.ResponseWriter, path string, body map[string]interface{}) {
				if path != PathRefund || body["originalExternalId"] != "EXT-1" || body["reason"] != "Customer complain" {
					t.Errorf("unexpected refund request %s: %v", path, body)
				}
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			})
			resp, err := client.Refund(context.Background(), creds, RefundRequest{
				OriginalPartnerReferenceNo: "ORD-1",
				OriginalReferenceNo:        "A1",
				OriginalExternalID:         "EXT-1",
				PartnerRefundNo:            "RFD-1",
				RefundAmount:               Amount{Value: "10000.00"},
				Reason:                     "Customer complain",
			})
			if tt.wantCaseCode != "" {
				var apiErr *APIError
				if !errors.As(err, &apiErr) || apiErr.CaseCode() != tt.wantCaseCode {
					t.Fatalf("err = %v, want APIError with case code %s", err, tt.wantCaseCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("Refund returned error: %v", err)
			}
			if resp.InProgress != tt.wantInProgress {
				t.Fatalf("InProgress = %v, want %v", resp.InProgress, tt.wantInProgress)
			}
		})
	}
}

func TestSignRequestMinifiesBody(t *testing.T) {
	pretty := []byte("{\n  \"a\": 1,\n  \"b\": \"x\"\n}")
	compact := []byte(`{"a":1,"b":"x"}`)
	s1, err := SignRequest("secret", "POST", PathQuery, "tok", pretty, "2025-11-27T08:10:57+07:00")
	if err != nil {
		t.Fatal(err)
	}
	s2, _ := SignRequest("secret", "POST", PathQuery, "tok", compact, "2025-11-27T08:10:57+07:00")
	if s1 != s2 || len(s1) != 128 {
		t.Fatalf("signatures differ or wrong length: %s vs %s", s1, s2)
	}
}

func TestVerifyNotificationSignature(t *testing.T) {
	body := []byte(`{"originalReferenceNo":"A1","originalPartnerReferenceNo":"ORD-1","latestTransactionStatus":"00"}`)
	timestamp := "2026-01-27T13:13:54+07:00"
	const requestPath = "/api/v1/webhooks/payments/manjo/v1.0/qr/qr-mpm-notify"

	header := func(signature string) http.Header {
		h := http.Header{}
		h.Set("Authorization", "Bearer tok-1")
		h.Set("X-TIMESTAMP", timestamp)
		h.Set("X-SIGNATURE", signature)
		h.Set("X-PARTNER-ID", "EP9613058999")
		return h
	}

	overRequestPath, _ := SignRequest("secret", "POST", requestPath, "tok-1", body, timestamp)
	overNotifyPath, _ := SignRequest("secret", "POST", PathNotify, "tok-1", body, timestamp)
	withPartnerID, _ := SignRequest("EP9613058999", "POST", PathNotify, "tok-1", body, timestamp)
	withoutToken, _ := SignRequest("secret", "POST", PathNotify, "", body, timestamp)

	if !VerifyNotificationSignature(header(overRequestPath), requestPath, body, "secret") {
		t.Fatal("signature over the request path was rejected")
	}
	if !VerifyNotificationSignature(header(strings.ToUpper(overNotifyPath)), requestPath, body, "secret") {
		t.Fatal("upper case signature over the docs' notify path was rejected")
	}
	if VerifyNotificationSignature(header(withPartnerID), requestPath, body, "secret") {
		t.Fatal("signature keyed by the X-PARTNER-ID header was accepted")
	}
	if VerifyNotificationSignature(header(withoutToken), requestPath, body, "secret") {
		t.Fatal("signature without the bearer token was accepted")
	}
	if VerifyNotificationSignature(header(overRequestPath), requestPath, []byte(`{"tampered":true}`), "secret") {
		t.Fatal("signature over another body was accepted")
	}
	if VerifyNotificationSignature(header(overRequestPath), requestPath, body, "") {
		t.Fatal("signature was accepted without a client secret")
	}
	if VerifyNotificationSignature(header(""), "/x", body, "secret") {
		t.Fatal("missing signature was accepted")
	}
}

func TestVerifyAccessTokenSignature(t *testing.T) {
	key := newTestKey(t)
	other := newTestKey(t)
	timestamp := "2026-11-09T10:00:00+07:00"
	signature, err := SignAccessToken(key.key, "EP1", timestamp)
	if err != nil {
		t.Fatal(err)
	}

	if !VerifyAccessTokenSignature(&key.key.PublicKey, "EP1", timestamp, signature) {
		t.Fatal("valid signature was rejected")
	}
	if VerifyAccessTokenSignature(&other.key.PublicKey, "EP1", timestamp, signature) {
		t.Fatal("signature was accepted with another public key")
	}
	if VerifyAccessTokenSignature(&key.key.PublicKey, "EP2", timestamp, signature) {
		t.Fatal("signature was accepted for another client key")
	}
	if VerifyAccessTokenSignature(&key.key.PublicKey, "EP1", timestamp, "not-hex") {
		t.Fatal("non-hex signature was accepted")
	}
}

func TestParseKeys(t *testing.T) {
	key := newTestKey(t)
	pkixDER, err := x509.MarshalPKIXPublicKey(&key.key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkixDER}))

	for name, value := range map[string]string{
		"pem":         publicPEM,
		"escaped pem": strings.ReplaceAll(publicPEM, "\n", `\n`),
		"bare base64": base64.StdEncoding.EncodeToString(pkixDER),
		"pkcs1 pem":   string(pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&key.key.PublicKey)})),
	} {
		got, err := ParsePublicKey(value)
		if err != nil || !got.Equal(&key.key.PublicKey) {
			t.Fatalf("%s: ParsePublicKey = %v", name, err)
		}
	}
	if _, err := ParsePrivateKey(strings.ReplaceAll(key.pem, "\n", `\n`)); err != nil {
		t.Fatalf("escaped private key: %v", err)
	}
	if _, err := ParsePublicKey("not a key"); err == nil {
		t.Fatal("garbage public key was accepted")
	}
}

func TestParseAmount(t *testing.T) {
	if n, err := ParseAmount("50450.00"); err != nil || n != 50450 {
		t.Fatalf("ParseAmount = %d, %v", n, err)
	}
	if _, err := ParseAmount("10.50"); err == nil {
		t.Fatal("fractional amount was accepted")
	}
	if FormatAmount(30000) != "30000.00" {
		t.Fatalf("FormatAmount = %s", FormatAmount(30000))
	}
}
