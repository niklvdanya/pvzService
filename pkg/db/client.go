package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
)

type ClientMode string

const (
	ModeRead  ClientMode = "read"
	ModeWrite ClientMode = "write"
)

type Client struct {
	readDB  *sql.DB
	writeDB *sql.DB
	logger  *slog.Logger
	txMutex sync.Mutex
}

type Config struct {
	ReadDSN      string
	WriteDSN     string
	MaxOpen      int
	MaxIdle      int
	MaxTxRetries int
}

func NewClient(cfg Config) (*Client, error) {
	logger := slog.Default()

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

	if err := readDB.Ping(); err != nil {
		readDB.Close()
		writeDB.Close()
		return nil, fmt.Errorf("read DB ping failed: %w", err)
	}
	if err := writeDB.Ping(); err != nil {
		readDB.Close()
		writeDB.Close()
		return nil, fmt.Errorf("write DB ping failed: %w", err)
	}

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
			if err != nil {
				err = fmt.Errorf("%v; failed to close write DB: %w", err, closeErr)
			} else {
				err = fmt.Errorf("failed to close write DB: %w", closeErr)
			}
		}
	}
	return err
}

func (c *Client) Exec(ctx context.Context, mode ClientMode, query string, args ...interface{}) (sql.Result, error) {
	db := c.getDB(mode)
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		c.logger.Error("Error executing query", "error", err)
		return nil, fmt.Errorf("exec query: %w", err)
	}
	c.logger.Info("Query executed successfully")
	return result, nil
}

func (c *Client) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := c.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		c.logger.Error("Error executing query", "error", err)
		return nil, fmt.Errorf("query: %w", err)
	}
	c.logger.Info("Query executed successfully")
	return rows, nil
}

func (c *Client) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	row := c.readDB.QueryRowContext(ctx, query, args...)
	return row
}

func (c *Client) BeginTx(ctx context.Context) (*Tx, error) {
	c.txMutex.Lock()
	defer c.txMutex.Unlock()

	tx, err := c.writeDB.BeginTx(ctx, nil)
	if err != nil {
		c.logger.Error("Error starting transaction", "error", err)
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	c.logger.Info("Transaction started")
	return &Tx{tx: tx, logger: c.logger}, nil
}

func (c *Client) getDB(mode ClientMode) *sql.DB {
	if mode == ModeRead {
		return c.readDB
	}
	return c.writeDB
}
