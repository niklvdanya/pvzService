package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"gitlab.ozon.dev/safariproxd/homework/internal/config"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/internal/infra"
	"gitlab.ozon.dev/safariproxd/homework/internal/repository/postgres"
	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		slog.Error("Config load failed", "error", err)
		os.Exit(1)
	}

	dbCfg := db.Config{
		ReadDSN:  cfg.ReadDSN(),
		WriteDSN: cfg.WriteDSN(),
		MaxOpen:  cfg.DB.Pool.MaxOpen,
		MaxIdle:  cfg.DB.Pool.MaxIdle,
	}
	dbClient, err := db.NewClient(dbCfg)
	if err != nil {
		slog.Error("DB client creation failed", "error", err)
		os.Exit(1)
	}
	defer dbClient.Close()

	outboxRepo := postgres.NewOutboxRepository(dbClient)
	kafkaProducer, err := infra.NewKafkaProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	if err != nil {
		slog.Error("Kafka producer creation failed", "error", err)
		os.Exit(1)
	}
	defer kafkaProducer.Close()

	worker := NewTwoPhaseOutboxWorker(outboxRepo, kafkaProducer, cfg.Outbox.BatchSize)

	slog.Info("Two-phase outbox worker started",
		"interval", cfg.Outbox.WorkerInterval,
		"batch_size", cfg.Outbox.BatchSize,
		"kafka_topic", cfg.Kafka.Topic)

	ctx := context.Background()
	ticker := time.NewTicker(cfg.Outbox.WorkerInterval)
	defer ticker.Stop()
	worker.ProcessOutboxMessages(ctx)

	for range ticker.C {
		worker.ProcessOutboxMessages(ctx)
	}
}

type TwoPhaseOutboxWorker struct {
	repo      *postgres.OutboxRepository
	producer  *infra.KafkaProducer
	batchSize int
}

func NewTwoPhaseOutboxWorker(repo *postgres.OutboxRepository, producer *infra.KafkaProducer, batchSize int) *TwoPhaseOutboxWorker {
	return &TwoPhaseOutboxWorker{
		repo:      repo,
		producer:  producer,
		batchSize: batchSize,
	}
}

func (w *TwoPhaseOutboxWorker) ProcessOutboxMessages(ctx context.Context) {
	now := time.Now()

	w.phaseOne(ctx)

	w.phaseTwo(ctx, now)
}

func (w *TwoPhaseOutboxWorker) phaseOne(ctx context.Context) {
	messages, err := w.repo.GetPendingMessages(ctx, w.batchSize)
	if err != nil {
		slog.Error("Failed to get pending messages", "error", err)
		return
	}

	if len(messages) == 0 {
		slog.Debug("No pending messages found in phase one")
		return
	}

	ids := make([]uuid.UUID, len(messages))
	for i, msg := range messages {
		ids[i] = msg.ID
	}

	if err := w.repo.SetProcessing(ctx, ids); err != nil {
		slog.Error("Failed to set processing status", "error", err, "count", len(ids))
		return
	}

	slog.Info("Phase one completed", "messages_set_to_processing", len(ids))
}

func (w *TwoPhaseOutboxWorker) phaseTwo(ctx context.Context, now time.Time) {
	messages, err := w.repo.GetProcessingMessages(ctx, w.batchSize, now)
	if err != nil {
		slog.Error("Failed to get processing messages", "error", err)
		return
	}

	if len(messages) == 0 {
		slog.Debug("No processing messages found in phase two")
		return
	}

	slog.Info("Phase two processing", "count", len(messages))

	for _, msg := range messages {
		w.processMessage(ctx, msg, now)
	}
}

func (w *TwoPhaseOutboxWorker) processMessage(ctx context.Context, msg domain.OutboxMessage, now time.Time) {
	if !msg.CanRetry(now) {
		slog.Debug("Message not ready for retry", "id", msg.ID, "attempts", msg.Attempts, "last_attempt", msg.LastAttemptAt)
		return
	}

	if msg.ShouldFail() {
		if err := w.repo.FailMessage(ctx, msg.ID, domain.NoAttemptsLeftError); err != nil {
			slog.Error("Failed to mark message as failed", "id", msg.ID, "error", err)
		} else {
			slog.Warn("Message marked as failed", "id", msg.ID, "attempts", msg.Attempts)
		}
		return
	}

	if err := w.repo.UpdateAttempt(ctx, msg.ID, now); err != nil {
		slog.Error("Failed to update attempt", "id", msg.ID, "error", err)
		return
	}

	if err := w.producer.Send(ctx, msg.Payload); err != nil {
		slog.Error("Failed to send message to Kafka",
			"id", msg.ID,
			"attempt", msg.Attempts+1,
			"error", err)

		if msg.Attempts+1 >= domain.MaxRetryAttempts {
			errorMsg := err.Error()
			if err := w.repo.UpdateStatus(ctx, msg.ID, domain.OutboxStatusFailed, &errorMsg); err != nil {
				slog.Error("Failed to update status to failed", "id", msg.ID, "error", err)
			}
		}
		return
	}

	if err := w.repo.UpdateStatus(ctx, msg.ID, domain.OutboxStatusCompleted, nil); err != nil {
		slog.Error("Failed to update status to completed", "id", msg.ID, "error", err)
		return
	}

	slog.Info("Message processed successfully",
		"id", msg.ID,
		"attempt", msg.Attempts+1)
}
