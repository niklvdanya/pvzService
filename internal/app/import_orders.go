package app

import (
	"context"
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"go.uber.org/multierr"
)

func (s *PVZService) ImportOrders(ctx context.Context, orders []domain.OrderToImport) (uint64, error) {
	var combinedErr error
	importedCount := uint64(0)
	for _, rawOrder := range orders {
		storageUntil, err := cli.MapStringToTime(rawOrder.StorageUntil)
		if err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("time.Parse: %w", err))
			continue
		}

		req := domain.AcceptOrderRequest{
			ReceiverID:   rawOrder.ReceiverID,
			OrderID:      rawOrder.OrderID,
			StorageUntil: storageUntil,
			Weight:       rawOrder.Weight,
			Price:        rawOrder.Price,
			PackageType:  rawOrder.PackageType,
		}
		_, err = s.AcceptOrder(ctx, req)
		if err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("AcceptOrder: %w", err))
			continue
		}
		importedCount++
	}
	return importedCount, combinedErr
}
