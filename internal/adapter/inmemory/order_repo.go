package inmemory

import (
	"gitlab.ozon.dev/safariproxd/homework/internal/app"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

// для каждого получателя храню id заказов, для каждого заказа подробную информацию о нем
// наверное можно было бы использовать какую-то одну структуру(что-то типа ClientID->OrderID->Order)
// но например в некоторых тасках нам дан только id заказа и в таком случае пришлось бы пробегаться по всем пользователям и искать подходящий id заказа
// и наоборот, для каждого пользователя мы должны знать какие у него заказы.
var (
	ordersByID       map[uint64]*domain.Order       = make(map[uint64]*domain.Order)
	ordersByReceiver map[uint64]map[uint64]struct{} = make(map[uint64]map[uint64]struct{})
)

type InMemoryOrderRepository struct{}

func NewInMemoryOrderRepository() *InMemoryOrderRepository {
	return &InMemoryOrderRepository{}
}

func (r *InMemoryOrderRepository) Save(order *domain.Order) error {
	if _, exists := ordersByID[order.OrderID]; exists {
		return app.ErrOrderAlreadyExists
	}
	ordersByID[order.OrderID] = order
	if _, exists := ordersByReceiver[order.ReceiverID]; !exists {
		ordersByReceiver[order.ReceiverID] = make(map[uint64]struct{})
	}
	ordersByReceiver[order.ReceiverID][order.OrderID] = struct{}{}
	return nil
}

func (r *InMemoryOrderRepository) GetByID(orderID uint64) (*domain.Order, error) {
	order, exists := ordersByID[orderID]
	if !exists {
		return nil, app.ErrOrderNotFound
	}
	return order, nil
}

func (r *InMemoryOrderRepository) GetByReceiverID(receiverID uint64) ([]*domain.Order, error) {
	var receiverOrders []*domain.Order
	if orderIDs, exists := ordersByReceiver[receiverID]; exists {
		for orderID := range orderIDs {
			if order, exists := ordersByID[orderID]; exists && order.IsBelongsToReciever(receiverID) {
				receiverOrders = append(receiverOrders, order)
			}
		}
	}
	return receiverOrders, nil
}

func (r *InMemoryOrderRepository) GetAll() ([]*domain.Order, error) {
	var allOrders []*domain.Order
	for _, order := range ordersByID {
		allOrders = append(allOrders, order)
	}
	return allOrders, nil
}

func (r *InMemoryOrderRepository) Update(order *domain.Order) error {
	if _, exists := ordersByID[order.OrderID]; !exists {
		return app.ErrOrderNotFound
	}
	ordersByID[order.OrderID] = order
	return nil
}

func (r *InMemoryOrderRepository) GetReturnedOrders() ([]*domain.Order, error) {
	var returnedOrders []*domain.Order
	for _, order := range ordersByID {
		if order.Status == domain.StatusReturnedFromClient || order.Status == domain.StatusGivenToCourier {
			returnedOrders = append(returnedOrders, order)
		}
	}
	return returnedOrders, nil
}
