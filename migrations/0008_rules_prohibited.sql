-- +goose Up
CREATE TABLE prohibited_categories (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id),
    org_unit_id TEXT NOT NULL,
    category TEXT NOT NULL,
    reason TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, org_unit_id, category),
    FOREIGN KEY (tenant_id, org_unit_id) REFERENCES org_units(tenant_id, id)
);
CREATE TABLE tenant_rules (
    tenant_id TEXT NOT NULL REFERENCES tenants(id),
    rule_key TEXT NOT NULL,
    rule_value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, rule_key)
);
