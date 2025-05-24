package app

import (
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"go.uber.org/multierr"
)

func (s *PVZService) ReturnOrdersFromClient(receiverID uint64, orderIDs []uint64) error {
	var combinedErr error
	currentTimeInMoscow := time.Now()

	for _, orderID := range orderIDs {
		order, err := s.orderRepo.GetByID(orderID)
		if err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("order %d: %w", orderID, ErrOrderNotFound))
			continue
		}

		if order.ReceiverID != receiverID {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf(
					"order %d: %w (expected %d, got %d)",
					orderID,
					ErrBelongsToDifferentReceiver,
					receiverID,
					order.ReceiverID,
				),
			)
			continue
		}

		if order.Status == domain.StatusInStorage {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf("order %d: %w", orderID, ErrAlreadyInStorage),
			)
			continue
		}

		timeSinceGiven := currentTimeInMoscow.Sub(order.LastUpdateTime)
		twoDaysLimit := 48 * time.Hour

		if timeSinceGiven > twoDaysLimit {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("order %d: %w (%.1f hours)",
				orderID, ErrReturnPeriodExpired, timeSinceGiven.Hours()))
			continue
		}

		order.Status = domain.StatusReturnedFromClient
		order.LastUpdateTime = currentTimeInMoscow
		if err := s.orderRepo.Update(order); err != nil {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf("order %d: failed to update order status to returned: %w", orderID, err),
			)
		}
	}
	return combinedErr
}
