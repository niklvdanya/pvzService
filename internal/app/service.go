package app

import (
	"context"
	"os"
	"sync/atomic"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/internal/metrics"
	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
	"golang.org/x/sync/errgroup"
)

type OrderRepository interface {
	Save(ctx context.Context, order domain.Order) error
	GetByID(ctx context.Context, orderID uint64) (domain.Order, error)
	Update(ctx context.Context, order domain.Order) error
	GetByReceiverID(ctx context.Context, receiverID uint64) ([]domain.Order, error)
	GetReturnedOrders(ctx context.Context) ([]domain.Order, error)
	GetAllOrders(ctx context.Context) ([]domain.Order, error)
	GetPackageRules(ctx context.Context, code string) ([]domain.PackageRules, error)
	SaveHistory(ctx context.Context, history domain.OrderHistory) error
	GetHistoryByOrderID(ctx context.Context, orderID uint64) ([]domain.OrderHistory, error)
	UpdateOrderInTx(ctx context.Context, tx *db.Tx, order domain.Order) error
	SaveOrderInTx(ctx context.Context, tx *db.Tx, order domain.Order) error
	SaveHistoryInTx(ctx context.Context, tx *db.Tx, history domain.OrderHistory) error
}

type OutboxRepository interface {
	Save(ctx context.Context, tx *db.Tx, event domain.Event) error
}

type PVZService struct {
	orderRepo       OrderRepository
	outboxRepo      OutboxRepository
	dbClient        *db.Client
	nowFn           func() time.Time
	workerLimit     int
	metricsProvider metrics.MetricsProvider
}

func NewPVZService(
	orderRepo OrderRepository,
	outboxRepo OutboxRepository,
	dbClient *db.Client,
	nowFn func() time.Time,
	limit int,
	metricsProvider metrics.MetricsProvider,
) *PVZService {
	if nowFn == nil {
		nowFn = time.Now
	}
	if limit <= 0 {
		limit = 1
	}
	return &PVZService{
		orderRepo:       orderRepo,
		outboxRepo:      outboxRepo,
		dbClient:        dbClient,
		nowFn:           nowFn,
		workerLimit:     limit,
		metricsProvider: metricsProvider,
	}
}

func Paginate[T any](items []T, currentPage, itemsPerPage uint64) []T {
	totalItems := uint64(len(items))

	if itemsPerPage == 0 {
		return []T{}
	}
	if currentPage == 0 {
		currentPage = 1
	}

	startIndex := (currentPage - 1) * itemsPerPage
	endIndex := startIndex + itemsPerPage

	if startIndex >= totalItems {
		return []T{}
	}
	if endIndex > totalItems {
		endIndex = totalItems
	}

	return items[startIndex:endIndex]
}

func processConcurrently[T any](
	parentCtx context.Context,
	items []T,
	workerLimit int,
	fn func(context.Context, T) error,
) (uint64, error) {
	g, ctx := errgroup.WithContext(parentCtx)
	sem := make(chan struct{}, workerLimit)

	var processed uint64

	for _, item := range items {
		g.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			}
			defer func() {
				<-sem
			}()

			if err := fn(ctx, item); err != nil {
				return err
			}
			atomic.AddUint64(&processed, 1)
			return nil
		})
	}

	err := g.Wait()
	return processed, err
}

func (s *PVZService) updateOrderStatusMetrics() {
	if os.Getenv("TESTING") == "true" {
		return
	}

	go func() {
		metricsCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		orders, err := s.orderRepo.GetAllOrders(metricsCtx)
		if err != nil {
			return
		}

		statusCounts := make(map[string]int)
		for _, order := range orders {
			statusCounts[order.GetStatusString()]++
		}

		for status, count := range statusCounts {
			metrics.OrdersByStatus.WithLabelValues(status).Set(float64(count))
		}
	}()
}
