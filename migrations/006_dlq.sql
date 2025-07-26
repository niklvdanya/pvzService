-- +goose Up
CREATE TABLE dlq (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_id UUID NOT NULL,
    payload JSONB NOT NULL,
    error TEXT NOT NULL,
    attempts INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    failed_at TIMESTAMPTZ NOT NULL,
    retry_after TIMESTAMPTZ NOT NULL,
    process_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3
);

CREATE INDEX idx_dlq_retry ON dlq (retry_after) WHERE process_count < max_retries;

-- +goose Down
DROP INDEX IF EXISTS idx_dlq_retry;
DROP TABLE IF EXISTS dlq;
