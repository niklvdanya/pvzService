package port

import (
	"gitlab.ozon.dev/safariproxd/homework/docs/homework-1/internal/domain"
)

type OrderRepository interface {
	Save(order *domain.Order) error
	GetByID(orderID uint64) (*domain.Order, error)
	Delete(orderID uint64) error
	GetByReceiverID(receiverID uint64) ([]*domain.Order, error)
	GetAll() ([]*domain.Order, error)
	Update(order *domain.Order) error
}

type ReturnedOrderRepository interface {
	Save(returnedOrder *domain.ReturnedOrder) error
	GetAll() ([]*domain.ReturnedOrder, error)
}
