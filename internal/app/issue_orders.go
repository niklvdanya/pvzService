package app

import (
	"context"
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"go.uber.org/multierr"
)

func (s *PVZService) IssueOrdersToClient(ctx context.Context, receiverID uint64, orderIDs []uint64) error {
	var combinedErr error
	currentTime := s.nowFn()

	for _, orderID := range orderIDs {
		order, err := s.orderRepo.GetByID(ctx, orderID)
		if err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("repo.GetByID: %w", err))
			continue
		}

		if order.ReceiverID != receiverID {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("validation: %w",
				domain.BelongsToDifferentReceiverError(orderID, receiverID, order.ReceiverID)))
			continue
		}

		if order.Status == domain.StatusGivenToClient {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("validation: %w",
				domain.OrderAlreadyGivenError(orderID)))
			continue
		}

		if order.Status == domain.StatusReturnedFromClient {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("validation: %w",
				domain.UnavaliableReturnedOrderError(orderID)))
			continue
		}

		if currentTime.After(order.StorageUntil) {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("validation: %w",
				domain.StorageExpiredError(orderID, cli.MapTimeToString(order.StorageUntil))))
			continue
		}

		order.Status = domain.StatusGivenToClient
		order.LastUpdateTime = currentTime
		if err := s.orderRepo.Update(ctx, order); err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("repo.Update: %w", err))
			continue
		}

		history := domain.OrderHistory{
			OrderID:   orderID,
			Status:    domain.StatusGivenToClient,
			ChangedAt: currentTime,
		}
		if err := s.orderRepo.SaveHistory(ctx, history); err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("repo.SaveHistory: %w", err))
			continue
		}
	}
	return combinedErr
}
