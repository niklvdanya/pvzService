package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"
)

type KafkaProducer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewKafkaProducer(brokers []string, topic string) (*KafkaProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Flush.Messages = 1
	config.Producer.Flush.Frequency = 10 * time.Millisecond
	config.Producer.Partitioner = func(topic string) sarama.Partitioner {
		return sarama.NewHashPartitioner(topic)
	}

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("create sync producer: %w", err)
	}

	return &KafkaProducer{
		producer: producer,
		topic:    topic,
	}, nil
}

func (p *KafkaProducer) Send(ctx context.Context, message []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	_, _, err := p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: p.topic,
		Value: sarama.ByteEncoder(message),
	})
	if err != nil {
		return fmt.Errorf("send message to kafka: %w", err)
	}
	return nil
}

func (p *KafkaProducer) Close() error {
	if err := p.producer.Close(); err != nil {
		return fmt.Errorf("close producer: %w", err)
	}
	return nil
}
