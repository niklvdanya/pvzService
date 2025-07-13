package infra

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

type TelegramNotifier struct {
	client    *TelegramClient
	formatter *MessageFormatter
}

type MessageFormatter struct {
	timeZone *time.Location
}

func NewTelegramNotifier(client *TelegramClient) *TelegramNotifier {
	timezone, _ := time.LoadLocation("UTC")

	return &TelegramNotifier{
		client: client,
		formatter: &MessageFormatter{
			timeZone: timezone,
		},
	}
}

func (n *TelegramNotifier) NotifyEvent(ctx context.Context, event *domain.Event) error {
	if !n.client.IsEnabled() {
		slog.Debug("Telegram notifications disabled, skipping")
		return nil
	}

	message := n.formatter.FormatEvent(event)

	if err := n.client.SendMessage(ctx, message); err != nil {
		return fmt.Errorf("send telegram notification: %w", err)
	}

	slog.Debug("Telegram notification sent",
		"event_id", event.EventID,
		"event_type", event.EventType,
		"order_id", event.Order.ID)

	return nil
}

func (f *MessageFormatter) FormatEvent(event *domain.Event) string {
	timestamp := event.Timestamp.In(f.timeZone).Format("15:04:05")

	switch event.EventType {
	case domain.EventTypeOrderAccepted:
		return fmt.Sprintf(
			"📦 <b>Заказ принят</b>\n\n"+
				"🆔 Заказ: <code>%d</code>\n"+
				"👤 Клиент: <code>%d</code>\n"+
				"👨‍💼 Курьер: <code>%d</code>\n"+
				"🕐 Время: %s\n\n"+
				"✅ Заказ успешно принят от курьера и размещен в ПВЗ",
			event.Order.ID, event.Order.UserID, event.Actor.ID, timestamp)

	case domain.EventTypeOrderIssued:
		return fmt.Sprintf(
			"✅ <b>Заказ выдан клиенту</b>\n\n"+
				"🆔 Заказ: <code>%d</code>\n"+
				"👤 Клиент: <code>%d</code>\n"+
				"🕐 Время: %s\n\n"+
				"🎉 Клиент получил свой заказ!",
			event.Order.ID, event.Order.UserID, timestamp)

	case domain.EventTypeOrderReturnedByClient:
		return fmt.Sprintf(
			"↩️ <b>Возврат от клиента</b>\n\n"+
				"🆔 Заказ: <code>%d</code>\n"+
				"👤 Клиент: <code>%d</code>\n"+
				"🕐 Время: %s\n\n"+
				"📥 Клиент вернул заказ в ПВЗ",
			event.Order.ID, event.Order.UserID, timestamp)

	case domain.EventTypeOrderReturnedToCourier:
		return fmt.Sprintf(
			"📮 <b>Возврат курьеру</b>\n\n"+
				"🆔 Заказ: <code>%d</code>\n"+
				"👤 Клиент: <code>%d</code>\n"+
				"🕐 Время: %s\n\n"+
				"⚠️ Заказ возвращен курьеру (истек срок хранения или возврат от клиента)",
			event.Order.ID, event.Order.UserID, timestamp)

	default:
		return fmt.Sprintf(
			"❓ <b>Неизвестное событие</b>\n\n"+
				"🔤 Тип: <code>%s</code>\n"+
				"🆔 Заказ: <code>%d</code>\n"+
				"👤 Клиент: <code>%d</code>\n"+
				"🕐 Время: %s",
			event.EventType, event.Order.ID, event.Order.UserID, timestamp)
	}
}

func (f *MessageFormatter) FormatStatistics(processedCount uint64, eventType domain.EventType) string {
	return fmt.Sprintf(
		"📊 <b>Статистика обработки</b>\n\n"+
			"✅ Обработано событий: <code>%d</code>\n"+
			"🔄 Последний тип: <code>%s</code>\n"+
			"🕐 Время: %s",
		processedCount, eventType, time.Now().Format("15:04:05"))
}

func (n *TelegramNotifier) NotifyStatistics(ctx context.Context, processedCount uint64, eventType domain.EventType) error {
	if !n.client.IsEnabled() {
		return nil
	}

	message := n.formatter.FormatStatistics(processedCount, eventType)

	if err := n.client.SendMessage(ctx, message); err != nil {
		return fmt.Errorf("send telegram statistics: %w", err)
	}

	return nil
}

func (n *TelegramNotifier) NotifyError(ctx context.Context, errorMsg string, eventID string) error {
	if !n.client.IsEnabled() {
		return nil
	}

	message := fmt.Sprintf(
		"❌ <b>Ошибка обработки</b>\n\n"+
			"🆔 Event ID: <code>%s</code>\n"+
			"⚠️ Ошибка: <code>%s</code>\n"+
			"🕐 Время: %s",
		eventID, errorMsg, time.Now().Format("15:04:05"))

	if err := n.client.SendMessage(ctx, message); err != nil {
		return fmt.Errorf("send telegram error: %w", err)
	}

	return nil
}
