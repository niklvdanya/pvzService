package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/internal/infra"
)

type EventHandler struct {
	processedCount    uint64
	telegramNotifier  *infra.TelegramNotifier
	lastStatisticTime time.Time
}

func NewEventHandler(telegramNotifier *infra.TelegramNotifier) *EventHandler {
	return &EventHandler{
		telegramNotifier:  telegramNotifier,
		lastStatisticTime: time.Now(),
	}
}

func (h *EventHandler) HandleMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	var event domain.Event
	if err := json.Unmarshal(message.Value, &event); err != nil {
		errorMsg := fmt.Sprintf("failed to unmarshal event: %v", err)
		slog.Error("Event parsing failed", "error", err, "raw_message", string(message.Value))
		if notifyErr := h.telegramNotifier.NotifyError(ctx, errorMsg, "unknown"); notifyErr != nil {
			slog.Error("Failed to send telegram error notification", "error", notifyErr)
		}

		return errors.New(errorMsg)
	}
	if err := h.validateEvent(&event); err != nil {
		errorMsg := fmt.Sprintf("invalid event: %v", err)
		slog.Error("Event validation failed", "error", err, "event_id", event.EventID)

		if notifyErr := h.telegramNotifier.NotifyError(ctx, errorMsg, event.EventID); notifyErr != nil {
			slog.Error("Failed to send telegram error notification", "error", notifyErr)
		}

		return errors.New(errorMsg)
	}
	if err := h.telegramNotifier.NotifyEvent(ctx, &event); err != nil {
		slog.Error("Failed to send telegram notification",
			"error", err,
			"event_id", event.EventID,
			"event_type", event.EventType)
	} else {
		slog.Info("Telegram notification sent successfully",
			"event_id", event.EventID,
			"event_type", event.EventType,
			"order_id", event.Order.ID,
			"user_id", event.Order.UserID)
	}

	h.logEvent(&event, message)
	h.processedCount++

	if h.processedCount%50 == 0 || time.Since(h.lastStatisticTime) > 10*time.Minute {
		if err := h.telegramNotifier.NotifyStatistics(ctx, h.processedCount, event.EventType); err != nil {
			slog.Error("Failed to send telegram statistics", "error", err)
		}
		h.lastStatisticTime = time.Now()

		slog.Info("Processing statistics",
			"total_processed", h.processedCount,
			"current_event_type", event.EventType)
	}

	return nil
}

func (h *EventHandler) validateEvent(event *domain.Event) error {
	if event.EventID == "" {
		return fmt.Errorf("missing event_id")
	}

	if event.EventType == "" {
		return fmt.Errorf("missing event_type")
	}

	if event.Source == "" {
		return fmt.Errorf("missing source")
	}

	if event.Order.ID == 0 {
		return fmt.Errorf("missing order.id")
	}

	if event.Order.UserID == 0 {
		return fmt.Errorf("missing order.user_id")
	}

	if time.Since(event.Timestamp) > 24*time.Hour {
		slog.Warn("Received old event",
			"event_id", event.EventID,
			"event_timestamp", event.Timestamp,
			"age_hours", time.Since(event.Timestamp).Hours())
	}

	return nil
}

func (h *EventHandler) logEvent(event *domain.Event, message *sarama.ConsumerMessage) {
	logger := slog.With(
		"event_id", event.EventID,
		"event_type", event.EventType,
		"source", event.Source,
		"timestamp", event.Timestamp.Format(time.RFC3339),
		"kafka_topic", message.Topic,
		"kafka_partition", message.Partition,
		"kafka_offset", message.Offset,
		"kafka_timestamp", message.Timestamp.Format(time.RFC3339),
	)

	logger = logger.With(
		"actor_type", event.Actor.Type,
		"actor_id", event.Actor.ID,
	)

	logger = logger.With(
		"order_id", event.Order.ID,
		"user_id", event.Order.UserID,
		"order_status", event.Order.Status,
	)

	switch event.EventType {
	case domain.EventTypeOrderAccepted:
		logger.Info("📦 Order accepted by courier",
			"message", fmt.Sprintf("Order %d accepted for user %d", event.Order.ID, event.Order.UserID))

	case domain.EventTypeOrderIssued:
		logger.Info("✅ Order issued to client",
			"message", fmt.Sprintf("Order %d issued to user %d", event.Order.ID, event.Order.UserID))

	case domain.EventTypeOrderReturnedByClient:
		logger.Info("↩️ Order returned by client",
			"message", fmt.Sprintf("Order %d returned by user %d", event.Order.ID, event.Order.UserID))

	case domain.EventTypeOrderReturnedToCourier:
		logger.Warn("📮 Order returned to courier",
			"message", fmt.Sprintf("Order %d returned to courier (user: %d)", event.Order.ID, event.Order.UserID))

	default:
		logger.Warn("❓ Unknown event type",
			"message", fmt.Sprintf("Unknown event type: %s for order %d", event.EventType, event.Order.ID))
	}

	eventJSON, _ := json.Marshal(event)
	slog.Debug("Raw event data",
		"event_id", event.EventID,
		"raw_json", string(eventJSON))
}

func (h *EventHandler) GetProcessedCount() uint64 {
	return h.processedCount
}
