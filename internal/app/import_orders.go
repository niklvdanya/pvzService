package app

import (
	"fmt"
	"time"

	"go.uber.org/multierr"
)

func (s *PVZService) ImportOrders(newOrders []struct {
	OrderID      uint64 `json:"order_id"`
	ReceiverID   uint64 `json:"receiver_id"`
	StorageUntil string `json:"storage_until"`
}) (uint64, error) {
	var importedCount uint64
	var combinedErr error

	for _, reqOrder := range newOrders {
		storageUntil, err := time.Parse("2006-01-02", reqOrder.StorageUntil)
		if err != nil {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf(
					"order %d: invalid storage until time format '%s': %w",
					reqOrder.OrderID,
					reqOrder.StorageUntil,
					err,
				),
			)
			continue
		}
		err = s.AcceptOrder(reqOrder.ReceiverID, reqOrder.OrderID, storageUntil)
		if err != nil {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf("order %d: failed to accept: %w", reqOrder.OrderID, err),
			)
			continue
		}
		importedCount++
	}
	return importedCount, combinedErr
}
