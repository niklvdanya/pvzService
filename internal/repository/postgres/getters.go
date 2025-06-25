package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (r *OrderRepository) GetByID(ctx context.Context, orderID uint64) (domain.Order, error) {
	query := `
		SELECT id, receiver_id, expires_at, status, accept_time, last_update_time, package_code, weight, price
		FROM orders
		WHERE id = $1
	`
	row := r.client.QueryRow(ctx, query, orderID)

	order, err := scanOrder(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Order{}, fmt.Errorf("get by id: %w", domain.EntityNotFoundError("Order", fmt.Sprintf("%d", orderID)))
	}
	if err != nil {
		return domain.Order{}, err
	}

	return order, nil
}

func (r *OrderRepository) GetByReceiverID(ctx context.Context, receiverID uint64) ([]domain.Order, error) {
	query := `
		SELECT id, receiver_id, expires_at, status, accept_time, last_update_time, package_code, weight, price
		FROM orders
		WHERE receiver_id = $1
	`
	rows, err := r.client.Query(ctx, query, receiverID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (r *OrderRepository) GetReturnedOrders(ctx context.Context) ([]domain.Order, error) {
	query := `
		SELECT id, receiver_id, expires_at, status, accept_time, last_update_time, package_code, weight, price
		FROM orders
		WHERE status IN ($1, $2)
		ORDER BY last_update_time DESC
	`
	rows, err := r.client.Query(ctx, query, domain.StatusReturnedFromClient, domain.StatusGivenToCourier)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (r *OrderRepository) GetAllOrders(ctx context.Context) ([]domain.Order, error) {
	query := `
		SELECT id, receiver_id, expires_at, status, accept_time, last_update_time, package_code, weight, price
		FROM orders
		ORDER BY last_update_time DESC
	`
	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (r *OrderRepository) GetPackageRules(ctx context.Context, code string) ([]domain.PackageRules, error) {
	query := `
		SELECT max_weight, extra_price
		FROM package_types
		WHERE code = $1
	`
	rows, err := r.client.Query(ctx, query, code)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var rules []domain.PackageRules
	for rows.Next() {
		var rule domain.PackageRules
		err := rows.Scan(&rule.MaxWeight, &rule.Price)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		rules = append(rules, rule)
	}

	if len(rules) == 0 {
		return nil, fmt.Errorf("no rules found for package code %s", code)
	}

	return rules, nil
}

func (r *OrderRepository) GetHistoryByOrderID(ctx context.Context, orderID uint64) ([]domain.OrderHistory, error) {
	query := `
        SELECT order_id, status, changed_at
        FROM order_history
        WHERE order_id = $1
        ORDER BY changed_at DESC
    `

	rows, err := r.client.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var history []domain.OrderHistory
	for rows.Next() {
		var h domain.OrderHistory
		err := rows.Scan(&h.OrderID, &h.Status, &h.ChangedAt)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		history = append(history, h)
	}

	return history, nil
}
