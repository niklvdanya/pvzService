package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeOrderAccepted          EventType = "order_accepted"
	EventTypeOrderReturnedToCourier EventType = "order_returned_to_courier"
	EventTypeOrderIssued            EventType = "order_issued"
	EventTypeOrderReturnedByClient  EventType = "order_returned_by_client"
)

type ActorType string

const (
	ActorTypeCourier ActorType = "courier"
	ActorTypeClient  ActorType = "client"
	ActorTypeSystem  ActorType = "system"
)

type Event struct {
	EventID   string    `json:"event_id"`
	EventType EventType `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Actor     Actor     `json:"actor"`
	Order     OrderInfo `json:"order"`
	Source    string    `json:"source"`
}

type Actor struct {
	Type ActorType `json:"type"`
	ID   uint64    `json:"id,string"`
}

type OrderInfo struct {
	ID     uint64 `json:"id,string"`
	UserID uint64 `json:"user_id,string"`
	Status string `json:"status"`
}

func NewEvent(eventType EventType, actor Actor, order OrderInfo) Event {
	return Event{
		EventID:   uuid.New().String(),
		EventType: eventType,
		Timestamp: time.Now(),
		Actor:     actor,
		Order:     order,
		Source:    "pvz-api",
	}
}

type OutboxStatus string

const (
	OutboxStatusCreated    OutboxStatus = "CREATED"
	OutboxStatusProcessing OutboxStatus = "PROCESSING"
	OutboxStatusCompleted  OutboxStatus = "COMPLETED"
	OutboxStatusFailed     OutboxStatus = "FAILED"
)

const (
	MaxRetryAttempts    = 3
	RetryDelay          = 2 * time.Second
	NoAttemptsLeftError = "NO_ATTEMPTS_LEFT"
)

type OutboxMessage struct {
	ID            uuid.UUID
	Payload       []byte
	Status        OutboxStatus
	Error         *string
	Attempts      int
	CreatedAt     time.Time
	SentAt        *time.Time
	LastAttemptAt *time.Time
}

func (m *OutboxMessage) CanRetry(now time.Time) bool {
	if m.Attempts >= MaxRetryAttempts {
		return false
	}
	if m.LastAttemptAt == nil {
		return true
	}

	return now.Sub(*m.LastAttemptAt) >= RetryDelay
}

func (m *OutboxMessage) ShouldFail() bool {
	return m.Attempts >= MaxRetryAttempts
}

func (m *OutboxMessage) IncrementAttempts(now time.Time) {
	m.Attempts++
	m.LastAttemptAt = &now
}
