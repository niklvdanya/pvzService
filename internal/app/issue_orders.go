package app

import (
	"context"
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"golang.org/x/sync/errgroup"
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
	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, s.workerLimit)
	now := s.nowFn()

	for _, id := range orderIDs {
		sem <- struct{}{}
		g.Go(func() error {
			defer func() { <-sem }()
			return s.issueSingle(ctx, receiverID, id, now)
		})
	}
	return g.Wait()
}
