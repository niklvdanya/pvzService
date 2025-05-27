package app

import (
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (s *PVZService) AcceptOrder(receiverID, orderID uint64, storageUntil time.Time) error {
	currentTime := time.Now()
	if storageUntil.Before(currentTime) {
		return fmt.Errorf("validation: %w", domain.ValidationFailedError(
			fmt.Sprintf("storage period already expired (current: %s, provided: %s)",
				currentTime.Format("2006-01-02"), storageUntil.Format("2006-01-02"))))
	}

	_, err := s.orderRepo.GetByID(orderID)
	if err == nil {
		return fmt.Errorf("repo.GetByID: %w", domain.OrderAlreadyExistsError(orderID))
	}
	if !isNotFoundError(err) {
		return fmt.Errorf("repo.GetByID: %w", err)
	}

	order := &domain.Order{
		OrderID:        orderID,
		ReceiverID:     receiverID,
		StorageUntil:   storageUntil,
		Status:         domain.StatusInStorage,
		AcceptTime:     currentTime,
		LastUpdateTime: currentTime,
	}
	if err := s.orderRepo.Save(order); err != nil {
		return fmt.Errorf("repo.Save: %w", err)
	}
	return nil
}
