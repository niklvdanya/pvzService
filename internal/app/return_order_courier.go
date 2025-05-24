package app

import (
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (s *PVZService) ReturnOrderToDelivery(orderID uint64) error {
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return fmt.Errorf("cannot return order %d to delivery: %w", orderID, err)
	}

	if order.Status != domain.StatusInStorage && order.Status != domain.StatusReturnedFromClient {
		return fmt.Errorf(
			"cannot return order %d to delivery: order is not in storage (current status: %s)",
			orderID,
			order.GetStatusString(),
		)
	}
	if time.Now().Before(order.StorageUntil) {
		return fmt.Errorf(
			"cannot return order %d to delivery: %w (until: %s)",
			orderID,
			ErrStorageNotExpired,
			order.StorageUntil.Format("2006-01-02"),
		)
	}
	if order.Status == domain.StatusInStorage {
		order.Status = domain.StatusReturnedWithoutClient
	} else {
		order.Status = domain.StatusGivenToCourier
	}
	order.LastUpdateTime = time.Now()

	return s.orderRepo.Update(order)
}
