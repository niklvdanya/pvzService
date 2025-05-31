package file

import (
	"encoding/json"
	"fmt"
	"os"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

type FileOrderRepository struct {
	ordersFilePath   string
	ordersByID       map[uint64]*domain.Order
	ordersByReceiver map[uint64]map[uint64]struct{}
}

func NewFileOrderRepository(path string) (*FileOrderRepository, error) {
	repo := &FileOrderRepository{
		ordersFilePath:   path,
		ordersByID:       make(map[uint64]*domain.Order),
		ordersByReceiver: make(map[uint64]map[uint64]struct{}),
	}

	if err := repo.loadFromFile(); err != nil {
		return nil, fmt.Errorf("loadFromFile: %w", err)
	}
	return repo, nil
}

func (r *FileOrderRepository) Save(order *domain.Order) error {
	if _, exists := r.ordersByID[order.OrderID]; exists {
		return fmt.Errorf("save: %w", domain.OrderAlreadyExistsError(order.OrderID))
	}
	r.ordersByID[order.OrderID] = order
	if _, exists := r.ordersByReceiver[order.ReceiverID]; !exists {
		r.ordersByReceiver[order.ReceiverID] = make(map[uint64]struct{})
	}
	r.ordersByReceiver[order.ReceiverID][order.OrderID] = struct{}{}

	if err := r.saveToFile(); err != nil {
		return fmt.Errorf("saveToFile: %w", err)
	}
	return nil
}

func (r *FileOrderRepository) GetByID(orderID uint64) (*domain.Order, error) {
	order, exists := r.ordersByID[orderID]
	if !exists {
		return nil, fmt.Errorf("getID: %w", domain.EntityNotFoundError("Order", fmt.Sprintf("%d", orderID)))
	}
	if order == nil {
		return nil, fmt.Errorf("getID: %w", domain.NilOrderError(orderID))
	}
	return order, nil
}

func (r *FileOrderRepository) GetByReceiverID(receiverID uint64) ([]*domain.Order, error) {
	orderIDs, exists := r.ordersByReceiver[receiverID]
	if !exists {
		return []*domain.Order{}, nil
	}

	orders := make([]*domain.Order, 0, len(orderIDs))
	for orderID := range orderIDs {
		if order, exists := r.ordersByID[orderID]; exists && order != nil && order.IsBelongsToReciever(receiverID) {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

func (r *FileOrderRepository) GetAllOrders() ([]*domain.Order, error) {
	orders := make([]*domain.Order, 0, len(r.ordersByID))
	for _, order := range r.ordersByID {
		if order != nil {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

func (r *FileOrderRepository) Update(order *domain.Order) error {
	if _, exists := r.ordersByID[order.OrderID]; !exists {
		return fmt.Errorf("update: %w", domain.EntityNotFoundError("Order", fmt.Sprintf("%d", order.OrderID)))
	}
	r.ordersByID[order.OrderID] = order

	if err := r.saveToFile(); err != nil {
		return fmt.Errorf("saveToFile: %w", err)
	}
	return nil
}

func (r *FileOrderRepository) GetReturnedOrders() ([]*domain.Order, error) {
	var returnedOrders []*domain.Order
	for _, order := range r.ordersByID {
		if order != nil && (order.Status == domain.StatusReturnedFromClient || order.Status == domain.StatusGivenToCourier) {
			returnedOrders = append(returnedOrders, order)
		}
	}
	return returnedOrders, nil
}

func (r *FileOrderRepository) loadFromFile() error {
	data, err := os.ReadFile(r.ordersFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("os.ReadFile: %w", err)
	}

	var orders []*domain.Order
	if err := json.Unmarshal(data, &orders); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	for _, order := range orders {
		if order == nil {
			continue
		}
		r.ordersByID[order.OrderID] = order
		if _, exists := r.ordersByReceiver[order.ReceiverID]; !exists {
			r.ordersByReceiver[order.ReceiverID] = make(map[uint64]struct{})
		}
		r.ordersByReceiver[order.ReceiverID][order.OrderID] = struct{}{}
	}
	return nil
}

func (r *FileOrderRepository) saveToFile() error {
	orders := make([]*domain.Order, 0, len(r.ordersByID))
	for _, order := range r.ordersByID {
		if order != nil {
			orders = append(orders, order)
		}
	}

	data, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	if err := os.WriteFile(r.ordersFilePath, data, 0644); err != nil {
		return fmt.Errorf("os.WriteFile: %w", err)
	}
	return nil
}
