package infra

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
)

type OutboxRepository interface {
	SaveInTx(ctx context.Context, tx *db.Tx, payload []byte) error
}

type EventProducer struct {
	outboxRepo OutboxRepository
}

func NewEventProducer(outboxRepo OutboxRepository) *EventProducer {
	return &EventProducer{
		outboxRepo: outboxRepo,
	}
}

func (p *EventProducer) ProduceInTx(ctx context.Context, tx *db.Tx, event domain.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	if err := p.outboxRepo.SaveInTx(ctx, tx, payload); err != nil {
		return fmt.Errorf("save to outbox: %w", err)
	}

	return nil
}
