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

// с возвращенными товарами работаем отдельно, потому что 1) удобнее будет вывести списко возвратов
// (чтобы не проходиться по всем товарам и проверять их статус)
// 2) возвращенные клиентами товары могут вернуть курьеру и тогда их надо удалять с хранилища
// а задание наверное предполгает хранить даже такие заказы

func (o *Order) GetStatusString() string {
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
		return "Given to courier"
	// по идее такого быть не может, но я добавил default на всякий случай
	default:
		return "Unknown Status"
	}
}

func (o *Order) IsBelongsToReciever(receiverID uint64) bool {
	return o.Status == StatusInStorage || o.Status == StatusGivenToClient
}
