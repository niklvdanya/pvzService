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

var errSaveHistory = errors.New("save history error")

func TestPVZService_AcceptOrder(t *testing.T) {
	t.Parallel()
	type testFixture struct {
		ctx          context.Context
		defaultReq   domain.AcceptOrderRequest
		fixedTime    time.Time
		packageRules []domain.PackageRules
	}

	fixture := testFixture{
		ctx: contextBack,
		defaultReq: domain.AcceptOrderRequest{
			OrderID:      1,
			ReceiverID:   someRecieverID,
			StorageUntil: someConstTime.Add(24 * time.Hour),
			Weight:       5.0,
			Price:        100.0,
			PackageType:  "bag",
		},
		fixedTime: someConstTime,
		packageRules: []domain.PackageRules{
			{MaxWeight: 10, Price: 5},
		},
	}

	expectOrderNotFound := func(repo *mock.OrderRepositoryMock, ctx context.Context, orderID uint64) {
		repo.GetByIDMock.Expect(ctx, orderID).Return(
			domain.Order{},
			domain.EntityNotFoundError("order", fmt.Sprintf("%d", orderID)),
		)
	}

	expectOrderExists := func(repo *mock.OrderRepositoryMock, ctx context.Context, orderID uint64) {
		repo.GetByIDMock.Expect(ctx, orderID).Return(
			domain.Order{OrderID: orderID},
			nil,
		)
	}

	expectPackageRules := func(repo *mock.OrderRepositoryMock, ctx context.Context, packageType string, rules []domain.PackageRules, err error) {
		repo.GetPackageRulesMock.Expect(ctx, packageType).Return(rules, err)
	}

	buildExpectedOrder := func(req domain.AcceptOrderRequest, totalPrice float64, tm time.Time) domain.Order {
		return domain.Order{
			OrderID:        req.OrderID,
			ReceiverID:     req.ReceiverID,
			StorageUntil:   req.StorageUntil,
			Status:         domain.StatusInStorage,
			AcceptTime:     tm,
			LastUpdateTime: tm,
			PackageType:    req.PackageType,
			Weight:         req.Weight,
			Price:          totalPrice,
		}
	}

	buildExpectedHistory := func(orderID uint64, tm time.Time) domain.OrderHistory {
		return domain.OrderHistory{
			OrderID:   orderID,
			Status:    domain.StatusInStorage,
			ChangedAt: tm,
		}
	}

	expectSaveOrder := func(repo *mock.OrderRepositoryMock, ctx context.Context, order domain.Order, err error) {
		repo.SaveMock.Expect(ctx, order).Return(err)
	}

	expectSaveHistory := func(repo *mock.OrderRepositoryMock, ctx context.Context, history domain.OrderHistory, err error) {
		repo.SaveHistoryMock.Expect(ctx, history).Return(err)
	}

	modifyRequest := func(base domain.AcceptOrderRequest, modifiers ...func(*domain.AcceptOrderRequest)) domain.AcceptOrderRequest {
		req := base
		for _, m := range modifiers {
			m(&req)
		}
		return req
	}

	withWeight := func(w float64) func(*domain.AcceptOrderRequest) {
		return func(r *domain.AcceptOrderRequest) { r.Weight = w }
	}

	withPackageType := func(pt string) func(*domain.AcceptOrderRequest) {
		return func(r *domain.AcceptOrderRequest) { r.PackageType = pt }
	}

	tests := []struct {
		name      string
		req       domain.AcceptOrderRequest
		prepare   func(*testing.T, *mock.OrderRepositoryMock, domain.AcceptOrderRequest)
		wantTotal float64
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "Success_AcceptOrder_WithPackageRules",
			req:  fixture.defaultReq,
			prepare: func(t *testing.T, repo *mock.OrderRepositoryMock, req domain.AcceptOrderRequest) {
				expectedOrder := buildExpectedOrder(req, req.Price+5, fixture.fixedTime)
				expectedHistory := buildExpectedHistory(req.OrderID, fixture.fixedTime)

				expectOrderNotFound(repo, fixture.ctx, req.OrderID)
				expectPackageRules(repo, fixture.ctx, req.PackageType, fixture.packageRules, nil)
				expectSaveOrder(repo, fixture.ctx, expectedOrder, nil)
				expectSaveHistory(repo, fixture.ctx, expectedHistory, nil)
			},
			wantTotal: 105.0,
			wantErr:   assert.NoError,
		},
		{
			name: "Fail_OrderAlreadyExists",
			req:  fixture.defaultReq,
			prepare: func(t *testing.T, repo *mock.OrderRepositoryMock, req domain.AcceptOrderRequest) {
				expectOrderExists(repo, fixture.ctx, req.OrderID)
			},
			wantTotal: 0,
			wantErr:   errIs(domain.OrderAlreadyExistsError(fixture.defaultReq.OrderID)),
		},
		{
			name: "Fail_InvalidPackageType",
			req:  fixture.defaultReq,
			prepare: func(t *testing.T, repo *mock.OrderRepositoryMock, req domain.AcceptOrderRequest) {
				expectOrderNotFound(repo, fixture.ctx, req.OrderID)
				expectPackageRules(repo, fixture.ctx, req.PackageType, nil, domain.InvalidPackageError(req.PackageType))
			},
			wantTotal: 0,
			wantErr:   errIs(domain.InvalidPackageError(fixture.defaultReq.PackageType)),
		},
		{
			name: "Fail_WeightTooHeavy",
			req:  modifyRequest(fixture.defaultReq, withWeight(15.0)),
			prepare: func(t *testing.T, repo *mock.OrderRepositoryMock, req domain.AcceptOrderRequest) {
				expectOrderNotFound(repo, fixture.ctx, req.OrderID)
				expectPackageRules(repo, fixture.ctx, req.PackageType, fixture.packageRules, nil)
			},
			wantTotal: 0,
			wantErr:   errIs(domain.WeightTooHeavyError(fixture.defaultReq.PackageType, 15.0, fixture.packageRules[0].MaxWeight)),
		},
		{
			name: "Success_AcceptOrder_WithoutPackageType",
			req:  modifyRequest(fixture.defaultReq, withPackageType("")),
			prepare: func(t *testing.T, repo *mock.OrderRepositoryMock, req domain.AcceptOrderRequest) {
				expectedOrder := buildExpectedOrder(req, req.Price, fixture.fixedTime)
				expectedHistory := buildExpectedHistory(req.OrderID, fixture.fixedTime)

				expectOrderNotFound(repo, fixture.ctx, req.OrderID)
				expectSaveOrder(repo, fixture.ctx, expectedOrder, nil)
				expectSaveHistory(repo, fixture.ctx, expectedHistory, nil)
			},
			wantTotal: 100.0,
			wantErr:   assert.NoError,
		},
		{
			name: "Fail_SaveHistory",
			req:  fixture.defaultReq,
			prepare: func(t *testing.T, repo *mock.OrderRepositoryMock, req domain.AcceptOrderRequest) {
				expectedOrder := buildExpectedOrder(req, req.Price+5, fixture.fixedTime)
				expectedHistory := buildExpectedHistory(req.OrderID, fixture.fixedTime)
				expectOrderNotFound(repo, fixture.ctx, req.OrderID)
				expectPackageRules(repo, fixture.ctx, req.PackageType, fixture.packageRules, nil)
				repo.SaveMock.Set(func(ctx context.Context, order domain.Order) error {
					if assert.Equal(t, expectedOrder, order) {
						return nil
					}
					return errors.New("unexpected order")
				})

				repo.SaveHistoryMock.Set(func(ctx context.Context, history domain.OrderHistory) error {
					if assert.Equal(t, expectedHistory, history) {
						return errSaveHistory
					}
					return errors.New("unexpected history")
				})
			},
			wantTotal: 0,
			wantErr:   errIs(errSaveHistory),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo, svc := NewEnv(t)
			tt.prepare(t, repo, tt.req)

			got, err := svc.AcceptOrder(fixture.ctx, tt.req)
			assert.Equal(t, tt.wantTotal, got)
			tt.wantErr(t, err)
		})
	}
}
