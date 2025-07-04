package app

import (
	"context"
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

// удалил часть валидации из бизнес логики, ибо в protoc validate она уже встроена
func (s *PVZService) AcceptOrder(ctx context.Context, req domain.AcceptOrderRequest) (float64, error) {
	currentTime := s.nowFn()

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

	history := domain.OrderHistory{
		OrderID:   req.OrderID,
		Status:    domain.StatusInStorage,
		ChangedAt: currentTime,
	}
	if err := s.orderRepo.SaveHistory(ctx, history); err != nil {
		return 0, fmt.Errorf("repo.SaveHistory: %w", err)
	}

	return totalPrice, nil
}
