package app

import (
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (s *PVZService) AcceptOrder(receiverID, orderID uint64, storageUntil time.Time, weight, price float64, packageType string) (float64, error) {
	currentTime := time.Now()
	if storageUntil.Before(currentTime) {
		return 0, fmt.Errorf("validation: %w", domain.ValidationFailedError(
			fmt.Sprintf("storage period already expired (current: %s, provided: %s)",
				currentTime.Format("2006-01-02"), storageUntil.Format("2006-01-02"))))
	}
	if weight <= 0 {
		return 0, fmt.Errorf("validation: %w", domain.ValidationFailedError("weight must be greater than 0"))
	}
	if price <= 0 {
		return 0, fmt.Errorf("validation: %w", domain.ValidationFailedError("price must be greater than 0"))
	}
	if !s.packageConfig.IsValidPackageType(packageType) {
		return 0, fmt.Errorf("validation: %w", domain.InvalidPackageError(packageType))
	}

	_, err := s.orderRepo.GetByID(orderID)
	if err == nil {
		return 0, fmt.Errorf("repo.GetByID: %w", domain.OrderAlreadyExistsError(orderID))
	}

	totalPrice := price
	if packageType != "" {
		rules, _ := s.packageConfig.GetRules(packageType)
		for _, rule := range rules {
			if rule.MaxWeight > 0 && weight > rule.MaxWeight {
				return 0, fmt.Errorf("validation: %w", domain.WeightTooHeavyError(packageType, weight, rule.MaxWeight))
			}
			totalPrice += rule.Price
		}
	}

	order := &domain.Order{
		OrderID:        orderID,
		ReceiverID:     receiverID,
		StorageUntil:   storageUntil,
		Status:         domain.StatusInStorage,
		AcceptTime:     currentTime,
		LastUpdateTime: currentTime,
		PackageType:    packageType,
		Weight:         weight,
		Price:          totalPrice,
	}
	if err := s.orderRepo.Save(order); err != nil {
		return 0, fmt.Errorf("repo.Save: %w", err)
	}
	return totalPrice, nil
}
