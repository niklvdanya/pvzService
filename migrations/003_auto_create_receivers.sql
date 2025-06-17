-- +goose Up

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS receivers (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION ensure_receiver_exists()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO receivers(id)
    VALUES (NEW.receiver_id)
    ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_ensure_receiver ON orders;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_ensure_receiver
    BEFORE INSERT OR UPDATE OF receiver_id ON orders
    FOR EACH ROW
    EXECUTE FUNCTION ensure_receiver_exists();
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_ensure_receiver ON orders;
-- +goose StatementEnd

-- +goose StatementBegin
DROP FUNCTION IF EXISTS ensure_receiver_exists();
-- +goose StatementEnd