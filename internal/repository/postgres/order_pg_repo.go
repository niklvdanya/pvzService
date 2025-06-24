package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
)

type Scanner interface {
	Scan(dest ...interface{}) error
}

func scanOrder(scanner Scanner) (domain.Order, error) {
	var order domain.Order
	var expiresAt, acceptTime, lastUpdateTime time.Time
	var packageCode sql.NullString

	err := scanner.Scan(
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
		return domain.Order{}, fmt.Errorf("scan: %w", err)
	}

	return domain.Order{
		OrderID:        order.OrderID,
		ReceiverID:     order.ReceiverID,
		StorageUntil:   expiresAt,
		Status:         order.Status,
		AcceptTime:     acceptTime,
		LastUpdateTime: lastUpdateTime,
		PackageType:    packageCode.String,
		Weight:         order.Weight,
		Price:          order.Price,
	}, nil
}

type OrderRepository struct {
	client *db.Client
}

func NewOrderRepository(client *db.Client) *OrderRepository {
	return &OrderRepository{client: client}
}

func (r *OrderRepository) Exists(ctx context.Context, orderID uint64) (bool, error) {
	query := `SELECT 1 FROM orders WHERE id = $1`
	var exists int
	err := r.client.QueryRow(ctx, query, orderID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check existence: %w", err)
	}
	return true, nil
}

func (r *OrderRepository) Save(ctx context.Context, order domain.Order) error {
	query := `
		INSERT INTO orders (id, receiver_id, expires_at, status, accept_time, last_update_time, package_code, weight, price)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	exists, err := r.Exists(ctx, order.OrderID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("check existence: %w", err)
	}
	if exists {
		tx.Rollback()
		return fmt.Errorf("save: %w", domain.OrderAlreadyExistsError(order.OrderID))
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

func (r *OrderRepository) Update(ctx context.Context, order domain.Order) error {
	query := `
		UPDATE orders
		SET receiver_id = $2, expires_at = $3, status = $4, accept_time = $5, last_update_time = $6, package_code = $7, weight = $8, price = $9
		WHERE id = $1
	`
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
