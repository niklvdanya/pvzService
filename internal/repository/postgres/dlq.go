package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
)

type DLQRepository struct {
	client *db.Client
}

func NewDLQRepository(client *db.Client) *DLQRepository {
	return &DLQRepository{client: client}
}

func (r *DLQRepository) Save(ctx context.Context, tx *db.Tx, msg domain.DLQMessage) error {
	const query = `
        INSERT INTO dlq (original_id, payload, error, attempts, failed_at, retry_after, max_retries)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `
	_, err := tx.Exec(ctx, query, msg.OriginalID, msg.Payload, msg.Error,
		msg.Attempts, msg.FailedAt, msg.RetryAfter, msg.MaxRetries)
	if err != nil {
		return fmt.Errorf("save dlq message: %w", err)
	}
	return nil
}

func (r *DLQRepository) GetRetryable(ctx context.Context, limit int) ([]domain.DLQMessage, error) {
	const query = `
        SELECT id, original_id, payload, error, attempts, created_at, failed_at, 
               retry_after, process_count, max_retries
        FROM dlq
        WHERE process_count < max_retries AND retry_after <= NOW()
        ORDER BY retry_after ASC
        LIMIT $1
        FOR UPDATE SKIP LOCKED
    `

	rows, err := r.client.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query retryable dlq messages: %w", err)
	}
	defer rows.Close()

	var messages []domain.DLQMessage
	for rows.Next() {
		var msg domain.DLQMessage
		err := rows.Scan(&msg.ID, &msg.OriginalID, &msg.Payload, &msg.Error,
			&msg.Attempts, &msg.CreatedAt, &msg.FailedAt, &msg.RetryAfter,
			&msg.ProcessCount, &msg.MaxRetries)
		if err != nil {
			return nil, fmt.Errorf("scan dlq message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (r *DLQRepository) UpdateRetry(ctx context.Context, id uuid.UUID, processCount int, retryAfter time.Time) error {
	const query = `UPDATE dlq SET process_count = $2, retry_after = $3 WHERE id = $1`
	_, err := r.client.Exec(ctx, db.ModeWrite, query, id, processCount, retryAfter)
	if err != nil {
		return fmt.Errorf("update dlq retry: %w", err)
	}
	return nil
}

func (r *DLQRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM dlq WHERE id = $1`
	_, err := r.client.Exec(ctx, db.ModeWrite, query, id)
	if err != nil {
		return fmt.Errorf("delete dlq message: %w", err)
	}
	return nil
}
