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

CREATE TABLE order_history (
    id            BIGSERIAL   PRIMARY KEY,
    order_id      BIGINT      NOT NULL REFERENCES orders(id),
    status        SMALLINT    NOT NULL,
    changed_at    TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS order_history;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS package_types;
DROP TABLE IF EXISTS receivers;