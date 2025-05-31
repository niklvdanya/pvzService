package app

import (
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"go.uber.org/multierr"
)

func (s *PVZService) IssueOrdersToClient(receiverID uint64, orderIDs []uint64) error {
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
		if err := s.orderRepo.Update(order); err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("repo.Update: %w", err))
		}
	}
	return combinedErr
}
