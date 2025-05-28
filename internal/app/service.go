package app

import (
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
	orderRepo OrderRepository
}

func NewPVZService(
	orderRepo OrderRepository,
) *PVZService {
	return &PVZService{
		orderRepo: orderRepo,
	}
}

func paginate[T any](items []T, currentPage, itemsPerPage uint64) []T {
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
