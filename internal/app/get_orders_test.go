package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gitlab.ozon.dev/safariproxd/homework/internal/app/mock"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func TestPVZService_GetReceiverOrders(t *testing.T) {
	t.Parallel()

	allOrders := []domain.Order{
		OrderInStorage(1, 1*time.Hour),
		BuildOrder(2, domain.StatusGivenToClient, 2*time.Hour, 0),
		OrderInStorage(3, 3*time.Hour),
		BuildOrder(4, domain.StatusReturnedFromClient, 4*time.Hour, 0),
		OrderInStorage(5, 5*time.Hour),
	}

	tests := []struct {
		name      string
		req       domain.ReceiverOrdersRequest
		setup     func(*mock.OrderRepositoryMock)
		wantIDs   []uint64
		wantTotal uint64
		assertE   assert.ErrorAssertionFunc
	}{
		{
			name: "All_NoFilter",
			req:  domain.ReceiverOrdersRequest{ReceiverID: someRecieverID, Page: 1, Limit: 100},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByReceiverIDMock.Expect(contextBack, someRecieverID).Return(allOrders, nil)
			},
			wantIDs:   []uint64{1, 2, 3, 4, 5},
			wantTotal: 5,
			assertE:   assert.NoError,
		},
		{
			name: "Filter_InPVZ",
			req:  domain.ReceiverOrdersRequest{ReceiverID: someRecieverID, InPVZ: true, Page: 1, Limit: 100},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByReceiverIDMock.Expect(contextBack, someRecieverID).Return(allOrders, nil)
			},
			wantIDs:   []uint64{1, 3, 5},
			wantTotal: 3,
			assertE:   assert.NoError,
		},
		{
			name: "LastN",
			req:  domain.ReceiverOrdersRequest{ReceiverID: someRecieverID, LastN: 2},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByReceiverIDMock.Expect(contextBack, someRecieverID).Return(allOrders, nil)
			},
			wantIDs:   []uint64{4, 5},
			wantTotal: 5,
			assertE:   assert.NoError,
		},
		{
			name: "Pagination_Page2",
			req:  domain.ReceiverOrdersRequest{ReceiverID: someRecieverID, Page: 2, Limit: 2},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByReceiverIDMock.Expect(contextBack, someRecieverID).Return(allOrders, nil)
			},
			wantIDs:   []uint64{3, 4},
			wantTotal: 5,
			assertE:   assert.NoError,
		},
		{
			name: "RepoError",
			req:  domain.ReceiverOrdersRequest{ReceiverID: someRecieverID, Page: 1, Limit: 10},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByReceiverIDMock.Expect(contextBack, someRecieverID).Return(nil, assert.AnError)
			},
			wantIDs:   nil,
			wantTotal: 0,
			assertE:   assert.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			tc.setup(repo)

			got, total, err := svc.GetReceiverOrders(contextBack, tc.req)

			tc.assertE(t, err)
			assert.Equal(t, tc.wantTotal, total)
			assert.ElementsMatch(t, tc.wantIDs, IdsOf(got))
		})
	}
}

