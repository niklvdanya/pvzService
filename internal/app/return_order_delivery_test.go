package app

import (
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
				upd := Updated(orig, domain.StatusReturnedWithoutClient, someConstTime)

				r.GetByIDMock.Expect(contextBack, orig.OrderID).Return(orig, nil)
				r.UpdateMock.Expect(contextBack, upd).Return(nil)
				r.SaveHistoryMock.Expect(contextBack, History(1, domain.StatusReturnedWithoutClient, 0)).Return(nil)
			},
			assertE: assert.NoError,
		},
		{
			name:    "Success_FromReturned",
			orderID: 2,
			setup: func(r *mock.OrderRepositoryMock) {
				orig := OrderReturned(2, -1*time.Hour)
				orig.StorageUntil = someConstTime.Add(-1 * time.Hour)
				upd := Updated(orig, domain.StatusGivenToCourier, someConstTime)

				r.GetByIDMock.Expect(contextBack, orig.OrderID).Return(orig, nil)
				r.UpdateMock.Expect(contextBack, upd).Return(nil)
				r.SaveHistoryMock.Expect(contextBack, History(2, domain.StatusGivenToCourier, 0)).Return(nil)
			},
			assertE: assert.NoError,
		},
		{
			name:    "Fail_RepoError",
			orderID: 3,
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Expect(contextBack, uint64(3)).Return(domain.Order{}, errDBDel)
			},
			assertE: errIs(errDBDel),
		},
		{
			name:    "Fail_StorageNotExpired",
			orderID: 4,
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Expect(contextBack, uint64(5)).Return(OrderInStorage(5, +1*time.Hour), nil)
			},
			assertE: errIs(domain.StorageNotExpiredError(5, DateString(+1*time.Hour))),
		},
		{
			name:    "Fail_UpdateError",
			orderID: 5,
			setup: func(r *mock.OrderRepositoryMock) {
				orig := OrderInStorage(6, -1*time.Hour)
				upd := Updated(orig, domain.StatusReturnedWithoutClient, someConstTime)

				r.GetByIDMock.Expect(contextBack, orig.OrderID).Return(orig, nil)
				r.UpdateMock.Expect(contextBack, upd).Return(errUpdDel)
			},
			assertE: errIs(errUpdDel),
		},
		{
			name:    "Fail_SaveHistoryError",
			orderID: 6,
			setup: func(r *mock.OrderRepositoryMock) {
				orig := OrderInStorage(7, -1*time.Hour)
				upd := Updated(orig, domain.StatusReturnedWithoutClient, someConstTime)

				r.GetByIDMock.Expect(contextBack, orig.OrderID).Return(orig, nil)
				r.UpdateMock.Expect(contextBack, upd).Return(nil)
				r.SaveHistoryMock.Expect(contextBack, History(7, domain.StatusReturnedWithoutClient, 0)).Return(errHistDel)
			},
			assertE: errIs(errHistDel),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			tc.setup(repo)

			err := svc.ReturnOrderToDelivery(contextBack, tc.orderID)
			tc.assertE(t, err)
		})
	}
}
