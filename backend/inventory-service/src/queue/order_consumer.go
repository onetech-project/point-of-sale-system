package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pos/backend/inventory-service/src/models"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

type OrderConsumerConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

type OrderConsumptionProcessor interface {
	ProcessFulfilledOrder(ctx context.Context, event models.OrderLifecycleEvent) error
}

type OrderConsumer struct {
	reader    *kafka.Reader
	processor OrderConsumptionProcessor
}

func NewOrderConsumer(config OrderConsumerConfig, processor OrderConsumptionProcessor) *OrderConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cleanBrokers(config.Brokers),
		Topic:          config.Topic,
		GroupID:        config.GroupID,
		StartOffset:    kafka.FirstOffset,
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        500 * time.Millisecond,
		CommitInterval: 0,
	})

	return &OrderConsumer{
		reader:    reader,
		processor: processor,
	}
}

func (c *OrderConsumer) Start(ctx context.Context) {
	log.Info().
		Str("topic", c.reader.Config().Topic).
		Str("group_id", c.reader.Config().GroupID).
		Msg("Order consumer started")

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Error().Err(err).Msg("Failed to fetch order event")
			time.Sleep(time.Second)
			continue
		}

		if err := c.processMessage(ctx, msg); err != nil {
			log.Error().
				Err(err).
				Int("partition", msg.Partition).
				Int64("offset", msg.Offset).
				Msg("Failed to process order event; offset will be retried")
			time.Sleep(2 * time.Second)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Error().Err(err).Msg("Failed to commit order event offset")
		}
	}
}

func (c *OrderConsumer) Close() error {
	return c.reader.Close()
}

func (c *OrderConsumer) processMessage(ctx context.Context, msg kafka.Message) error {
	eventType := eventTypeFromHeaders(msg.Headers)
	if eventType == "" {
		var envelope struct {
			EventType string `json:"event_type"`
		}
		if err := json.Unmarshal(msg.Value, &envelope); err != nil {
			log.Error().
				Err(err).
				Str("raw_message", string(msg.Value)).
				Msg("Skipping malformed order event")
			return nil
		}
		eventType = envelope.EventType
	}

	if eventType != models.OrderFulfilledEventType {
		return nil
	}

	var event models.OrderLifecycleEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Error().
			Err(err).
			Str("raw_message", string(msg.Value)).
			Msg("Skipping malformed order.fulfilled event")
		return nil
	}
	if event.EventType == "" {
		event.EventType = eventType
	}

	if err := c.processor.ProcessFulfilledOrder(ctx, event); err != nil {
		return fmt.Errorf("failed to consume fulfilled order %s: %w", event.OrderID, err)
	}
	return nil
}

func eventTypeFromHeaders(headers []kafka.Header) string {
	for i := range headers {
		if strings.EqualFold(headers[i].Key, "event-type") {
			return string(headers[i].Value)
		}
	}
	return ""
}

func cleanBrokers(brokers []string) []string {
	cleaned := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		broker = strings.TrimSpace(broker)
		if broker != "" {
			cleaned = append(cleaned, broker)
		}
	}
	return cleaned
}
