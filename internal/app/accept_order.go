package app

import (
	"context"
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (s *PVZService) AcceptOrder(ctx context.Context, req domain.AcceptOrderRequest) (float64, error) {
	currentTime := time.Now()
	if req.StorageUntil.Before(currentTime) {
		return 0, fmt.Errorf("validation: %w",
			domain.ValidationFailedError(
				fmt.Sprintf("storage period already expired (current: %s, provided: %s)",
					cli.MapTimeToString(currentTime), cli.MapTimeToString(req.StorageUntil))))
	}
	if req.Weight <= 0 {
		return 0, fmt.Errorf("validation: %w",
			domain.ValidationFailedError("weight must be greater than 0"))
	}
	if req.Price <= 0 {
		return 0, fmt.Errorf("validation: %w",
			domain.ValidationFailedError("price must be greater than 0"))
	}

	if _, err := s.orderRepo.GetByID(ctx, req.OrderID); err == nil {
		return 0, fmt.Errorf("repo.GetByID: %w",
			domain.OrderAlreadyExistsError(req.OrderID))
	}

	var rules []domain.PackageRules
	if req.PackageType != "" {
		var err error
		rules, err = s.orderRepo.GetPackageRules(ctx, req.PackageType)
		if err != nil {
			return 0, fmt.Errorf("validation: %w", err)
		}
	}

	totalPrice := req.Price
	for _, r := range rules {
		if r.MaxWeight > 0 && req.Weight > r.MaxWeight {
			return 0, fmt.Errorf("validation: %w",
				domain.WeightTooHeavyError(req.PackageType, req.Weight, r.MaxWeight))
		}
		totalPrice += r.Price
	}

	order := domain.Order{
		OrderID:        req.OrderID,
		ReceiverID:     req.ReceiverID,
		StorageUntil:   req.StorageUntil,
		Status:         domain.StatusInStorage,
		AcceptTime:     currentTime,
		LastUpdateTime: currentTime,
		PackageType:    req.PackageType,
		Weight:         req.Weight,
		Price:          totalPrice,
	}
	if err := s.orderRepo.Save(ctx, order); err != nil {
		return 0, fmt.Errorf("repo.Save: %w", err)
	}

	return totalPrice, nil
}
