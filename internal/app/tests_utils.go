package app

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"gitlab.ozon.dev/safariproxd/homework/internal/app/mock"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
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
		LastUpdateTime: someConstTime.Add(time.Duration(id) * time.Minute),
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
	svc := &PVZService{orderRepo: repo, nowFn: func() time.Time { return someConstTime }}
	return repo, svc
}
