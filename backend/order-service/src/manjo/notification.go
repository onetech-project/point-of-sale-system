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
//	HMAC_SHA512(key, POST:path:accessToken:lowercase(hex(sha256(minify(body)))):X-TIMESTAMP)
//
// The docs (v1.1) say the key is the X-PARTNER-ID value, while the generic
// formula names the client secret, and the path is either the path Manjo called
// or Manjo's own /v1.0/qr/qr-mpm-notify. All those combinations are accepted.
// The access token is the Authorization bearer token, which is optional.
func VerifyNotificationSignature(header http.Header, requestPath string, body []byte, clientSecret string) bool {
	signature := strings.TrimSpace(header.Get("X-SIGNATURE"))
	timestamp := header.Get("X-TIMESTAMP")
	if signature == "" || timestamp == "" {
		return false
	}
	accessToken := strings.TrimSpace(strings.TrimPrefix(header.Get("Authorization"), "Bearer"))

	keys := []string{}
	if clientSecret != "" {
		keys = append(keys, clientSecret)
	}
	if partnerID := header.Get("X-PARTNER-ID"); partnerID != "" {
		keys = append(keys, partnerID)
	}
	paths := []string{requestPath}
	if requestPath != PathNotify {
		paths = append(paths, PathNotify)
	}

	for _, key := range keys {
		for _, path := range paths {
			expected, err := SignRequest(key, http.MethodPost, path, accessToken, body, timestamp)
			if err != nil {
				return false
			}
			if hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected)) {
				return true
			}
		}
	}
	return false
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

var (
	AckSuccess      = NotificationAck{ResponseCode: "2005200", ResponseMessage: "Successful"}
	AckInvalidBody  = NotificationAck{ResponseCode: "4005202", ResponseMessage: "Invalid Mandatory Field"}
	AckUnauthorized = NotificationAck{ResponseCode: "4015200", ResponseMessage: "Unauthorized. Invalid signature"}
)
