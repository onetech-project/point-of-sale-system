package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"github.com/pos/notification-service/src/queue"
)

// This integration test publishes an `order_paid` event to Kafka and checks
// that a notification record is created and a redis stream entry is present.
func TestOrderPaidIntegration(t *testing.T) {
	brokers := os.Getenv("KAFKA_BROKERS")
	dsn := os.Getenv("DATABASE_URL")
	redisHost := os.Getenv("REDIS_HOST")

	if brokers == "" || dsn == "" || redisHost == "" {
		t.Skip("KAFKA_BROKERS, DATABASE_URL or REDIS_HOST not set; skipping integration test")
	}

	brokerList := strings.Split(brokers, ",")
	producer := queue.NewKafkaProducer(brokerList, "orders.events")
	defer producer.Close()

	// Build event payload
	event := map[string]interface{}{
		"event_id":   "itest-" + time.Now().Format("20060102150405"),
		"event_type": "order_paid",
		"order_id":   "itest-order-1",
		"tenant_id":  "tenant:demo",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"payload": map[string]interface{}{
			"reference":    "REF-IT-1",
			"total_amount": 12345,
		},
	}

	ctx := context.Background()
	if err := producer.Publish(ctx, event["event_id"].(string), event); err != nil {
		t.Fatalf("failed to publish kafka message: %v", err)
	}

	// Wait and poll DB for notification record
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	found := false
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		var id string
		// metadata->>'order_id' may be inside metadata JSON; try to match reference instead
		err := db.QueryRowContext(ctx, `SELECT id FROM notifications WHERE metadata->>'order_id' = $1 LIMIT 1`, "itest-order-1").Scan(&id)
		if err == nil {
			found = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !found {
		t.Fatalf("notification record for order not found in DB after timeout")
	}

	// Check redis stream for tenant
	rclient := redis.NewClient(&redis.Options{Addr: redisHost})
	defer rclient.Close()

	stream := "tenant:tenant:demo:stream"
	// Retry reading the redis stream for a short window to handle timing/race conditions.
	foundRedis := false
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		msgs, err := rclient.XRevRangeN(ctx, stream, "+", "-", 20).Result()
		if err == nil {
			for _, m := range msgs {
				for _, v := range m.Values {
					switch vv := v.(type) {
					case string:
						var js map[string]interface{}
						if err := json.Unmarshal([]byte(vv), &js); err == nil {
							if data, ok := js["data"].(map[string]interface{}); ok {
								if oid, ok := data["order_id"].(string); ok && oid == "itest-order-1" {
									foundRedis = true
									break
								}
							}
						}
					}
				}
				if foundRedis {
					break
				}
			}
		}
		if foundRedis {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if !foundRedis {
		t.Fatalf("expected redis stream to contain our event")
	}
}
