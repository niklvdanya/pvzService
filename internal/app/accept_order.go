package app

import (
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (s *PVZService) acceptOrder(req domain.AcceptOrderRequest) (float64, error) {
	currentTime := time.Now()
	if req.StorageUntil.Before(currentTime) {
		return 0, fmt.Errorf("validation: %w", domain.ValidationFailedError(
			fmt.Sprintf("storage period already expired (current: %s, provided: %s)",
				cli.MapTimeToString(currentTime), cli.MapTimeToString(req.StorageUntil))))
	}
	if req.Weight <= 0 {
		return 0, fmt.Errorf("validation: %w", domain.ValidationFailedError("weight must be greater than 0"))
	}
	if req.Price <= 0 {
		return 0, fmt.Errorf("validation: %w", domain.ValidationFailedError("price must be greater than 0"))
	}
	if !s.packageConfig.IsValidPackageType(req.PackageType) {
		return 0, fmt.Errorf("validation: %w", domain.InvalidPackageError(req.PackageType))
	}

	_, err := s.orderRepo.GetByID(req.OrderID)
	if err == nil {
		return 0, fmt.Errorf("repo.GetByID: %w", domain.OrderAlreadyExistsError(req.OrderID))
	}

	totalPrice := req.Price
	if req.PackageType != "" {
		rules, _ := s.packageConfig.GetRules(req.PackageType)
		for _, rule := range rules {
			if rule.MaxWeight > 0 && req.Weight > rule.MaxWeight {
				return 0, fmt.Errorf("validation: %w", domain.WeightTooHeavyError(req.PackageType, req.Weight, rule.MaxWeight))
			}
			totalPrice += rule.Price
		}
	}

	order := &domain.Order{
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
	if err := s.orderRepo.Save(order); err != nil {
		return 0, fmt.Errorf("repo.Save: %w", err)
	}
	return totalPrice, nil
}
