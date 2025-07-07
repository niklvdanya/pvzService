package app

import (
	"context"
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (s *PVZService) importSingle(ctx context.Context, raw domain.OrderToImport) error {
	storageUntil, err := cli.MapStringToTime(raw.StorageUntil)
	if err != nil {
		return fmt.Errorf("time.Parse: %w", err)
	}
	req := domain.AcceptOrderRequest{
		ReceiverID:   raw.ReceiverID,
		OrderID:      raw.OrderID,
		StorageUntil: storageUntil,
		Weight:       raw.Weight,
		Price:        raw.Price,
		PackageType:  raw.PackageType,
	}
	_, err = s.AcceptOrder(ctx, req)
	return err
}

func (s *PVZService) ImportOrders(
	ctx context.Context,
	orders []domain.OrderToImport,
) (uint64, error) {
	return processConcurrently(ctx, orders, s.workerLimit, s.importSingle)
}
