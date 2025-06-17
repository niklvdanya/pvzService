package postgres

import (
	"database/sql"
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

type OrderPGRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderPGRepository {
	return &OrderPGRepository{db: db}
}

const orderFields = `
    id, receiver_id, status, expires_at,
    accept_time, last_update_time,
    package_code, weight, price
`

func scanOrder(row scannable) (*domain.Order, error) {
	var o domain.Order
	var pkg sql.NullString
	if err := row.Scan(
		&o.OrderID, &o.ReceiverID, &o.Status, &o.StorageUntil,
		&o.AcceptTime, &o.LastUpdateTime,
		&pkg, &o.Weight, &o.Price,
	); err != nil {
		return nil, err
	}
	if pkg.Valid {
		o.PackageType = pkg.String
	}
	return &o, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func (r *OrderPGRepository) Save(order *domain.Order) error {
	const q = `INSERT INTO orders (` + orderFields + `)
               VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err := r.db.Exec(
		q,
		order.OrderID, order.ReceiverID, order.Status, order.StorageUntil,
		order.AcceptTime, order.LastUpdateTime,
		nullable(order.PackageType), order.Weight, order.Price,
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}

func (r *OrderPGRepository) GetByID(orderID uint64) (*domain.Order, error) {
	const q = `SELECT ` + orderFields + ` FROM orders WHERE id=$1`
	row := r.db.QueryRow(q, orderID)
	o, err := scanOrder(row)
	if err == sql.ErrNoRows {
		return nil, domain.EntityNotFoundError("Order", fmt.Sprintf("%d", orderID))
	}
	return o, err
}

func (r *OrderPGRepository) Update(o *domain.Order) error {
	const q = `
        UPDATE orders
           SET status=$2, expires_at=$3, last_update_time=$4,
               package_code=$5, weight=$6, price=$7
         WHERE id=$1`
	_, err := r.db.Exec(q,
		o.OrderID, o.Status, o.StorageUntil, o.LastUpdateTime,
		nullable(o.PackageType), o.Weight, o.Price,
	)
	return err
}

func (r *OrderPGRepository) Delete(orderID uint64) error {
	_, err := r.db.Exec(`DELETE FROM orders WHERE id=$1`, orderID)
	return err
}

func (r *OrderPGRepository) GetByReceiverID(receiverID uint64) ([]*domain.Order, error) {
	const q = `SELECT ` + orderFields + ` FROM orders
               WHERE receiver_id=$1 AND status IN (0,1)`
	rows, err := r.db.Query(q, receiverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*domain.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, rows.Err()
}

func (r *OrderPGRepository) GetAllOrders() ([]*domain.Order, error) {
	rows, err := r.db.Query(`SELECT ` + orderFields + ` FROM orders`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*domain.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, rows.Err()
}

func (r *OrderPGRepository) GetReturnedOrders() ([]*domain.Order, error) {
	const q = `SELECT ` + orderFields + ` FROM orders
               WHERE status IN (2,4)`
	rows, err := r.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*domain.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, rows.Err()
}

func (r *OrderPGRepository) GetPackageRules(code string) ([]domain.PackageRules, error) {
	const q = `SELECT max_weight, extra_price FROM package_types WHERE code = $1`
	rows, err := r.db.Query(q, code)
	if err != nil {
		return nil, err
	}

	var res []domain.PackageRules
	for rows.Next() {
		var pr domain.PackageRules
		if err := rows.Scan(&pr.MaxWeight, &pr.Price); err != nil {
			return nil, err
		}
		res = append(res, pr)
	}
	if len(res) == 0 {
		return nil, domain.InvalidPackageError(code)
	}
	return res, nil
}

func nullable(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
