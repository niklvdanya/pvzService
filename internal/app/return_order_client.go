package app

import (
	"context"
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"golang.org/x/sync/errgroup"
)

func (s *PVZService) returnSingle(ctx context.Context, receiverID uint64, orderID uint64, now time.Time) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("repo.GetByID: %w", err)
	}

	if order.ReceiverID != receiverID {
		return domain.BelongsToDifferentReceiverError(orderID, receiverID, order.ReceiverID)
	}
	if order.Status == domain.StatusInStorage {
		return domain.AlreadyInStorageError(orderID)
	}
	if now.Sub(order.LastUpdateTime) > 48*time.Hour {
		return domain.ReturnPeriodExpiredError(orderID, now.Sub(order.LastUpdateTime).Hours())
	}

	order.Status = domain.StatusReturnedFromClient
	order.LastUpdateTime = now

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("repo.Update: %w", err)
	}

	hist := domain.OrderHistory{
		OrderID:   orderID,
		Status:    domain.StatusReturnedFromClient,
		ChangedAt: now,
	}
	return s.orderRepo.SaveHistory(ctx, hist)
}

func (s *PVZService) ReturnOrdersFromClient(ctx context.Context, receiverID uint64, orderIDs []uint64) error {
	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, parallelWorkers)
	now := time.Now()

	for _, id := range orderIDs {
		sem <- struct{}{}
		g.Go(func() error {
			defer func() {
				<-sem
			}()
			return s.returnSingle(ctx, receiverID, id, now)
		})
	}
	return g.Wait()
}
