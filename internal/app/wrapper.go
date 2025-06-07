package app

import (
	"context"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (s *PVZService) AcceptOrder(ctx context.Context, req domain.AcceptOrderRequest) (float64, error) {
	return WithTimeoutAndContextCheck(ctx, s.serviceTimeout, func(ctx context.Context) (float64, error) {
		return s.acceptOrder(req)
	})
}

func (s *PVZService) ReturnOrderToDelivery(ctx context.Context, orderID uint64) error {
	_, err := WithTimeoutAndContextCheck(ctx, s.serviceTimeout, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, s.returnOrderToDelivery(orderID)
	})
	return err
}

func (s *PVZService) IssueOrdersToClient(ctx context.Context, receiverID uint64, orderIDs []uint64) error {
	_, err := WithTimeoutAndContextCheck(ctx, s.serviceTimeout, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, s.issueOrdersToClient(receiverID, orderIDs)
	})
	return err
}

func (s *PVZService) ReturnOrdersFromClient(ctx context.Context, receiverID uint64, orderIDs []uint64) error {
	_, err := WithTimeoutAndContextCheck(ctx, s.serviceTimeout, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, s.returnOrdersFromClient(receiverID, orderIDs)
	})
	return err
}

type OrdersWithTotal struct {
	Orders []*domain.Order
	Total  uint64
}

func (s *PVZService) GetReceiverOrders(ctx context.Context, receiverID uint64, inPVZ bool, lastN, page, limit uint64) ([]*domain.Order, uint64, error) {
	result, err := WithTimeoutAndContextCheck(ctx, s.serviceTimeout, func(ctx context.Context) (OrdersWithTotal, error) {
		orders, total, err := s.getReceiverOrders(receiverID, inPVZ, lastN, page, limit)
		return OrdersWithTotal{Orders: orders, Total: total}, err
	})
	if err != nil {
		return nil, 0, err
	}
	return result.Orders, result.Total, nil
}

func (s *PVZService) GetReceiverOrdersScroll(ctx context.Context, receiverID uint64, lastID, limit uint64) ([]*domain.Order, uint64, error) {
	result, err := WithTimeoutAndContextCheck(ctx, s.serviceTimeout, func(ctx context.Context) (OrdersWithTotal, error) {
		orders, total, err := s.getReceiverOrdersScroll(receiverID, lastID, limit)
		return OrdersWithTotal{Orders: orders, Total: total}, err
	})
	if err != nil {
		return nil, 0, err
	}
	return result.Orders, result.Total, nil
}

func (s *PVZService) GetReturnedOrders(ctx context.Context, page, limit uint64) ([]*domain.Order, uint64, error) {
	result, err := WithTimeoutAndContextCheck(ctx, s.serviceTimeout, func(ctx context.Context) (OrdersWithTotal, error) {
		orders, total, err := s.getReturnedOrders(page, limit)
		return OrdersWithTotal{Orders: orders, Total: total}, err
	})
	if err != nil {
		return nil, 0, err
	}
	return result.Orders, result.Total, nil
}

func (s *PVZService) GetOrderHistory(ctx context.Context) ([]*domain.Order, error) {
	return WithTimeoutAndContextCheck(ctx, s.serviceTimeout, func(ctx context.Context) ([]*domain.Order, error) {
		return s.getOrderHistory()
	})
}

func (s *PVZService) ImportOrders(ctx context.Context, orders []domain.OrderToImport) (uint64, error) {
	return WithTimeoutAndContextCheck(ctx, s.serviceTimeout, func(ctx context.Context) (uint64, error) {
		return s.importOrders(orders)
	})
}
