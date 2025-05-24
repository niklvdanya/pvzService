package app

import (
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"go.uber.org/multierr"
)

func (s *PVZService) ReturnOrdersFromClient(receiverID uint64, orderIDs []uint64) error {
	var combinedErr error
	currentTime := time.Now()

	for _, orderID := range orderIDs {
		order, err := s.orderRepo.GetByID(orderID)
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
		if err := s.orderRepo.Update(order); err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("repo.Update: %w", err))
		}
	}
	return combinedErr
}
