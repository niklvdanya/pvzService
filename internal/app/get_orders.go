package app

import (
	"fmt"
	"sort"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (s *PVZService) GetReturnedOrders(page, limit uint64) ([]*domain.Order, uint64, error) {
	returnOrders, err := s.orderRepo.GetReturnedOrders()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get all returned orders: %w", err)
	}

	paginated := paginate(returnOrders, page, limit)
	return paginated, uint64(len(returnOrders)), nil
}

func (s *PVZService) GetReceiverOrders(
	receiverID uint64,
	inPVZ bool,
	lastN uint64,
	page, limit uint64,
) ([]*domain.Order, uint64, error) {
	receiverOrders, err := s.orderRepo.GetByReceiverID(receiverID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get orders for receiver %d: %w", receiverID, err)
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
		paginatedOrders = paginate(filteredOrders, page, limit)
	}

	return paginatedOrders, totalItems, nil
}

func (s *PVZService) GetOrderHistory() ([]*domain.Order, error) {
	allOrders, err := s.orderRepo.GetAllOrders()
	if err != nil {
		return nil, fmt.Errorf("failed to get all orders for history: %w", err)
	}
	sort.Slice(allOrders, func(i, j int) bool {
		return allOrders[i].LastUpdateTime.After(allOrders[j].LastUpdateTime)
	})
	return allOrders, nil
}

// по сути аналог GetReceiverOrders, но здесь нам важно сохранять последний полученный с функции заказ
func (s *PVZService) GetReceiverOrdersScroll(
	receiverID uint64,
	lastID, limit uint64,
) ([]*domain.Order, uint64, error) {
	receiverOrders, err := s.orderRepo.GetByReceiverID(receiverID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get orders for receiver %d: %w", receiverID, err)
	}

	filteredOrders := receiverOrders
	// для этого метода решил сделать сортировку для удобства, хоть это и не прописано в задании
	sort.Slice(filteredOrders, func(i, j int) bool {
		return filteredOrders[i].OrderID < filteredOrders[j].OrderID
	})

	totalItems := uint64(len(filteredOrders))
	var resultOrders []*domain.Order
	var nextLastID uint64 = 0

	startIndex := -1
	if lastID > 0 {
		for i, order := range filteredOrders {
			if order.OrderID == lastID {
				startIndex = i
				break
			}
		}

		if startIndex == -1 {
			return []*domain.Order{}, totalItems, nil
		}
	}
	// каждый раз берем n = limit заказов и если что проверяем границы
	// и обновляем постоянно lastID
	startOffset := startIndex + 1
	if startOffset >= len(filteredOrders) {
		return []*domain.Order{}, totalItems, nil
	}

	endOffset := startOffset + int(limit)
	if endOffset > len(filteredOrders) {
		endOffset = len(filteredOrders)
	}
	resultOrders = filteredOrders[startOffset:endOffset]

	if len(resultOrders) > 0 && endOffset < len(filteredOrders) {
		nextLastID = resultOrders[len(resultOrders)-1].OrderID
	} else {
		nextLastID = 0
	}

	return resultOrders, nextLastID, nil
}
