package app

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"go.uber.org/multierr"
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

const parallelWorkers = 8

func (s *PVZService) ImportOrders(ctx context.Context, orders []domain.OrderToImport) (uint64, error) {
	sem := make(chan struct{}, parallelWorkers)

	var processed uint64
	var combined error
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, o := range orders {
		sem <- struct{}{}
		wg.Add(1)

		go func(ord domain.OrderToImport) {
			defer func() {
				<-sem
				wg.Done()
			}()
			if err := s.importSingle(ctx, ord); err != nil {
				mu.Lock()
				combined = multierr.Append(combined, err)
				mu.Unlock()
				return
			}
			atomic.AddUint64(&processed, 1)
		}(o)
	}

	wg.Wait()
	return processed, combined
}
