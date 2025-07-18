package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
)

type KafkaConsumerConfig struct {
	Brokers         []string
	Topic           string
	ConsumerGroup   string
	AutoOffsetReset string
}

type KafkaConsumer struct {
	consumerGroup sarama.ConsumerGroup
	topic         string
	config        KafkaConsumerConfig
}

type MessageHandler interface {
	HandleMessage(ctx context.Context, message *sarama.ConsumerMessage) error
}

func NewKafkaConsumer(cfg KafkaConsumerConfig) (*KafkaConsumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.MaxVersion

	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	if cfg.AutoOffsetReset == "latest" {
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	config.Consumer.Group.ResetInvalidOffsets = true

	config.Consumer.Offsets.AutoCommit.Enable = true
	config.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second

	config.Consumer.Group.Session.Timeout = 60 * time.Second
	config.Consumer.Group.Heartbeat.Interval = 3 * time.Second
	config.Consumer.Group.Rebalance.Timeout = 60 * time.Second

	config.Consumer.Return.Errors = true

	consumerGroup, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.ConsumerGroup, config)
	if err != nil {
		return nil, fmt.Errorf("create consumer group: %w", err)
	}

	return &KafkaConsumer{
		consumerGroup: consumerGroup,
		topic:         cfg.Topic,
		config:        cfg,
	}, nil
}

func (c *KafkaConsumer) Consume(ctx context.Context, handler MessageHandler) error {
	consumer := &ConsumerGroupHandler{
		handler: handler,
		ready:   make(chan bool),
	}

	go func() {
		for err := range c.consumerGroup.Errors() {
			slog.Error("Consumer group error", "error", err)
		}
	}()

	for {
		if err := c.consumerGroup.Consume(ctx, []string{c.topic}, consumer); err != nil {
			slog.Error("Error from consumer", "error", err)
			return fmt.Errorf("consume error: %w", err)
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		slog.Info("Consumer session ended, restarting...")
		consumer.ready = make(chan bool)
	}
}

func (c *KafkaConsumer) Close() error {
	return c.consumerGroup.Close()
}

type ConsumerGroupHandler struct {
	handler MessageHandler
	ready   chan bool
}

func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			slog.Debug("Received message",
				"topic", message.Topic,
				"partition", message.Partition,
				"offset", message.Offset,
				"timestamp", message.Timestamp,
				"key", string(message.Key))

			if err := h.handler.HandleMessage(session.Context(), message); err != nil {
				slog.Error("Failed to handle message",
					"error", err,
					"topic", message.Topic,
					"partition", message.Partition,
					"offset", message.Offset)
				continue
			}

			session.MarkMessage(message, "")

			slog.Debug("Message processed successfully",
				"topic", message.Topic,
				"partition", message.Partition,
				"offset", message.Offset)

		case <-session.Context().Done():
			return nil
		}
	}
}
