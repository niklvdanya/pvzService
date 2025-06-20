package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

type ClientMode string

const (
	ModeRead  ClientMode = "read"
	ModeWrite ClientMode = "write"
)

type Client struct {
	readDB  *sql.DB
	writeDB *sql.DB
	logger  *log.Logger
}

type Config struct {
	ReadDSN  string
	WriteDSN string
	MaxOpen  int
	MaxIdle  int
	LogFile  string
}

func NewClient(cfg Config) (*Client, error) {
	logFile, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	logger := log.New(logFile, "DB: ", 0)

	readDB, err := sql.Open("postgres", cfg.ReadDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open read DB: %w", err)
	}
	readDB.SetMaxOpenConns(cfg.MaxOpen)
	readDB.SetMaxIdleConns(cfg.MaxIdle)

	writeDB, err := sql.Open("postgres", cfg.WriteDSN)
	if err != nil {
		readDB.Close()
		return nil, fmt.Errorf("failed to open write DB: %w", err)
	}
	writeDB.SetMaxOpenConns(cfg.MaxOpen)
	writeDB.SetMaxIdleConns(cfg.MaxIdle)

	return &Client{
		readDB:  readDB,
		writeDB: writeDB,
		logger:  logger,
	}, nil
}

func (c *Client) Close() error {
	var err error
	if c.readDB != nil {
		if closeErr := c.readDB.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close read DB: %w", closeErr)
		}
	}
	if c.writeDB != nil {
		if closeErr := c.writeDB.Close(); closeErr != nil {
			err = fmt.Errorf("%v; failed to close write DB: %w", err, closeErr)
		}
	}
	return err
}

func (c *Client) Exec(ctx context.Context, mode ClientMode, query string, args ...interface{}) (sql.Result, error) {
	db := c.getDB(mode)
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		c.logger.Printf("Error executing query: %v", err)
		return nil, fmt.Errorf("exec query: %w", err)
	}
	c.logger.Printf("Query executed successfully")
	return result, nil
}

func (c *Client) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := c.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		c.logger.Printf("Error executing query: %v", err)
		return nil, fmt.Errorf("query: %w", err)
	}
	c.logger.Printf("Query executed successfully")
	return rows, nil
}

func (c *Client) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	row := c.readDB.QueryRowContext(ctx, query, args...)
	return row
}

func (c *Client) BeginTx(ctx context.Context) (*Tx, error) {
	tx, err := c.writeDB.BeginTx(ctx, nil)
	if err != nil {
		c.logger.Printf("Error starting transaction: %v", err)
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	c.logger.Printf("Transaction started")
	return &Tx{tx: tx, logger: c.logger}, nil
}

func (c *Client) getDB(mode ClientMode) *sql.DB {
	if mode == ModeRead {
		return c.readDB
	}
	return c.writeDB
}

type Tx struct {
	tx     *sql.Tx
	logger *log.Logger
}

func (t *Tx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		t.logger.Printf("Error executing query: %v", err)
		return nil, fmt.Errorf("exec query: %w", err)
	}
	t.logger.Printf("Query executed successfully")
	return result, nil
}

func (t *Tx) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		t.logger.Printf("Error executing query: %v", err)
		return nil, fmt.Errorf("query: %w", err)
	}
	t.logger.Printf("Query executed successfully")
	return rows, nil
}

func (t *Tx) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	row := t.tx.QueryRowContext(ctx, query, args...)
	return row
}

func (t *Tx) Commit() error {
	err := t.tx.Commit()
	if err != nil {
		t.logger.Printf("Error committing transaction: %v", err)
		return fmt.Errorf("commit transaction: %w", err)
	}
	t.logger.Printf("Transaction committed")
	return nil
}

func (t *Tx) Rollback() error {
	err := t.tx.Rollback()
	if err != nil {
		t.logger.Printf("Error rolling back transaction: %v", err)
		return fmt.Errorf("rollback transaction: %w", err)
	}
	t.logger.Printf("Transaction rolled back")
	return nil
}
