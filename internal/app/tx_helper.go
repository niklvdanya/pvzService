package app

import (
	"context"
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
)

func saveOrderInTx(ctx context.Context, tx *db.Tx, o domain.Order) error {
	const query = `
        INSERT INTO orders (
            id, receiver_id, expires_at, status,
            accept_time, last_update_time, package_code, weight, price)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
        ON CONFLICT (id) DO NOTHING`

	res, err := tx.Exec(ctx, query,
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

func updateOrderInTx(ctx context.Context, tx *db.Tx, o domain.Order) error {
	const query = `
        UPDATE orders
        SET receiver_id = $2, expires_at = $3, status = $4,
            accept_time = $5, last_update_time = $6,
            package_code = $7, weight = $8, price = $9
        WHERE id = $1`

	res, err := tx.Exec(ctx, query,
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

func saveHistoryInTx(ctx context.Context, tx *db.Tx, h domain.OrderHistory) error {
	const query = `INSERT INTO order_history (order_id, status, changed_at) VALUES ($1,$2,$3)`
	_, err := tx.Exec(ctx, query, h.OrderID, h.Status, h.ChangedAt)
	if err != nil {
		return fmt.Errorf("exec insert history: %w", err)
	}
	return nil
}

func (s *PVZService) withTransaction(ctx context.Context, fn func(*db.Tx) error) error {
	tx, err := s.dbClient.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
