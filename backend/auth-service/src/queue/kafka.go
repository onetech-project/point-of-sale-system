package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader  *kafka.Reader
	handler func(context.Context, []byte) error
}

func NewKafkaConsumer(brokers []string, topic string, groupID string, handler func(context.Context, []byte) error) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e1,  // 100B
		MaxBytes:       10e6,  // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})

	return &KafkaConsumer{
		reader:  reader,
		handler: handler,
	}
}

func (c *KafkaConsumer) Start(ctx context.Context) {
	log.Printf("Starting Kafka consumer for topic: %s", c.reader.Config().Topic)

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down Kafka consumer...")
			c.reader.Close()
			return
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			log.Printf("Received message: topic=%s partition=%d offset=%d",
				msg.Topic, msg.Partition, msg.Offset)

			if err := c.handler(ctx, msg.Value); err != nil {
				log.Printf("Error handling message: %v", err)
				// Don't commit on error - will be reprocessed
				continue
			}
		}
	}
}

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}

// KafkaProducer for publishing events
type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		Async:                  false,
	}

	return &KafkaProducer{writer: writer}
}

func (p *KafkaProducer) Publish(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
	}

	return p.writer.WriteMessages(ctx, msg)
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
