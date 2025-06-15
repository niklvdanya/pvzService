package app

import (
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

type OrderRepository interface {
	Save(order *domain.Order) error
	GetByID(orderID uint64) (*domain.Order, error)
	Update(order *domain.Order) error
	GetByReceiverID(receiverID uint64) ([]*domain.Order, error)
	GetReturnedOrders() ([]*domain.Order, error)
	GetAllOrders() ([]*domain.Order, error)
}

type PVZService struct {
	orderRepo     OrderRepository
	packageConfig domain.PackageConfig
}

func NewPVZService(orderRepo OrderRepository, configPath string) (*PVZService, error) {
	packageConfig, err := domain.LoadPackageConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load package config: %w", err)
	}

	return &PVZService{
		orderRepo:     orderRepo,
		packageConfig: packageConfig,
	}, nil
}

func Paginate[T any](items []T, currentPage, itemsPerPage uint64) []T {
	totalItems := uint64(len(items))

	if itemsPerPage == 0 {
		return []T{}
	}
	if currentPage == 0 {
		currentPage = 1
	}

	startIndex := (currentPage - 1) * itemsPerPage
	endIndex := startIndex + itemsPerPage

	if startIndex >= totalItems {
		return []T{}
	}
	if endIndex > totalItems {
		endIndex = totalItems
	}

	return items[startIndex:endIndex]
}
