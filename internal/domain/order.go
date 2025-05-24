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
