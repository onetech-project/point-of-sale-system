// Package manjo is a client for the Manjo (manjo.co.id) QRIS MPM API:
// B2B access token, QR generate, query payment and refund, plus verification
// of the payment notifications Manjo sends to the merchant.
//
// Docs: https://docs.manjo.co.id/docs/qris/api-access-token-b2b and siblings.
package manjo

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	PathAccessToken = "/v1.0/access-token/b2b"
	PathGenerate    = "/v1.0/qr/qr-mpm-generate"
	PathQuery       = "/v1.0/qr/qr-mpm-query"
	PathRefund      = "/v1.0/qr/qr-mpm-refund"
	PathNotify      = "/v1.0/qr/qr-mpm-notify"

	// PaymentID is the Manjo transaction code for QRIS ("99").
	PaymentID = "99"
	// ChannelID is the device identification Manjo asks for ("05").
	ChannelID = "05"

	timestampLayout = "2006-01-02T15:04:05-07:00"
)

// Credentials are one tenant's Manjo credentials. The RSA key that signs
// access token requests is the platform's, held by the Client.
type Credentials struct {
	ClientID     string // X-CLIENT-KEY for the access token
	ClientSecret string // HMAC-SHA512 key for request and notification signatures
	MerchantID   string // merchantId in request bodies
	PartnerID    string // X-PARTNER-ID; ClientID when empty
	Environment  string // "sandbox" or "production"
}

func (c Credentials) partnerID() string {
	if c.PartnerID != "" {
		return c.PartnerID
	}
	return c.ClientID
}

// Client calls the Manjo API. It caches access tokens per client ID and is safe
// for concurrent use.
type Client struct {
	baseURL    string
	httpClient *http.Client
	ipAddress  string
	privateKey *rsa.PrivateKey
	now        func() time.Time
	location   *time.Location

	mu     sync.Mutex
	tokens map[string]cachedToken
}

type cachedToken struct {
	token     string
	expiresAt time.Time
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithClock sets the clock, for tests.
func WithClock(now func() time.Time) Option {
	return func(c *Client) { c.now = now }
}

// WithIPAddress sets the X-IP-ADDRESS header value (merchant server IP).
func WithIPAddress(ip string) Option {
	return func(c *Client) { c.ipAddress = ip }
}

// WithPrivateKey sets the platform RSA key that signs access token requests.
// Manjo holds the matching public key.
func WithPrivateKey(key *rsa.PrivateKey) Option {
	return func(c *Client) { c.privateKey = key }
}

// NewClient returns a client for the Manjo API at baseURL, e.g. https://api.manjo.co.id.
func NewClient(baseURL string, opts ...Option) *Client {
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		location = time.FixedZone("WIB", 7*60*60)
	}
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 15 * time.Second},
		ipAddress:  "127.0.0.1",
		now:        time.Now,
		location:   location,
		tokens:     map[string]cachedToken{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// APIError is a non-successful Manjo response.
type APIError struct {
	HTTPStatus      int
	ResponseCode    string
	ResponseMessage string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("manjo: HTTP %d, responseCode %s: %s", e.HTTPStatus, e.ResponseCode, e.ResponseMessage)
}

// CaseCode returns the last two digits of the response code, e.g. "13" for 4047813.
func (e *APIError) CaseCode() string {
	if len(e.ResponseCode) != 7 {
		return ""
	}
	return e.ResponseCode[5:]
}

// Amount is a Manjo amount object. Value has two decimals, e.g. "50000.00".
type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

// FormatAmount formats whole rupiah as a Manjo amount value.
func FormatAmount(rupiah int) string {
	return strconv.Itoa(rupiah) + ".00"
}

// ParseAmount parses a Manjo amount value into whole rupiah. Cents are rejected,
// since IDR orders never have them.
func ParseAmount(value string) (int, error) {
	whole, frac, _ := strings.Cut(strings.TrimSpace(value), ".")
	if strings.Trim(frac, "0") != "" {
		return 0, fmt.Errorf("manjo: amount %q has a fractional rupiah part", value)
	}
	n, err := strconv.Atoi(whole)
	if err != nil {
		return 0, fmt.Errorf("manjo: invalid amount %q", value)
	}
	return n, nil
}

// ---------------------------------------------------------------------------
// Access token
// ---------------------------------------------------------------------------

type accessTokenResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	AccessToken     string `json:"accessToken"`
	TokenType       string `json:"tokenType"`
	ExpiresIn       string `json:"expiresIn"`
}

