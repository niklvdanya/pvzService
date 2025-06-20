-- +goose Up
CREATE TABLE receivers (
    id            BIGSERIAL PRIMARY KEY
);

CREATE TABLE package_types (
    code          TEXT PRIMARY KEY,          
    max_weight    NUMERIC(10,2) NOT NULL,
    extra_price   NUMERIC(10,2) NOT NULL
);

CREATE TABLE orders (
    id               BIGINT       PRIMARY KEY,
    receiver_id      BIGINT       NOT NULL REFERENCES receivers(id),
    status           SMALLINT     NOT NULL,
    expires_at       TIMESTAMPTZ  NOT NULL,
    accept_time      TIMESTAMPTZ  NOT NULL,
    last_update_time TIMESTAMPTZ  NOT NULL,
    package_code     TEXT         REFERENCES package_types(code),
    weight           NUMERIC(10,2) NOT NULL,
    price            NUMERIC(10,2) NOT NULL
);

CREATE INDEX idx_orders_receiver_status ON orders (receiver_id, status);

-- +goose Down
DROP INDEX IF EXISTS idx_orders_receiver_status;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS package_types;
DROP TABLE IF EXISTS receivers;