package app

import (
	"fmt"
	"time"

	"go.uber.org/multierr"
)

func (s *PVZService) ImportOrders(orders []struct {
	OrderID      uint64  `json:"order_id"`
	ReceiverID   uint64  `json:"receiver_id"`
	StorageUntil string  `json:"storage_until"`
	PackageType  string  `json:"package_type"`
	Weight       float64 `json:"weight"`
	Price        float64 `json:"price"`
}) (uint64, error) {
	var combinedErr error
	importedCount := uint64(0)
	for _, rawOrder := range orders {
		storageUntil, err := time.Parse("2006-01-02", rawOrder.StorageUntil)
		if err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("time.Parse: %w", err))
			continue
		}

		_, err = s.AcceptOrder(rawOrder.ReceiverID, rawOrder.OrderID, storageUntil, rawOrder.Weight, rawOrder.Price, rawOrder.PackageType)
		if err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("AcceptOrder: %w", err))
			continue
		}
		importedCount++
	}
	return importedCount, combinedErr
}
