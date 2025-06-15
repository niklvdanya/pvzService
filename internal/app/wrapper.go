package app

import (
	"context"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (s *PVZService) AcceptOrder(ctx context.Context, req domain.AcceptOrderRequest) (float64, error) {
	return s.acceptOrder(req)
}

func (s *PVZService) ReturnOrderToDelivery(ctx context.Context, orderID uint64) error {
	return s.returnOrderToDelivery(orderID)
}

func (s *PVZService) IssueOrdersToClient(ctx context.Context, receiverID uint64, orderIDs []uint64) error {
	return s.issueOrdersToClient(receiverID, orderIDs)
}

func (s *PVZService) ReturnOrdersFromClient(ctx context.Context, receiverID uint64, orderIDs []uint64) error {
	return s.returnOrdersFromClient(receiverID, orderIDs)
}

func (s *PVZService) GetReceiverOrders(
	ctx context.Context,
	receiverID uint64,
	inPVZ bool,
	lastN, page, limit uint64,
) ([]*domain.Order, uint64, error) {
	return s.getReceiverOrders(receiverID, inPVZ, lastN, page, limit)
}

func (s *PVZService) GetReceiverOrdersScroll(
	ctx context.Context,
	receiverID uint64,
	lastID, limit uint64,
) ([]*domain.Order, uint64, error) {
	return s.getReceiverOrdersScroll(receiverID, lastID, limit)
}

func (s *PVZService) GetReturnedOrders(
	ctx context.Context,
	page, limit uint64,
) ([]*domain.Order, uint64, error) {
	return s.getReturnedOrders(page, limit)
}

func (s *PVZService) GetOrderHistory(ctx context.Context) ([]*domain.Order, error) {
	return s.getOrderHistory()
}

func (s *PVZService) ImportOrders(ctx context.Context, orders []domain.OrderToImport) (uint64, error) {
	return s.importOrders(orders)
}
