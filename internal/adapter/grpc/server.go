package server

import (
	"gitlab.ozon.dev/safariproxd/homework/internal/app"
	"gitlab.ozon.dev/safariproxd/homework/pkg/api"
	"google.golang.org/grpc"
)

type OrdersServer struct {
	api.UnimplementedOrdersServiceServer
	service *app.PVZService
}

func NewOrdersServer(service *app.PVZService) *OrdersServer {
	return &OrdersServer{
		service: service,
	}
}

func (s *OrdersServer) Register(grpcServer *grpc.Server) {
	api.RegisterOrdersServiceServer(grpcServer, s)
}
