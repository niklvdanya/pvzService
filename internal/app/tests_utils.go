package app

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/app/mock"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/internal/metrics"
)

var (
	someConstTime  = time.Date(2025, time.June, 28, 3, 26, 0, 0, time.UTC)
	contextBack    = context.Background()
	someRecieverID = uint64(100)
)

func Stored(id uint64, status domain.OrderStatus) domain.Order {
	return domain.Order{
		OrderID:        id,
		ReceiverID:     someRecieverID,
		StorageUntil:   someConstTime.Add(24 * time.Hour),
		Status:         status,
		LastUpdateTime: someConstTime.Add(time.Duration(int64(id)) * time.Minute),
	}
}

func IdsOf(ord []domain.Order) (ids []uint64) {
	for _, o := range ord {
		ids = append(ids, o.OrderID)
	}
	return
}

func NewEnv(t *testing.T) (*mock.OrderRepositoryMock, *PVZService) {
	ctrl := minimock.NewController(t)
	repo := mock.NewOrderRepositoryMock(ctrl)
	const testWorkerLimit = 8
	noOpMetrics := metrics.NewNoOpProvider()
	svc := NewPVZService(repo, nil, nil, func() time.Time { return someConstTime }, testWorkerLimit, noOpMetrics)
	return repo, svc
}

func BuildOrder(id uint64, status domain.OrderStatus,
	storageOff, updateOff time.Duration) domain.Order {

	o := Stored(id, status)
	o.StorageUntil = someConstTime.Add(storageOff)
	o.LastUpdateTime = someConstTime.Add(updateOff)
	return o
}

func OrderInStorage(id uint64, storageOff time.Duration) domain.Order {
	return BuildOrder(id, domain.StatusInStorage, storageOff, 0)
}

func OrderGiven(id uint64, lastUpdOff time.Duration) domain.Order {
	return BuildOrder(id, domain.StatusGivenToClient, 24*time.Hour, lastUpdOff)
}

func OrderReturned(id uint64, lastUpdOff time.Duration) domain.Order {
	return BuildOrder(id, domain.StatusReturnedFromClient, 24*time.Hour, lastUpdOff)
}

func Updated(o domain.Order, newStatus domain.OrderStatus, t time.Time) domain.Order {
	o.Status, o.LastUpdateTime = newStatus, t
	return o
}
func DateString(off time.Duration) string {
	return cli.MapTimeToString(someConstTime.Add(off))
}

func History(orderID uint64, status domain.OrderStatus,
	changedOff time.Duration) domain.OrderHistory {

	return domain.OrderHistory{
		OrderID:   orderID,
		Status:    status,
		ChangedAt: someConstTime.Add(changedOff),
	}
}

func TimesOf(h []domain.OrderHistory) (ts []time.Time) {
	for _, rec := range h {
		ts = append(ts, rec.ChangedAt)
	}
	return
}

func DTO(id uint64, pkg string, off time.Duration) domain.OrderToImport {
	return domain.OrderToImport{
		OrderID:      id,
		ReceiverID:   someRecieverID,
		StorageUntil: DateString(off),
		PackageType:  pkg,
		Weight:       5,
		Price:        100,
	}
}

func errIs(target error) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, _ ...interface{}) bool {
		return assert.ErrorIs(t, err, target)
	}
}
