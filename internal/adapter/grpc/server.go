package server

import (
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/pkg/api"
	"google.golang.org/grpc"
)

type IOrderService interface {
	AcceptOrder(req domain.AcceptOrderRequest) (float64, error)
	ReturnOrderToDelivery(orderID uint64) error
	IssueOrdersToClient(receiverID uint64, orderIDs []uint64) error
	ReturnOrdersFromClient(receiverID uint64, orderIDs []uint64) error
	GetReceiverOrders(receiverID uint64, inPVZ bool, lastN, page, limit uint64) ([]*domain.Order, uint64, error)
	GetReceiverOrdersScroll(receiverID uint64, lastID, limit uint64) ([]*domain.Order, uint64, error)
	GetReturnedOrders(page, limit uint64) ([]*domain.Order, uint64, error)
	GetOrderHistory() ([]*domain.Order, error)
	ImportOrders(orders []domain.OrderToImport) (uint64, error)
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
