package server

import (
	"context"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/app"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/pkg/api"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *OrdersServer) AcceptOrder(ctx context.Context, req *api.AcceptOrderRequest) (*api.OrderResponse, error) {
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
	_, err := s.service.AcceptOrder(ctx, acceptReq)
	if err != nil {
		return nil, err
	}
	return &api.OrderResponse{
		Status:  api.OrderStatus_ORDER_STATUS_EXPECTS,
		OrderId: req.OrderId,
	}, nil
}

func (s *OrdersServer) ReturnOrder(ctx context.Context, req *api.OrderIdRequest) (*api.OrderResponse, error) {
	err := s.service.ReturnOrderToDelivery(ctx, req.OrderId)
	if err != nil {
		return nil, err
	}
	return &api.OrderResponse{
		Status:  api.OrderStatus_ORDER_STATUS_DELETED,
		OrderId: req.OrderId,
	}, nil
}

func (s *OrdersServer) ProcessOrders(ctx context.Context, req *api.ProcessOrdersRequest) (*api.ProcessResult, error) {
	var err error
	if req.Action == api.ActionType_ACTION_TYPE_ISSUE {
		err = s.service.IssueOrdersToClient(ctx, req.UserId, req.OrderIds)
	} else {
		err = s.service.ReturnOrdersFromClient(ctx, req.UserId, req.OrderIds)
	}
	if err != nil {
		return processErrors(err, req.OrderIds)
	}
	return &api.ProcessResult{Processed: req.OrderIds}, nil
}

func (s *OrdersServer) ListOrders(ctx context.Context, req *api.ListOrdersRequest) (*api.OrdersList, error) {
	var page, limit, lastN uint64
	if req.Pagination != nil {
		page = uint64(req.Pagination.Page)
		limit = uint64(req.Pagination.CountOnPage)
	}
	if req.LastN != nil {
		lastN = uint64(*req.LastN)
	}
	ordersReq := domain.ReceiverOrdersRequest{
		ReceiverID: req.UserId,
		InPVZ:      req.InPvz,
		LastN:      lastN,
		Page:       page,
		Limit:      limit,
	}
	orders, total, err := s.service.GetReceiverOrders(ctx, ordersReq)
	if err != nil {
		return nil, err
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
	orders, _, err := s.service.GetReturnedOrders(ctx, page, limit)
	if err != nil {
		return nil, err
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
	orders, err := s.service.GetOrderHistory(ctx)
	if err != nil {
		return nil, err
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

func (s *OrdersServer) GetOrderHistory(ctx context.Context, req *api.OrderHistoryRequest) (*api.OrderHistoryResponse, error) {
	history, err := s.service.GetOrderHistoryByID(ctx, req.OrderId)
	if err != nil {
		return nil, err
	}
	protoHistory := make([]*api.OrderHistory, len(history))
	for i, record := range history {
		protoHistory[i] = &api.OrderHistory{
			OrderId:   record.OrderID,
			Status:    mapDomainStatusToProto(record.Status),
			CreatedAt: timestamppb.New(record.ChangedAt),
		}
	}
	return &api.OrderHistoryResponse{History: protoHistory}, nil
}

func (s *OrdersServer) ImportOrders(ctx context.Context, req *api.ImportOrdersRequest) (*api.ImportResult, error) {
	orders := make([]domain.OrderToImport, len(req.Orders))
	for i, order := range req.Orders {
		var packageType string
		if order.Package != nil {
			packageType = mapPackageTypeToString(*order.Package)
		}
		orders[i] = domain.OrderToImport{
			OrderID:      order.OrderId,
			ReceiverID:   order.UserId,
			StorageUntil: order.ExpiresAt.AsTime().Format(cli.TimeFormat),
			PackageType:  packageType,
			Weight:       float64(order.Weight),
			Price:        float64(order.Price),
		}
	}
	imported, err := s.service.ImportOrders(ctx, orders)
	if err != nil {
		return processImportErrors(err, orders), nil
	}
	return &api.ImportResult{Imported: int32(imported)}, nil
}
