package app

import (
	"fmt"
	"sort"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (s *PVZService) getReceiverOrders(receiverID uint64, inPVZ bool, lastN, page, limit uint64) ([]*domain.Order, uint64, error) {
	if receiverID == 0 {
		return nil, 0, fmt.Errorf("validation: %w", domain.ValidationFailedError("receiver ID cannot be empty"))
	}

	receiverOrders, err := s.orderRepo.GetByReceiverID(receiverID)
	if err != nil {
		return nil, 0, fmt.Errorf("repo.GetByReceiverID: %w", err)
	}

	var filteredOrders []*domain.Order
	for _, order := range receiverOrders {
		if inPVZ && order.Status != domain.StatusInStorage {
			continue
		}
		filteredOrders = append(filteredOrders, order)
	}

	var paginatedOrders []*domain.Order
	totalItems := uint64(len(filteredOrders))

	if lastN > 0 {
		if totalItems > lastN {
			paginatedOrders = filteredOrders[totalItems-lastN:]
		} else {
			paginatedOrders = filteredOrders
		}
	} else {
		paginatedOrders = Paginate(filteredOrders, page, limit)
	}

	return paginatedOrders, totalItems, nil
}

func (s *PVZService) getReturnedOrders(page, limit uint64) ([]*domain.Order, uint64, error) {
	returnOrders, err := s.orderRepo.GetReturnedOrders()
	if err != nil {
		return nil, 0, fmt.Errorf("repo.GetReturnedOrders: %w", err)
	}

	paginated := Paginate(returnOrders, page, limit)
	return paginated, uint64(len(returnOrders)), nil
}

func (s *PVZService) getOrderHistory() ([]*domain.Order, error) {
	orders, err := s.orderRepo.GetAllOrders()
	if err != nil {
		return nil, fmt.Errorf("repo.GetAllOrders: %w", err)
	}

	sort.Slice(orders, func(i, j int) bool {
		return orders[i].LastUpdateTime.After(orders[j].LastUpdateTime)
	})
	return orders, nil
}

func (s *PVZService) getReceiverOrdersScroll(receiverID uint64, lastID, limit uint64) ([]*domain.Order, uint64, error) {
	if receiverID == 0 {
		return nil, 0, fmt.Errorf("validation: %w", domain.ValidationFailedError("receiver ID cannot be empty"))
	}

	receiverOrders, err := s.orderRepo.GetByReceiverID(receiverID)
	if err != nil {
		return nil, 0, fmt.Errorf("repo.GetByReceiverID: %w", err)
	}

	sort.Slice(receiverOrders, func(i, j int) bool {
		return receiverOrders[i].OrderID < receiverOrders[j].OrderID
	})

	totalItems := uint64(len(receiverOrders))
	var resultOrders []*domain.Order
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
			return []*domain.Order{}, totalItems, nil
		}
	}

	startOffset := startIndex + 1
	if startOffset >= len(receiverOrders) {
		return []*domain.Order{}, totalItems, nil
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
