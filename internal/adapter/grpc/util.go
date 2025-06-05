package server

import (
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/pkg/api"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
