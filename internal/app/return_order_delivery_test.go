package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gitlab.ozon.dev/safariproxd/homework/internal/app/mock"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

var (
	errDBDel   = errors.New("db err")
	errUpdDel  = errors.New("upd err")
	errHistDel = errors.New("hist err")
)

func TestPVZService_ReturnOrderToDelivery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		orderID uint64
		setup   func(*mock.OrderRepositoryMock)
		assertE assert.ErrorAssertionFunc
	}{
		{
			name:    "Success_FromStorage",
			orderID: 1,
			setup: func(r *mock.OrderRepositoryMock) {
				orig := OrderInStorage(1, -1*time.Hour)
				r.GetByIDMock.Set(func(ctx context.Context, id uint64) (domain.Order, error) {
					if id == 1 {
						return orig, nil
					}
					return domain.Order{}, errors.New("unexpected id")
				})
				r.UpdateMock.Set(func(ctx context.Context, o domain.Order) error {
					if o.OrderID == 1 && o.Status == domain.StatusReturnedWithoutClient {
						return nil
					}
					return errors.New("unexpected update params")
				})
				r.SaveHistoryMock.Set(func(ctx context.Context, h domain.OrderHistory) error {
					if h.OrderID == 1 && h.Status == domain.StatusReturnedWithoutClient {
						return nil
					}
					return errors.New("unexpected history params")
				})
			},
			assertE: assert.NoError,
		},
		{
			name:    "Success_FromReturned",
			orderID: 2,
			setup: func(r *mock.OrderRepositoryMock) {
				orig := OrderReturned(2, -1*time.Hour)
				orig.StorageUntil = someConstTime.Add(-1 * time.Hour)

				r.GetByIDMock.Set(func(ctx context.Context, id uint64) (domain.Order, error) {
					if id == 2 {
						return orig, nil
					}
					return domain.Order{}, errors.New("unexpected id")
				})
				r.UpdateMock.Set(func(ctx context.Context, o domain.Order) error {
					if o.OrderID == 2 && o.Status == domain.StatusGivenToCourier {
						return nil
					}
					return errors.New("unexpected update params")
				})
				r.SaveHistoryMock.Set(func(ctx context.Context, h domain.OrderHistory) error {
					if h.OrderID == 2 && h.Status == domain.StatusGivenToCourier {
						return nil
					}
					return errors.New("unexpected history params")
				})
			},
			assertE: assert.NoError,
		},
		{
			name:    "Fail_RepoError",
			orderID: 3,
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Set(func(ctx context.Context, id uint64) (domain.Order, error) {
					if id == 3 {
						return domain.Order{}, errDBDel
					}
					return domain.Order{}, errors.New("unexpected id")
				})
			},
			assertE: errIs(errDBDel),
		},
		{
			name:    "Fail_StorageNotExpired",
			orderID: 4,
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Set(func(ctx context.Context, id uint64) (domain.Order, error) {
					if id == 4 {
						return OrderInStorage(4, +1*time.Hour), nil
					}
					return domain.Order{}, errors.New("unexpected id")
				})
			},
			assertE: errIs(domain.StorageNotExpiredError(4, DateString(+1*time.Hour))),
		},
		{
			name:    "Fail_UpdateError",
			orderID: 5,
			setup: func(r *mock.OrderRepositoryMock) {
				orig := OrderInStorage(5, -1*time.Hour)

				r.GetByIDMock.Set(func(ctx context.Context, id uint64) (domain.Order, error) {
					if id == 5 {
						return orig, nil
					}
					return domain.Order{}, errors.New("unexpected id")
				})
				r.UpdateMock.Set(func(ctx context.Context, o domain.Order) error {
					if o.OrderID == 5 && o.Status == domain.StatusReturnedWithoutClient {
						return errUpdDel
					}
					return errors.New("unexpected update params")
				})
			},
			assertE: errIs(errUpdDel),
		},
		{
			name:    "Fail_SaveHistoryError",
			orderID: 6,
			setup: func(r *mock.OrderRepositoryMock) {
				orig := OrderInStorage(6, -1*time.Hour)

				r.GetByIDMock.Set(func(ctx context.Context, id uint64) (domain.Order, error) {
					if id == 6 {
						return orig, nil
					}
					return domain.Order{}, errors.New("unexpected id")
				})
				r.UpdateMock.Set(func(ctx context.Context, o domain.Order) error {
					if o.OrderID == 6 && o.Status == domain.StatusReturnedWithoutClient {
						return nil
					}
					return errors.New("unexpected update params")
				})
				r.SaveHistoryMock.Set(func(ctx context.Context, h domain.OrderHistory) error {
					if h.OrderID == 6 && h.Status == domain.StatusReturnedWithoutClient {
						return errHistDel
					}
					return errors.New("unexpected history params")
				})
			},
			assertE: errIs(errHistDel),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			tc.setup(repo)

			err := svc.ReturnOrderToDelivery(context.Background(), tc.orderID)
			tc.assertE(t, err)
		})
	}
}