func TestPVZService_GetReturnedOrders(t *testing.T) {
	t.Parallel()

	returned := []domain.Order{
		OrderReturned(1, -1*time.Minute),
		BuildOrder(2, domain.StatusGivenToCourier, 0, -2*time.Minute),
		BuildOrder(3, domain.StatusGivenToCourier, 0, -3*time.Minute),
		OrderReturned(4, -4*time.Minute),
	}

	tests := []struct {
		name      string
		page, lim uint64
		setup     func(*mock.OrderRepositoryMock)
		wantIDs   []uint64
		wantTotal uint64
		assertE   assert.ErrorAssertionFunc
	}{
		{
			name: "All_NoPagination",
			page: 1, lim: 100,
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetReturnedOrdersMock.Expect(contextBack).Return(returned, nil)
			},
			wantIDs:   []uint64{1, 2, 3, 4},
			wantTotal: 4,
			assertE:   assert.NoError,
		},
		{
			name: "Pagination_Page2",
			page: 2, lim: 2,
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetReturnedOrdersMock.Expect(contextBack).Return(returned, nil)
			},
			wantIDs:   []uint64{3, 4},
			wantTotal: 4,
			assertE:   assert.NoError,
		},
		{
			name: "PageBeyondRange",
			page: 3, lim: 2,
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetReturnedOrdersMock.Expect(contextBack).Return(returned, nil)
			},
			wantIDs:   nil,
			wantTotal: 4,
			assertE:   assert.NoError,
		},
		{
			name: "LimitZero",
			page: 1, lim: 0,
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetReturnedOrdersMock.Expect(contextBack).Return(returned, nil)
			},
			wantIDs:   nil,
			wantTotal: 4,
			assertE:   assert.NoError,
		},
		{
			name: "RepoError",
			page: 1, lim: 10,
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetReturnedOrdersMock.Expect(contextBack).Return(nil, assert.AnError)
			},
			wantIDs:   nil,
			wantTotal: 0,
			assertE:   assert.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			tc.setup(repo)

			got, total, err := svc.GetReturnedOrders(contextBack, tc.page, tc.lim)

			tc.assertE(t, err)
			assert.Equal(t, tc.wantTotal, total)
			assert.ElementsMatch(t, tc.wantIDs, IdsOf(got))
		})
	}
}

func TestPVZService_GetOrderHistory(t *testing.T) {
	t.Parallel()

	input := []domain.Order{
		BuildOrder(1, domain.StatusGivenToClient, 0, -3*time.Hour),
		BuildOrder(2, domain.StatusGivenToClient, 0, -1*time.Hour),
		BuildOrder(3, domain.StatusGivenToClient, 0, -2*time.Hour),
	}
	want := []uint64{2, 3, 1}

	tests := []struct {
		name    string
		setup   func(*mock.OrderRepositoryMock)
		wantIDs []uint64
		assertE assert.ErrorAssertionFunc
	}{
		{
			name: "Success",
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetAllOrdersMock.Expect(contextBack).Return(input, nil)
			},
			wantIDs: want,
			assertE: assert.NoError,
		},
		{
			name: "RepoError",
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetAllOrdersMock.Expect(contextBack).Return(nil, assert.AnError)
			},
			wantIDs: nil,
			assertE: assert.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			tc.setup(repo)

			got, err := svc.GetOrderHistory(contextBack)

			tc.assertE(t, err)
			assert.Equal(t, tc.wantIDs, IdsOf(got))
		})
	}
}

func TestPVZService_GetOrderHistoryByID(t *testing.T) {
	t.Parallel()

	id := uint64(42)
	h := []domain.OrderHistory{
		History(id, domain.StatusReturnedFromClient, -3*time.Hour),
		History(id, domain.StatusReturnedFromClient, -1*time.Hour),
		History(id, domain.StatusReturnedFromClient, -2*time.Hour),
	}
	wantTimes := TimesOf([]domain.OrderHistory{h[1], h[2], h[0]})

	tests := []struct {
		name    string
		setup   func(*mock.OrderRepositoryMock)
		want    []time.Time
		assertE assert.ErrorAssertionFunc
	}{
		{
			name: "Success",
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetHistoryByOrderIDMock.Expect(contextBack, id).Return(h, nil)
			},
			want:    wantTimes,
			assertE: assert.NoError,
		},
		{
			name: "EmptyHistory",
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetHistoryByOrderIDMock.Expect(contextBack, id).Return([]domain.OrderHistory{}, nil)
			},
			want:    nil,
			assertE: assert.Error,
		},
		{
			name: "RepoError",
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetHistoryByOrderIDMock.Expect(contextBack, id).Return(nil, assert.AnError)
			},
			want:    nil,
			assertE: assert.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			tc.setup(repo)

			got, err := svc.GetOrderHistoryByID(contextBack, id)

			tc.assertE(t, err)
			assert.Equal(t, tc.want, TimesOf(got))
		})
	}
}
