package postgres

import (
	"context"
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/internal/metrics"
	"gitlab.ozon.dev/safariproxd/homework/pkg/cache"
	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
)

type CachedOrderRepository struct {
	repo              *OrderRepository
	orderCache        *cache.LRUCache[string, domain.Order]
	receiverCache     *cache.LRUCache[string, []domain.Order]
	historyCache      *cache.LRUCache[string, []domain.OrderHistory]
	packageRulesCache *cache.LRUCache[string, []domain.PackageRules]
}

func NewCachedOrderRepository(client *db.Client, cacheConfig cache.Config) *CachedOrderRepository {
	repo := NewOrderRepository(client)

	return &CachedOrderRepository{
		repo:              repo,
		orderCache:        cache.New[string, domain.Order](cacheConfig),
		receiverCache:     cache.New[string, []domain.Order](cacheConfig),
		historyCache:      cache.New[string, []domain.OrderHistory](cacheConfig),
		packageRulesCache: cache.New[string, []domain.PackageRules](cacheConfig),
	}
}

func (r *CachedOrderRepository) Exists(ctx context.Context, orderID uint64) (bool, error) {
	key := r.orderKey(orderID)
	if _, found := r.orderCache.Get(key); found {
		metrics.CacheHits.WithLabelValues("orders", "hit").Inc()
		return true, nil
	}

	metrics.CacheHits.WithLabelValues("orders", "miss").Inc()
	return r.repo.Exists(ctx, orderID)
}

func (r *CachedOrderRepository) Save(ctx context.Context, order domain.Order) error {
	if err := r.repo.Save(ctx, order); err != nil {
		return err
	}
	r.invalidateOrderCaches(order.OrderID, order.ReceiverID)

	r.orderCache.Set(r.orderKey(order.OrderID), order)

	return nil
}

func (r *CachedOrderRepository) GetByID(ctx context.Context, orderID uint64) (domain.Order, error) {
	key := r.orderKey(orderID)

	if order, found := r.orderCache.Get(key); found {
		metrics.CacheHits.WithLabelValues("orders", "hit").Inc()
		return order, nil
	}

	metrics.CacheHits.WithLabelValues("orders", "miss").Inc()
	order, err := r.repo.GetByID(ctx, orderID)
	if err != nil {
		return order, err
	}

	r.orderCache.Set(key, order)
	return order, nil
}

func (r *CachedOrderRepository) Update(ctx context.Context, order domain.Order) error {
	if err := r.repo.Update(ctx, order); err != nil {
		return err
	}
	r.invalidateOrderCaches(order.OrderID, order.ReceiverID)
	r.orderCache.Set(r.orderKey(order.OrderID), order)

	return nil
}

func (r *CachedOrderRepository) GetByReceiverID(ctx context.Context, receiverID uint64) ([]domain.Order, error) {
	key := r.receiverKey(receiverID)

	if orders, found := r.receiverCache.Get(key); found {
		metrics.CacheHits.WithLabelValues("receivers", "hit").Inc()
		return orders, nil
	}

	metrics.CacheHits.WithLabelValues("receivers", "miss").Inc()
	orders, err := r.repo.GetByReceiverID(ctx, receiverID)
	if err != nil {
		return orders, err
	}

	r.receiverCache.Set(key, orders)
	return orders, nil
}

func (r *CachedOrderRepository) GetReturnedOrders(ctx context.Context) ([]domain.Order, error) {
	key := "returned_orders"

	if orders, found := r.receiverCache.Get(key); found {
		metrics.CacheHits.WithLabelValues("receivers", "hit").Inc()
		return orders, nil
	}

	metrics.CacheHits.WithLabelValues("receivers", "miss").Inc()
	orders, err := r.repo.GetReturnedOrders(ctx)
	if err != nil {
		return orders, err
	}

	r.receiverCache.Set(key, orders)
	return orders, nil
}

func (r *CachedOrderRepository) GetAllOrders(ctx context.Context) ([]domain.Order, error) {
	key := "all_orders"

	if orders, found := r.receiverCache.Get(key); found {
		metrics.CacheHits.WithLabelValues("receivers", "hit").Inc()
		return orders, nil
	}

	metrics.CacheHits.WithLabelValues("receivers", "miss").Inc()
	orders, err := r.repo.GetAllOrders(ctx)
	if err != nil {
		return orders, err
	}

	r.receiverCache.Set(key, orders)
	return orders, nil
}

func (r *CachedOrderRepository) GetPackageRules(ctx context.Context, code string) ([]domain.PackageRules, error) {
	key := fmt.Sprintf("package_rules:%s", code)

	if rules, found := r.packageRulesCache.Get(key); found {
		metrics.CacheHits.WithLabelValues("package_rules", "hit").Inc()
		return rules, nil
	}

	metrics.CacheHits.WithLabelValues("package_rules", "miss").Inc()
	rules, err := r.repo.GetPackageRules(ctx, code)
	if err != nil {
		return rules, err
	}

	r.packageRulesCache.Set(key, rules)
	return rules, nil
}

func (r *CachedOrderRepository) SaveHistory(ctx context.Context, history domain.OrderHistory) error {
	if err := r.repo.SaveHistory(ctx, history); err != nil {
		return err
	}
	r.historyCache.Delete(r.historyKey(history.OrderID))

	return nil
}

func (r *CachedOrderRepository) GetHistoryByOrderID(ctx context.Context, orderID uint64) ([]domain.OrderHistory, error) {
	key := r.historyKey(orderID)

	if history, found := r.historyCache.Get(key); found {
		metrics.CacheHits.WithLabelValues("history", "hit").Inc()
		return history, nil
	}

	metrics.CacheHits.WithLabelValues("history", "miss").Inc()
	history, err := r.repo.GetHistoryByOrderID(ctx, orderID)
	if err != nil {
		return history, err
	}

	r.historyCache.Set(key, history)
	return history, nil
}

func (r *CachedOrderRepository) CleanupExpired() {
	r.orderCache.CleanupExpired()
	r.receiverCache.CleanupExpired()
	r.historyCache.CleanupExpired()
	r.packageRulesCache.CleanupExpired()
}

func (r *CachedOrderRepository) ClearCache() {
	r.orderCache.Clear()
	r.receiverCache.Clear()
	r.historyCache.Clear()
	r.packageRulesCache.Clear()
}

func (r *CachedOrderRepository) GetCacheStats() map[string]int {
	return map[string]int{
		"orders":        r.orderCache.Size(),
		"receivers":     r.receiverCache.Size(),
		"history":       r.historyCache.Size(),
		"package_rules": r.packageRulesCache.Size(),
	}
}

func (r *CachedOrderRepository) orderKey(orderID uint64) string {
	return fmt.Sprintf("order:%d", orderID)
}

func (r *CachedOrderRepository) receiverKey(receiverID uint64) string {
	return fmt.Sprintf("receiver:%d", receiverID)
}

func (r *CachedOrderRepository) historyKey(orderID uint64) string {
	return fmt.Sprintf("history:%d", orderID)
}

func (r *CachedOrderRepository) invalidateOrderCaches(orderID uint64, receiverID uint64) {
	r.orderCache.Delete(r.orderKey(orderID))
	r.receiverCache.Delete(r.receiverKey(receiverID))

	r.receiverCache.Delete("returned_orders")
	r.receiverCache.Delete("all_orders")

	r.historyCache.Delete(r.historyKey(orderID))
}
