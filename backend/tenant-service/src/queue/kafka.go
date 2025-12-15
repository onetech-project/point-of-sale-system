package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaProducer for publishing events
type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		RequiredAcks:           kafka.RequireOne,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
	}

	return &KafkaProducer{writer: writer}
}

func (p *KafkaProducer) Publish(
	ctx context.Context, topic string, key string, value interface{}) error {
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

type KafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(
	brokers []string,
	groupID string,
	topics []string,
) *KafkaConsumer {

	return &KafkaConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			GroupID:     groupID,
			GroupTopics: topics,
		}),
	}
}

func (k *KafkaConsumer) Start(
	ctx context.Context,
	handler func(context.Context, kafka.Message) error,
) {
	for {
		msg, err := k.reader.FetchMessage(ctx)
		if err != nil {
			return
		}

		if err := handler(ctx, msg); err == nil {
			k.reader.CommitMessages(ctx, msg)
		}
	}
}

func (k *KafkaConsumer) Stop() error {
	return k.reader.Close()
}
