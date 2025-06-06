package server

import (
	"github.com/ulule/limiter/v3"
	"gitlab.ozon.dev/safariproxd/homework/internal/app"
	"gitlab.ozon.dev/safariproxd/homework/pkg/api"
	"google.golang.org/grpc"
)

type OrdersServer struct {
	api.UnimplementedOrdersServiceServer
	service *app.PVZService
	limiter *limiter.Limiter
}

func NewOrdersServer(service *app.PVZService, limiter *limiter.Limiter) *OrdersServer {
	return &OrdersServer{
		service: service,
		limiter: limiter,
	}
}

func (s *OrdersServer) Register(grpcServer *grpc.Server) {
	api.RegisterOrdersServiceServer(grpcServer, s)
}
