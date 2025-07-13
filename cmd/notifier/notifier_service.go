package main

import (
	"context"
	"fmt"
	"log/slog"

	"gitlab.ozon.dev/safariproxd/homework/internal/infra"
)

type NotifierService struct {
	consumer *infra.KafkaConsumer
	handler  *EventHandler
	done     chan struct{}
}

func NewNotifierService(consumer *infra.KafkaConsumer, handler *EventHandler) *NotifierService {
	return &NotifierService{
		consumer: consumer,
		handler:  handler,
		done:     make(chan struct{}),
	}
}

func (s *NotifierService) Start(ctx context.Context) error {
	slog.Info("Starting Kafka consumer...")

	if err := s.consumer.Consume(ctx, s.handler); err != nil {
		return fmt.Errorf("consumer error: %w", err)
	}

	return nil
}

func (s *NotifierService) Shutdown(_ context.Context) error {
	slog.Info("Shutting down notifier service...")

	if err := s.consumer.Close(); err != nil {
		slog.Error("Error closing consumer", "error", err)
		return err
	}

	slog.Info("Notifier service shutdown complete",
		"total_processed_events", s.handler.GetProcessedCount())

	close(s.done)
	return nil
}

func (s *NotifierService) Wait() {
	<-s.done
}
