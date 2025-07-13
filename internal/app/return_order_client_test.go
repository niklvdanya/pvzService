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
						return OrderGiven(1, -10*time.Hour), nil
					case 2:
						return OrderGiven(2, -8*time.Hour), nil
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
						bad := OrderGiven(4, -4*time.Hour)
						bad.ReceiverID = 999
						return bad, nil
					}
					return domain.Order{}, fmt.Errorf("unexpected id %d", id)
				})
			},
			assertE: errIs(domain.BelongsToDifferentReceiverError(4, someRecieverID, 999)),
		},
		{
			name:     "Fail_AlreadyInStorage",
			orderIDs: []uint64{5},
			setup: func(r *mock.OrderRepositoryMock, ctx context.Context) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					if id == 5 {
						return OrderInStorage(5, 0), nil
					}
					return domain.Order{}, fmt.Errorf("unexpected id %d", id)
				})
			},
			assertE: errIs(domain.AlreadyInStorageError(5)),
		},
		{
			name:     "Fail_UpdateError",
			orderIDs: []uint64{6},
			setup: func(r *mock.OrderRepositoryMock, ctx context.Context) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					if id == 6 {
						return OrderGiven(6, -2*time.Hour), nil
					}
					return domain.Order{}, fmt.Errorf("unexpected id %d", id)
				})
				r.UpdateMock.Set(func(_ context.Context, o domain.Order) error {
					if o.OrderID == 6 {
						return errUpdRet
					}
					return fmt.Errorf("unexpected order %d", o.OrderID)
				})
			},
			assertE: errIs(errUpdRet),
		},
		{
			name:     "Fail_SaveHistoryError",
			orderIDs: []uint64{7},
			setup: func(r *mock.OrderRepositoryMock, ctx context.Context) {
				r.GetByIDMock.Set(func(_ context.Context, id uint64) (domain.Order, error) {
					if id == 7 {
						return OrderGiven(7, -3*time.Hour), nil
					}
					return domain.Order{}, fmt.Errorf("unexpected id %d", id)
				})
				r.UpdateMock.Set(func(_ context.Context, o domain.Order) error {
					if o.OrderID == 7 {
						return nil
					}
					return fmt.Errorf("unexpected order %d", o.OrderID)
				})
				r.SaveHistoryMock.Set(func(_ context.Context, h domain.OrderHistory) error {
					if h.OrderID == 7 {
						return errHistRet
					}
					return fmt.Errorf("unexpected history %d", h.OrderID)
				})
			},
			assertE: errIs(errHistRet),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			tc.setup(repo, ctx)

			err := svc.ReturnOrdersFromClient(ctx, someRecieverID, tc.orderIDs)
			tc.assertE(t, err)
		})
	}
}
