-- +goose Up
ALTER TABLE outbox ADD COLUMN attempts INT DEFAULT 0;
ALTER TABLE outbox ADD COLUMN last_attempt_at TIMESTAMPTZ;

-- Обновим индекс для эффективного поиска задач для повторной обработки
DROP INDEX IF EXISTS idx_outbox_status;
CREATE INDEX idx_outbox_status_retry ON outbox (status, created_at) WHERE status = 'CREATED';
CREATE INDEX idx_outbox_processing_retry ON outbox (status, last_attempt_at) WHERE status = 'PROCESSING';

-- +goose Down
DROP INDEX IF EXISTS idx_outbox_processing_retry;
DROP INDEX IF EXISTS idx_outbox_status_retry;
CREATE INDEX idx_outbox_status ON outbox (status, created_at);

ALTER TABLE outbox DROP COLUMN IF EXISTS last_attempt_at;
ALTER TABLE outbox DROP COLUMN IF EXISTS attempts;