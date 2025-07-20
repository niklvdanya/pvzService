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
	"gitlab.ozon.dev/safariproxd/homework/internal/infra/kafka"
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
	dlqRepo := postgres.NewDLQRepository(dbClient)

	kafkaProducer, err := kafka.NewKafkaProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	if err != nil {
		slog.Error("Kafka producer creation failed", "error", err)
		os.Exit(1)
	}
	defer kafkaProducer.Close()

	worker := NewTwoPhaseOutboxWorker(outboxRepo, dlqRepo, kafkaProducer, cfg.Outbox.BatchSize, dbClient)
	dlqWorker := NewDLQWorker(dlqRepo, kafkaProducer)

	slog.Info("Outbox worker with DLQ started",
		"interval", cfg.Outbox.WorkerInterval,
		"batch_size", cfg.Outbox.BatchSize,
		"kafka_topic", cfg.Kafka.Topic)

	ctx := context.Background()
	ticker := time.NewTicker(cfg.Outbox.WorkerInterval)
	dlqTicker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	defer dlqTicker.Stop()

	for {
		select {
		case <-ticker.C:
			worker.ProcessOutboxMessages(ctx)
		case <-dlqTicker.C:
			dlqWorker.ProcessDLQ(ctx)
		}
	}
}

type TwoPhaseOutboxWorker struct {
	repo      *postgres.OutboxRepository
	dlqRepo   *postgres.DLQRepository
	producer  *kafka.KafkaProducer
	batchSize int
	dbClient  *db.Client
}

func NewTwoPhaseOutboxWorker(repo *postgres.OutboxRepository, dlqRepo *postgres.DLQRepository, producer *kafka.KafkaProducer, batchSize int, dbClient *db.Client) *TwoPhaseOutboxWorker {
	return &TwoPhaseOutboxWorker{
		repo:      repo,
		dlqRepo:   dlqRepo,
		producer:  producer,
		batchSize: batchSize,
		dbClient:  dbClient,
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
		return
	}

	ids := make([]uuid.UUID, len(messages))
	for i, msg := range messages {
		ids[i] = msg.ID
	}

	if err := w.repo.SetProcessing(ctx, ids); err != nil {
		slog.Error("Failed to set processing status", "error", err)
		return
	}

	slog.Debug("Phase one completed", "messages_set_to_processing", len(ids))
}

func (w *TwoPhaseOutboxWorker) phaseTwo(ctx context.Context, now time.Time) {
	messages, err := w.repo.GetProcessingMessages(ctx, w.batchSize, now)
	if err != nil {
		slog.Error("Failed to get processing messages", "error", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	slog.Debug("Phase two processing", "count", len(messages))

	for _, msg := range messages {
		w.processMessage(ctx, msg, now)
	}
}

func (w *TwoPhaseOutboxWorker) processMessage(ctx context.Context, msg domain.OutboxMessage, now time.Time) {
	if !msg.CanRetry(now) {
		return
	}

	if msg.ShouldFail() {
		w.moveToDLQ(ctx, msg)
		return
	}

	if err := w.repo.UpdateAttempt(ctx, msg.ID, now); err != nil {
		slog.Error("Failed to update attempt", "id", msg.ID, "error", err)
		return
	}

	if err := w.producer.Send(ctx, msg.Payload); err != nil {
		slog.Error("Failed to send message to Kafka", "id", msg.ID, "error", err)

		if msg.Attempts+1 >= domain.MaxRetryAttempts {
			w.moveToDLQ(ctx, msg)
		}
		return
	}

	if err := w.repo.UpdateStatus(ctx, msg.ID, domain.OutboxStatusCompleted, nil); err != nil {
		slog.Error("Failed to update status to completed", "id", msg.ID, "error", err)
		return
	}

	slog.Debug("Message processed successfully", "id", msg.ID)
}

func (w *TwoPhaseOutboxWorker) moveToDLQ(ctx context.Context, msg domain.OutboxMessage) {
	err := w.withTransaction(ctx, func(tx *db.Tx) error {
		dlqMsg := domain.DLQMessage{
			OriginalID: msg.ID,
			Payload:    msg.Payload,
			Error:      "Max retries exceeded",
			Attempts:   msg.Attempts,
			FailedAt:   time.Now(),
			RetryAfter: time.Now().Add(domain.DLQRetryDelay),
			MaxRetries: domain.DLQMaxRetries,
		}

		if err := w.dlqRepo.Save(ctx, tx, dlqMsg); err != nil {
			return err
		}

		return w.repo.UpdateStatus(ctx, msg.ID, domain.OutboxStatusFailed, &dlqMsg.Error)
	})

	if err != nil {
		slog.Error("Failed to move message to DLQ", "id", msg.ID, "error", err)
	} else {
		slog.Info("Message moved to DLQ", "id", msg.ID)
	}
}

func (w *TwoPhaseOutboxWorker) withTransaction(ctx context.Context, fn func(*db.Tx) error) error {
	tx, err := w.dbClient.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err = fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}
