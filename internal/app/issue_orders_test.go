package app

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"gitlab.ozon.dev/safariproxd/homework/internal/app/mock"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

var (
	errUpdate = errors.New("update err")
	errHist   = errors.New("hist err")
)

func TestPVZService_IssueOrdersToClient(t *testing.T) {
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
						return OrderInStorage(1, 24*time.Hour), nil
					case 2:
						return OrderInStorage(2, 48*time.Hour), nil
					}
					return domain.Order{}, fmt.Errorf("unexpected id %d", id)
				})
				r.UpdateMock.Set(func(_ context.Context, _ domain.Order) error { return nil })
				r.SaveHistoryMock.Set(func(_ context.Context, _ domain.OrderHistory) error { return nil })
			},
			assertE: assert.NoError,
		},
		{
			name:     "Partial_SecondAlreadyGiven",
			orderIDs: []uint64{1, 2},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					if id == 1 {
						return OrderInStorage(1, 24*time.Hour), nil
					}
					return OrderGiven(2, 0), nil
				})
				ok := OrderInStorage(1, 24*time.Hour)
				r.UpdateMock.Expect(minimock.AnyContext, Updated(ok, domain.StatusGivenToClient, someConstTime)).Return(nil)
				r.SaveHistoryMock.Expect(minimock.AnyContext, History(1, domain.StatusGivenToClient, 0)).Return(nil)
			},
			assertE: errIs(domain.OrderAlreadyGivenError(2)),
		},
		{
			name:     "Fail_OrderNotFound",
			orderIDs: []uint64{3},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Expect(minimock.AnyContext, uint64(3)).Return(domain.Order{}, domain.EntityNotFoundError("Order", "3"))
			},
			assertE: errIs(domain.EntityNotFoundError("Order", "3")),
		},
		{
			name:     "Fail_BelongsToDifferentReceiver",
			orderIDs: []uint64{4},
			setup: func(r *mock.OrderRepositoryMock) {
				bad := OrderInStorage(4, 24*time.Hour)
				bad.ReceiverID = 999
				r.GetByIDMock.Expect(minimock.AnyContext, uint64(4)).Return(bad, nil)
			},
			assertE: errIs(domain.BelongsToDifferentReceiverError(4, someRecieverID, 999)),
		},
		{
			name:     "Fail_AlreadyGiven",
			orderIDs: []uint64{5},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Expect(minimock.AnyContext, uint64(5)).Return(OrderGiven(5, 0), nil)
			},
			assertE: errIs(domain.OrderAlreadyGivenError(5)),
		},
		{
			name:     "Fail_ReturnedOrderUnavailable",
			orderIDs: []uint64{6},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Expect(minimock.AnyContext, uint64(6)).Return(OrderReturned(6, 0), nil)
			},
			assertE: errIs(domain.UnavaliableReturnedOrderError(6)),
		},
		{
			name:     "Fail_UpdateError",
			orderIDs: []uint64{7},
			setup: func(r *mock.OrderRepositoryMock) {
				o := OrderInStorage(7, 24*time.Hour)
				r.GetByIDMock.Expect(minimock.AnyContext, uint64(7)).Return(o, nil)
				r.UpdateMock.Expect(minimock.AnyContext, Updated(o, domain.StatusGivenToClient, someConstTime)).Return(errUpdate)
			},
			assertE: errIs(errUpdate),
		},
		{
			name:     "Fail_SaveHistoryError",
			orderIDs: []uint64{8},
			setup: func(r *mock.OrderRepositoryMock) {
				o := OrderInStorage(8, 24*time.Hour)
				r.GetByIDMock.Expect(minimock.AnyContext, uint64(8)).Return(o, nil)
				r.UpdateMock.Expect(minimock.AnyContext, Updated(o, domain.StatusGivenToClient, someConstTime)).Return(nil)
				r.SaveHistoryMock.Expect(minimock.AnyContext, History(8, domain.StatusGivenToClient, 0)).Return(errHist)
			},
			assertE: errIs(errHist),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			tc.setup(repo)

			err := svc.IssueOrdersToClient(contextBack, someRecieverID, tc.orderIDs)
			tc.assertE(t, err)
		})
	}
}
