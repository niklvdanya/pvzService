package app

import (
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (s *PVZService) AcceptOrder(receiverID, orderID uint64, storageUntil time.Time) error {
	currentTime := time.Now()

	if storageUntil.Before(currentTime) {
		return fmt.Errorf(
			"cannot accept order %d: storage period already expired. Current time: %s, Provided until: %s",
			orderID,
			currentTime.Format("2006-01-02"),
			storageUntil.Format("2006-01-02"),
		)
	}

	_, err := s.orderRepo.GetByID(orderID)
	if err == nil {
		return fmt.Errorf("cannot accept order %d: %w", orderID, ErrOrderAlreadyExists)
	}

	order := &domain.Order{
		OrderID:        orderID,
		ReceiverID:     receiverID,
		StorageUntil:   storageUntil,
		Status:         domain.StatusInStorage,
		AcceptTime:     currentTime,
		LastUpdateTime: currentTime,
	}
	return s.orderRepo.Save(order)
}
