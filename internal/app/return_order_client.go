package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"go.uber.org/multierr"
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
	sem := make(chan struct{}, s.workerLimit)
	now := s.nowFn()

	var combined error
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, id := range orderIDs {
		sem <- struct{}{}
		wg.Add(1)
		go func(oid uint64) {
			defer func() {
				<-sem
				wg.Done()
			}()
			if err := s.returnSingle(ctx, receiverID, oid, now); err != nil {
				mu.Lock()
				combined = multierr.Append(combined, err)
				mu.Unlock()
			}
		}(id)
	}

	wg.Wait()
	return combined
}
