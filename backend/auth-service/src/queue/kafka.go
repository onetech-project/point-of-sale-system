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
		MinBytes:       10e1, // 100B
		MaxBytes:       10e6, // 10MB
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

// KafkaProducerConfig holds configuration for Kafka producer
type KafkaProducerConfig struct {
	Brokers              []string
	Topic                string
	Balancer             kafka.Balancer
	MaxAttempts          int
	RequiredAcks         kafka.RequiredAcks
	Async                bool
	Compression          kafka.Compression
	AllowAutoTopicCreate bool
}

// NewKafkaProducer creates a Kafka producer with default configuration
func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	config := KafkaProducerConfig{
		Brokers:              brokers,
		Topic:                topic,
		Balancer:             &kafka.LeastBytes{},
		MaxAttempts:          3,
		RequiredAcks:         kafka.RequireOne,
		Async:                false,
		Compression:          kafka.Snappy,
		AllowAutoTopicCreate: true,
	}
	return NewKafkaProducerWithConfig(config)
}

// NewKafkaProducerWithConfig creates a Kafka producer with custom configuration
func NewKafkaProducerWithConfig(config KafkaProducerConfig) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(config.Brokers...),
		Topic:                  config.Topic,
		Balancer:               config.Balancer,
		MaxAttempts:            config.MaxAttempts,
		RequiredAcks:           config.RequiredAcks,
		Async:                  config.Async,
		Compression:            config.Compression,
		AllowAutoTopicCreation: config.AllowAutoTopicCreate,
	}

	return &KafkaProducer{writer: writer}
}

// Publish publishes a single message to Kafka
func (p *KafkaProducer) Publish(ctx context.Context, key string, value interface{}) error {
	var data []byte
	var err error

	// If value is already []byte, use it directly (avoid double marshaling)
	if b, ok := value.([]byte); ok {
		data = b
	} else {
		data, err = json.Marshal(value)
		if err != nil {
			return err
		}
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
	}

	return p.writer.WriteMessages(ctx, msg)
}

// PublishWithHeaders publishes a message with custom headers
func (p *KafkaProducer) PublishWithHeaders(ctx context.Context, key string, value interface{}, headers []kafka.Header) error {
	var data []byte
	var err error

	// If value is already []byte, use it directly (avoid double marshaling)
	if b, ok := value.([]byte); ok {
		data = b
	} else {
		data, err = json.Marshal(value)
		if err != nil {
			return err
		}
	}

	msg := kafka.Message{
		Key:     []byte(key),
		Value:   data,
		Time:    time.Now(),
		Headers: headers,
	}

	return p.writer.WriteMessages(ctx, msg)
}

// PublishBatch publishes multiple messages in a single batch
func (p *KafkaProducer) PublishBatch(ctx context.Context, messages []kafka.Message) error {
	return p.writer.WriteMessages(ctx, messages...)
}

// Close closes the Kafka writer
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
