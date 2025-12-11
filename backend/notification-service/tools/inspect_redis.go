package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

func main() {
	ctx := context.Background()
	addr := os.Getenv("REDIS_HOST")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()

	stream := "tenant:tenant:demo:stream"
	fmt.Printf("Reading stream: %s\n", stream)
	msgs, err := client.XRevRangeN(ctx, stream, "+", "-", 20).Result()
	if err != nil {
		fmt.Printf("XRevRangeN error: %v\n", err)
		return
	}

	fmt.Printf("Found %d messages:\n", len(msgs))
	for i, m := range msgs {
		fmt.Printf("Message %d ID=%s\n", i, m.ID)
		for k, v := range m.Values {
			fmt.Printf("  %s (%T) = %v\n", k, v, v)
		}
	}

	// Sleep briefly to allow manual re-run if desired
	time.Sleep(100 * time.Millisecond)
}
