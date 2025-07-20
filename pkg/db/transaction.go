package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Tx struct {
	tx     *sql.Tx
	logger *slog.Logger
	tracer trace.Tracer
}

func (t *Tx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, span := t.tracer.Start(ctx, "tx.exec",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", "exec"),
			attribute.String("db.statement", query),
		),
	)
	defer span.End()

	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		t.logger.Error("Error executing query", "error", err)
		return nil, fmt.Errorf("exec query: %w", err)
	}

	if rowsAffected, err := result.RowsAffected(); err == nil {
		span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))
	}

	span.SetStatus(codes.Ok, "")
	t.logger.Info("Query executed successfully")
	return result, nil
}

func (t *Tx) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, span := t.tracer.Start(ctx, "tx.query",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", "query"),
			attribute.String("db.statement", query),
		),
	)
	defer span.End()

	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		t.logger.Error("Error executing query", "error", err)
		return nil, fmt.Errorf("query: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	t.logger.Info("Query executed successfully")
	return rows, nil
}

func (t *Tx) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, span := t.tracer.Start(ctx, "tx.query_row",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", "query_row"),
			attribute.String("db.statement", query),
		),
	)
	defer span.End()

	row := t.tx.QueryRowContext(ctx, query, args...)
	span.SetStatus(codes.Ok, "")
	return row
}

func (t *Tx) Commit() error {
	_, span := t.tracer.Start(context.Background(), "tx.commit",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", "commit"),
		),
	)
	defer span.End()

	err := t.tx.Commit()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		t.logger.Error("Error committing transaction", "error", err)
		return fmt.Errorf("commit transaction: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	t.logger.Info("Transaction committed")
	return nil
}

func (t *Tx) Rollback() error {
	_, span := t.tracer.Start(context.Background(), "tx.rollback",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", "rollback"),
		),
	)
	defer span.End()

	err := t.tx.Rollback()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		t.logger.Error("Error rolling back transaction", "error", err)
		return fmt.Errorf("rollback transaction: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	t.logger.Info("Transaction rolled back")
	return nil
}
