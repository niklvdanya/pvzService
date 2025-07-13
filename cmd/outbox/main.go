package main

import (
	"context"
	"log/slog"
	"os"
	"time"

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

	slog.Info("Outbox worker started",
		"interval", cfg.Outbox.WorkerInterval,
		"batch_size", cfg.Outbox.BatchSize,
		"kafka_topic", cfg.Kafka.Topic)

	ctx := context.Background()
	ticker := time.NewTicker(cfg.Outbox.WorkerInterval)
	defer ticker.Stop()

	processOutboxMessages(ctx, outboxRepo, kafkaProducer, cfg.Outbox.BatchSize)

	for range ticker.C {
		processOutboxMessages(ctx, outboxRepo, kafkaProducer, cfg.Outbox.BatchSize)
	}
}

func processOutboxMessages(ctx context.Context, repo *postgres.OutboxRepository, producer *infra.KafkaProducer, batchSize int) {
	messages, err := repo.GetPendingMessages(ctx, batchSize)
	if err != nil {
		slog.Error("Failed to get pending messages", "error", err)
		return
	}
	if len(messages) == 0 {
		slog.Debug("No pending messages found")
		return
	}

	slog.Info("Processing outbox messages", "count", len(messages))
	for _, msg := range messages {
		if err := repo.UpdateStatus(ctx, msg.ID, domain.OutboxStatusProcessing, nil); err != nil {
			slog.Error("Failed to update status to processing", "id", msg.ID, "error", err)
			continue
		}

		if err := producer.Send(ctx, msg.Payload); err != nil {
			errMsg := err.Error()
			if err := repo.UpdateStatus(ctx, msg.ID, domain.OutboxStatusFailed, &errMsg); err != nil {
				slog.Error("Failed to update status to failed", "id", msg.ID, "error", err)
			}
			slog.Error("Failed to send message to Kafka", "id", msg.ID, "error", err)
			continue
		}

		if err := repo.UpdateStatus(ctx, msg.ID, domain.OutboxStatusCompleted, nil); err != nil {
			slog.Error("Failed to update status to completed", "id", msg.ID, "error", err)
			continue
		}
		slog.Info("Message processed successfully", "id", msg.ID)
	}
}
