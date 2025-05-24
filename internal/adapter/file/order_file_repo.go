package file

import (
	"encoding/json"
	"fmt"
	"os"

	"gitlab.ozon.dev/safariproxd/homework/internal/app"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

const (
	ordersFilePath = "data/orders.json"
)

func makeDir() error {
	return os.MkdirAll("data", 0755)
}

type fileOrderData struct {
	Orders           map[uint64]*domain.Order
	OrdersByReceiver map[uint64]map[uint64]struct{}
}

type FileOrderRepository struct {
	cache *fileOrderData
}

func NewFileOrderRepository() (*FileOrderRepository, error) {
	if err := makeDir(); err != nil {
		return nil, fmt.Errorf("failed to ensure data directory: %w", err)
	}
	repo := &FileOrderRepository{
		cache: &fileOrderData{
			Orders:           make(map[uint64]*domain.Order),
			OrdersByReceiver: make(map[uint64]map[uint64]struct{}),
		},
	}
	if err := repo.loadFromFile(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load orders from file: %w", err)
	}
	return repo, nil
}

func (r *FileOrderRepository) loadFromFile() error {
	data, err := os.ReadFile(ordersFilePath)
	if err != nil {
		return err
	}

	var loadedData fileOrderData
	if err := json.Unmarshal(data, &loadedData); err != nil {
		return fmt.Errorf("failed to unmarshal orders data: %w", err)
	}
	r.cache = &loadedData
	return nil
}

func (r *FileOrderRepository) saveToFile() error {
	data, err := json.MarshalIndent(r.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal orders data: %w", err)
	}

	tmpFilePath := ordersFilePath + ".tmp"
	if err := os.WriteFile(tmpFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary orders file: %w", err)
	}
	if err := os.Rename(tmpFilePath, ordersFilePath); err != nil {
		return fmt.Errorf("failed to rename temporary orders file: %w", err)
	}
	return nil
}

func (r *FileOrderRepository) Save(order *domain.Order) error {
	if _, exists := r.cache.Orders[order.OrderID]; exists {
		return app.ErrOrderAlreadyExists
	}
	r.cache.Orders[order.OrderID] = order
	if _, exists := r.cache.OrdersByReceiver[order.ReceiverID]; !exists {
		r.cache.OrdersByReceiver[order.ReceiverID] = make(map[uint64]struct{})
	}
	r.cache.OrdersByReceiver[order.ReceiverID][order.OrderID] = struct{}{}

	return r.saveToFile()
}

func (r *FileOrderRepository) GetByID(orderID uint64) (*domain.Order, error) {
	order, exists := r.cache.Orders[orderID]
	if !exists {
		return nil, app.ErrOrderNotFound
	}
	return order, nil
}

func (r *FileOrderRepository) Update(order *domain.Order) error {
	if _, exists := r.cache.Orders[order.OrderID]; !exists {
		return app.ErrOrderNotFound
	}
	r.cache.Orders[order.OrderID] = order
	return r.saveToFile()
}

func (r *FileOrderRepository) GetByReceiverID(receiverID uint64) ([]*domain.Order, error) {
	orderIDs, exists := r.cache.OrdersByReceiver[receiverID]
	if !exists {
		return []*domain.Order{}, nil
	}

	orders := make([]*domain.Order, 0, len(orderIDs))
	for orderID := range orderIDs {
		if order, exists := r.cache.Orders[orderID]; exists && order.IsBelongsToReciever(receiverID) {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

func (r *FileOrderRepository) GetReturnedOrders() ([]*domain.Order, error) {
	var returnedOrders []*domain.Order
	for _, order := range r.cache.Orders {
		// заказ из списка возвратов, если 1) его вернул клиент и он в хранилище
		// 2) его вернул клиент и он был выдан курьеру
		// если заказ был принят курьером и сразу выдан обратно, то я такие заказы не считаю возвратами
		if order.Status == domain.StatusReturnedFromClient || order.Status == domain.StatusGivenToCourier {
			returnedOrders = append(returnedOrders, order)
		}
	}
	return returnedOrders, nil
}

func (r *FileOrderRepository) GetAllOrders() ([]*domain.Order, error) {
	orders := make([]*domain.Order, 0, len(r.cache.Orders))
	for _, order := range r.cache.Orders {
		orders = append(orders, order)
	}
	return orders, nil
}
