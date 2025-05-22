package app

import (
	"fmt"
	"sort"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/docs/homework-1/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/docs/homework-1/internal/port"
	"gitlab.ozon.dev/safariproxd/homework/docs/homework-1/internal/util"

	"go.uber.org/multierr"
)

type PVZService struct {
	orderRepo         port.OrderRepository
	returnedOrderRepo port.ReturnedOrderRepository
}

func NewPVZService(orderRepo port.OrderRepository, returnedOrderRepo port.ReturnedOrderRepository) port.OrderService {
	return &PVZService{
		orderRepo:         orderRepo,
		returnedOrderRepo: returnedOrderRepo,
	}
}

func (s *PVZService) AcceptOrder(receiverID, orderID uint64, storageUntil time.Time) error {
	currentTimeInMoscow := util.NowInMoscow()

	if storageUntil.Before(currentTimeInMoscow) {
		return fmt.Errorf("cannot accept order %d: storage period already expired. Current time: %s, Provided until: %s", orderID, currentTimeInMoscow.Format("2006-01-02 15:04"), storageUntil.Format("2006-01-02 15:04"))
	}

	_, err := s.orderRepo.GetByID(orderID)
	if err == nil {
		return fmt.Errorf("cannot accept order %d: %w", orderID, ErrOrderAlreadyExists)
	}

	order := &domain.Order{
		OrderID:        orderID,
		ReceiverID:     receiverID,
		StorageUntil:   storageUntil,
		Status:         domain.StatusInStorage,
		AcceptTime:     currentTimeInMoscow,
		LastUpdateTime: currentTimeInMoscow,
	}
	return s.orderRepo.Save(order)
}

func (s *PVZService) ReturnOrderToDelivery(orderID uint64) error {
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return fmt.Errorf("cannot return order %d to delivery: %w", orderID, err)
	}

	if order.Status != domain.StatusInStorage && order.Status != domain.StatusReturnedFromClient {
		return fmt.Errorf("cannot return order %d to delivery: order is not in storage (current status: %s)", orderID, order.GetStatusString())
	}
	if util.NowInMoscow().Before(order.StorageUntil) {
		return fmt.Errorf("cannot return order %d to delivery: %w (until: %s)", orderID, ErrStorageNotExpired, order.StorageUntil.Format("2006-01-02 15:04"))
	}

	return s.orderRepo.Delete(orderID)
}

func (s *PVZService) IssueOrdersToClient(receiverID uint64, orderIDs []uint64) error {
	var combinedErr error
	currentTime := util.NowInMoscow()

	for _, orderID := range orderIDs {
		order, err := s.orderRepo.GetByID(orderID)
		if err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("order %d: %w", orderID, ErrOrderNotFound))
			continue
		}

		if order.ReceiverID != receiverID {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("order %d: %w (expected %d, got %d)", orderID, ErrBelongsToDifferentReceiver, receiverID, order.ReceiverID))
			continue
		}

		if order.Status == domain.StatusGivenToClient {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("order %d: %w", orderID, ErrOrderAlreadyGiven))
			continue
		}
		// не может же клиент вернуть заказа и потом снова его забрать :)
		if order.Status == domain.StatusReturnedFromClient {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("order %d: %w", orderID, ErrUnavaliableReturnedOrder))
			continue
		}
		if currentTime.After(order.StorageUntil) {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("order %d: %w (%s)", orderID, ErrStorageExpired, order.StorageUntil.Format("2006-01-02 15:04")))
			continue
		}

		order.Status = domain.StatusGivenToClient
		order.LastUpdateTime = currentTime
		if err := s.orderRepo.Update(order); err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("order %d: failed to update status: %w", orderID, err))
		}
	}
	return combinedErr
}

func (s *PVZService) ReturnOrdersFromClient(receiverID uint64, orderIDs []uint64) error {
	var combinedErr error
	currentTimeInMoscow := util.NowInMoscow()

	for _, orderID := range orderIDs {
		order, err := s.orderRepo.GetByID(orderID)
		if err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("order %d: %w", orderID, ErrOrderNotFound))
			continue
		}

		if order.ReceiverID != receiverID {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("order %d: %w (expected %d, got %d)", orderID, ErrBelongsToDifferentReceiver, receiverID, order.ReceiverID))
			continue
		}

		if order.Status == domain.StatusInStorage {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("order %d: %w", orderID, ErrAlreadyInStorage))
			continue
		}

		timeSinceGiven := currentTimeInMoscow.Sub(order.LastUpdateTime)
		twoDaysLimit := 48 * time.Hour

		if timeSinceGiven > twoDaysLimit {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("order %d: %w (%.1f hours)",
				orderID, ErrReturnPeriodExpired, timeSinceGiven.Hours()))
			continue
		}

		returnedOrder := &domain.ReturnedOrder{
			OrderID:    order.OrderID,
			ReceiverID: order.ReceiverID,
			ReturnedAt: currentTimeInMoscow,
		}
		if err := s.returnedOrderRepo.Save(returnedOrder); err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("order %d: failed to save returned order: %w", orderID, err))
			continue
		}

		order.Status = domain.StatusReturnedFromClient
		order.LastUpdateTime = currentTimeInMoscow
		if err := s.orderRepo.Update(order); err != nil {
			combinedErr = multierr.Append(combinedErr, fmt.Errorf("order %d: failed to update order status to returned: %w", orderID, err))
		}
	}
	return combinedErr
}

func (s *PVZService) GetReturnedOrders(page, limit uint64) ([]*domain.ReturnedOrder, uint64, error) {
	allReturned, err := s.returnedOrderRepo.GetAll()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get all returned orders: %w", err)
	}

	paginated := paginate(allReturned, page, limit)
	return paginated, uint64(len(allReturned)), nil
}

func (s *PVZService) GetReceiverOrders(receiverID uint64, inPVZ bool, page, limit uint64) ([]*domain.Order, uint64, error) {
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

	paginatedOrders = paginate(filteredOrders, page, limit)

	return paginatedOrders, totalItems, nil
}

func (s *PVZService) GetOrderHistory() ([]*domain.Order, error) {
	allOrders, err := s.orderRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all orders for history: %w", err)
	}
	sort.Slice(allOrders, func(i, j int) bool {
		return allOrders[i].LastUpdateTime.After(allOrders[j].LastUpdateTime)
	})
	return allOrders, nil
}

// мб надо вынести в pkg или utils
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
