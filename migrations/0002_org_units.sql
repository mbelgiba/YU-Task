-- +goose Up
CREATE TABLE org_units (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id),
    parent_id TEXT,
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('BUILDING','FLOOR','DEAN_OFFICE','DEPARTMENT','CLUB')),
    building TEXT,
    floor INTEGER,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, parent_id) REFERENCES org_units(tenant_id, id)
);
CREATE INDEX org_units_tenant_kind_idx ON org_units(tenant_id, kind);
