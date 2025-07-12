-- +goose Up
CREATE TYPE outbox_status AS ENUM ('CREATED', 'PROCESSING', 'COMPLETED', 'FAILED');

CREATE TABLE outbox (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    payload JSONB NOT NULL,
    status outbox_status NOT NULL,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_status ON outbox (status, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_outbox_status;
DROP TABLE IF EXISTS outbox;
DROP TYPE IF EXISTS outbox_status;