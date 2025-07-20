package main

import (
	"context"
	"encoding/json"
	"log/slog"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/internal/infra/kafka"
	"gitlab.ozon.dev/safariproxd/homework/internal/repository/postgres"
)

type DLQWorker struct {
	repo     *postgres.DLQRepository
	producer *kafka.KafkaProducer
}

func NewDLQWorker(repo *postgres.DLQRepository, producer *kafka.KafkaProducer) *DLQWorker {
	return &DLQWorker{
		repo:     repo,
		producer: producer,
	}
}

func (w *DLQWorker) ProcessDLQ(ctx context.Context) {
	messages, err := w.repo.GetRetryable(ctx, 10)
	if err != nil {
		slog.Error("Failed to get DLQ messages", "error", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	slog.Info("Processing DLQ messages", "count", len(messages))

	for _, msg := range messages {
		w.processDLQMessage(ctx, msg)
	}
}

func (w *DLQWorker) processDLQMessage(ctx context.Context, msg domain.DLQMessage) {
	var event domain.Event
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		slog.Error("Failed to unmarshal DLQ event", "id", msg.ID, "error", err)
		return
	}

	if err := w.producer.Send(ctx, msg.Payload); err != nil {
		msg.IncrementRetry()
		if err := w.repo.UpdateRetry(ctx, msg.ID, msg.ProcessCount, msg.RetryAfter); err != nil {
			slog.Error("Failed to update DLQ retry", "id", msg.ID, "error", err)
		}
		slog.Warn("Failed to resend DLQ message",
			"id", msg.ID,
			"attempt", msg.ProcessCount,
			"error", err)
		return
	}

	if err := w.repo.Delete(ctx, msg.ID); err != nil {
		slog.Error("Failed to delete processed DLQ message", "id", msg.ID, "error", err)
		return
	}

	slog.Info("Successfully processed DLQ message",
		"id", msg.ID,
		"event_id", event.EventID,
		"attempts", msg.Attempts)
}
