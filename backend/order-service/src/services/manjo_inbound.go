package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/point-of-sale-system/order-service/src/manjo"
)

// Manjo calls the merchant in two steps (docs: API Payment Notification, "Flow
// for Notification"): it first asks our /v1.0/access-token/b2b for a token,
// signing with SHA256withRSA that we verify with Manjo's public key, then posts
// the notification to /v1.0/qr/qr-mpm-notify with that token as bearer.

const (
	// manjoInboundTokenTTL is the lifetime of tokens we issue to Manjo.
	manjoInboundTokenTTL = 15 * time.Minute
	// manjoTimestampSkew bounds how far an access token request's X-TIMESTAMP
	// may be from our clock, so a captured request cannot be replayed later.
	manjoTimestampSkew = 5 * time.Minute
	// manjoExternalIDTTL keeps notification X-EXTERNAL-IDs past the day they
	// must be unique in.
	manjoExternalIDTTL = 25 * time.Hour
)

// ManjoTokenHolder is who an inbound access token was issued to.
type ManjoTokenHolder struct {
	ClientID    string `json:"client_id"`
	Environment string `json:"environment"`
}

// manjoTokenStore keeps inbound access tokens and notification external IDs.
type manjoTokenStore interface {
	SaveToken(ctx context.Context, token string, holder ManjoTokenHolder, ttl time.Duration) error
	// Token returns nil when the token is unknown or expired.
	Token(ctx context.Context, token string) (*ManjoTokenHolder, error)
	// ClaimExternalID records an X-EXTERNAL-ID and reports whether it was new.
	ClaimExternalID(ctx context.Context, scope, externalID string, ttl time.Duration) (bool, error)
}

// WithManjoInbound enables Manjo's calls to us: publicKeys maps an environment
// ("sandbox", "production") to the Manjo public key that signs its access
// token requests there.
func (s *PaymentService) WithManjoInbound(store manjoTokenStore, publicKeys map[string]*rsa.PublicKey) *PaymentService {
	s.manjoTokens = store
	s.manjoPublicKeys = publicKeys
	return s
}

// IssueManjoAccessToken answers Manjo's POST /v1.0/access-token/b2b. It returns
// the HTTP status and body to send.
func (s *PaymentService) IssueManjoAccessToken(ctx context.Context, header http.Header, body []byte) (int, manjo.AccessTokenResponse) {
	clientKey := strings.TrimSpace(header.Get("X-CLIENT-KEY"))
	timestamp := strings.TrimSpace(header.Get("X-TIMESTAMP"))
	signature := strings.TrimSpace(header.Get("X-SIGNATURE"))
	if clientKey == "" || timestamp == "" || signature == "" {
		return http.StatusBadRequest, manjo.TokenInvalidMandatory
	}

	var req struct {
		GrantType string `json:"grantType"`
	}
	if err := json.Unmarshal(body, &req); err != nil || req.GrantType == "" {
		return http.StatusBadRequest, manjo.TokenInvalidMandatory
	}
	if req.GrantType != "client_credentials" {
		return http.StatusBadRequest, manjo.TokenInvalidFormat
	}

	sentAt, err := manjo.ParseTimestamp(timestamp)
	if err != nil {
		return http.StatusBadRequest, manjo.TokenInvalidFormat
	}
	if skew := s.clock().Sub(sentAt); skew > manjoTimestampSkew || skew < -manjoTimestampSkew {
		log.Warn().Str("client_key", clientKey).Str("timestamp", timestamp).Msg("Manjo access token request outside the allowed clock skew")
		return http.StatusUnauthorized, manjo.TokenUnauthorized
	}

	environment := ""
	for env, key := range s.manjoPublicKeys {
		if manjo.VerifyAccessTokenSignature(key, clientKey, timestamp, signature) {
			environment = env
			break
		}
	}
	if environment == "" {
		log.Warn().Str("client_key", clientKey).Msg("Invalid Manjo access token request signature")
		return http.StatusUnauthorized, manjo.TokenUnauthorized
	}
	if s.manjoTokens == nil {
		log.Error().Msg("Manjo access token requested but no token store is configured")
		return http.StatusInternalServerError, manjo.TokenGeneralError
	}

	token, err := newManjoInboundToken()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate Manjo access token")
		return http.StatusInternalServerError, manjo.TokenGeneralError
	}
	holder := ManjoTokenHolder{ClientID: clientKey, Environment: environment}
	if err := s.manjoTokens.SaveToken(ctx, token, holder, manjoInboundTokenTTL); err != nil {
		log.Error().Err(err).Msg("Failed to store Manjo access token")
		return http.StatusInternalServerError, manjo.TokenGeneralError
	}

	log.Info().Str("client_key", clientKey).Str("environment", environment).Msg("Issued Manjo access token")
	return http.StatusOK, manjo.AccessTokenResponse{
		ResponseCode:    "2007300",
		ResponseMessage: "Successful",
		AccessToken:     token,
		TokenType:       "Bearer",
		ExpiresIn:       strconv.Itoa(int(manjoInboundTokenTTL.Seconds())),
	}
}

// manjoTokenHolder returns who the notification's bearer token was issued to,
// or nil when the token is missing, unknown or expired.
func (s *PaymentService) manjoTokenHolder(ctx context.Context, header http.Header) (*ManjoTokenHolder, error) {
	token := manjo.BearerToken(header)
	if token == "" || s.manjoTokens == nil {
		return nil, nil
	}
	return s.manjoTokens.Token(ctx, token)
}

// claimManjoExternalID enforces Manjo's "X-EXTERNAL-ID unique per day" rule.
func (s *PaymentService) claimManjoExternalID(ctx context.Context, clientID, externalID string) (bool, error) {
	day := s.clock().In(manjoLocation).Format("20060102")
	return s.manjoTokens.ClaimExternalID(ctx, clientID+":"+day, externalID, manjoExternalIDTTL)
}

func (s *PaymentService) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

var manjoLocation = func() *time.Location {
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return location
}()

func newManjoInboundToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// RedisManjoTokenStore keeps Manjo inbound tokens in Redis. Tokens are stored
// under their SHA-256, so the keys themselves are not usable tokens.
type RedisManjoTokenStore struct {
	rdb *redis.Client
}

func NewRedisManjoTokenStore(rdb *redis.Client) *RedisManjoTokenStore {
	return &RedisManjoTokenStore{rdb: rdb}
}

func manjoTokenKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "manjo:inbound_token:" + hex.EncodeToString(sum[:])
}

func (r *RedisManjoTokenStore) SaveToken(ctx context.Context, token string, holder ManjoTokenHolder, ttl time.Duration) error {
	value, err := json.Marshal(holder)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, manjoTokenKey(token), value, ttl).Err()
}

func (r *RedisManjoTokenStore) Token(ctx context.Context, token string) (*ManjoTokenHolder, error) {
	value, err := r.rdb.Get(ctx, manjoTokenKey(token)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read Manjo token: %w", err)
	}
	var holder ManjoTokenHolder
	if err := json.Unmarshal(value, &holder); err != nil {
		return nil, fmt.Errorf("failed to decode Manjo token: %w", err)
	}
	return &holder, nil
}

func (r *RedisManjoTokenStore) ClaimExternalID(ctx context.Context, scope, externalID string, ttl time.Duration) (bool, error) {
	return r.rdb.SetNX(ctx, "manjo:notify_external_id:"+scope+":"+externalID, 1, ttl).Result()
}
