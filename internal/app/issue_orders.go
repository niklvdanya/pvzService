package app

import (
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"go.uber.org/multierr"
)

func (s *PVZService) IssueOrdersToClient(receiverID uint64, orderIDs []uint64) error {
	var combinedErr error
	currentTime := time.Now()

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

		if order.Status == domain.StatusGivenToClient {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf("order %d: %w", orderID, ErrOrderAlreadyGiven),
			)
			continue
		}
		// не может же клиент вернуть заказа и потом снова его забрать :)
		if order.Status == domain.StatusReturnedFromClient {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf("order %d: %w", orderID, ErrUnavaliableReturnedOrder),
			)
			continue
		}
		if currentTime.After(order.StorageUntil) {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf(
					"order %d: %w (%s)",
					orderID,
					ErrStorageExpired,
					order.StorageUntil.Format("2006-01-02"),
				),
			)
			continue
		}

		order.Status = domain.StatusGivenToClient
		order.LastUpdateTime = currentTime
		if err := s.orderRepo.Update(order); err != nil {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf("order %d: failed to update status: %w", orderID, err),
			)
		}
	}
	return combinedErr
}
