package app

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gitlab.ozon.dev/safariproxd/homework/internal/app/mock"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

var (
	errUpdRet  = errors.New("update err")
	errHistRet = errors.New("hist err")
)

func TestPVZService_ReturnOrdersFromClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		orderIDs []uint64
		setup    func(*mock.OrderRepositoryMock)
		assertE  assert.ErrorAssertionFunc
	}{
		{
			name:     "Success_MultipleOrders",
			orderIDs: []uint64{1, 2},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					switch id {
					case 1:
						return OrderGiven(1, -10*time.Hour), nil
					case 2:
						return OrderGiven(2, -8*time.Hour), nil
					}
					return domain.Order{}, fmt.Errorf("unexpected id %d", id)
				})
				r.UpdateMock.Set(func(_ context.Context, _ domain.Order) error { return nil })
				r.SaveHistoryMock.Set(func(_ context.Context, _ domain.OrderHistory) error { return nil })
			},
			assertE: assert.NoError,
		},
		{
			name:     "Partial_SecondAlreadyInStorage",
			orderIDs: []uint64{1, 2},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					if id == 1 {
						return OrderGiven(1, -5*time.Hour), nil
					}
					return OrderInStorage(2, 0), nil
				})
				ok := OrderGiven(1, -5*time.Hour)
				r.UpdateMock.Expect(contextBack, Updated(ok, domain.StatusReturnedFromClient, someConstTime)).Return(nil)
				r.SaveHistoryMock.Expect(contextBack, History(1, domain.StatusReturnedFromClient, 0)).Return(nil)
			},
			assertE: errIs(domain.AlreadyInStorageError(2)),
		},
		{
			name:     "Fail_OrderNotFound",
			orderIDs: []uint64{3},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Expect(contextBack, uint64(3)).Return(domain.Order{}, domain.EntityNotFoundError("Order", "3"))
			},
			assertE: errIs(domain.EntityNotFoundError("Order", "3")),
		},
		{
			name:     "Fail_BelongsToDifferentReceiver",
			orderIDs: []uint64{4},
			setup: func(r *mock.OrderRepositoryMock) {
				bad := OrderGiven(4, -4*time.Hour)
				bad.ReceiverID = 999
				r.GetByIDMock.Expect(contextBack, uint64(4)).Return(bad, nil)
			},
			assertE: errIs(domain.BelongsToDifferentReceiverError(4, someRecieverID, 999)),
		},
		{
			name:     "Fail_AlreadyInStorage",
			orderIDs: []uint64{5},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Expect(contextBack, uint64(5)).Return(OrderInStorage(5, 0), nil)
			},
			assertE: errIs(domain.AlreadyInStorageError(5)),
		},
		{
			name:     "Fail_UpdateError",
			orderIDs: []uint64{6},
			setup: func(r *mock.OrderRepositoryMock) {
				o := OrderGiven(7, -2*time.Hour)
				r.GetByIDMock.Expect(contextBack, uint64(7)).Return(o, nil)
				r.UpdateMock.Expect(contextBack, Updated(o, domain.StatusReturnedFromClient, someConstTime)).Return(errUpdRet)
			},
			assertE: errIs(errUpdRet),
		},
		{
			name:     "Fail_SaveHistoryError",
			orderIDs: []uint64{7},
			setup: func(r *mock.OrderRepositoryMock) {
				o := OrderGiven(8, -3*time.Hour)
				r.GetByIDMock.Expect(contextBack, uint64(8)).Return(o, nil)
				r.UpdateMock.Expect(contextBack, Updated(o, domain.StatusReturnedFromClient, someConstTime)).Return(nil)
				r.SaveHistoryMock.Expect(contextBack, History(8, domain.StatusReturnedFromClient, 0)).Return(errHistRet)
			},
			assertE: errIs(errHistRet),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			tc.setup(repo)

			err := svc.ReturnOrdersFromClient(contextBack, someRecieverID, tc.orderIDs)
			tc.assertE(t, err)
		})
	}
}
