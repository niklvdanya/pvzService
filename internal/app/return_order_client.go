package app

import (
	"context"
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/internal/metrics"
	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
)

func (s *PVZService) returnSingle(ctx context.Context, receiverID uint64, orderID uint64, now time.Time) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("repo.GetByID: %w", err)
	}

	if order.ReceiverID != receiverID {
		return domain.BelongsToDifferentReceiverError(orderID, receiverID, order.ReceiverID)
	}
	if order.Status == domain.StatusInStorage || order.Status == domain.StatusReturnedFromClient {
		return domain.AlreadyInStorageError(orderID)
	}
	if now.Sub(order.LastUpdateTime) > 48*time.Hour {
		return domain.ReturnPeriodExpiredError(orderID, now.Sub(order.LastUpdateTime).Hours())
	}

	order.Status = domain.StatusReturnedFromClient
	order.LastUpdateTime = now

	hist := domain.OrderHistory{
		OrderID:   orderID,
		Status:    domain.StatusReturnedFromClient,
		ChangedAt: now,
	}

	event := domain.NewEvent(
		domain.EventTypeOrderReturnedByClient,
		domain.Actor{
			Type: domain.ActorTypeClient,
			ID:   receiverID,
		},
		domain.OrderInfo{
			ID:     orderID,
			UserID: receiverID,
			Status: "returned_by_client",
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
		return s.orderRepo.SaveHistory(ctx, history)
	}
	return s.dbClient.WithTransaction(ctx, func(tx *db.Tx) error {
		if err := s.orderRepo.UpdateOrderInTx(ctx, tx, order); err != nil {
			return fmt.Errorf("update order: %w", err)
		}

		if err := s.orderRepo.SaveHistoryInTx(ctx, tx, hist); err != nil {
			return fmt.Errorf("save history: %w", err)
		}

		if err := s.outboxRepo.Save(ctx, tx, event); err != nil {
			return fmt.Errorf("save event: %w", err)
		}

		return nil
	})
}

func (s *PVZService) ReturnOrdersFromClient(
	ctx context.Context,
	receiverID uint64,
	orderIDs []uint64,
) error {
	now := s.nowFn()
	processed, err := processConcurrently(ctx, orderIDs, s.workerLimit, func(c context.Context, id uint64) error {
		return s.returnSingle(c, receiverID, id, now)
	})

	metrics.OrdersReturnedTotal.WithLabelValues("by_client").Add(float64(processed))
	s.updateOrderStatusMetrics()

	return err
}
