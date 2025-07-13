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
	errUpdate = errors.New("update err")
	errHist   = errors.New("hist err")
)

func TestPVZService_IssueOrdersToClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		orderIDs []uint64
		setup    func(*mock.OrderRepositoryMock, context.Context)
		assertE  assert.ErrorAssertionFunc
	}{
		{
			name:     "Success_MultipleOrders",
			orderIDs: []uint64{1, 2},
			setup: func(r *mock.OrderRepositoryMock, ctx context.Context) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					switch id {
					case 1:
						return OrderInStorage(1, 24*time.Hour), nil
					case 2:
						return OrderInStorage(2, 48*time.Hour), nil
					default:
						return domain.Order{}, fmt.Errorf("unexpected id %d", id)
					}
				})
				r.UpdateMock.Set(func(_ context.Context, o domain.Order) error {
					if o.OrderID == 1 || o.OrderID == 2 {
						return nil
					}
					return fmt.Errorf("unexpected order %d", o.OrderID)
				})
				r.SaveHistoryMock.Set(func(_ context.Context, h domain.OrderHistory) error {
					if h.OrderID == 1 || h.OrderID == 2 {
						return nil
					}
					return fmt.Errorf("unexpected history %d", h.OrderID)
				})
			},
			assertE: assert.NoError,
		},
		{
			name:     "Fail_OrderNotFound",
			orderIDs: []uint64{3},
			setup: func(r *mock.OrderRepositoryMock, ctx context.Context) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					if id == 3 {
						return domain.Order{}, domain.EntityNotFoundError("Order", "3")
					}
					return domain.Order{}, fmt.Errorf("unexpected id %d", id)
				})
			},
			assertE: errIs(domain.EntityNotFoundError("Order", "3")),
		},
		{
			name:     "Fail_BelongsToDifferentReceiver",
			orderIDs: []uint64{4},
			setup: func(r *mock.OrderRepositoryMock, ctx context.Context) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					if id == 4 {
						bad := OrderInStorage(4, 24*time.Hour)
						bad.ReceiverID = 999
						return bad, nil
					}
					return domain.Order{}, fmt.Errorf("unexpected id %d", id)
				})
			},
			assertE: errIs(domain.BelongsToDifferentReceiverError(4, someRecieverID, 999)),
		},
		{
			name:     "Fail_AlreadyGiven",
			orderIDs: []uint64{5},
			setup: func(r *mock.OrderRepositoryMock, ctx context.Context) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					if id == 5 {
						return OrderGiven(5, 0), nil
					}
					return domain.Order{}, fmt.Errorf("unexpected id %d", id)
				})
			},
			assertE: errIs(domain.OrderAlreadyGivenError(5)),
		},
		{
			name:     "Fail_ReturnedOrderUnavailable",
			orderIDs: []uint64{6},
			setup: func(r *mock.OrderRepositoryMock, ctx context.Context) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					if id == 6 {
						return OrderReturned(6, 0), nil
					}
					return domain.Order{}, fmt.Errorf("unexpected id %d", id)
				})
			},
			assertE: errIs(domain.UnavaliableReturnedOrderError(6)),
		},
		{
			name:     "Fail_UpdateError",
			orderIDs: []uint64{7},
			setup: func(r *mock.OrderRepositoryMock, ctx context.Context) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					if id == 7 {
						return OrderInStorage(7, 24*time.Hour), nil
					}
					return domain.Order{}, fmt.Errorf("unexpected id %d", id)
				})
				r.UpdateMock.Set(func(_ context.Context, o domain.Order) error {
					if o.OrderID == 7 {
						return errUpdate
					}
					return fmt.Errorf("unexpected order %d", o.OrderID)
				})
			},
			assertE: errIs(errUpdate),
		},
		{
			name:     "Fail_SaveHistoryError",
			orderIDs: []uint64{8},
			setup: func(r *mock.OrderRepositoryMock, ctx context.Context) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					if id == 8 {
						return OrderInStorage(8, 24*time.Hour), nil
					}
					return domain.Order{}, fmt.Errorf("unexpected id %d", id)
				})
				r.UpdateMock.Set(func(_ context.Context, o domain.Order) error {
					if o.OrderID == 8 {
						return nil
					}
					return fmt.Errorf("unexpected order %d", o.OrderID)
				})
				r.SaveHistoryMock.Set(func(_ context.Context, h domain.OrderHistory) error {
					if h.OrderID == 8 {
						return errHist
					}
					return fmt.Errorf("unexpected history %d", h.OrderID)
				})
			},
			assertE: errIs(errHist),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			tc.setup(repo, ctx)

			err := svc.IssueOrdersToClient(ctx, someRecieverID, tc.orderIDs)
			tc.assertE(t, err)
		})
	}
}
