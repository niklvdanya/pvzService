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
	bagRules = []domain.PackageRules{{MaxWeight: 10, Price: 5}}
	errDB    = errors.New("db err")
)

func TestPVZService_ImportOrders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        []domain.OrderToImport
		setup        func(*mock.OrderRepositoryMock, context.Context)
		wantImported uint64
		assertE      assert.ErrorAssertionFunc
	}{
		{
			name: "Success_Multiple",
			input: []domain.OrderToImport{
				DTO(1, "bag", 24*time.Hour),
				DTO(2, "bag", 24*time.Hour),
			},
			setup: func(r *mock.OrderRepositoryMock, ctx context.Context) {
				r.GetByIDMock.Set(func(_ context.Context, _ uint64) (domain.Order, error) {
					return domain.Order{}, domain.EntityNotFoundError("Order", "x")
				})
				r.GetPackageRulesMock.Set(func(_ context.Context, _ string) ([]domain.PackageRules, error) {
					return bagRules, nil
				})
				r.SaveMock.Set(func(_ context.Context, _ domain.Order) error { return nil })
				r.SaveHistoryMock.Set(func(_ context.Context, _ domain.OrderHistory) error { return nil })
			},
			wantImported: 2,
			assertE:      assert.NoError,
		},
		{
			name:  "Fail_SaveError",
			input: []domain.OrderToImport{DTO(6, "bag", 24*time.Hour)},
			setup: func(r *mock.OrderRepositoryMock, ctx context.Context) {
				r.GetByIDMock.Set(func(_ context.Context, _ uint64) (domain.Order, error) {
					return domain.Order{}, domain.EntityNotFoundError("Order", "6")
				})
				r.GetPackageRulesMock.Set(func(_ context.Context, _ string) ([]domain.PackageRules, error) {
					return bagRules, nil
				})
				r.SaveMock.Set(func(_ context.Context, _ domain.Order) error { return errDB })
			},
			wantImported: 0,
			assertE:      errIs(errDB),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			tc.setup(repo, ctx)
			got, err := svc.ImportOrders(ctx, tc.input)
			assert.Equal(t, tc.wantImported, got)
			tc.assertE(t, err)
		})
	}
}
