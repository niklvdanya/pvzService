package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/app"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/pkg/api"
	"go.uber.org/multierr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrdersServer struct {
	api.UnimplementedOrdersServiceServer
	service *app.PVZService
}

func NewOrdersServer(service *app.PVZService) *OrdersServer {
	return &OrdersServer{service: service}
}

func (s *OrdersServer) Register(grpcServer *grpc.Server) {
	api.RegisterOrdersServiceServer(grpcServer, s)
}

func (s *OrdersServer) AcceptOrder(ctx context.Context, req *api.AcceptOrderRequest) (*api.OrderResponse, error) {
	if req.OrderId == 0 || req.UserId == 0 || req.ExpiresAt == nil {
		return nil, status.Error(codes.InvalidArgument, "order_id, user_id, and expires_at are required")
	}
	packageType := mapPackageTypeToString(*req.Package)
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
	var page, limit uint64
	if req.Pagination != nil {
		page = uint64(req.Pagination.Page)
		limit = uint64(req.Pagination.CountOnPage)
	}
	orders, total, err := s.service.GetReceiverOrders(req.UserId, req.InPvz, uint64(*req.LastN), page, limit)
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

// Helper functions
func mapPackageTypeToString(pt api.PackageType) string {
	switch pt {
	case api.PackageType_PACKAGE_TYPE_BAG:
		return "bag"
	case api.PackageType_PACKAGE_TYPE_BOX:
		return "box"
	case api.PackageType_PACKAGE_TYPE_TAPE:
		return "film"
	case api.PackageType_PACKAGE_TYPE_BAG_TAPE:
		return "bag+film"
	case api.PackageType_PACKAGE_TYPE_BOX_TAPE:
		return "box+film"
	default:
		return ""
	}
}

func mapDomainStatusToProto(status domain.OrderStatus) api.OrderStatus {
	switch status {
	case domain.StatusInStorage:
		return api.OrderStatus_ORDER_STATUS_EXPECTS
	case domain.StatusGivenToClient:
		return api.OrderStatus_ORDER_STATUS_ACCEPTED
	case domain.StatusReturnedFromClient:
		return api.OrderStatus_ORDER_STATUS_RETURNED
	case domain.StatusReturnedWithoutClient, domain.StatusGivenToCourier:
		return api.OrderStatus_ORDER_STATUS_DELETED
	default:
		return api.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func mapDomainOrderToProto(order *domain.Order) *api.Order {
	pkgType := mapStringToPackageType(order.PackageType)
	return &api.Order{
		OrderId:    order.OrderID,
		UserId:     order.ReceiverID,
		Status:     mapDomainStatusToProto(order.Status),
		ExpiresAt:  timestamppb.New(order.StorageUntil),
		Weight:     float32(order.Weight),
		TotalPrice: float32(order.Price),
		Package:    &pkgType,
	}
}

func mapStringToPackageType(pt string) api.PackageType {
	switch pt {
	case "bag":
		return api.PackageType_PACKAGE_TYPE_BAG
	case "box":
		return api.PackageType_PACKAGE_TYPE_BOX
	case "film":
		return api.PackageType_PACKAGE_TYPE_TAPE
	case "bag+film":
		return api.PackageType_PACKAGE_TYPE_BAG_TAPE
	case "box+film":
		return api.PackageType_PACKAGE_TYPE_BOX_TAPE
	default:
		return api.PackageType_PACKAGE_TYPE_UNSPECIFIED
	}
}

func mapErrorToGRPCStatus(err error) error {
	var domainErr domain.Error
	if errors.As(err, &domainErr) {
		switch domainErr.Code {
		case domain.ErrorCodeNotFound:
			return status.Error(codes.NotFound, domainErr.Message)
		case domain.ErrorCodeAlreadyExists:
			return status.Error(codes.AlreadyExists, domainErr.Message)
		case domain.ErrorCodeStorageExpired, domain.ErrorCodeStorageNotExpired:
			return status.Error(codes.FailedPrecondition, domainErr.Message)
		case domain.ErrorCodeValidationFailed, domain.ErrorCodeInvalidPackage, domain.ErrorCodeWeightTooHeavy:
			return status.Error(codes.InvalidArgument, domainErr.Message)
		default:
			return status.Error(codes.Internal, domainErr.Message)
		}
	}
	if multierr.Errors(err) != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return status.Error(codes.Internal, err.Error())
}

func processErrors(err error, orderIDs []uint64) (*api.ProcessResult, error) {
	var processed, errors []uint64
	multiErrs := multierr.Errors(err)
	if len(multiErrs) == 0 {
		return &api.ProcessResult{Processed: orderIDs}, nil
	}
	for _, orderID := range orderIDs {
		found := false
		for _, e := range multiErrs {
			if strings.Contains(e.Error(), fmt.Sprintf("Order %d", orderID)) {
				errors = append(errors, orderID)
				found = true
				break
			}
		}
		if !found {
			processed = append(processed, orderID)
		}
	}
	return &api.ProcessResult{Processed: processed, Errors: errors}, status.Error(codes.InvalidArgument, err.Error())
}

func processImportErrors(err error, orders []domain.OrderToImport) *api.ImportResult {
	var errors []uint64
	multiErrs := multierr.Errors(err)
	for _, order := range orders {
		for _, e := range multiErrs {
			if strings.Contains(e.Error(), fmt.Sprintf("Order %d", order.OrderID)) {
				errors = append(errors, order.OrderID)
				break
			}
		}
	}
	return &api.ImportResult{Imported: int32(len(orders) - len(errors)), Errors: errors}
}
