package app

import (
	"context"
	"fmt"
	"sync"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"golang.org/x/sync/errgroup"
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

func (s *PVZService) ImportOrders(ctx context.Context, orders []domain.OrderToImport) (uint64, error) {
	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, s.workerLimit)

	var cnt uint64
	var mu sync.Mutex

	for _, o := range orders {
		sem <- struct{}{}
		g.Go(func() error {
			defer func() { <-sem }()
			if err := s.importSingle(ctx, o); err != nil {
				return err
			}
			mu.Lock()
			cnt++
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return cnt, err
	}
	return cnt, nil
}
