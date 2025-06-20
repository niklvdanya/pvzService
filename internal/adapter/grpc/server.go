package server

import (
	"context"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/pkg/api"
	"google.golang.org/grpc"
)

type IOrderService interface {
	AcceptOrder(ctx context.Context, req domain.AcceptOrderRequest) (float64, error)
	ReturnOrderToDelivery(ctx context.Context, orderID uint64) error
	IssueOrdersToClient(ctx context.Context, receiverID uint64, orderIDs []uint64) error
	ReturnOrdersFromClient(ctx context.Context, receiverID uint64, orderIDs []uint64) error
	GetReceiverOrders(ctx context.Context, receiverID uint64, inPVZ bool, lastN, page, limit uint64) ([]*domain.Order, uint64, error)
	GetReceiverOrdersScroll(ctx context.Context, receiverID uint64, lastID, limit uint64) ([]*domain.Order, uint64, error)
	GetReturnedOrders(ctx context.Context, page, limit uint64) ([]*domain.Order, uint64, error)
	GetOrderHistory(ctx context.Context) ([]*domain.Order, error)
	GetOrderHistoryByID(ctx context.Context, orderID uint64) ([]*domain.OrderHistory, error)
	ImportOrders(ctx context.Context, orders []domain.OrderToImport) (uint64, error)
}

type OrdersServer struct {
	api.UnimplementedOrdersServiceServer
	service IOrderService
}

func NewOrdersServer(service IOrderService) *OrdersServer {
	return &OrdersServer{
		service: service,
	}
}

func (s *OrdersServer) Register(grpcServer *grpc.Server) {
	api.RegisterOrdersServiceServer(grpcServer, s)
}
