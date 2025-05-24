package app

import (
	"fmt"
	"sort"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/internal/port"
	"gitlab.ozon.dev/safariproxd/homework/internal/util"

	"go.uber.org/multierr"
)

type PVZService struct {
	orderRepo         port.OrderRepository
	returnedOrderRepo port.ReturnedOrderRepository
}

func NewPVZService(
	orderRepo port.OrderRepository,
	returnedOrderRepo port.ReturnedOrderRepository,
) port.OrderService {
	return &PVZService{
		orderRepo:         orderRepo,
		returnedOrderRepo: returnedOrderRepo,
	}
}

func (s *PVZService) AcceptOrder(receiverID, orderID uint64, storageUntil time.Time) error {
	currentTimeInMoscow := util.NowInMoscow()

	if storageUntil.Before(currentTimeInMoscow) {
		return fmt.Errorf(
			"cannot accept order %d: storage period already expired. Current time: %s, Provided until: %s",
			orderID,
			currentTimeInMoscow.Format("2006-01-02 15:04"),
			storageUntil.Format("2006-01-02 15:04"),
		)
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
		return fmt.Errorf(
			"cannot return order %d to delivery: order is not in storage (current status: %s)",
			orderID,
			order.GetStatusString(),
		)
	}
	if util.NowInMoscow().Before(order.StorageUntil) {
		return fmt.Errorf(
			"cannot return order %d to delivery: %w (until: %s)",
			orderID,
			ErrStorageNotExpired,
			order.StorageUntil.Format("2006-01-02 15:04"),
		)
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
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf(
					"order %d: %w (expected %d, got %d)",
					orderID,
					ErrBelongsToDifferentReceiver,
					receiverID,
					order.ReceiverID,
				),
			)
			continue
		}

		if order.Status == domain.StatusGivenToClient {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf("order %d: %w", orderID, ErrOrderAlreadyGiven),
			)
			continue
		}
		// не может же клиент вернуть заказа и потом снова его забрать :)
		if order.Status == domain.StatusReturnedFromClient {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf("order %d: %w", orderID, ErrUnavaliableReturnedOrder),
			)
			continue
		}
		if currentTime.After(order.StorageUntil) {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf(
					"order %d: %w (%s)",
					orderID,
					ErrStorageExpired,
					order.StorageUntil.Format("2006-01-02 15:04"),
				),
			)
			continue
		}

		order.Status = domain.StatusGivenToClient
		order.LastUpdateTime = currentTime
		if err := s.orderRepo.Update(order); err != nil {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf("order %d: failed to update status: %w", orderID, err),
			)
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
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf(
					"order %d: %w (expected %d, got %d)",
					orderID,
					ErrBelongsToDifferentReceiver,
					receiverID,
					order.ReceiverID,
				),
			)
			continue
		}

		if order.Status == domain.StatusInStorage {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf("order %d: %w", orderID, ErrAlreadyInStorage),
			)
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
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf("order %d: failed to save returned order: %w", orderID, err),
			)
			continue
		}

		order.Status = domain.StatusReturnedFromClient
		order.LastUpdateTime = currentTimeInMoscow
		if err := s.orderRepo.Update(order); err != nil {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf("order %d: failed to update order status to returned: %w", orderID, err),
			)
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
	allOrders, err := s.orderRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all orders for history: %w", err)
	}
	sort.Slice(allOrders, func(i, j int) bool {
		return allOrders[i].LastUpdateTime.After(allOrders[j].LastUpdateTime)
	})
	return allOrders, nil
}

func (s *PVZService) ImportOrders(newOrders []struct {
	OrderID      uint64 `json:"order_id"`
	ReceiverID   uint64 `json:"receiver_id"`
	StorageUntil string `json:"storage_until"`
}) (uint64, error) {
	var importedCount uint64
	var combinedErr error

	moscowLoc := util.GetMoscowLocation()

	for _, reqOrder := range newOrders {
		storageUntil, err := time.ParseInLocation("2006-01-02_15:04", reqOrder.StorageUntil, moscowLoc)
		if err != nil {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf(
					"order %d: invalid storage until time format '%s': %w",
					reqOrder.OrderID,
					reqOrder.StorageUntil,
					err,
				),
			)
			continue
		}
		err = s.AcceptOrder(reqOrder.ReceiverID, reqOrder.OrderID, storageUntil)
		if err != nil {
			combinedErr = multierr.Append(
				combinedErr,
				fmt.Errorf("order %d: failed to accept: %w", reqOrder.OrderID, err),
			)
			continue
		}
		importedCount++
	}
	return importedCount, combinedErr
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
