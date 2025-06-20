package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
)

type OrderRepository struct {
	client *db.Client
}

func NewOrderRepository(client *db.Client) *OrderRepository {
	return &OrderRepository{client: client}
}

func (r *OrderRepository) Save(order *domain.Order) error {
	query := `
		INSERT INTO orders (id, receiver_id, expires_at, status, accept_time, last_update_time, package_code, weight, price)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	ctx := context.Background()
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	var exists int
	err = tx.QueryRow(ctx, `
		SELECT 1 FROM orders WHERE id = $1
	`, order.OrderID).Scan(&exists)
	if err == nil {
		tx.Rollback()
		return fmt.Errorf("save: %w", domain.OrderAlreadyExistsError(order.OrderID))
	}
	if err != sql.ErrNoRows {
		tx.Rollback()
		return fmt.Errorf("check existence: %w", err)
	}

	_, err = tx.Exec(ctx, query,
		order.OrderID,
		order.ReceiverID,
		order.StorageUntil,
		order.Status,
		order.AcceptTime,
		order.LastUpdateTime,
		order.PackageType,
		order.Weight,
		order.Price,
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("exec insert: %w", err)
	}

	return tx.Commit()
}

func (r *OrderRepository) GetByID(orderID uint64) (*domain.Order, error) {
	query := `
		SELECT id, receiver_id, expires_at, status, accept_time, last_update_time, package_code, weight, price
		FROM orders
		WHERE id = $1
	`
	ctx := context.Background()
	row := r.client.QueryRow(ctx, query, orderID)

	var order domain.Order
	var expiresAt, acceptTime, lastUpdateTime time.Time
	var packageCode sql.NullString

	err := row.Scan(
		&order.OrderID,
		&order.ReceiverID,
		&expiresAt,
		&order.Status,
		&acceptTime,
		&lastUpdateTime,
		&packageCode,
		&order.Weight,
		&order.Price,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("get by id: %w", domain.EntityNotFoundError("Order", fmt.Sprintf("%d", orderID)))
	}
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	order.StorageUntil = expiresAt
	order.AcceptTime = acceptTime
	order.LastUpdateTime = lastUpdateTime
	order.PackageType = packageCode.String

	return &order, nil
}

func (r *OrderRepository) Update(order *domain.Order) error {
	query := `
		UPDATE orders
		SET receiver_id = $2, expires_at = $3, status = $4, accept_time = $5, last_update_time = $6, package_code = $7, weight = $8, price = $9
		WHERE id = $1
	`
	ctx := context.Background()
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	result, err := tx.Exec(ctx, query,
		order.OrderID,
		order.ReceiverID,
		order.StorageUntil,
		order.Status,
		order.AcceptTime,
		order.LastUpdateTime,
		order.PackageType,
		order.Weight,
		order.Price,
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("exec update: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("update: %w", domain.EntityNotFoundError("Order", fmt.Sprintf("%d", order.OrderID)))
	}

	return tx.Commit()
}

func (r *OrderRepository) GetByReceiverID(receiverID uint64) ([]*domain.Order, error) {
	query := `
		SELECT id, receiver_id, expires_at, status, accept_time, last_update_time, package_code, weight, price
		FROM orders
		WHERE receiver_id = $1
	`
	ctx := context.Background()
	rows, err := r.client.Query(ctx, query, receiverID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var order domain.Order
		var expiresAt, acceptTime, lastUpdateTime time.Time
		var packageCode sql.NullString

		err := rows.Scan(
			&order.OrderID,
			&order.ReceiverID,
			&expiresAt,
			&order.Status,
			&acceptTime,
			&lastUpdateTime,
			&packageCode,
			&order.Weight,
			&order.Price,
		)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		order.StorageUntil = expiresAt
		order.AcceptTime = acceptTime
		order.LastUpdateTime = lastUpdateTime
		order.PackageType = packageCode.String
		orders = append(orders, &order)
	}

	return orders, nil
}

func (r *OrderRepository) GetReturnedOrders() ([]*domain.Order, error) {
	query := `
		SELECT id, receiver_id, expires_at, status, accept_time, last_update_time, package_code, weight, price
		FROM orders
		WHERE status IN ($1, $2)
		ORDER BY last_update_time DESC
	`
	ctx := context.Background()
	rows, err := r.client.Query(ctx, query, domain.StatusReturnedFromClient, domain.StatusGivenToCourier)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var order domain.Order
		var expiresAt, acceptTime, lastUpdateTime time.Time
		var packageCode sql.NullString

		err := rows.Scan(
			&order.OrderID,
			&order.ReceiverID,
			&expiresAt,
			&order.Status,
			&acceptTime,
			&lastUpdateTime,
			&packageCode,
			&order.Weight,
			&order.Price,
		)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		order.StorageUntil = expiresAt
		order.AcceptTime = acceptTime
		order.LastUpdateTime = lastUpdateTime
		order.PackageType = packageCode.String
		orders = append(orders, &order)
	}

	return orders, nil
}

func (r *OrderRepository) GetAllOrders() ([]*domain.Order, error) {
	query := `
		SELECT id, receiver_id, expires_at, status, accept_time, last_update_time, package_code, weight, price
		FROM orders
		ORDER BY last_update_time DESC
	`
	ctx := context.Background()
	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var order domain.Order
		var expiresAt, acceptTime, lastUpdateTime time.Time
		var packageCode sql.NullString

		err := rows.Scan(
			&order.OrderID,
			&order.ReceiverID,
			&expiresAt,
			&order.Status,
			&acceptTime,
			&lastUpdateTime,
			&packageCode,
			&order.Weight,
			&order.Price,
		)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		order.StorageUntil = expiresAt
		order.AcceptTime = acceptTime
		order.LastUpdateTime = lastUpdateTime
		order.PackageType = packageCode.String
		orders = append(orders, &order)
	}

	return orders, nil
}

func (r *OrderRepository) GetPackageRules(code string) ([]domain.PackageRules, error) {
	query := `
		SELECT max_weight, extra_price
		FROM package_types
		WHERE code = $1
	`
	ctx := context.Background()
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
