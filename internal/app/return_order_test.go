package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.ozon.dev/safariproxd/homework/internal/app/mock"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func TestPVZService_ReturnOrderToDelivery(t *testing.T) {
	t.Parallel()

	expired := someConstTime.Add(-1 * time.Hour)
	notExpired := someConstTime.Add(1 * time.Hour)

	makeOrder := func(id uint64, status domain.OrderStatus, until time.Time) domain.Order {
		o := Stored(id, status)
		o.StorageUntil = until
		return o
	}

	tests := []struct {
		name    string
		orderID uint64
		order   domain.Order
		prepare func(r *mock.OrderRepositoryMock, o domain.Order)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "Success_FromStorage",
			orderID: 1,
			order:   makeOrder(1, domain.StatusInStorage, expired),
			prepare: func(r *mock.OrderRepositoryMock, o domain.Order) {
				updated := o
				updated.Status = domain.StatusReturnedWithoutClient
				updated.LastUpdateTime = someConstTime
				r.GetByIDMock.Expect(contextBack, o.OrderID).Return(o, nil)
				r.UpdateMock.Expect(contextBack, updated).Return(nil)
				r.SaveHistoryMock.Expect(contextBack, domain.OrderHistory{OrderID: o.OrderID, Status: updated.Status, ChangedAt: someConstTime}).Return(nil)
			},
			wantErr: assert.NoError,
		},
		{
			name:    "Success_FromReturned",
			orderID: 2,
			order:   makeOrder(2, domain.StatusReturnedFromClient, expired),
			prepare: func(r *mock.OrderRepositoryMock, o domain.Order) {
				updated := o
				updated.Status = domain.StatusGivenToCourier
				updated.LastUpdateTime = someConstTime
				r.GetByIDMock.Expect(contextBack, o.OrderID).Return(o, nil)
				r.UpdateMock.Expect(contextBack, updated).Return(nil)
				r.SaveHistoryMock.Expect(contextBack, domain.OrderHistory{OrderID: o.OrderID, Status: updated.Status, ChangedAt: someConstTime}).Return(nil)
			},
			wantErr: assert.NoError,
		},
		{
			name:    "Fail_RepoError",
			orderID: 3,
			prepare: func(r *mock.OrderRepositoryMock, _ domain.Order) {
				r.GetByIDMock.Expect(contextBack, uint64(3)).Return(domain.Order{}, fmt.Errorf("db err"))
			},
			wantErr: assert.Error,
		},
		{
			name:    "Fail_WrongStatus",
			orderID: 4,
			order:   makeOrder(4, domain.StatusGivenToClient, expired),
			prepare: func(r *mock.OrderRepositoryMock, o domain.Order) {
				r.GetByIDMock.Expect(contextBack, o.OrderID).Return(o, nil)
			},
			wantErr: assert.Error,
		},
		{
			name:    "Fail_StorageNotExpired",
			orderID: 5,
			order:   makeOrder(5, domain.StatusInStorage, notExpired),
			prepare: func(r *mock.OrderRepositoryMock, o domain.Order) {
				r.GetByIDMock.Expect(contextBack, o.OrderID).Return(o, nil)
			},
			wantErr: assert.Error,
		},
		{
			name:    "Fail_UpdateError",
			orderID: 6,
			order:   makeOrder(6, domain.StatusInStorage, expired),
			prepare: func(r *mock.OrderRepositoryMock, o domain.Order) {
				updated := o
				updated.Status = domain.StatusReturnedWithoutClient
				updated.LastUpdateTime = someConstTime
				r.GetByIDMock.Expect(contextBack, o.OrderID).Return(o, nil)
				r.UpdateMock.Expect(contextBack, updated).Return(fmt.Errorf("upd err"))
			},
			wantErr: assert.Error,
		},
		{
			name:    "Fail_SaveHistoryError",
			orderID: 7,
			order:   makeOrder(7, domain.StatusInStorage, expired),
			prepare: func(r *mock.OrderRepositoryMock, o domain.Order) {
				updated := o
				updated.Status = domain.StatusReturnedWithoutClient
				updated.LastUpdateTime = someConstTime
				r.GetByIDMock.Expect(contextBack, o.OrderID).Return(o, nil)
				r.UpdateMock.Expect(contextBack, updated).Return(nil)
				r.SaveHistoryMock.Expect(contextBack, domain.OrderHistory{OrderID: o.OrderID, Status: updated.Status, ChangedAt: someConstTime}).Return(fmt.Errorf("hist err"))
			},
			wantErr: assert.Error,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			if tc.prepare != nil {
				tc.prepare(repo, tc.order)
			}
			err := svc.ReturnOrderToDelivery(contextBack, tc.orderID)
			tc.wantErr(t, err)
		})
	}
}
