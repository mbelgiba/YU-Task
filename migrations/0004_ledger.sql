-- +goose Up
CREATE TABLE ledger_operations (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id),
    idempotency_key TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('ISSUE','REWARD','SPEND','EXPIRE')),
    reference_id TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, idempotency_key),
    UNIQUE (tenant_id, id)
);

CREATE TABLE ledger_entries (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id),
    operation_id TEXT NOT NULL,
    account_id TEXT NOT NULL,
    amount_ec INTEGER NOT NULL CHECK (amount_ec <> 0),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id, operation_id) REFERENCES ledger_operations(tenant_id, id)
);
CREATE INDEX ledger_entries_tenant_account_idx ON ledger_entries(tenant_id, account_id, created_at);

CREATE TRIGGER ledger_entries_no_update BEFORE UPDATE ON ledger_entries
BEGIN SELECT RAISE(ABORT, 'ledger entries are immutable'); END;
CREATE TRIGGER ledger_entries_no_delete BEFORE DELETE ON ledger_entries
BEGIN SELECT RAISE(ABORT, 'ledger entries are immutable'); END;
CREATE TRIGGER ledger_operations_no_update BEFORE UPDATE ON ledger_operations
BEGIN SELECT RAISE(ABORT, 'ledger operations are immutable'); END;
CREATE TRIGGER ledger_operations_no_delete BEFORE DELETE ON ledger_operations
BEGIN SELECT RAISE(ABORT, 'ledger operations are immutable'); END;
