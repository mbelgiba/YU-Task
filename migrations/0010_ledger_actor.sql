-- +goose Up
ALTER TABLE ledger_operations ADD COLUMN actor_id TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE ledger_operations DROP COLUMN actor_id;
