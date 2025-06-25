package app

import (
	"context"
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (s *PVZService) ReturnOrderToDelivery(ctx context.Context, orderID uint64) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("repo.GetByID: %w", err)
	}

	if order.Status != domain.StatusInStorage && order.Status != domain.StatusReturnedFromClient {
		return fmt.Errorf("validation: %w", domain.ValidationFailedError(
			fmt.Sprintf("order is not in storage (current status: %s)", order.GetStatusString())))
	}
	if time.Now().Before(order.StorageUntil) {
		return fmt.Errorf("validation: %w", domain.StorageNotExpiredError(
			orderID, cli.MapTimeToString(order.StorageUntil)))
	}

	newStatus := domain.StatusReturnedWithoutClient
	if order.Status == domain.StatusReturnedFromClient {
		newStatus = domain.StatusGivenToCourier
	}
	order.Status = newStatus
	order.LastUpdateTime = time.Now()

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("repo.Update: %w", err)
	}

	history := domain.OrderHistory{
		OrderID:   orderID,
		Status:    newStatus,
		ChangedAt: order.LastUpdateTime,
	}
	if err := s.orderRepo.SaveHistory(ctx, history); err != nil {
		return fmt.Errorf("repo.SaveHistory: %w", err)
	}

	return nil
}
