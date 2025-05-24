package inmemory

import (
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

type InMemoryOrderRepository struct {
	ordersByID       map[uint64]*domain.Order
	ordersByReceiver map[uint64]map[uint64]struct{}
}

func NewInMemoryOrderRepository() *InMemoryOrderRepository {
	return &InMemoryOrderRepository{
		ordersByID:       make(map[uint64]*domain.Order),
		ordersByReceiver: make(map[uint64]map[uint64]struct{}),
	}
}

func (r *InMemoryOrderRepository) Save(order *domain.Order) error {
	if _, exists := r.ordersByID[order.OrderID]; exists {
		return fmt.Errorf("save: %w", domain.OrderAlreadyExistsError(order.OrderID))
	}
	r.ordersByID[order.OrderID] = order
	if _, exists := r.ordersByReceiver[order.ReceiverID]; !exists {
		r.ordersByReceiver[order.ReceiverID] = make(map[uint64]struct{})
	}
	r.ordersByReceiver[order.ReceiverID][order.OrderID] = struct{}{}
	return nil
}

func (r *InMemoryOrderRepository) GetByID(orderID uint64) (*domain.Order, error) {
	order, exists := r.ordersByID[orderID]
	if !exists {
		return nil, fmt.Errorf("get: %w", domain.EntityNotFoundError("Order", fmt.Sprintf("%d", orderID)))
	}
	return order, nil
}

func (r *InMemoryOrderRepository) GetByReceiverID(receiverID uint64) ([]*domain.Order, error) {
	orderIDs, exists := r.ordersByReceiver[receiverID]
	if !exists {
		return []*domain.Order{}, nil
	}

	orders := make([]*domain.Order, 0, len(orderIDs))
	for orderID := range orderIDs {
		if order, exists := r.ordersByID[orderID]; exists && order.IsBelongsToReciever(receiverID) {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

func (r *InMemoryOrderRepository) GetAllOrders() ([]*domain.Order, error) {
	orders := make([]*domain.Order, 0, len(r.ordersByID))
	for _, order := range r.ordersByID {
		orders = append(orders, order)
	}
	return orders, nil
}

func (r *InMemoryOrderRepository) Update(order *domain.Order) error {
	if _, exists := r.ordersByID[order.OrderID]; !exists {
		return fmt.Errorf("update: %w", domain.EntityNotFoundError("Order", fmt.Sprintf("%d", order.OrderID)))
	}
	r.ordersByID[order.OrderID] = order
	return nil
}

func (r *InMemoryOrderRepository) GetReturnedOrders() ([]*domain.Order, error) {
	var returnedOrders []*domain.Order
	for _, order := range r.ordersByID {
		if order.Status == domain.StatusReturnedFromClient || order.Status == domain.StatusGivenToCourier {
			returnedOrders = append(returnedOrders, order)
		}
	}
	return returnedOrders, nil
}
