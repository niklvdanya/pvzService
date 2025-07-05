package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"go.uber.org/multierr"
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

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("repo.Update: %w", err)
	}

	hist := domain.OrderHistory{
		OrderID:   orderID,
		Status:    domain.StatusGivenToClient,
		ChangedAt: now,
	}
	return s.orderRepo.SaveHistory(ctx, hist)
}

func (s *PVZService) IssueOrdersToClient(ctx context.Context, receiverID uint64, orderIDs []uint64) error {
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
			if err := s.issueSingle(ctx, receiverID, oid, now); err != nil {
				mu.Lock()
				combined = multierr.Append(combined, err)
				mu.Unlock()
			}
		}(id)
	}

	wg.Wait()
	return combined
}
