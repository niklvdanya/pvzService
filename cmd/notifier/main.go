package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/config"
	"gitlab.ozon.dev/safariproxd/homework/internal/infra"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		slog.Error("Config load failed", "error", err)
		os.Exit(1)
	}

	telegramClient := infra.NewTelegramClient(cfg.Telegram)
	telegramNotifier := infra.NewTelegramNotifier(telegramClient)

	if !telegramClient.IsEnabled() {
		slog.Warn("Telegram notifications disabled",
			"bot_token_configured", cfg.Telegram.BotToken != "",
			"chat_id_configured", cfg.Telegram.ChatID != 0)
	} else {
		slog.Info("Telegram notifications enabled",
			"chat_id", cfg.Telegram.ChatID)
	}

	consumerConfig := infra.KafkaConsumerConfig{
		Brokers:         cfg.Kafka.Brokers,
		Topic:           cfg.Kafka.Topic,
		ConsumerGroup:   "pvz-notifier",
		AutoOffsetReset: "earliest",
	}

	consumer, err := infra.NewKafkaConsumer(consumerConfig)
	if err != nil {
		slog.Error("Failed to create Kafka consumer", "error", err)
		os.Exit(1)
	}
	defer consumer.Close()

	eventHandler := NewEventHandler(telegramNotifier)
	notifier := NewNotifierService(consumer, eventHandler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("Notifier service started",
		"consumer_group", consumerConfig.ConsumerGroup,
		"topic", consumerConfig.Topic,
		"brokers", consumerConfig.Brokers,
		"telegram_enabled", telegramClient.IsEnabled())

	if telegramClient.IsEnabled() {
		startupMsg := "🚀 <b>Notifier запущен</b>\n\n" +
			"📡 Подключен к Kafka\n" +
			"🔔 Уведомления включены\n" +
			"⏰ Время запуска: " + time.Now().Format("15:04:05")

		if err := telegramClient.SendMessage(ctx, startupMsg); err != nil {
			slog.Error("Failed to send startup notification", "error", err)
		}
	}

	go func() {
		if err := notifier.Start(ctx); err != nil {
			slog.Error("Notifier service error", "error", err)
			cancel()
		}
	}()

	select {
	case sig := <-sigChan:
		slog.Info("Received shutdown signal", "signal", sig)
	case <-ctx.Done():
		slog.Info("Context canceled")
	}

	slog.Info("Shutting down notifier service...")

	if telegramClient.IsEnabled() {
		shutdownMsg := "🛑 <b>Notifier остановлен</b>\n\n" +
			fmt.Sprintf("📊 Обработано событий: <code>%d</code>\n", eventHandler.GetProcessedCount()) +
			"⏰ Время остановки: " + time.Now().Format("15:04:05")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := telegramClient.SendMessage(shutdownCtx, shutdownMsg); err != nil {
			slog.Error("Failed to send shutdown notification", "error", err)
		}
		shutdownCancel()
	}

	cancel()

	gracefulCtx, gracefulCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer gracefulCancel()

	if err := notifier.Shutdown(gracefulCtx); err != nil {
		slog.Error("Error during shutdown", "error", err)
	}

	slog.Info("Notifier service stopped")
}
