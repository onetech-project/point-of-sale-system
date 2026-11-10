package manjo

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
)

// Notification is the QR MPM payment notification Manjo posts to the merchant.
//
// The docs name the status field both transactionStatus and
// latestTransactionStatus and give two different status code tables, so the
// notification status is not trusted: the receiver confirms the payment with
// the query API (see QueryStatus) before acting on it.
type Notification struct {
	OriginalReferenceNo        string  `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string  `json:"originalPartnerReferenceNo"`
	LatestTransactionStatus    string  `json:"latestTransactionStatus"`
	TransactionStatus          string  `json:"transactionStatus"`
	TransactionStatusDesc      string  `json:"transactionStatusDesc"`
	Amount                     *Amount `json:"amount,omitempty"`
	AdditionalInfo             struct {
		MerchantCode string `json:"merchantCode"`
		RRN          string `json:"rrn"`
		IssuerName   string `json:"issuerName"`
	} `json:"additionalInfo"`
}

// ParseNotification decodes a notification body.
func ParseNotification(body []byte) (*Notification, error) {
	var n Notification
	if err := json.Unmarshal(body, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

// VerifyNotificationSignature checks the X-SIGNATURE of a notification:
//
//	HMAC_SHA512(clientSecret, POST:path:accessToken:lowercase(hex(sha256(minify(body)))):X-TIMESTAMP)
//
// accessToken is the bearer token Manjo got from our access token endpoint.
// The path Manjo signs is either the full path it called or the relative
// /v1.0/qr/qr-mpm-notify of the docs; both are accepted, keyed by the secret.
func VerifyNotificationSignature(header http.Header, requestPath string, body []byte, clientSecret string) bool {
	signature := strings.ToLower(strings.TrimSpace(header.Get("X-SIGNATURE")))
	timestamp := header.Get("X-TIMESTAMP")
	if signature == "" || timestamp == "" || clientSecret == "" {
		return false
	}
	accessToken := BearerToken(header)

	paths := []string{requestPath}
	if requestPath != PathNotify {
		paths = append(paths, PathNotify)
	}
	for _, path := range paths {
		expected, err := SignRequest(clientSecret, http.MethodPost, path, accessToken, body, timestamp)
		if err != nil {
			return false
		}
		if hmac.Equal([]byte(signature), []byte(expected)) {
			return true
		}
	}
	return false
}

// BearerToken returns the token of an "Authorization: Bearer <token>" header.
func BearerToken(header http.Header) string {
	value := strings.TrimSpace(header.Get("Authorization"))
	if len(value) > 7 && strings.EqualFold(value[:7], "Bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return ""
}

func hmacSHA512Hex(key, message string) string {
	mac := hmac.New(sha512.New, []byte(key))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// Payment statuses of the query API.
const (
	QueryStatusSuccess   = "00"
	QueryStatusInitiated = "01"
	QueryStatusPaying    = "02"
	QueryStatusPending   = "03"
	QueryStatusRefunded  = "04"
	QueryStatusCanceled  = "05"
	QueryStatusFailed    = "06"
	QueryStatusNotFound  = "07"
)

// Payment statuses stored in payment_transactions.transaction_status, shared
// with Midtrans so both gateways read the same.
const (
	StatusSettlement = "settlement"
	StatusPending    = "pending"
	StatusRefund     = "refund"
	StatusCancel     = "cancel"
	StatusFailure    = "failure"
	StatusExpire     = "expire"
)

// StatusFromQuery maps a query API latestTransactionStatus to the stored status.
// Not found is treated as pending: the QR may simply not be paid yet.
func StatusFromQuery(code string) string {
	switch code {
	case QueryStatusSuccess:
		return StatusSettlement
	case QueryStatusRefunded:
		return StatusRefund
	case QueryStatusCanceled:
		return StatusCancel
	case QueryStatusFailed:
		return StatusFailure
	default:
		return StatusPending
	}
}

// NotificationAck is the body returned to Manjo for a notification.
type NotificationAck struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
}

// Notification answers (service code 52).
var (
	AckSuccess      = NotificationAck{ResponseCode: "2005200", ResponseMessage: "Successful"}
	AckInvalidBody  = NotificationAck{ResponseCode: "4005202", ResponseMessage: "Invalid Mandatory Field"}
	AckUnauthorized = NotificationAck{ResponseCode: "4015200", ResponseMessage: "Unauthorized. Invalid signature"}
	AckInvalidToken = NotificationAck{ResponseCode: "4015201", ResponseMessage: "Invalid Token (B2B)"}
	AckConflict     = NotificationAck{ResponseCode: "4095200", ResponseMessage: "Conflict"}
)

// AccessTokenResponse answers an access token request Manjo sends to the
// merchant (service code 73).
type AccessTokenResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	AccessToken     string `json:"accessToken,omitempty"`
	TokenType       string `json:"tokenType,omitempty"`
	ExpiresIn       string `json:"expiresIn,omitempty"`
}

var (
	TokenInvalidFormat    = AccessTokenResponse{ResponseCode: "4007301", ResponseMessage: "Invalid Field Format"}
	TokenInvalidMandatory = AccessTokenResponse{ResponseCode: "4007302", ResponseMessage: "Invalid Mandatory Field"}
	TokenUnauthorized     = AccessTokenResponse{ResponseCode: "4017300", ResponseMessage: "Unauthorized. Invalid signature"}
	TokenGeneralError     = AccessTokenResponse{ResponseCode: "5007300", ResponseMessage: "General Error"}
)
