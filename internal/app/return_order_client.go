package app

import (
	"context"
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"go.uber.org/multierr"
)

func (s *PVZService) ReturnOrdersFromClient(ctx context.Context, receiverID uint64, orderIDs []uint64) error {
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

		if order.Status == domain.StatusInStorage {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("validation: %w",
				domain.AlreadyInStorageError(orderID)))
			continue
		}

		timeSinceGiven := currentTime.Sub(order.LastUpdateTime)
		twoDaysLimit := 48 * time.Hour
		if timeSinceGiven > twoDaysLimit {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("validation: %w",
				domain.ReturnPeriodExpiredError(orderID, timeSinceGiven.Hours())))
			continue
		}

		order.Status = domain.StatusReturnedFromClient
		order.LastUpdateTime = currentTime
		if err := s.orderRepo.Update(ctx, order); err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("repo.Update: %w", err))
			continue
		}

		history := domain.OrderHistory{
			OrderID:   orderID,
			Status:    domain.StatusReturnedFromClient,
			ChangedAt: currentTime,
		}
		if err := s.orderRepo.SaveHistory(ctx, history); err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("repo.SaveHistory: %w", err))
			continue
		}
	}
	return combinedErr
}
