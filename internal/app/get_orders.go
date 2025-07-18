package app

import (
	"context"
	"fmt"
	"sort"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (s *PVZService) GetReceiverOrders(ctx context.Context, req domain.ReceiverOrdersRequest) ([]domain.Order, uint64, error) {
	receiverOrders, err := s.orderRepo.GetByReceiverID(ctx, req.ReceiverID)
	if err != nil {
		return nil, 0, fmt.Errorf("repo.GetByReceiverID: %w", err)
	}

	var filteredOrders []domain.Order
	for _, order := range receiverOrders {
		if req.InPVZ && order.Status != domain.StatusInStorage {
			continue
		}
		filteredOrders = append(filteredOrders, order)
	}

	var paginatedOrders []domain.Order
	totalItems := uint64(len(filteredOrders))

	if req.LastN > 0 {
		if totalItems > req.LastN {
			paginatedOrders = filteredOrders[totalItems-req.LastN:]
		} else {
			paginatedOrders = filteredOrders
		}
	} else {
		paginatedOrders = Paginate(filteredOrders, req.Page, req.Limit)
	}

	return paginatedOrders, totalItems, nil
}

func (s *PVZService) GetReturnedOrders(ctx context.Context, page, limit uint64) ([]domain.Order, uint64, error) {
	returnOrders, err := s.orderRepo.GetReturnedOrders(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("repo.GetReturnedOrders: %w", err)
	}

	paginated := Paginate(returnOrders, page, limit)
	return paginated, uint64(len(returnOrders)), nil
}

func (s *PVZService) GetOrderHistory(ctx context.Context) ([]domain.Order, error) {
	orders, err := s.orderRepo.GetAllOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("repo.GetAllOrders: %w", err)
	}

	sort.Slice(orders, func(i, j int) bool {
		return orders[i].LastUpdateTime.After(orders[j].LastUpdateTime)
	})
	return orders, nil
}

func (s *PVZService) GetOrderHistoryByID(ctx context.Context, orderID uint64) ([]domain.OrderHistory, error) {
	history, err := s.orderRepo.GetHistoryByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("repo.GetHistoryByOrderID: %w", err)
	}

	if len(history) == 0 {
		return nil, fmt.Errorf("history: %w", domain.EntityNotFoundError("Order", fmt.Sprintf("%d", orderID)))
	}
	sort.Slice(history, func(i, j int) bool {
		return history[i].ChangedAt.After(history[j].ChangedAt)
	})

	return history, nil
}
