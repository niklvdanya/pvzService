package app

import (
	"context"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

type OrderRepository interface {
	Save(ctx context.Context, order domain.Order) error
	GetByID(ctx context.Context, orderID uint64) (domain.Order, error)
	Update(ctx context.Context, order domain.Order) error
	GetByReceiverID(ctx context.Context, receiverID uint64) ([]domain.Order, error)
	GetReturnedOrders(ctx context.Context) ([]domain.Order, error)
	GetAllOrders(ctx context.Context) ([]domain.Order, error)
	GetPackageRules(ctx context.Context, code string) ([]domain.PackageRules, error)
	SaveHistory(ctx context.Context, history domain.OrderHistory) error
	GetHistoryByOrderID(ctx context.Context, orderID uint64) ([]domain.OrderHistory, error)
}

type PVZService struct {
	orderRepo OrderRepository
	nowFn     func() time.Time
}

func NewPVZService(orderRepo OrderRepository, nowFn func() time.Time) *PVZService {
	if nowFn == nil {
		nowFn = time.Now // Значение по умолчанию — time.Now
	}
	return &PVZService{
		orderRepo: orderRepo,
		nowFn:     nowFn,
	}
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
