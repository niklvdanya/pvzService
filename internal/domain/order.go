package domain

import (
	"time"
)

type OrderStatus uint8

const (
	StatusInStorage OrderStatus = iota
	StatusGivenToClient
	StatusReturnedFromClient
	StatusReturnedWithoutClient
	StatusGivenToCourier
)

type Order struct {
	OrderID        uint64
	ReceiverID     uint64
	StorageUntil   time.Time
	Status         OrderStatus
	AcceptTime     time.Time
	LastUpdateTime time.Time
	PackageType    string
	Weight         float64
	Price          float64
}

type OrderHistory struct {
	OrderID   uint64
	Status    OrderStatus
	ChangedAt time.Time
}

type OrderToImport struct {
	OrderID      uint64  `json:"order_id"`
	ReceiverID   uint64  `json:"receiver_id"`
	StorageUntil string  `json:"storage_until"`
	PackageType  string  `json:"package_type"`
	Weight       float64 `json:"weight"`
	Price        float64 `json:"price"`
}

var OrdersToImport []OrderToImport

type AcceptOrderRequest struct {
	ReceiverID   uint64
	OrderID      uint64
	StorageUntil time.Time
	Weight       float64
	Price        float64
	PackageType  string
}

func (o Order) GetStatusString() string {
	switch o.Status {
	case StatusInStorage:
		return "In Storage"
	case StatusGivenToClient:
		return "Given to client"
	case StatusReturnedFromClient:
		return "Returned from client"
	case StatusGivenToCourier:
		return "Given to courier"
	case StatusReturnedWithoutClient:
		return "Given to courier without client"
	default:
		return "Unknown Status"
	}
}

func (o Order) IsBelongsToReciever(receiverID uint64) bool {
	return o.Status == StatusInStorage || o.Status == StatusGivenToClient
}
