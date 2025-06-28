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
	if orderID == 0 {
		return nil, fmt.Errorf("validation: %w", domain.ValidationFailedError("order ID cannot be empty"))
	}

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
func (s *PVZService) GetReceiverOrdersScroll(ctx context.Context, receiverID uint64, lastID, limit uint64) ([]domain.Order, uint64, error) {
	if receiverID == 0 {
		return nil, 0, fmt.Errorf("validation: %w", domain.ValidationFailedError("receiver ID cannot be empty"))
	}

	receiverOrders, err := s.orderRepo.GetByReceiverID(ctx, receiverID)
	if err != nil {
		return nil, 0, fmt.Errorf("repo.GetByReceiverID: %w", err)
	}

	sort.Slice(receiverOrders, func(i, j int) bool {
		return receiverOrders[i].OrderID < receiverOrders[j].OrderID
	})

	totalItems := uint64(len(receiverOrders))
	var resultOrders []domain.Order
	var nextLastID uint64

	startIndex := -1
	if lastID > 0 {
		for i, order := range receiverOrders {
			if order.OrderID == lastID {
				startIndex = i
				break
			}
		}
		if startIndex == -1 {
			return []domain.Order{}, totalItems, nil
		}
	}

	startOffset := startIndex + 1
	if startOffset >= len(receiverOrders) {
		return []domain.Order{}, totalItems, nil
	}

	endOffset := startOffset + int(limit)
	if endOffset > len(receiverOrders) {
		endOffset = len(receiverOrders)
	}
	resultOrders = receiverOrders[startOffset:endOffset]

	if len(resultOrders) > 0 && endOffset < len(receiverOrders) {
		nextLastID = resultOrders[len(resultOrders)-1].OrderID
	}

	return resultOrders, nextLastID, nil
}
