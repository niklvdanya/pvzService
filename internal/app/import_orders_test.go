package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/app/mock"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

var (
	bagRules = []domain.PackageRules{{MaxWeight: 10, Price: 5}}
	errDB    = errors.New("db err")
)

func expOrder(dto domain.OrderToImport, extra float64) domain.Order {
	st, _ := cli.MapStringToTime(dto.StorageUntil)
	return domain.Order{
		OrderID:        dto.OrderID,
		ReceiverID:     dto.ReceiverID,
		StorageUntil:   st,
		Status:         domain.StatusInStorage,
		AcceptTime:     someConstTime,
		LastUpdateTime: someConstTime,
		PackageType:    dto.PackageType,
		Weight:         dto.Weight,
		Price:          dto.Price + extra,
	}
}

func TestPVZService_ImportOrders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        []domain.OrderToImport
		setup        func(*mock.OrderRepositoryMock)
		wantImported uint64
		assertE      assert.ErrorAssertionFunc
	}{
		{
			name: "Success_Multiple",
			input: []domain.OrderToImport{
				DTO(1, "bag", 24*time.Hour),
				DTO(2, "bag", 24*time.Hour),
			},
			setup: func(r *mock.OrderRepositoryMock) {
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
			name: "Partial_InvalidPackage",
			input: []domain.OrderToImport{
				DTO(3, "bag", 24*time.Hour),
				DTO(4, "unknown", 24*time.Hour),
			},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Set(func(_ context.Context, _ uint64) (domain.Order, error) {
					return domain.Order{}, domain.EntityNotFoundError("Order", "x")
				})
				r.GetPackageRulesMock.Set(func(_ context.Context, code string) ([]domain.PackageRules, error) {
					if code == "bag" {
						return bagRules, nil
					}
					return nil, domain.InvalidPackageError(code)
				})
				want := expOrder(DTO(3, "bag", 24*time.Hour), 5)
				r.SaveMock.Expect(contextBack, want).Return(nil)
				r.SaveHistoryMock.Expect(contextBack, History(3, domain.StatusInStorage, 0)).Return(nil)
			},
			wantImported: 1,
			assertE:      errIs(domain.InvalidPackageError("unknown")),
		},
		{
			name:  "Fail_SaveError",
			input: []domain.OrderToImport{DTO(6, "bag", 24*time.Hour)},
			setup: func(r *mock.OrderRepositoryMock) {
				r.GetByIDMock.Set(func(_ context.Context, _ uint64) (domain.Order, error) {
					return domain.Order{}, domain.EntityNotFoundError("Order", "6")
				})
				r.GetPackageRulesMock.Set(func(_ context.Context, _ string) ([]domain.PackageRules, error) {
					return bagRules, nil
				})
				r.SaveMock.Expect(contextBack, expOrder(DTO(6, "bag", 24*time.Hour), 5)).Return(errDB)
			},
			wantImported: 0,
			assertE:      errIs(errDB),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			tc.setup(repo)

			got, err := svc.ImportOrders(contextBack, tc.input)
			assert.Equal(t, tc.wantImported, got)
			tc.assertE(t, err)
		})
	}
}