// accessToken returns a cached token, or requests a new one. Tokens are renewed
// a minute before they expire.
func (c *Client) accessToken(ctx context.Context, creds Credentials) (string, error) {
	c.mu.Lock()
	cached, ok := c.tokens[creds.ClientID]
	c.mu.Unlock()
	if ok && c.now().Before(cached.expiresAt) {
		return cached.token, nil
	}

	if c.privateKey == nil {
		return "", errors.New("manjo: no private key configured for access token requests")
	}
	timestamp := c.timestamp()
	signature, err := SignAccessToken(c.privateKey, creds.ClientID, timestamp)
	if err != nil {
		return "", err
	}

	body := []byte(`{"grantType":"client_credentials"}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+PathAccessToken, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("manjo: build access token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TIMESTAMP", timestamp)
	req.Header.Set("X-CLIENT-KEY", creds.ClientID)
	req.Header.Set("X-SIGNATURE", signature)

	var resp accessTokenResponse
	status, err := c.do(req, &resp)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK || resp.AccessToken == "" {
		return "", &APIError{HTTPStatus: status, ResponseCode: resp.ResponseCode, ResponseMessage: resp.ResponseMessage}
	}

	expiresIn, err := strconv.Atoi(resp.ExpiresIn)
	if err != nil || expiresIn <= 0 {
		expiresIn = 900
	}
	ttl := time.Duration(expiresIn)*time.Second - time.Minute
	if ttl <= 0 {
		ttl = time.Duration(expiresIn) * time.Second / 2
	}

	c.mu.Lock()
	c.tokens[creds.ClientID] = cachedToken{token: resp.AccessToken, expiresAt: c.now().Add(ttl)}
	c.mu.Unlock()
	return resp.AccessToken, nil
}

func (c *Client) dropToken(clientID string) {
	c.mu.Lock()
	delete(c.tokens, clientID)
	c.mu.Unlock()
}

// SignAccessToken signs clientID + "|" + timestamp with SHA256withRSA and hex
// encodes the signature, as the access token API requires.
func SignAccessToken(key *rsa.PrivateKey, clientID, timestamp string) (string, error) {
	digest := sha256.Sum256([]byte(clientID + "|" + timestamp))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("manjo: sign access token: %w", err)
	}
	return hex.EncodeToString(signature), nil
}

// VerifyAccessTokenSignature checks the hex X-SIGNATURE of an access token
// request Manjo sends to the merchant: SHA256withRSA over clientKey + "|" +
// timestamp, verified with Manjo's public key.
func VerifyAccessTokenSignature(key *rsa.PublicKey, clientKey, timestamp, signatureHex string) bool {
	signature, err := hex.DecodeString(strings.TrimSpace(signatureHex))
	if err != nil || len(signature) == 0 {
		return false
	}
	digest := sha256.Sum256([]byte(clientKey + "|" + timestamp))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature) == nil
}

// ---------------------------------------------------------------------------
// Transactional requests
// ---------------------------------------------------------------------------

// GenerateRequest is the body of the QR MPM generate API.
type GenerateRequest struct {
	PartnerReferenceNo string                 `json:"partnerReferenceNo"`
	Amount             Amount                 `json:"amount"`
	MerchantID         string                 `json:"merchantId"`
	SubMerchantID      string                 `json:"subMerchantId,omitempty"`
	StoreID            string                 `json:"storeId,omitempty"`
	TerminalID         string                 `json:"terminalId,omitempty"`
	ValidityPeriod     string                 `json:"validityPeriod"`
	AdditionalInfo     GenerateAdditionalInfo `json:"additionalInfo"`
}

type GenerateAdditionalInfo struct {
	PaymentID     string `json:"paymentId"`
	DynamicAmount string `json:"dynamicAmount"`
	ProdDesc      string `json:"prodDesc"`
}

// GenerateResponse is the QR MPM generate response.
type GenerateResponse struct {
	ResponseCode       string `json:"responseCode"`
	ResponseMessage    string `json:"responseMessage"`
	ReferenceNo        string `json:"referenceNo"`
	PartnerReferenceNo string `json:"partnerReferenceNo"`
	QRContent          string `json:"qrContent"`
	QRURL              string `json:"qrUrl"`
	QRImage            string `json:"qrImage"`
	RedirectURL        string `json:"redirectUrl"`
	MerchantName       string `json:"merchantName"`
	StoreID            string `json:"storeId"`
	TerminalID         string `json:"terminalId"`
	AdditionalInfo     struct {
		PaymentID      string `json:"paymentId"`
		MerchantCode   string `json:"merchantCode"`
		ExpiryDuration string `json:"expiryDuration"`
		ExpireDate     string `json:"expireDate"` // yyyyMMddHHmmss, WIB
		Amount         string `json:"amount"`
	} `json:"additionalInfo"`

	// ExternalID is the X-EXTERNAL-ID the request was sent with; Manjo asks for
	// it again as originalExternalId when refunding.
	ExternalID string `json:"-"`
}

// ExpiresAt parses additionalInfo.expireDate (WIB). It returns false when the
// field is missing or malformed.
func (r *GenerateResponse) ExpiresAt(location *time.Location) (time.Time, bool) {
	if location == nil {
		location = time.FixedZone("WIB", 7*60*60)
	}
	t, err := time.ParseInLocation("20060102150405", r.AdditionalInfo.ExpireDate, location)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// Generate creates a dynamic QRIS MPM code.
func (c *Client) Generate(ctx context.Context, creds Credentials, req GenerateRequest) (*GenerateResponse, error) {
	req.MerchantID = creds.MerchantID
	if req.Amount.Currency == "" {
		req.Amount.Currency = "IDR"
	}
	if req.AdditionalInfo.PaymentID == "" {
		req.AdditionalInfo.PaymentID = PaymentID
	}
	if req.AdditionalInfo.DynamicAmount == "" {
		req.AdditionalInfo.DynamicAmount = "N"
	}

	externalID := NewExternalID()
	var resp GenerateResponse
	status, err := c.post(ctx, creds, PathGenerate, externalID, false, req, &resp)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK || !isSuccess(resp.ResponseCode) || resp.QRContent == "" {
		return nil, &APIError{HTTPStatus: status, ResponseCode: resp.ResponseCode, ResponseMessage: resp.ResponseMessage}
	}
	resp.ExternalID = externalID
	return &resp, nil
}

// QueryRequest is the body of the query payment API.
type QueryRequest struct {
	OriginalReferenceNo        string              `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string              `json:"originalPartnerReferenceNo"`
	OriginalExternalID         string              `json:"originalExternalId,omitempty"`
	ServiceCode                string              `json:"serviceCode"`
	MerchantID                 string              `json:"merchantId"`
	AdditionalInfo             QueryAdditionalInfo `json:"additionalInfo"`
}

type QueryAdditionalInfo struct {
	Currency string `json:"currency"`
}

// QueryResponse is the query payment response.
type QueryResponse struct {
	ResponseCode               string  `json:"responseCode"`
	ResponseMessage            string  `json:"responseMessage"`
	OriginalReferenceNo        string  `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string  `json:"originalPartnerReferenceNo"`
	OriginalExternalID         string  `json:"originalExternalId"`
	ServiceCode                string  `json:"serviceCode"`
	LatestTransactionStatus    string  `json:"latestTransactionStatus"`
	TransactionStatusDesc      string  `json:"transactionStatusDesc"`
	PaidTime                   string  `json:"paidTime"`
	Amount                     *Amount `json:"amount,omitempty"`
	FeeAmount                  *Amount `json:"feeAmount,omitempty"`
	TerminalID                 string  `json:"terminalId"`
}

// Query fetches the latest status of a generated QR payment. A transaction
// Manjo does not know is reported as status "07" (not found), not as an error.
func (c *Client) Query(ctx context.Context, creds Credentials, req QueryRequest) (*QueryResponse, error) {
	req.MerchantID = creds.MerchantID
	req.ServiceCode = PaymentID
	if req.AdditionalInfo.Currency == "" {
		req.AdditionalInfo.Currency = "IDR"
	}

	var resp QueryResponse
	status, err := c.post(ctx, creds, PathQuery, NewExternalID(), true, req, &resp)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound && resp.LatestTransactionStatus != "" {
		return &resp, nil
	}
	if status != http.StatusOK || !isSuccess(resp.ResponseCode) {
		return nil, &APIError{HTTPStatus: status, ResponseCode: resp.ResponseCode, ResponseMessage: resp.ResponseMessage}
	}
	return &resp, nil
}

// RefundRequest is the body of the refund API.
type RefundRequest struct {
	MerchantID                 string               `json:"merchantId"`
	ExternalStoreID            string               `json:"externalStoreId,omitempty"`
	OriginalPartnerReferenceNo string               `json:"originalPartnerReferenceNo"`
	OriginalReferenceNo        string               `json:"originalReferenceNo"`
	OriginalExternalID         string               `json:"originalExternalId"`
	PartnerRefundNo            string               `json:"partnerRefundNo"`
	RefundAmount               Amount               `json:"refundAmount"`
	Reason                     string               `json:"reason"`
	AdditionalInfo             RefundAdditionalInfo `json:"additionalInfo"`
}

type RefundAdditionalInfo struct {
	PaymentID string `json:"paymentId"`
}

// RefundResponse is the refund response.
type RefundResponse struct {
	ResponseCode               string  `json:"responseCode"`
	ResponseMessage            string  `json:"responseMessage"`
	OriginalPartnerReferenceNo string  `json:"originalPartnerReferenceNo"`
	OriginalReferenceNo        string  `json:"originalReferenceNo"`
	OriginalExternalID         string  `json:"originalExternalId"`
	RefundNo                   string  `json:"refundNo"`
	PartnerRefundNo            string  `json:"partnerRefundNo"`
	RefundAmount               *Amount `json:"refundAmount,omitempty"`
	RefundTime                 string  `json:"refundTime"`

	// InProgress is true for HTTP 202 "Request In Progress": Manjo accepted the
	// refund but has not finished it.
	InProgress bool `json:"-"`
}

// Refund refunds a paid QR payment, fully or partially.
func (c *Client) Refund(ctx context.Context, creds Credentials, req RefundRequest) (*RefundResponse, error) {
	req.MerchantID = creds.MerchantID
	req.AdditionalInfo.PaymentID = PaymentID
	if req.RefundAmount.Currency == "" {
		req.RefundAmount.Currency = "IDR"
	}

	var resp RefundResponse
	status, err := c.post(ctx, creds, PathRefund, NewExternalID(), false, req, &resp)
	if err != nil {
		return nil, err
	}
	switch {
	case status == http.StatusOK && isSuccess(resp.ResponseCode):
		return &resp, nil
	case status == http.StatusAccepted:
		resp.InProgress = true
		return &resp, nil
	default:
		return nil, &APIError{HTTPStatus: status, ResponseCode: resp.ResponseCode, ResponseMessage: resp.ResponseMessage}
	}
}

// post sends a signed transactional request. On 401 the cached token is
// dropped and the request retried once with a new token.
func (c *Client) post(ctx context.Context, creds Credentials, path, externalID string, withClientKey bool, body, out interface{}) (int, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return 0, fmt.Errorf("manjo: encode request: %w", err)
	}

	for attempt := 0; ; attempt++ {
		token, err := c.accessToken(ctx, creds)
		if err != nil {
			return 0, err
		}

		timestamp := c.timestamp()
		signature, err := SignRequest(creds.ClientSecret, http.MethodPost, path, token, payload, timestamp)
		if err != nil {
			return 0, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return 0, fmt.Errorf("manjo: build request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-TIMESTAMP", timestamp)
		req.Header.Set("X-SIGNATURE", signature)
		req.Header.Set("X-PARTNER-ID", creds.partnerID())
		req.Header.Set("X-EXTERNAL-ID", externalID)
		req.Header.Set("X-IP-ADDRESS", c.ipAddress)
		req.Header.Set("CHANNEL-ID", ChannelID)
		if withClientKey {
			req.Header.Set("X-CLIENT-KEY", creds.ClientID)
		}

		status, err := c.do(req, out)
		if err != nil {
			return 0, err
		}
		if status == http.StatusUnauthorized && attempt == 0 {
			c.dropToken(creds.ClientID)
			continue
		}
		return status, nil
	}
}

func (c *Client) do(req *http.Request, out interface{}) (int, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("manjo: %s %s: %w", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, fmt.Errorf("manjo: read response: %w", err)
	}
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, out); err != nil {
			return resp.StatusCode, fmt.Errorf("manjo: decode %s response (HTTP %d): %w", req.URL.Path, resp.StatusCode, err)
		}
	}
	return resp.StatusCode, nil
}

func (c *Client) timestamp() string {
	return c.now().In(c.location).Format(timestampLayout)
}

// ParseTimestamp parses an X-TIMESTAMP header (yyyy-MM-ddTHH:mm:ssTZD).
func ParseTimestamp(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, strings.TrimSpace(value))
}

// Location is the Manjo timezone (WIB).
func (c *Client) Location() *time.Location {
	return c.location
}

func isSuccess(responseCode string) bool {
	return strings.HasPrefix(responseCode, "200")
}

// ---------------------------------------------------------------------------
// Signatures
// ---------------------------------------------------------------------------

// SignRequest computes the symmetric X-SIGNATURE of a transactional request:
//
//	stringToSign = METHOD:path:accessToken:lowercase(hex(sha256(minify(body)))):timestamp
//	signature    = hex(HMAC_SHA512(clientSecret, stringToSign))
func SignRequest(clientSecret, method, path, accessToken string, body []byte, timestamp string) (string, error) {
	minified, err := minifyJSON(body)
	if err != nil {
		return "", err
	}
	bodyHash := sha256.Sum256(minified)
	stringToSign := strings.Join([]string{
		method,
		path,
		accessToken,
		strings.ToLower(hex.EncodeToString(bodyHash[:])),
		timestamp,
	}, ":")
	return hmacSHA512Hex(clientSecret, stringToSign), nil
}

func minifyJSON(body []byte) ([]byte, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return []byte{}, nil
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, body); err != nil {
		return nil, fmt.Errorf("manjo: minify body: %w", err)
	}
	return buf.Bytes(), nil
}

// NewExternalID returns a random X-EXTERNAL-ID: 32 hex characters, unique per day
// as Manjo requires.
func NewExternalID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand does not fail on supported platforms; fall back to the clock.
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(b[:])
}
