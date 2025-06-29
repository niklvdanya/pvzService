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
				r.SaveHistoryMock.Expect(contextBack,
					History(1, domain.StatusReturnedWithoutClient, 0)).Return(nil)
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
				r.SaveHistoryMock.Expect(contextBack,
					History(2, domain.StatusGivenToCourier, 0)).Return(nil)
			},
			assertE: assert.NoError,
		},
		{
			name:    "Fail_RepoError",
			orderID: 3,
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Expect(contextBack, uint64(3)).Return(domain.Order{}, fmt.Errorf("db err"))
			},
			assertE: assert.Error,
		},
		{
			name:    "Fail_WrongStatus",
			orderID: 4,
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Expect(contextBack, uint64(4)).Return(OrderGiven(4, 0), nil)
			},
			assertE: assert.Error,
		},
		{
			name:    "Fail_StorageNotExpired",
			orderID: 5,
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Expect(contextBack, uint64(5)).Return(OrderInStorage(5, +1*time.Hour), nil)
			},
			assertE: assert.Error,
		},
		{
			name:    "Fail_UpdateError",
			orderID: 6,
			setup: func(r *mock.OrderRepositoryMock) {
				orig := OrderInStorage(6, -1*time.Hour)
				upd := Updated(orig, domain.StatusReturnedWithoutClient, someConstTime)

				r.GetByIDMock.Expect(contextBack, orig.OrderID).Return(orig, nil)
				r.UpdateMock.Expect(contextBack, upd).Return(fmt.Errorf("upd err"))
			},
			assertE: assert.Error,
		},
		{
			name:    "Fail_SaveHistoryError",
			orderID: 7,
			setup: func(r *mock.OrderRepositoryMock) {
				orig := OrderInStorage(7, -1*time.Hour)
				upd := Updated(orig, domain.StatusReturnedWithoutClient, someConstTime)

				r.GetByIDMock.Expect(contextBack, orig.OrderID).Return(orig, nil)
				r.UpdateMock.Expect(contextBack, upd).Return(nil)
				r.SaveHistoryMock.Expect(contextBack,
					History(7, domain.StatusReturnedWithoutClient, 0)).Return(fmt.Errorf("hist err"))
			},
			assertE: assert.Error,
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
