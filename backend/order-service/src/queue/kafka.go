package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

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
	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("ERROR: Failed to marshal Kafka message: %v", err)
		return err
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
	}

	log.Printf("DEBUG: Publishing message to Kafka - Topic: %s, Key: %s, Size: %d bytes",
		p.writer.Topic, key, len(data))

	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		log.Printf("ERROR: Failed to write message to Kafka: %v", err)
	} else {
		log.Printf("DEBUG: Message successfully written to Kafka")
	}

	return err
}

// PublishWithHeaders publishes a message with custom headers
func (p *KafkaProducer) PublishWithHeaders(ctx context.Context, key string, value interface{}, headers []kafka.Header) error {
	var data []byte
	var err error

	// Check if value is already marshaled ([]byte)
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

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
