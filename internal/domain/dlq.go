package domain

import (
	"time"

	"github.com/google/uuid"
)

type DLQMessage struct {
	ID           uuid.UUID `json:"id"`
	OriginalID   uuid.UUID `json:"original_id"`
	Payload      []byte    `json:"payload"`
	Error        string    `json:"error"`
	Attempts     int       `json:"attempts"`
	CreatedAt    time.Time `json:"created_at"`
	FailedAt     time.Time `json:"failed_at"`
	RetryAfter   time.Time `json:"retry_after"`
	ProcessCount int       `json:"process_count"`
	MaxRetries   int       `json:"max_retries"`
}

const (
	DLQMaxRetries = 3
	DLQRetryDelay = 30 * time.Minute
)

func (m *DLQMessage) CanRetry() bool {
	return m.ProcessCount < m.MaxRetries && time.Now().After(m.RetryAfter)
}

func (m *DLQMessage) IncrementRetry() {
	m.ProcessCount++
	m.RetryAfter = time.Now().Add(DLQRetryDelay * time.Duration(m.ProcessCount))
}
