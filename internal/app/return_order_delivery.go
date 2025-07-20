package app

import (
	"context"
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/internal/metrics"
	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
)

func (s *PVZService) ReturnOrderToDelivery(ctx context.Context, orderID uint64) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("repo.GetByID: %w", err)
	}

	if order.Status != domain.StatusInStorage && order.Status != domain.StatusReturnedFromClient {
		return fmt.Errorf("validation: %w", domain.ValidationFailedError(
			fmt.Sprintf("order is not in storage (current status: %s)", order.GetStatusString())))
	}
	if s.nowFn().Before(order.StorageUntil) {
		return fmt.Errorf("validation: %w", domain.StorageNotExpiredError(
			orderID, cli.MapTimeToString(order.StorageUntil)))
	}

	newStatus := domain.StatusReturnedWithoutClient
	if order.Status == domain.StatusReturnedFromClient {
		newStatus = domain.StatusGivenToCourier
	}
	order.Status = newStatus
	order.LastUpdateTime = s.nowFn()

	history := domain.OrderHistory{
		OrderID:   orderID,
		Status:    newStatus,
		ChangedAt: order.LastUpdateTime,
	}

	event := domain.NewEvent(
		domain.EventTypeOrderReturnedToCourier,
		domain.Actor{
			Type: domain.ActorTypeSystem,
			ID:   0,
		},
		domain.OrderInfo{
			ID:     orderID,
			UserID: order.ReceiverID,
			Status: "returned_to_courier",
		},
	)
	if s.dbClient == nil {
		if err := s.orderRepo.Update(ctx, order); err != nil {
			return fmt.Errorf("failed to update order: %w", err)
		}
		history := domain.OrderHistory{
			OrderID:   order.OrderID,
			Status:    order.Status,
			ChangedAt: order.AcceptTime,
		}

		metrics.OrdersReturnedTotal.WithLabelValues("to_courier").Inc()
		s.updateOrderStatusMetrics()
		return s.orderRepo.SaveHistory(ctx, history)
	}
	return s.withTransaction(ctx, func(tx *db.Tx) error {
		if err := updateOrderInTx(ctx, tx, order); err != nil {
			return fmt.Errorf("update order: %w", err)
		}

		if err := saveHistoryInTx(ctx, tx, history); err != nil {
			return fmt.Errorf("save history: %w", err)
		}

		if err := s.outboxRepo.Save(ctx, tx, event); err != nil {
			return fmt.Errorf("save event: %w", err)
		}

		metrics.OrdersReturnedTotal.WithLabelValues("to_courier").Inc()
		s.updateOrderStatusMetrics()
		return nil
	})
}
