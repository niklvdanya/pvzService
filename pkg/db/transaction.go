package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

type Tx struct {
	tx     *sql.Tx
	logger *slog.Logger
}

func (t *Tx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		t.logger.Error("Error executing query", "error", err)
		return nil, fmt.Errorf("exec query: %w", err)
	}
	t.logger.Info("Query executed successfully")
	return result, nil
}

func (t *Tx) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		t.logger.Error("Error executing query", "error", err)
		return nil, fmt.Errorf("query: %w", err)
	}
	t.logger.Info("Query executed successfully")
	return rows, nil
}

func (t *Tx) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	row := t.tx.QueryRowContext(ctx, query, args...)
	return row
}

func (t *Tx) Commit() error {
	err := t.tx.Commit()
	if err != nil {
		t.logger.Error("Error committing transaction", "error", err)
		return fmt.Errorf("commit transaction: %w", err)
	}
	t.logger.Info("Transaction committed")
	return nil
}

func (t *Tx) Rollback() error {
	err := t.tx.Rollback()
	if err != nil {
		t.logger.Error("Error rolling back transaction", "error", err)
		return fmt.Errorf("rollback transaction: %w", err)
	}
	t.logger.Info("Transaction rolled back")
	return nil
}
