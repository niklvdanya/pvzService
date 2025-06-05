package server

import (
	"context"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/app"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/pkg/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *OrdersServer) AcceptOrder(ctx context.Context, req *api.AcceptOrderRequest) (*api.OrderResponse, error) {
	if req.OrderId == 0 || req.UserId == 0 || req.ExpiresAt == nil {
		return nil, status.Error(codes.InvalidArgument, "order_id, user_id, and expires_at are required")
	}
	var packageType string
	if req.Package != nil {
		packageType = mapPackageTypeToString(*req.Package)
	}
	acceptReq := domain.AcceptOrderRequest{
		OrderID:      req.OrderId,
		ReceiverID:   req.UserId,
		StorageUntil: req.ExpiresAt.AsTime(),
		Weight:       float64(req.Weight),
		Price:        float64(req.Price),
		PackageType:  packageType,
	}
	_, err := s.service.AcceptOrder(acceptReq)
	if err != nil {
		return nil, mapErrorToGRPCStatus(err)
	}
	return &api.OrderResponse{
		Status:  api.OrderStatus_ORDER_STATUS_EXPECTS,
		OrderId: req.OrderId,
	}, nil
}

func (s *OrdersServer) ReturnOrder(ctx context.Context, req *api.OrderIdRequest) (*api.OrderResponse, error) {
	if req.OrderId == 0 {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}
	err := s.service.ReturnOrderToDelivery(req.OrderId)
	if err != nil {
		return nil, mapErrorToGRPCStatus(err)
	}
	return &api.OrderResponse{
		Status:  api.OrderStatus_ORDER_STATUS_DELETED,
		OrderId: req.OrderId,
	}, nil
}

func (s *OrdersServer) ProcessOrders(ctx context.Context, req *api.ProcessOrdersRequest) (*api.ProcessResult, error) {
	if req.UserId == 0 || len(req.OrderIds) == 0 || req.Action == api.ActionType_ACTION_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "user_id, order_ids, and action are required")
	}
	var err error
	if req.Action == api.ActionType_ACTION_TYPE_ISSUE {
		err = s.service.IssueOrdersToClient(req.UserId, req.OrderIds)
	} else {
		err = s.service.ReturnOrdersFromClient(req.UserId, req.OrderIds)
	}
	if err != nil {
		return processErrors(err, req.OrderIds)
	}
	return &api.ProcessResult{Processed: req.OrderIds}, nil
}

func (s *OrdersServer) ListOrders(ctx context.Context, req *api.ListOrdersRequest) (*api.OrdersList, error) {
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	var page, limit, lastN uint64
	if req.Pagination != nil {
		page = uint64(req.Pagination.Page)
		limit = uint64(req.Pagination.CountOnPage)
	}
	if req.LastN != nil {
		lastN = uint64(*req.LastN)
	}
	orders, total, err := s.service.GetReceiverOrders(req.UserId, req.InPvz, lastN, page, limit)
	if err != nil {
		return nil, mapErrorToGRPCStatus(err)
	}
	protoOrders := make([]*api.Order, len(orders))
	for i, order := range orders {
		protoOrders[i] = mapDomainOrderToProto(order)
	}
	return &api.OrdersList{
		Orders: protoOrders,
		Total:  int32(total),
	}, nil
}

func (s *OrdersServer) ListReturns(ctx context.Context, req *api.ListReturnsRequest) (*api.ReturnsList, error) {
	var page, limit uint64
	if req.Pagination != nil {
		page = uint64(req.Pagination.Page)
		limit = uint64(req.Pagination.CountOnPage)
	}
	orders, _, err := s.service.GetReturnedOrders(page, limit)
	if err != nil {
		return nil, mapErrorToGRPCStatus(err)
	}
	protoOrders := make([]*api.Order, len(orders))
	for i, order := range orders {
		protoOrders[i] = mapDomainOrderToProto(order)
	}
	return &api.ReturnsList{Returns: protoOrders}, nil
}

func (s *OrdersServer) GetHistory(ctx context.Context, req *api.GetHistoryRequest) (*api.OrderHistoryList, error) {
	var page, limit uint64
	if req.Pagination != nil {
		page = uint64(req.Pagination.Page)
		limit = uint64(req.Pagination.CountOnPage)
	}
	orders, err := s.service.GetOrderHistory()
	if err != nil {
		return nil, mapErrorToGRPCStatus(err)
	}
	paginated := app.Paginate(orders, page, limit)
	history := make([]*api.OrderHistory, len(paginated))
	for i, order := range paginated {
		history[i] = &api.OrderHistory{
			OrderId:   order.OrderID,
			Status:    mapDomainStatusToProto(order.Status),
			CreatedAt: timestamppb.New(order.LastUpdateTime),
		}
	}
	return &api.OrderHistoryList{History: history}, nil
}

func (s *OrdersServer) ImportOrders(ctx context.Context, req *api.ImportOrdersRequest) (*api.ImportResult, error) {
	orders := make([]domain.OrderToImport, len(req.Orders))
	for i, order := range req.Orders {
		orders[i] = domain.OrderToImport{
			OrderID:      order.OrderId,
			ReceiverID:   order.UserId,
			StorageUntil: order.ExpiresAt.AsTime().Format(cli.TimeFormat),
			PackageType:  mapPackageTypeToString(*order.Package),
			Weight:       float64(order.Weight),
			Price:        float64(order.Price),
		}
	}
	imported, err := s.service.ImportOrders(orders)
	if err != nil {
		return processImportErrors(err, orders), nil
	}
	return &api.ImportResult{Imported: int32(imported)}, nil
}
