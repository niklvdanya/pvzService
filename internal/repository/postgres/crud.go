package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

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

func (r *OrderRepository) SaveHistory(ctx context.Context, history domain.OrderHistory) error {
	query := `
		INSERT INTO order_history (order_id, status, changed_at)
		VALUES ($1, $2, $3)
	`

	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	_, err = tx.Exec(ctx, query, history.OrderID, history.Status, history.ChangedAt)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("exec insert: %w", err)
	}

	return tx.Commit()
}
