// internal/repository/postgres/outbox.go (обновленная версия)
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
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

// GetPendingMessages возвращает сообщения со статусом CREATED, готовые для обработки
func (r *OutboxRepository) GetPendingMessages(ctx context.Context, limit int) ([]domain.OutboxMessage, error) {
	const query = `
		SELECT id, payload, status, error, attempts, created_at, sent_at, last_attempt_at
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

	return r.scanMessages(rows)
}

// GetProcessingMessages возвращает сообщения со статусом PROCESSING, готовые для отправки
func (r *OutboxRepository) GetProcessingMessages(ctx context.Context, limit int, now time.Time) ([]domain.OutboxMessage, error) {
	const query = `
		SELECT id, payload, status, error, attempts, created_at, sent_at, last_attempt_at
		FROM outbox
		WHERE status = $1 
		  AND (last_attempt_at IS NULL OR last_attempt_at <= $2)
		  AND attempts < $3
		ORDER BY created_at ASC
		LIMIT $4
		FOR UPDATE SKIP LOCKED
	`

	// Вычисляем время, до которого сообщения могут быть обработаны (с учетом задержки retry)
	cutoffTime := now.Add(-domain.RetryDelay)

	rows, err := r.client.Query(ctx, query, domain.OutboxStatusProcessing, cutoffTime, domain.MaxRetryAttempts, limit)
	if err != nil {
		return nil, fmt.Errorf("query processing messages: %w", err)
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

func (r *OutboxRepository) scanMessages(rows *sql.Rows) ([]domain.OutboxMessage, error) {
	var messages []domain.OutboxMessage
	for rows.Next() {
		var msg domain.OutboxMessage
		var errorStr sql.NullString
		var sentAt sql.NullTime
		var lastAttemptAt sql.NullTime

		err := rows.Scan(&msg.ID, &msg.Payload, &msg.Status, &errorStr, &msg.Attempts, &msg.CreatedAt, &sentAt, &lastAttemptAt)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}

		if errorStr.Valid {
			msg.Error = &errorStr.String
		}
		if sentAt.Valid {
			msg.SentAt = &sentAt.Time
		}
		if lastAttemptAt.Valid {
			msg.LastAttemptAt = &lastAttemptAt.Time
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// SetProcessing переводит сообщения в статус PROCESSING
func (r *OutboxRepository) SetProcessing(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	// Преобразуем UUID в строки для pq.Array
	stringIDs := make([]string, len(ids))
	for i, id := range ids {
		stringIDs[i] = id.String()
	}

	query := `UPDATE outbox SET status = $1 WHERE id = ANY($2)`
	_, err := r.client.Exec(ctx, db.ModeWrite, query, domain.OutboxStatusProcessing, pq.Array(stringIDs))
	if err != nil {
		return fmt.Errorf("set processing status: %w", err)
	}

	return nil
}

// UpdateAttempt обновляет счетчик попыток и время последней попытки
func (r *OutboxRepository) UpdateAttempt(ctx context.Context, id uuid.UUID, now time.Time) error {
	const query = `
		UPDATE outbox 
		SET attempts = attempts + 1, last_attempt_at = $2
		WHERE id = $1
	`

	res, err := r.client.Exec(ctx, db.ModeWrite, query, id, now)
	if err != nil {
		return fmt.Errorf("update attempt: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message not found: %s", id)
	}

	return nil
}

// UpdateStatus обновляет статус сообщения
func (r *OutboxRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OutboxStatus, errorMsg *string) error {
	var query string
	var args []interface{}

	switch status {
	case domain.OutboxStatusCompleted:
		query = `UPDATE outbox SET status = $2, sent_at = NOW() WHERE id = $1`
		args = []interface{}{id, status}
	case domain.OutboxStatusFailed:
		query = `UPDATE outbox SET status = $2, error = $3 WHERE id = $1`
		args = []interface{}{id, status, errorMsg}
	default:
		return fmt.Errorf("invalid status for UpdateStatus: %s", status)
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

// FailMessage помечает сообщение как FAILED с указанием причины
func (r *OutboxRepository) FailMessage(ctx context.Context, id uuid.UUID, reason string) error {
	return r.UpdateStatus(ctx, id, domain.OutboxStatusFailed, &reason)
}
