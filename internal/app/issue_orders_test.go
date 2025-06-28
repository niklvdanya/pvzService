package app

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.ozon.dev/safariproxd/homework/internal/app/mock"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func TestPVZService_IssueOrdersToClient(t *testing.T) {
	t.Parallel()

	buildStoredOrder := func(id uint64, until time.Time) domain.Order {
		return domain.Order{OrderID: id, ReceiverID: someRecieverID, StorageUntil: until, Status: domain.StatusInStorage}
	}
	buildUpdatedOrder := func(o domain.Order) domain.Order {
		o.Status, o.LastUpdateTime = domain.StatusGivenToClient, someConstTime
		return o
	}
	buildHistory := func(id uint64) domain.OrderHistory {
		return domain.OrderHistory{OrderID: id, Status: domain.StatusGivenToClient, ChangedAt: someConstTime}
	}

	tests := []struct {
		name     string
		orderIDs []uint64
		prepare  func(*mock.OrderRepositoryMock)
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name:     "Success_MultipleOrders",
			orderIDs: []uint64{1, 2},
			prepare: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					switch id {
					case 1:
						return buildStoredOrder(1, someConstTime.Add(24*time.Hour)), nil
					case 2:
						return buildStoredOrder(2, someConstTime.Add(48*time.Hour)), nil
					default:
						return domain.Order{}, fmt.Errorf("unexpected id %d", id)
					}
				})
				r.UpdateMock.Set(func(_ context.Context, _ domain.Order) error { return nil })
				r.SaveHistoryMock.Set(func(_ context.Context, _ domain.OrderHistory) error { return nil })
			},
			wantErr: assert.NoError,
		},
		{
			name:     "Partial_MultipleOrders_SecondAlreadyGiven",
			orderIDs: []uint64{1, 2},
			prepare: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					if id == 1 {
						return buildStoredOrder(1, someConstTime.Add(24*time.Hour)), nil
					}
					return domain.Order{OrderID: 2, ReceiverID: someRecieverID, StorageUntil: someConstTime.Add(48 * time.Hour), Status: domain.StatusGivenToClient}, nil
				})
				okOrd := buildStoredOrder(1, someConstTime.Add(24*time.Hour))
				r.UpdateMock.Expect(contextBack, buildUpdatedOrder(okOrd)).Return(nil)
				r.SaveHistoryMock.Expect(contextBack, buildHistory(1)).Return(nil)
			},
			wantErr: assert.Error,
		},
		{
			name:     "Fail_OrderNotFound",
			orderIDs: []uint64{3},
			prepare: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Expect(contextBack, uint64(3)).Return(domain.Order{}, domain.EntityNotFoundError("order", "3"))
			},
			wantErr: assert.Error,
		},
		{
			name:     "Fail_BelongsToDifferentReceiver",
			orderIDs: []uint64{4},
			prepare: func(r *mock.OrderRepositoryMock) {
				bad := domain.Order{OrderID: 4, ReceiverID: 999, StorageUntil: someConstTime.Add(24 * time.Hour), Status: domain.StatusInStorage}
				r.GetByIDMock.Expect(contextBack, uint64(4)).Return(bad, nil)
			},
			wantErr: assert.Error,
		},
		{
			name:     "Fail_AlreadyGiven",
			orderIDs: []uint64{5},
			prepare: func(r *mock.OrderRepositoryMock) {
				given := domain.Order{OrderID: 5, ReceiverID: someRecieverID, StorageUntil: someConstTime.Add(24 * time.Hour), Status: domain.StatusGivenToClient}
				r.GetByIDMock.Expect(contextBack, uint64(5)).Return(given, nil)
			},
			wantErr: assert.Error,
		},
		{
			name:     "Fail_ReturnedOrderUnavailable",
			orderIDs: []uint64{6},
			prepare: func(r *mock.OrderRepositoryMock) {
				ret := domain.Order{OrderID: 6, ReceiverID: someRecieverID, StorageUntil: someConstTime.Add(24 * time.Hour), Status: domain.StatusReturnedFromClient}
				r.GetByIDMock.Expect(contextBack, uint64(6)).Return(ret, nil)
			},
			wantErr: assert.Error,
		},
		{
			name:     "Fail_StorageExpired",
			orderIDs: []uint64{7},
			prepare: func(r *mock.OrderRepositoryMock) {
				exp := buildStoredOrder(7, someConstTime.Add(-24*time.Hour))
				r.GetByIDMock.Expect(contextBack, uint64(7)).Return(exp, nil)
			},
			wantErr: assert.Error,
		},
		{
			name:     "Fail_UpdateError",
			orderIDs: []uint64{8},
			prepare: func(r *mock.OrderRepositoryMock) {
				ord := buildStoredOrder(8, someConstTime.Add(24*time.Hour))
				r.GetByIDMock.Expect(contextBack, uint64(8)).Return(ord, nil)
				r.UpdateMock.Expect(contextBack, buildUpdatedOrder(ord)).Return(fmt.Errorf("update error"))
			},
			wantErr: assert.Error,
		},
		{
			name:     "Fail_SaveHistoryError",
			orderIDs: []uint64{9},
			prepare: func(r *mock.OrderRepositoryMock) {
				ord := buildStoredOrder(9, someConstTime.Add(24*time.Hour))
				r.GetByIDMock.Expect(contextBack, uint64(9)).Return(ord, nil)
				r.UpdateMock.Expect(contextBack, buildUpdatedOrder(ord)).Return(nil)
				r.SaveHistoryMock.Expect(contextBack, buildHistory(9)).Return(fmt.Errorf("history error"))
			},
			wantErr: assert.Error,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			tc.prepare(repo)
			err := svc.IssueOrdersToClient(contextBack, someRecieverID, tc.orderIDs)
			tc.wantErr(t, err)
		})
	}
}
