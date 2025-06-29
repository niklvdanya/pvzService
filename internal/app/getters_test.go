package app

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func Test_GetReceiverOrders(t *testing.T) {
	t.Parallel()

	orders := []domain.Order{Stored(1, domain.StatusInStorage), Stored(2, domain.StatusGivenToClient), Stored(3, domain.StatusInStorage)}

	tests := []struct {
		name    string
		req     domain.ReceiverOrdersRequest
		wantIDs []uint64
		repoErr bool
		wantErr bool
	}{
		{"Filter_IN_PVZ", domain.ReceiverOrdersRequest{ReceiverID: someRecieverID, InPVZ: true, Page: 1, Limit: 5}, []uint64{1, 3}, false, false},
		{"LastN", domain.ReceiverOrdersRequest{ReceiverID: someRecieverID, LastN: 2}, []uint64{2, 3}, false, false},
		{"RepoErr", domain.ReceiverOrdersRequest{ReceiverID: someRecieverID}, nil, true, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, svc := NewEnv(t)
			if tt.repoErr {
				repo.GetByReceiverIDMock.Expect(contextBack, someRecieverID).Return(nil, fmt.Errorf("db"))
			} else {
				repo.GetByReceiverIDMock.Expect(contextBack, someRecieverID).Return(orders, nil)
			}
			got, _, err := svc.GetReceiverOrders(contextBack, tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantIDs, IdsOf(got))
		})
	}
}

func Test_GetReturnedOrders(t *testing.T) {
	t.Parallel()

	returns := []domain.Order{Stored(5, domain.StatusReturnedFromClient), Stored(6, domain.StatusReturnedFromClient)}
	repo, svc := NewEnv(t)
	repo.GetReturnedOrdersMock.Expect(contextBack).Return(returns, nil)
	got, total, err := svc.GetReturnedOrders(contextBack, 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), total)
	assert.Equal(t, []uint64{5}, IdsOf(got))
}

func Test_GetOrderHistory_Sorted(t *testing.T) {
	t.Parallel()

	unsorted := []domain.Order{Stored(10, 0), Stored(12, 0), Stored(11, 0)}
	repo, svc := NewEnv(t)
	repo.GetAllOrdersMock.Expect(contextBack).Return(unsorted, nil)
	got, err := svc.GetOrderHistory(contextBack)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{12, 11, 10}, IdsOf(got))
}

func Test_GetOrderHistoryByID(t *testing.T) {
	t.Parallel()

	h := []domain.OrderHistory{{OrderID: 1, Status: 2, ChangedAt: someConstTime.Add(time.Hour)}, {OrderID: 1, Status: 1, ChangedAt: someConstTime}}
	repo, svc := NewEnv(t)
	repo.GetHistoryByOrderIDMock.Set(func(_ context.Context, id uint64) ([]domain.OrderHistory, error) {
		if id == 1 {
			return h, nil
		}
		return nil, nil
	})
	got, err := svc.GetOrderHistoryByID(contextBack, 1)
	assert.NoError(t, err)
	assert.Equal(t, []domain.OrderHistory{h[0], h[1]}, got)
	_, err = svc.GetOrderHistoryByID(contextBack, 2)
	assert.Error(t, err)
}

func Test_GetReceiverOrdersScroll(t *testing.T) {
	t.Parallel()

	orders := []domain.Order{Stored(1, 0), Stored(2, 0), Stored(3, 0)}
	repo, svc := NewEnv(t)
	repo.GetByReceiverIDMock.Set(func(_ context.Context, _ uint64) ([]domain.Order, error) { return orders, nil })
	page1, next, _ := svc.GetReceiverOrdersScroll(contextBack, someRecieverID, 0, 2)
	assert.Equal(t, []uint64{1, 2}, IdsOf(page1))
	page2, next2, _ := svc.GetReceiverOrdersScroll(contextBack, someRecieverID, next, 2)
	assert.Equal(t, []uint64{3}, IdsOf(page2))
	assert.Equal(t, uint64(0), next2)
}

func Test_ReturnOrderToDelivery(t *testing.T) {
	t.Parallel()

	expired := someConstTime.Add(-time.Hour)

	makeOrd := func(id uint64, st domain.OrderStatus, until time.Time) domain.Order {
		o := Stored(id, st)
		o.StorageUntil = until
		return o
	}

	tests := []struct {
		name   string
		order  domain.Order
		wantOK bool
	}{
		{"FromStorage", makeOrd(1, domain.StatusInStorage, expired), true},
		{"FromReturned", makeOrd(2, domain.StatusReturnedFromClient, expired), true},
		{"WrongStatus", makeOrd(3, domain.StatusGivenToClient, expired), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, svc := NewEnv(t)
			repo.GetByIDMock.Expect(contextBack, tt.order.OrderID).Return(tt.order, nil)
			if tt.wantOK {
				upd := tt.order
				upd.Status = domain.StatusReturnedWithoutClient
				if tt.order.Status == domain.StatusReturnedFromClient {
					upd.Status = domain.StatusGivenToCourier
				}
				upd.LastUpdateTime = someConstTime
				repo.UpdateMock.Expect(contextBack, upd).Return(nil)
				repo.SaveHistoryMock.Expect(contextBack, domain.OrderHistory{OrderID: upd.OrderID, Status: upd.Status, ChangedAt: someConstTime}).Return(nil)
			}
			err := svc.ReturnOrderToDelivery(contextBack, tt.order.OrderID)
			if tt.wantOK {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
