package app

import (
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (s *PVZService) ReturnOrderToDelivery(orderID uint64) error {
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return fmt.Errorf("repo.GetByID: %w", err)
	}

	if order.Status != domain.StatusInStorage && order.Status != domain.StatusReturnedFromClient {
		return fmt.Errorf("validation: %w", domain.ValidationFailedError(
			fmt.Sprintf("order is not in storage (current status: %s)", order.GetStatusString())))
	}
	if time.Now().Before(order.StorageUntil) {
		return fmt.Errorf("validation: %w", domain.StorageNotExpiredError(
			orderID, order.StorageUntil.Format("2006-01-02")))
	}

	if order.Status == domain.StatusInStorage {
		order.Status = domain.StatusReturnedWithoutClient
	} else {
		order.Status = domain.StatusGivenToCourier
	}
	order.LastUpdateTime = time.Now()

	if err := s.orderRepo.Update(order); err != nil {
		return fmt.Errorf("repo.Update: %w", err)
	}
	return nil
}
