package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
)

type OutboxRepository struct {
	client *db.Client
}

func NewOutboxRepository(client *db.Client) *OutboxRepository {
	return &OutboxRepository{client: client}
}

func (r *OutboxRepository) Save(ctx context.Context, tx *db.Tx, event domain.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	const query = `
        INSERT INTO outbox (id, payload, status, created_at)
        VALUES (gen_random_uuid(), $1, $2, NOW())
    `
	_, err = tx.Exec(ctx, query, payload, domain.OutboxStatusCreated)
	if err != nil {
		return fmt.Errorf("save outbox message: %w", err)
	}
	return nil
}

func (r *OutboxRepository) GetPendingMessages(ctx context.Context, limit int) ([]domain.OutboxMessage, error) {
	const query = `
		SELECT id, payload, status, error, created_at, sent_at
		FROM outbox
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`

	rows, err := r.client.Query(ctx, query, domain.OutboxStatusCreated, limit)
	if err != nil {
		return nil, fmt.Errorf("query pending messages: %w", err)
	}
	defer rows.Close()

	var messages []domain.OutboxMessage
	for rows.Next() {
		var msg domain.OutboxMessage
		var errorStr sql.NullString
		var sentAt sql.NullTime

		err := rows.Scan(&msg.ID, &msg.Payload, &msg.Status, &errorStr, &msg.CreatedAt, &sentAt)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}

		if errorStr.Valid {
			msg.Error = &errorStr.String
		}
		if sentAt.Valid {
			msg.SentAt = &sentAt.Time
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (r *OutboxRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OutboxStatus, errorMsg *string) error {
	var query string
	var args []interface{}

	switch status {
	case domain.OutboxStatusProcessing:
		query = `UPDATE outbox SET status = $2 WHERE id = $1`
		args = []interface{}{id, status}
	case domain.OutboxStatusCompleted:
		query = `UPDATE outbox SET status = $2, sent_at = NOW() WHERE id = $1`
		args = []interface{}{id, status}
	case domain.OutboxStatusFailed:
		query = `UPDATE outbox SET status = $2, error = $3 WHERE id = $1`
		args = []interface{}{id, status, errorMsg}
	default:
		return fmt.Errorf("invalid status: %s", status)
	}

	res, err := r.client.Exec(ctx, db.ModeWrite, query, args...)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message not found: %s", id)
	}

	return nil
}
