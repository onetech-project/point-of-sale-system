package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pos/notification-service/src/providers"
)

// RedisProvider holds the provider used by handlers (injected from main)
var RedisProvider *providers.RedisProvider

// SSEAuthMiddleware validates a simple HMAC-signed JWT in Authorization header.
// Test-first implementation: verifies signature using TEST_JWT_SECRET (or JWT_SECRET),
// extracts tenant_id from payload, and enforces it matches ALLOWED_TENANT (default "tenant:demo").
func SSEAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		auth := c.Request().Header.Get("Authorization")
		if auth == "" {
			return c.NoContent(http.StatusUnauthorized)
		}
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			return c.NoContent(http.StatusUnauthorized)
		}
		token := strings.TrimPrefix(auth, prefix)

		parts := strings.SplitN(token, ".", 3)
		if len(parts) != 3 {
			return c.NoContent(http.StatusUnauthorized)
		}

		hdrPayload := parts[0] + "." + parts[1]
		sig := parts[2]

		secret := os.Getenv("TEST_JWT_SECRET")
		if secret == "" {
			secret = os.Getenv("JWT_SECRET")
		}
		if secret == "" {
			secret = "test-secret"
		}

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(hdrPayload))
		expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
		if sig != expected {
			return c.NoContent(http.StatusUnauthorized)
		}

		payloadB, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return c.NoContent(http.StatusUnauthorized)
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(payloadB, &payload); err != nil {
			return c.NoContent(http.StatusUnauthorized)
		}
		tenantID, _ := payload["tenant_id"].(string)
		allowed := os.Getenv("ALLOWED_TENANT")
		if allowed == "" {
			allowed = "tenant:demo"
		}
		if tenantID != allowed {
			return c.NoContent(http.StatusForbidden)
		}

		c.Set("tenant_id", tenantID)
		c.Set("claims", payload)
		return next(c)
	}
}

// SSEHandler streams Redis tenant stream entries as Server-Sent Events.
// If RedisProvider is not set or unavailable it falls back to a heartbeat/snapshot message.
func SSEHandler(c echo.Context) error {
	r := c.Response()
	r.Header().Set(echo.HeaderContentType, "text/event-stream")
	r.Header().Set("Cache-Control", "no-cache")
	r.WriteHeader(http.StatusOK)

	ctx := c.Request().Context()
	tenantID := c.Request().Header.Get("X-Tenant-ID")

	if RedisProvider == nil || tenantID == "" {
		// Fallback behavior: instruct snapshot if Last-Event-ID provided
		lastEventID := c.Request().Header.Get("Last-Event-ID")
		if lastEventID != "" && tenantID != "" {
			snapshotURL := fmt.Sprintf("/api/orders/snapshot?tenant_id=%s", tenantID)
			payload := map[string]interface{}{"type": "resync", "method": "snapshot", "snapshot_url": snapshotURL}
			b, _ := json.Marshal(payload)
			if _, err := r.Write([]byte("data: " + string(b) + "\n\n")); err != nil {
				return err
			}
			if flusher, ok := r.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
			return nil
		}
		if _, err := r.Write([]byte("data: heartbeat\n\n")); err != nil {
			return err
		}
		if flusher, ok := r.Writer.(http.Flusher); ok {
			flusher.Flush()
		}
		return nil
	}

	stream := fmt.Sprintf("tenant:%s:stream", tenantID)
	lastID := c.Request().Header.Get("Last-Event-ID")
	if lastID == "" {
		lastID = "$"
	}

	// Determine connected user id (if forwarded by gateway or present in claims)
	connectedUser := c.Request().Header.Get("X-User-ID")
	if connectedUser == "" {
		if claims := c.Get("claims"); claims != nil {
			if mp, ok := claims.(map[string]interface{}); ok {
				if uid, _ := mp["user_id"].(string); uid != "" {
					connectedUser = uid
				}
			}
		}
	}

	var userStream string
	var lastIDUser string
	if connectedUser != "" {
		userStream = fmt.Sprintf("tenant:%s:user:%s:stream", tenantID, connectedUser)
		lastIDUser = c.Request().Header.Get("Last-Event-ID-User")
		if lastIDUser == "" {
			lastIDUser = "$"
		}
	}

	// Adaptive block duration for XREAD blocking to reduce CPU when idle
	blockDur := 5 * time.Second
	maxBlock := 60 * time.Second
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			msgs, err := RedisProvider.ReadStream(ctx, stream, lastID, blockDur)
			if err != nil {
				// write error event; increase backoff
				errPayload := map[string]string{"error": err.Error()}
				b, _ := json.Marshal(errPayload)
				r.Write([]byte("event: error\n"))
				r.Write([]byte("data: " + string(b) + "\n\n"))
				if flusher, ok := r.Writer.(http.Flusher); ok {
					flusher.Flush()
				}
				// exponential backoff on error
				blockDur = blockDur * 2
				if blockDur > maxBlock {
					blockDur = maxBlock
				}
				time.Sleep(500 * time.Millisecond)
				continue
			}

			if len(msgs) == 0 {
				// heartbeat; gradually increase blockDur to reduce polls when idle
				r.Write([]byte("data: heartbeat\n\n"))
				if flusher, ok := r.Writer.(http.Flusher); ok {
					flusher.Flush()
				}
				// increase block duration up to max
				if blockDur < maxBlock {
					blockDur = min(blockDur*2, maxBlock)
				}
				continue
			}

			// messages received; reset block duration
			blockDur = 5 * time.Second
			for _, m := range msgs {
				b, _ := json.Marshal(m.Values)
				r.Write([]byte("id: " + m.ID + "\n"))
				if ev, ok := m.Values["event"]; ok {
					r.Write([]byte("event: " + fmt.Sprint(ev) + "\n"))
				}
				r.Write([]byte("data: " + string(b) + "\n\n"))
				if flusher, ok := r.Writer.(http.Flusher); ok {
					flusher.Flush()
				}
				lastID = m.ID
			}

			// If there is a user-specific stream, also try reading it immediately
			if userStream != "" {
				userMsgs, uerr := RedisProvider.ReadStream(ctx, userStream, lastIDUser, 0)
				if uerr == nil && len(userMsgs) > 0 {
					for _, um := range userMsgs {
						ub, _ := json.Marshal(um.Values)
						r.Write([]byte("id: " + um.ID + "\n"))
						if ev, ok := um.Values["event"]; ok {
							r.Write([]byte("event: " + fmt.Sprint(ev) + "\n"))
						}
						r.Write([]byte("data: " + string(ub) + "\n\n"))
						if flusher, ok := r.Writer.(http.Flusher); ok {
							flusher.Flush()
						}
						lastIDUser = um.ID
					}
				}
			}
		}
	}
}
