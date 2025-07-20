package app

import (
	"context"
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
)

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
