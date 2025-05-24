package file

import (
	"encoding/json"
	"fmt"
	"os"

	"gitlab.ozon.dev/safariproxd/homework/internal/app"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/internal/port"
)

const (
	ordersFilePath         = "data/orders.json"
	returnedOrdersFilePath = "data/returned_orders.json"
)

func makeDir() error {
	return os.MkdirAll("data", 0755)
}

type fileOrderData struct {
	Orders           map[uint64]*domain.Order       `json:"orders"`
	OrdersByReceiver map[uint64]map[uint64]struct{} `json:"orders_by_receiver"`
}

type FileOrderRepository struct {
	cache *fileOrderData
}

func NewFileOrderRepository() (port.OrderRepository, error) {
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

func (r *FileOrderRepository) Delete(orderID uint64) error {
	order, exists := r.cache.Orders[orderID]
	if !exists {
		return app.ErrOrderNotFound
	}
	delete(r.cache.Orders, orderID)
	delete(r.cache.OrdersByReceiver[order.ReceiverID], orderID)
	if len(r.cache.OrdersByReceiver[order.ReceiverID]) == 0 {
		delete(r.cache.OrdersByReceiver, order.ReceiverID)
	}
	return r.saveToFile()
}

func (r *FileOrderRepository) GetByReceiverID(receiverID uint64) ([]*domain.Order, error) {
	var receiverOrders []*domain.Order
	if orderIDs, exists := r.cache.OrdersByReceiver[receiverID]; exists {
		for orderID := range orderIDs {
			if order, exists := r.cache.Orders[orderID]; exists {
				receiverOrders = append(receiverOrders, order)
			}
		}
	}
	return receiverOrders, nil
}

func (r *FileOrderRepository) GetAll() ([]*domain.Order, error) {
	var allOrders []*domain.Order
	for _, order := range r.cache.Orders {
		allOrders = append(allOrders, order)
	}
	return allOrders, nil
}

func (r *FileOrderRepository) Update(order *domain.Order) error {
	if _, exists := r.cache.Orders[order.OrderID]; !exists {
		return app.ErrOrderNotFound
	}
	r.cache.Orders[order.OrderID] = order
	return r.saveToFile()
}

type fileReturnedOrderData struct {
	ReturnedOrders map[uint64]*domain.ReturnedOrder `json:"returned_orders"`
}

type FileReturnedOrderRepository struct {
	cache *fileReturnedOrderData
}

func NewFileReturnedOrderRepository() (port.ReturnedOrderRepository, error) {
	if err := makeDir(); err != nil {
		return nil, fmt.Errorf("failed to ensure data directory: %w", err)
	}
	repo := &FileReturnedOrderRepository{
		cache: &fileReturnedOrderData{
			ReturnedOrders: make(map[uint64]*domain.ReturnedOrder),
		},
	}
	if err := repo.loadFromFile(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load returned orders from file: %w", err)
	}
	return repo, nil
}

func (r *FileReturnedOrderRepository) loadFromFile() error {
	data, err := os.ReadFile(returnedOrdersFilePath)
	if err != nil {
		return err
	}
	var loadedData fileReturnedOrderData
	if err := json.Unmarshal(data, &loadedData); err != nil {
		return fmt.Errorf("failed to unmarshal returned orders data: %w", err)
	}
	r.cache = &loadedData
	return nil
}

func (r *FileReturnedOrderRepository) saveToFile() error {
	data, err := json.MarshalIndent(r.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal returned orders data: %w", err)
	}
	tmpFilePath := returnedOrdersFilePath + ".tmp"
	if err := os.WriteFile(tmpFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary returned orders file: %w", err)
	}
	if err := os.Rename(tmpFilePath, returnedOrdersFilePath); err != nil {
		return fmt.Errorf("failed to rename temporary returned orders file: %w", err)
	}
	return nil
}

func (r *FileReturnedOrderRepository) Save(returnedOrder *domain.ReturnedOrder) error {
	r.cache.ReturnedOrders[returnedOrder.OrderID] = returnedOrder
	return r.saveToFile()
}

func (r *FileReturnedOrderRepository) GetAll() ([]*domain.ReturnedOrder, error) {
	var allReturned []*domain.ReturnedOrder
	for _, order := range r.cache.ReturnedOrders {
		allReturned = append(allReturned, order)
	}
	return allReturned, nil
}
