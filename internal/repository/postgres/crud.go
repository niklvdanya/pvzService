package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
)

// транзакции внутри каждого CRUD решил удалить — как будто с ними могут быть проблемы при параллельных запросах
// сами транзакции сейчас не используются, но планирую в бизнес логике их использовать
// (например атомарно выполнять методы Save и SaveHistory)
func (r *OrderRepository) Exists(ctx context.Context, orderID uint64) (bool, error) {
	const query = `SELECT 1 FROM orders WHERE id = $1`
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

func (r *OrderRepository) Save(ctx context.Context, o domain.Order) error {
	const query = `
        INSERT INTO orders (
            id, receiver_id, expires_at, status,
            accept_time, last_update_time, package_code, weight, price)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
        ON CONFLICT (id) DO NOTHING`

	res, err := r.client.Exec(ctx, db.ModeWrite, query,
		o.OrderID, o.ReceiverID, o.StorageUntil, o.Status,
		o.AcceptTime, o.LastUpdateTime, o.PackageType, o.Weight, o.Price,
	)
	if err != nil {
		return fmt.Errorf("exec insert: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.OrderAlreadyExistsError(o.OrderID)
	}
	return nil
}

func (r *OrderRepository) Update(ctx context.Context, o domain.Order) error {
	const query = `
        UPDATE orders
        SET receiver_id = $2, expires_at = $3, status = $4,
            accept_time = $5, last_update_time = $6,
            package_code = $7, weight = $8, price = $9
        WHERE id = $1`

	res, err := r.client.Exec(ctx, db.ModeWrite, query,
		o.OrderID, o.ReceiverID, o.StorageUntil, o.Status,
		o.AcceptTime, o.LastUpdateTime, o.PackageType, o.Weight, o.Price,
	)
	if err != nil {
		return fmt.Errorf("exec update: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.EntityNotFoundError("Order", fmt.Sprintf("%d", o.OrderID))
	}
	return nil
}

func (r *OrderRepository) SaveHistory(ctx context.Context, h domain.OrderHistory) error {
	const query = `INSERT INTO order_history (order_id, status, changed_at) VALUES ($1,$2,$3)`
	_, err := r.client.Exec(ctx, db.ModeWrite, query, h.OrderID, h.Status, h.ChangedAt)
	if err != nil {
		return fmt.Errorf("exec insert history: %w", err)
	}
	return nil
}

func (r *OrderRepository) SaveOrderInTx(ctx context.Context, tx *db.Tx, order domain.Order) error {
	const query = `
        INSERT INTO orders (
            id, receiver_id, expires_at, status,
            accept_time, last_update_time, package_code, weight, price)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
        ON CONFLICT (id) DO NOTHING`

	res, err := tx.Exec(ctx, query,
		order.OrderID, order.ReceiverID, order.StorageUntil, order.Status,
		order.AcceptTime, order.LastUpdateTime, order.PackageType, order.Weight, order.Price,
	)
	if err != nil {
		return fmt.Errorf("exec insert: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.OrderAlreadyExistsError(order.OrderID)
	}

	return nil
}

func (r *OrderRepository) UpdateOrderInTx(ctx context.Context, tx *db.Tx, order domain.Order) error {
	const query = `
        UPDATE orders
        SET receiver_id = $2, expires_at = $3, status = $4,
            accept_time = $5, last_update_time = $6,
            package_code = $7, weight = $8, price = $9
        WHERE id = $1`

	res, err := tx.Exec(ctx, query,
		order.OrderID, order.ReceiverID, order.StorageUntil, order.Status,
		order.AcceptTime, order.LastUpdateTime, order.PackageType, order.Weight, order.Price,
	)
	if err != nil {
		return fmt.Errorf("exec update: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.EntityNotFoundError("Order", fmt.Sprintf("%d", order.OrderID))
	}

	return nil
}

func (r *OrderRepository) SaveHistoryInTx(ctx context.Context, tx *db.Tx, history domain.OrderHistory) error {
	const query = `INSERT INTO order_history (order_id, status, changed_at) VALUES ($1,$2,$3)`

	_, err := tx.Exec(ctx, query, history.OrderID, history.Status, history.ChangedAt)
	if err != nil {
		return fmt.Errorf("exec insert history: %w", err)
	}

	return nil
}
