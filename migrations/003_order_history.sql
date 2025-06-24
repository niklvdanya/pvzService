-- +goose Up
CREATE TABLE order_history (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id        BIGINT NOT NULL REFERENCES orders(id),
    status          SMALLINT NOT NULL,
    changed_at      TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);

CREATE INDEX idx_order_history_order_id ON order_history (order_id);

-- +goose Down
DROP INDEX IF EXISTS idx_order_history_order_id;
DROP TABLE IF EXISTS order_history;