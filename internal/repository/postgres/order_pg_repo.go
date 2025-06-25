package postgres

import (
	"database/sql"
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
