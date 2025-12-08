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

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
