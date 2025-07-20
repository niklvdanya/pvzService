package app

import (
	"context"
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/internal/metrics"
	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
)

func (s *PVZService) issueSingle(ctx context.Context, receiverID uint64, orderID uint64, now time.Time) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("repo.GetByID: %w", err)
	}

	if order.ReceiverID != receiverID {
		return domain.BelongsToDifferentReceiverError(orderID, receiverID, order.ReceiverID)
	}
	if order.Status == domain.StatusGivenToClient {
		return domain.OrderAlreadyGivenError(orderID)
	}
	if order.Status == domain.StatusReturnedFromClient {
		return domain.UnavaliableReturnedOrderError(orderID)
	}
	if now.After(order.StorageUntil) {
		return domain.StorageExpiredError(orderID, cli.MapTimeToString(order.StorageUntil))
	}

	order.Status = domain.StatusGivenToClient
	order.LastUpdateTime = now

	hist := domain.OrderHistory{
		OrderID:   orderID,
		Status:    domain.StatusGivenToClient,
		ChangedAt: now,
	}

	event := domain.NewEvent(
		domain.EventTypeOrderIssued,
		domain.Actor{
			Type: domain.ActorTypeClient,
			ID:   receiverID,
		},
		domain.OrderInfo{
			ID:     orderID,
			UserID: receiverID,
			Status: "issued",
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
	return s.withTransaction(ctx, func(tx *db.Tx) error {
		if err := updateOrderInTx(ctx, tx, order); err != nil {
			return fmt.Errorf("update order: %w", err)
		}

		if err := saveHistoryInTx(ctx, tx, hist); err != nil {
			return fmt.Errorf("save history: %w", err)
		}

		if err := s.outboxRepo.Save(ctx, tx, event); err != nil {
			return fmt.Errorf("save event: %w", err)
		}

		return nil
	})
}

func (s *PVZService) IssueOrdersToClient(
	ctx context.Context,
	receiverID uint64,
	orderIDs []uint64,
) error {
	processed, err := processConcurrently(ctx, orderIDs, s.workerLimit, func(c context.Context, id uint64) error {
		return s.issueSingle(c, receiverID, id, s.nowFn())
	})

	metrics.OrdersIssuedTotal.Add(float64(processed))
	s.updateOrderStatusMetrics()

	return err
}
