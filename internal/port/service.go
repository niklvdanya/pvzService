package port

import (
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

type OrderService interface {
	AcceptOrder(receiverID, orderID uint64, storageUntil time.Time) error
	ReturnOrderToDelivery(orderID uint64) error
	IssueOrdersToClient(receiverID uint64, orderIDs []uint64) error
	ReturnOrdersFromClient(receiverID uint64, orderIDs []uint64) error
	GetReceiverOrders(
		receiverID uint64,
		inPVZ bool,
		lastN, page, limit uint64,
	) ([]*domain.Order, uint64, error)
	GetReceiverOrdersScroll(receiverID uint64, lastID, limit uint64) ([]*domain.Order, uint64, error)
	GetReturnedOrders(page, limit uint64) ([]*domain.ReturnedOrder, uint64, error)
	GetOrderHistory() ([]*domain.Order, error)
	ImportOrders(orders []struct {
		OrderID      uint64 `json:"order_id"`
		ReceiverID   uint64 `json:"receiver_id"`
		StorageUntil string `json:"storage_until"`
	}) (uint64, error)
}
